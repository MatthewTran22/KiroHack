package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEmbeddingServiceIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if required environment variables are set
	geminiAPIKey := os.Getenv("LLM_API_KEY")
	if geminiAPIKey == "" {
		t.Skip("LLM_API_KEY not set, skipping integration test")
	}

	t.Logf("Using LLM_API_KEY: %s...", geminiAPIKey[:10]) // Log first 10 chars for verification

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	redisAddr := os.Getenv("REDIS_HOST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Setup MongoDB connection
	mongoConfig := &database.Config{
		URI:          mongoURI,
		DatabaseName: "ai_government_consultant_test",
	}

	mongodb, err := database.NewMongoDB(mongoConfig)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Setup Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   1, // Use different DB for tests
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

	// Create repository
	repo := embedding.NewRepository(mongodb.Database)

	t.Run("GenerateEmbedding", func(t *testing.T) {
		text := "This is a test document about government policy and strategic planning."

		embeddings, err := service.GenerateEmbedding(ctx, text)
		if err != nil {
			t.Fatalf("Failed to generate embedding: %v", err)
		}

		if len(embeddings) == 0 {
			t.Error("Expected non-empty embeddings")
		}

		// Embeddings should be normalized (values between -1 and 1)
		for i, val := range embeddings {
			if val < -1.0 || val > 1.0 {
				t.Errorf("Embedding value %d out of range [-1, 1]: %f", i, val)
			}
		}

		t.Logf("Generated embedding with %d dimensions", len(embeddings))
	})

	t.Run("GenerateDocumentEmbedding", func(t *testing.T) {
		// Create a test document
		document := &models.Document{
			ID:               primitive.NewObjectID(),
			Name:             "test-policy.pdf",
			Content:          "This document outlines the government's new digital transformation policy. It includes guidelines for technology adoption, security requirements, and implementation timelines.",
			ContentType:      "application/pdf",
			Size:             1024,
			UploadedBy:       primitive.NewObjectID(),
			UploadedAt:       time.Now(),
			ProcessingStatus: models.ProcessingStatusCompleted,
			Classification: models.SecurityClassification{
				Level: "PUBLIC",
			},
			Metadata: models.DocumentMetadata{
				Category: models.DocumentCategoryPolicy,
				Tags:     []string{"digital", "transformation", "policy"},
				Language: "en",
			},
		}

		// Insert document into database
		collection := mongodb.Database.Collection("documents")
		_, err := collection.InsertOne(ctx, document)
		if err != nil {
			t.Fatalf("Failed to insert test document: %v", err)
		}
		defer collection.DeleteOne(ctx, map[string]interface{}{"_id": document.ID})

		// Generate embedding for document
		err = service.GenerateDocumentEmbedding(ctx, document.ID)
		if err != nil {
			t.Fatalf("Failed to generate document embedding: %v", err)
		}

		// Verify document was updated with embeddings
		var updatedDoc models.Document
		err = collection.FindOne(ctx, map[string]interface{}{"_id": document.ID}).Decode(&updatedDoc)
		if err != nil {
			t.Fatalf("Failed to retrieve updated document: %v", err)
		}

		if len(updatedDoc.Embeddings) == 0 {
			t.Error("Document should have embeddings after processing")
		}

		if updatedDoc.ProcessingTimestamp == nil {
			t.Error("Document should have processing timestamp")
		}

		t.Logf("Document embedding generated with %d dimensions", len(updatedDoc.Embeddings))
	})

	t.Run("GenerateKnowledgeEmbedding", func(t *testing.T) {
		// Create a test knowledge item
		knowledge := &models.KnowledgeItem{
			ID:       primitive.NewObjectID(),
			Title:    "Digital Transformation Best Practices",
			Content:  "Government agencies should follow these best practices when implementing digital transformation initiatives: 1) Conduct thorough security assessments, 2) Ensure compliance with data protection regulations, 3) Provide adequate training for staff.",
			Type:     models.KnowledgeTypeBestPractice,
			Category: "technology",
			Tags:     []string{"digital", "transformation", "best-practices"},
			Source: models.KnowledgeSource{
				Type:        "manual",
				Reference:   "IT Guidelines v2.1",
				Reliability: 0.9,
			},
			Confidence: 0.85,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			CreatedBy:  primitive.NewObjectID(),
			Version:    1,
			IsActive:   true,
		}

		// Insert knowledge item into database
		collection := mongodb.Database.Collection("knowledge_items")
		_, err := collection.InsertOne(ctx, knowledge)
		if err != nil {
			t.Fatalf("Failed to insert test knowledge item: %v", err)
		}
		defer collection.DeleteOne(ctx, map[string]interface{}{"_id": knowledge.ID})

		// Generate embedding for knowledge item
		err = service.GenerateKnowledgeEmbedding(ctx, knowledge.ID)
		if err != nil {
			t.Fatalf("Failed to generate knowledge embedding: %v", err)
		}

		// Verify knowledge item was updated with embeddings
		var updatedKnowledge models.KnowledgeItem
		err = collection.FindOne(ctx, map[string]interface{}{"_id": knowledge.ID}).Decode(&updatedKnowledge)
		if err != nil {
			t.Fatalf("Failed to retrieve updated knowledge item: %v", err)
		}

		if len(updatedKnowledge.Embeddings) == 0 {
			t.Error("Knowledge item should have embeddings after processing")
		}

		t.Logf("Knowledge embedding generated with %d dimensions", len(updatedKnowledge.Embeddings))
	})

	t.Run("VectorSearch", func(t *testing.T) {
		// Create test documents with embeddings
		documents := []*models.Document{
			{
				ID:               primitive.NewObjectID(),
				Name:             "policy1.pdf",
				Content:          "Government digital transformation policy focusing on cloud adoption and cybersecurity measures.",
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
				Metadata: models.DocumentMetadata{
					Category: models.DocumentCategoryPolicy,
					Tags:     []string{"digital", "cloud", "security"},
				},
			},
			{
				ID:               primitive.NewObjectID(),
				Name:             "strategy1.pdf",
				Content:          "Strategic planning document for government modernization and citizen service improvement.",
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
				Metadata: models.DocumentMetadata{
					Category: models.DocumentCategoryStrategy,
					Tags:     []string{"modernization", "citizen-services"},
				},
			},
		}

		collection := mongodb.Database.Collection("documents")

		// Generate embeddings for test documents
		for _, doc := range documents {
			_, err := collection.InsertOne(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to insert test document: %v", err)
			}
			defer collection.DeleteOne(ctx, map[string]interface{}{"_id": doc.ID})

			err = service.GenerateDocumentEmbedding(ctx, doc.ID)
			if err != nil {
				t.Fatalf("Failed to generate embedding for test document: %v", err)
			}
		}

		// Perform vector search
		searchOptions := &embedding.SearchOptions{
			Limit:      5,
			Threshold:  0.3, // Lower threshold for testing
			Collection: "documents",
		}

		results, err := service.VectorSearch(ctx, "digital transformation policy", searchOptions)
		if err != nil {
			t.Fatalf("Vector search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected search results but got none")
		}

		// Verify results are sorted by score
		for i := 1; i < len(results); i++ {
			if results[i-1].Score < results[i].Score {
				t.Error("Results should be sorted by score in descending order")
			}
		}

		// The first document should be more relevant to "digital transformation policy"
		if len(results) > 0 {
			t.Logf("Top result: %s (score: %.3f)", results[0].Document.Name, results[0].Score)
			if results[0].Score < 0.3 {
				t.Errorf("Top result score too low: %.3f", results[0].Score)
			}
		}
	})

	t.Run("BatchGenerateEmbeddings", func(t *testing.T) {
		texts := []string{
			"Government policy on data protection and privacy",
			"Strategic planning for digital infrastructure",
			"Operational guidelines for remote work",
			"Technology adoption framework for agencies",
		}

		embeddings, err := service.BatchGenerateEmbeddings(ctx, texts)
		if err != nil {
			t.Fatalf("Batch embedding generation failed: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		// Verify each embedding has the same dimension
		expectedDim := len(embeddings[0])
		for i, embedding := range embeddings {
			if len(embedding) != expectedDim {
				t.Errorf("Embedding %d has dimension %d, expected %d", i, len(embedding), expectedDim)
			}
		}

		t.Logf("Generated %d embeddings with %d dimensions each", len(embeddings), expectedDim)
	})

	t.Run("EmbeddingPipeline", func(t *testing.T) {
		// Create pipeline
		pipelineConfig := &embedding.PipelineConfig{
			BatchSize:     2,
			MaxWorkers:    2,
			RetryAttempts: 1,
			RetryDelay:    time.Second,
		}

		pipeline := embedding.NewPipeline(service, repo, logger.NewTestLogger(), pipelineConfig)

		// Create test documents without embeddings
		testDocs := []*models.Document{
			{
				ID:               primitive.NewObjectID(),
				Name:             "pipeline-test1.pdf",
				Content:          "Test document for pipeline processing - policy analysis",
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
			},
			{
				ID:               primitive.NewObjectID(),
				Name:             "pipeline-test2.pdf",
				Content:          "Test document for pipeline processing - strategic planning",
				ProcessingStatus: models.ProcessingStatusCompleted,
				Classification:   models.SecurityClassification{Level: "PUBLIC"},
			},
		}

		collection := mongodb.Database.Collection("documents")
		var docIDs []primitive.ObjectID

		for _, doc := range testDocs {
			_, err := collection.InsertOne(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to insert test document: %v", err)
			}
			docIDs = append(docIDs, doc.ID)
			defer collection.DeleteOne(ctx, map[string]interface{}{"_id": doc.ID})
		}

		// Process specific documents
		result, err := pipeline.ProcessSpecificDocuments(ctx, docIDs)
		if err != nil {
			t.Fatalf("Pipeline processing failed: %v", err)
		}

		if result.TotalProcessed != len(docIDs) {
			t.Errorf("Expected %d processed, got %d", len(docIDs), result.TotalProcessed)
		}

		if result.Successful != len(docIDs) {
			t.Errorf("Expected %d successful, got %d", len(docIDs), result.Successful)
		}

		if result.Failed != 0 {
			t.Errorf("Expected 0 failed, got %d", result.Failed)
		}

		t.Logf("Pipeline processed %d documents in %v", result.TotalProcessed, result.Duration)

		// Verify documents now have embeddings
		for _, docID := range docIDs {
			var doc models.Document
			err := collection.FindOne(ctx, map[string]interface{}{"_id": docID}).Decode(&doc)
			if err != nil {
				t.Fatalf("Failed to retrieve processed document: %v", err)
			}

			if len(doc.Embeddings) == 0 {
				t.Errorf("Document %s should have embeddings after pipeline processing", doc.Name)
			}
		}
	})

	t.Run("EmbeddingStats", func(t *testing.T) {
		stats, err := repo.GetEmbeddingStats(ctx)
		if err != nil {
			t.Fatalf("Failed to get embedding stats: %v", err)
		}

		t.Logf("Embedding Stats:")
		t.Logf("  Documents with embeddings: %d/%d", stats.DocumentsWithEmbeddings, stats.TotalDocuments)
		t.Logf("  Knowledge items with embeddings: %d/%d", stats.KnowledgeWithEmbeddings, stats.TotalKnowledgeItems)

		if stats.TotalDocuments < 0 {
			t.Error("Total documents should not be negative")
		}
		if stats.DocumentsWithEmbeddings > stats.TotalDocuments {
			t.Error("Documents with embeddings should not exceed total documents")
		}
	})

	// Clean up Redis cache if available
	if redisClient != nil {
		t.Run("ClearCache", func(t *testing.T) {
			err := service.ClearCache(ctx)
			if err != nil {
				t.Errorf("Failed to clear cache: %v", err)
			}
		})
	}
}

func TestEmbeddingServiceWithoutExternalDependencies(t *testing.T) {
	// This test runs even without external services
	// It tests the service creation and basic validation

	t.Run("ServiceCreationWithoutAPIKey", func(t *testing.T) {
		config := &embedding.Config{
			Logger: logger.NewTestLogger(),
		}

		_, err := embedding.NewService(config)
		if err == nil {
			t.Error("Expected error when creating service without API key")
		}
	})

	t.Run("ServiceCreationWithAPIKey", func(t *testing.T) {
		config := &embedding.Config{
			GeminiAPIKey: "test-api-key",
			Logger:       logger.NewTestLogger(),
		}

		service, err := embedding.NewService(config)
		if err != nil {
			t.Errorf("Unexpected error creating service: %v", err)
		}

		if service == nil {
			t.Error("Expected service but got nil")
		}
	})

	t.Run("DefaultPipelineConfig", func(t *testing.T) {
		config := embedding.DefaultPipelineConfig()

		if config.BatchSize <= 0 {
			t.Error("Default batch size should be positive")
		}
		if config.MaxWorkers <= 0 {
			t.Error("Default max workers should be positive")
		}
		if config.RetryAttempts < 0 {
			t.Error("Default retry attempts should not be negative")
		}
		if config.RetryDelay <= 0 {
			t.Error("Default retry delay should be positive")
		}
	})
}
