package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ai-government-consultant/internal/api"
	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEmbeddingAPIIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if required environment variables are set
	geminiAPIKey := os.Getenv("LLM_API_KEY")
	if geminiAPIKey == "" {
		t.Skip("LLM_API_KEY not set, skipping integration test")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisAddr := redisHost + ":" + redisPort

	t.Logf("Testing with LLM_API_KEY: %s...", geminiAPIKey[:10])
	t.Logf("MongoDB URI: %s", mongoURI)
	t.Logf("Redis Address: %s", redisAddr)

	// Setup MongoDB connection
	mongoConfig := &database.Config{
		URI:          mongoURI,
		DatabaseName: "ai_government_consultant_api_test",
	}

	mongodb, err := database.NewMongoDB(mongoConfig)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Setup Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   2, // Use different DB for API tests
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Logf("Redis not available, continuing without cache: %v", err)
		redisClient = nil
	}

	// Create embedding service
	embeddingConfig := &embedding.Config{
		GeminiAPIKey: geminiAPIKey,
		MongoDB:      mongodb.Database,
		Redis:        redisClient,
		Logger:       logger.NewTestLogger(),
	}

	service, err := embedding.NewService(embeddingConfig)
	if err != nil {
		t.Fatalf("Failed to create embedding service: %v", err)
	}

	// Create repository and pipeline
	repo := embedding.NewRepository(mongodb.Database)
	pipelineConfig := &embedding.PipelineConfig{
		BatchSize:     5,
		MaxWorkers:    2,
		RetryAttempts: 1,
		RetryDelay:    time.Second,
	}
	pipeline := embedding.NewPipeline(service, repo, logger.NewTestLogger(), pipelineConfig)

	// Create API handler
	handler := api.NewEmbeddingHandler(service, repo, pipeline)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api/v1")
	handler.RegisterRoutes(apiGroup)

	// Test 1: Generate Embedding API
	t.Run("POST /api/v1/embeddings/generate", func(t *testing.T) {
		requestBody := api.GenerateEmbeddingRequest{
			Text: "This is a test document about government policy and digital transformation.",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/embeddings/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response api.GenerateEmbeddingResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		if response.Dimensions == 0 {
			t.Error("Expected non-zero dimensions")
		}
		if len(response.Embeddings) == 0 {
			t.Error("Expected non-empty embeddings")
		}
		if response.Text != requestBody.Text {
			t.Errorf("Expected text %s, got %s", requestBody.Text, response.Text)
		}

		t.Logf("Generated embedding with %d dimensions", response.Dimensions)
	})

	// Test 2: Create test documents for search
	var testDocumentIDs []primitive.ObjectID
	t.Run("Setup test documents", func(t *testing.T) {
		documents := []*models.Document{
			{
				ID:               primitive.NewObjectID(),
				Name:             "digital-policy.pdf",
				Content:          "This document outlines the government's comprehensive digital transformation policy. It includes guidelines for technology adoption, cybersecurity requirements, and implementation timelines for all government agencies.",
				ContentType:      "application/pdf",
				Size:             2048,
				UploadedBy:       primitive.NewObjectID(),
				UploadedAt:       time.Now(),
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
				Metadata: models.DocumentMetadata{
					Category: models.DocumentCategoryPolicy,
					Tags:     []string{"digital", "transformation", "policy"},
					Language: "en",
				},
			},
			{
				ID:               primitive.NewObjectID(),
				Name:             "cloud-strategy.pdf",
				Content:          "Strategic planning document for government cloud adoption. This document provides a roadmap for migrating government services to cloud infrastructure while maintaining security and compliance standards.",
				ContentType:      "application/pdf",
				Size:             1536,
				UploadedBy:       primitive.NewObjectID(),
				UploadedAt:       time.Now(),
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
				Metadata: models.DocumentMetadata{
					Category: models.DocumentCategoryStrategy,
					Tags:     []string{"cloud", "strategy", "infrastructure"},
					Language: "en",
				},
			},
		}

		collection := mongodb.Database.Collection("documents")
		for _, doc := range documents {
			_, err := collection.InsertOne(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to insert test document: %v", err)
			}
			testDocumentIDs = append(testDocumentIDs, doc.ID)

			// Clean up after test
			defer collection.DeleteOne(ctx, map[string]interface{}{"_id": doc.ID})
		}

		t.Logf("Created %d test documents", len(testDocumentIDs))
	})

	// Test 3: Process Embeddings API
	t.Run("POST /api/v1/embeddings/process", func(t *testing.T) {
		// Convert ObjectIDs to strings
		var documentIDStrings []string
		for _, id := range testDocumentIDs {
			documentIDStrings = append(documentIDStrings, id.Hex())
		}

		requestBody := api.ProcessEmbeddingsRequest{
			DocumentIDs: documentIDStrings,
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/embeddings/process", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response api.ProcessEmbeddingsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		if response.TotalProcessed != len(testDocumentIDs) {
			t.Errorf("Expected %d processed, got %d", len(testDocumentIDs), response.TotalProcessed)
		}
		if response.Successful != len(testDocumentIDs) {
			t.Errorf("Expected %d successful, got %d", len(testDocumentIDs), response.Successful)
		}

		t.Logf("Processed %d documents successfully in %s", response.Successful, response.Duration)
	})

	// Test 4: Vector Search API
	t.Run("POST /api/v1/embeddings/search", func(t *testing.T) {
		requestBody := api.VectorSearchRequest{
			Query:     "digital transformation policy",
			Limit:     5,
			Threshold: 0.3, // Lower threshold for testing
			Filters: map[string]interface{}{
				"processing_status": "completed",
			},
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/embeddings/search", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response api.VectorSearchResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		if response.Query != requestBody.Query {
			t.Errorf("Expected query %s, got %s", requestBody.Query, response.Query)
		}

		if len(response.Results) == 0 {
			t.Error("Expected search results but got none")
		} else {
			// Verify results are sorted by score
			for i := 1; i < len(response.Results); i++ {
				if response.Results[i-1].Score < response.Results[i].Score {
					t.Error("Results should be sorted by score in descending order")
				}
			}

			t.Logf("Found %d search results, top score: %.3f", len(response.Results), response.Results[0].Score)
		}
	})

	// Test 5: Get Embedding Stats API
	t.Run("GET /api/v1/embeddings/stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/embeddings/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var stats embedding.EmbeddingStats
		err := json.Unmarshal(w.Body.Bytes(), &stats)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		if stats.TotalDocuments < 0 {
			t.Error("Total documents should not be negative")
		}
		if stats.DocumentsWithEmbeddings > stats.TotalDocuments {
			t.Error("Documents with embeddings should not exceed total documents")
		}

		t.Logf("Stats: %d/%d documents have embeddings", stats.DocumentsWithEmbeddings, stats.TotalDocuments)
	})

	// Test 6: Generate Document Embedding API
	t.Run("POST /api/v1/embeddings/documents/{id}/embedding", func(t *testing.T) {
		if len(testDocumentIDs) == 0 {
			t.Skip("No test documents available")
		}

		documentID := testDocumentIDs[0]
		url := fmt.Sprintf("/api/v1/embeddings/documents/%s/embedding", documentID.Hex())

		req := httptest.NewRequest("POST", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response api.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		if response.Message == "" {
			t.Error("Expected success message")
		}

		t.Logf("Generated embedding for document %s", documentID.Hex())
	})

	// Test 7: Get Similar Documents API
	t.Run("GET /api/v1/embeddings/documents/{id}/similar", func(t *testing.T) {
		if len(testDocumentIDs) == 0 {
			t.Skip("No test documents available")
		}

		documentID := testDocumentIDs[0]
		url := fmt.Sprintf("/api/v1/embeddings/documents/%s/similar?limit=3", documentID.Hex())

		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response api.VectorSearchResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			return
		}

		// Should find similar documents (excluding the source document)
		t.Logf("Found %d similar documents", response.Count)
	})

	// Test 8: Clear Cache API
	if redisClient != nil {
		t.Run("DELETE /api/v1/embeddings/cache", func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/v1/embeddings/cache", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
				return
			}

			var response api.SuccessResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			if response.Message == "" {
				t.Error("Expected success message")
			}

			t.Log("Successfully cleared embedding cache")
		})
	}

	// Test 9: Error Handling
	t.Run("Error handling tests", func(t *testing.T) {
		// Test invalid JSON
		req := httptest.NewRequest("POST", "/api/v1/embeddings/generate", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}

		// Test empty text
		requestBody := api.GenerateEmbeddingRequest{Text: ""}
		body, _ := json.Marshal(requestBody)
		req = httptest.NewRequest("POST", "/api/v1/embeddings/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty text, got %d", w.Code)
		}

		// Test invalid document ID
		req = httptest.NewRequest("POST", "/api/v1/embeddings/documents/invalid-id/embedding", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid document ID, got %d", w.Code)
		}

		t.Log("Error handling tests passed")
	})

	// Clean up Redis cache if available
	if redisClient != nil {
		redisClient.FlushDB(ctx)
	}
}

func TestEmbeddingAPIWithoutExternalDependencies(t *testing.T) {
	// Test API structure without external dependencies

	t.Run("API Handler Creation", func(t *testing.T) {
		handler := api.NewEmbeddingHandler(nil, nil, nil)
		if handler == nil {
			t.Error("Expected handler but got nil")
		}
	})

	t.Run("Request/Response Types", func(t *testing.T) {
		// Test GenerateEmbeddingRequest
		req := api.GenerateEmbeddingRequest{
			Text: "test text",
		}
		if req.Text != "test text" {
			t.Errorf("Expected 'test text', got %s", req.Text)
		}

		// Test VectorSearchRequest
		searchReq := api.VectorSearchRequest{
			Query:     "test query",
			Limit:     10,
			Threshold: 0.7,
			Filters:   map[string]interface{}{"category": "test"},
		}
		if searchReq.Query != "test query" {
			t.Errorf("Expected 'test query', got %s", searchReq.Query)
		}
		if searchReq.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", searchReq.Limit)
		}

		// Test ProcessEmbeddingsRequest
		processReq := api.ProcessEmbeddingsRequest{
			DocumentIDs: []string{"id1", "id2"},
			ProcessAll:  false,
		}
		if len(processReq.DocumentIDs) != 2 {
			t.Errorf("Expected 2 document IDs, got %d", len(processReq.DocumentIDs))
		}
	})
}
