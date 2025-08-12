package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-government-consultant/internal/api"
	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDocumentProcessingIntegration tests the full document processing workflow
// This test interacts with the actual Docker container setup
func TestDocumentProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup database connection to Docker container
	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/test_integration?authSource=admin",
		DatabaseName:   "test_integration",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping integration test - MongoDB Docker container not available: %v", err)
	}

	// Clean up test data
	ctx := context.Background()
	defer func() {
		mongodb.Database.Drop(ctx)
		mongodb.Close(ctx)
	}()

	// Initialize database
	err = mongodb.CreateIndexes(ctx)
	if err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}

	// Create services
	documentService := document.NewService(mongodb.Database)
	documentRepo := document.NewRepository(mongodb.Database)

	// Create API handler
	handler := api.NewDocumentHandler(documentService)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock user authentication middleware
	testUserID := primitive.NewObjectID()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})

	apiGroup := router.Group("/api")
	handler.RegisterRoutes(apiGroup)

	// Test document content
	testContent := `
	Government Cybersecurity Policy Document
	
	This document outlines the comprehensive cybersecurity policy for government agencies.
	
	Contact Information:
	- Email: security@government.gov
	- Phone: 555-123-4567
	- Budget: $2,500,000
	- Effective Date: 01/01/2024
	
	The policy includes regulations for data protection, compliance requirements,
	and security procedures for all government personnel and contractors.
	
	Key areas covered:
	- Risk assessment and management
	- Incident response procedures
	- Access control and authentication
	- Data encryption and protection
	- Regular security audits and reviews
	`

	t.Run("Full Document Processing Workflow", func(t *testing.T) {
		// Step 1: Upload document
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file
		part, err := writer.CreateFormFile("file", "cybersecurity_policy.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte(testContent))

		// Add metadata
		writer.WriteField("title", "Government Cybersecurity Policy")
		writer.WriteField("department", "IT Security")
		writer.WriteField("category", "policy")
		writer.WriteField("tags", "cybersecurity,policy,government,security")
		writer.WriteField("language", "en")

		writer.Close()

		req, err := http.NewRequest("POST", "/api/documents/upload", body)
		if err != nil {
			t.Fatalf("Failed to create upload request: %v", err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Upload failed with status %d: %s", w.Code, w.Body.String())
		}

		var uploadResult document.ProcessingResult
		err = json.Unmarshal(w.Body.Bytes(), &uploadResult)
		if err != nil {
			t.Fatalf("Failed to unmarshal upload response: %v", err)
		}

		if uploadResult.Status != "uploaded" {
			t.Fatalf("Upload failed: %s", uploadResult.Message)
		}

		documentID := uploadResult.DocumentID.Hex()
		t.Logf("Document uploaded successfully with ID: %s", documentID)

		// Step 2: Check initial processing status
		req, err = http.NewRequest("GET", "/api/documents/"+documentID+"/status", nil)
		if err != nil {
			t.Fatalf("Failed to create status request: %v", err)
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status check failed with status %d: %s", w.Code, w.Body.String())
		}

		var statusResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal status response: %v", err)
		}

		t.Logf("Initial processing status: %v", statusResponse["processing_status"])

		// Step 3: Process document synchronously
		req, err = http.NewRequest("POST", "/api/documents/"+documentID+"/process", nil)
		if err != nil {
			t.Fatalf("Failed to create process request: %v", err)
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Processing failed with status %d: %s", w.Code, w.Body.String())
		}

		var processedDoc models.Document
		err = json.Unmarshal(w.Body.Bytes(), &processedDoc)
		if err != nil {
			t.Fatalf("Failed to unmarshal processed document: %v", err)
		}

		t.Logf("Document processed with status: %s", processedDoc.ProcessingStatus)

		// Step 4: Verify processing results
		// The document should be processed or at least in processing state
		validStatuses := []models.ProcessingStatus{
			models.ProcessingStatusCompleted,
			models.ProcessingStatusProcessing,
		}

		statusValid := false
		for _, validStatus := range validStatuses {
			if processedDoc.ProcessingStatus == validStatus {
				statusValid = true
				break
			}
		}

		if !statusValid {
			t.Errorf("Expected processing status to be completed or processing, got %s", processedDoc.ProcessingStatus)
		}

		// Check metadata extraction
		if processedDoc.Metadata.Category != models.DocumentCategoryPolicy {
			t.Errorf("Expected category policy, got %s", processedDoc.Metadata.Category)
		}

		if processedDoc.Metadata.Title == nil || *processedDoc.Metadata.Title == "" {
			t.Errorf("Expected title to be extracted")
		} else {
			t.Logf("Extracted title: %s", *processedDoc.Metadata.Title)
		}

		// Check entity extraction
		if len(processedDoc.ExtractedEntities) == 0 {
			t.Errorf("Expected entities to be extracted")
		} else {
			t.Logf("Extracted %d entities", len(processedDoc.ExtractedEntities))
			for _, entity := range processedDoc.ExtractedEntities {
				t.Logf("  - %s: %s (confidence: %.2f)", entity.Type, entity.Value, entity.Confidence)
			}
		}

		// Verify specific entities
		entityTypes := make(map[string]int)
		for _, entity := range processedDoc.ExtractedEntities {
			entityTypes[entity.Type]++
		}

		expectedEntityTypes := []string{"email", "phone", "money"}
		for _, expectedType := range expectedEntityTypes {
			if count, exists := entityTypes[expectedType]; !exists || count == 0 {
				t.Errorf("Expected to find %s entities, but found %d", expectedType, count)
			}
		}

		// Step 5: Test search functionality
		req, err = http.NewRequest("GET", "/api/documents/search?query=cybersecurity", nil)
		if err != nil {
			t.Fatalf("Failed to create search request: %v", err)
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Search failed with status %d: %s", w.Code, w.Body.String())
		}

		var searchResponse api.SearchResponse
		err = json.Unmarshal(w.Body.Bytes(), &searchResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal search response: %v", err)
		}

		if searchResponse.Total == 0 {
			t.Errorf("Expected search to find documents, but got 0 results")
		} else {
			t.Logf("Search found %d documents", searchResponse.Total)
		}

		// Step 6: Test repository statistics
		stats, err := documentRepo.GetStatistics(ctx)
		if err != nil {
			t.Fatalf("Failed to get statistics: %v", err)
		}

		t.Logf("Repository statistics: %+v", stats)

		if total, ok := stats["total"].(int64); !ok || total == 0 {
			t.Errorf("Expected at least 1 document in statistics, got %v", stats["total"])
		}

		// Step 7: Test document retrieval
		retrievedDoc, err := documentRepo.GetByID(ctx, uploadResult.DocumentID)
		if err != nil {
			t.Fatalf("Failed to retrieve document: %v", err)
		}

		if retrievedDoc.ID != uploadResult.DocumentID {
			t.Errorf("Retrieved document ID mismatch: expected %s, got %s",
				uploadResult.DocumentID.Hex(), retrievedDoc.ID.Hex())
		}

		// The retrieved document should be processed (completed or processing)
		if retrievedDoc.ProcessingStatus != models.ProcessingStatusCompleted &&
			retrievedDoc.ProcessingStatus != models.ProcessingStatusProcessing {
			t.Errorf("Retrieved document should be completed or processing, got %s", retrievedDoc.ProcessingStatus)
		}

		t.Logf("Integration test completed successfully!")
	})

	t.Run("Document Validation", func(t *testing.T) {
		// Test validation endpoint
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte("Test content for validation"))
		writer.Close()

		req, err := http.NewRequest("POST", "/api/documents/validate", body)
		if err != nil {
			t.Fatalf("Failed to create validation request: %v", err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Validation failed with status %d: %s", w.Code, w.Body.String())
		}

		var validationResult document.ValidationResult
		err = json.Unmarshal(w.Body.Bytes(), &validationResult)
		if err != nil {
			t.Fatalf("Failed to unmarshal validation response: %v", err)
		}

		if !validationResult.Valid {
			t.Errorf("Expected document to be valid, but got: %v", validationResult.Errors)
		}

		t.Logf("Document validation passed: format=%s, size=%d",
			validationResult.Format, validationResult.Size)
	})

	t.Run("Error Handling", func(t *testing.T) {
		// Test invalid document ID
		req, err := http.NewRequest("GET", "/api/documents/invalid-id", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 for invalid ID, got %d", w.Code)
		}

		// Test non-existent document
		nonExistentID := primitive.NewObjectID().Hex()
		req, err = http.NewRequest("GET", "/api/documents/"+nonExistentID, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent document, got %d", w.Code)
		}

		t.Logf("Error handling tests passed")
	})
}
