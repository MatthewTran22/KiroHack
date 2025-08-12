package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func setupAPITestDB(t *testing.T) *mongo.Database {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}

	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/test_ai_government_consultant?authSource=admin",
		DatabaseName:   "test_ai_government_consultant_api",
		ConnectTimeout: 5 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping API tests - MongoDB not available: %v", err)
	}

	// Clean up any existing test data
	ctx := context.Background()
	mongodb.Database.Collection("documents").Drop(ctx)

	t.Cleanup(func() {
		mongodb.Database.Drop(ctx)
		mongodb.Close(ctx)
	})

	return mongodb.Database
}

func createTestRouter(handler *DocumentHandler) (*gin.Engine, primitive.ObjectID) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a consistent user ID for all requests
	testUserID := primitive.NewObjectID()

	// Add middleware to set user ID
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})

	api := router.Group("/api")
	handler.RegisterRoutes(api)

	return router, testUserID
}

func createMultipartRequest(method, url string, fieldName, fileName, content string, extraFields map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, err
	}
	part.Write([]byte(content))

	// Add extra fields
	for key, value := range extraFields {
		writer.WriteField(key, value)
	}

	writer.Close()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestDocumentHandler_ValidateDocument(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	tests := []struct {
		name           string
		fileName       string
		content        string
		expectedStatus int
		expectValid    bool
	}{
		{
			name:           "valid document",
			fileName:       "test.txt",
			content:        "This is a test document",
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "invalid format",
			fileName:       "test.xyz",
			content:        "Invalid format",
			expectedStatus: http.StatusOK,
			expectValid:    false,
		},
		{
			name:           "empty file",
			fileName:       "empty.txt",
			content:        "",
			expectedStatus: http.StatusOK,
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := createMultipartRequest("POST", "/api/documents/validate", "file", tt.fileName, tt.content, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result document.ValidationResult
				err := json.Unmarshal(w.Body.Bytes(), &result)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if result.Valid != tt.expectValid {
					t.Errorf("Expected valid=%v, got valid=%v", tt.expectValid, result.Valid)
				}
			}
		})
	}
}

func TestDocumentHandler_UploadDocument(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	tests := []struct {
		name           string
		fileName       string
		content        string
		extraFields    map[string]string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:     "successful upload",
			fileName: "policy.txt",
			content:  "This is a government policy document",
			extraFields: map[string]string{
				"title":      "Test Policy",
				"department": "IT",
				"category":   "policy",
				"tags":       "policy,government",
				"language":   "en",
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid file format",
			fileName:       "invalid.xyz",
			content:        "Invalid content",
			extraFields:    map[string]string{"language": "en"},
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := createMultipartRequest("POST", "/api/documents/upload", "file", tt.fileName, tt.content, tt.extraFields)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var result document.ProcessingResult
				err := json.Unmarshal(w.Body.Bytes(), &result)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if tt.expectSuccess && result.Status != "uploaded" {
					t.Errorf("Expected status 'uploaded', got '%s': %s", result.Status, result.Message)
				}

				if !tt.expectSuccess && result.Status == "uploaded" {
					t.Errorf("Expected upload to fail, but got status 'uploaded'")
				}
			}
		})
	}
}

func TestDocumentHandler_GetDocument(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	// First upload a document
	req, err := createMultipartRequest("POST", "/api/documents/upload", "file", "test.txt", "Test content", map[string]string{"language": "en"})
	if err != nil {
		t.Fatalf("Failed to create upload request: %v", err)
	}

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

	// Now test getting the document
	documentID := uploadResult.DocumentID.Hex()

	tests := []struct {
		name           string
		documentID     string
		expectedStatus int
	}{
		{
			name:           "get existing document",
			documentID:     documentID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent document",
			documentID:     primitive.NewObjectID().Hex(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid document ID",
			documentID:     "invalid-id",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/documents/"+tt.documentID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var doc models.Document
				err := json.Unmarshal(w.Body.Bytes(), &doc)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if doc.ID.Hex() != tt.documentID {
					t.Errorf("Expected document ID %s, got %s", tt.documentID, doc.ID.Hex())
				}
			}
		})
	}
}

func TestDocumentHandler_GetProcessingStatus(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	// First upload a document
	req, err := createMultipartRequest("POST", "/api/documents/upload", "file", "test.txt", "Test content", map[string]string{"language": "en"})
	if err != nil {
		t.Fatalf("Failed to create upload request: %v", err)
	}

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

	// Test getting processing status
	documentID := uploadResult.DocumentID.Hex()
	req, err = http.NewRequest("GET", "/api/documents/"+documentID+"/status", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var statusResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check that status response contains expected fields
	expectedFields := []string{"id", "name", "processing_status", "uploaded_at"}
	for _, field := range expectedFields {
		if _, exists := statusResponse[field]; !exists {
			t.Errorf("Expected field '%s' in status response", field)
		}
	}
}

func TestDocumentHandler_SearchDocuments(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	// Create repository and ensure indexes
	repo := document.NewRepository(db)
	ctx := context.Background()
	err := repo.CreateIndexes(ctx)
	if err != nil {
		t.Logf("Warning: Could not create indexes: %v", err)
	}

	// Upload a few test documents
	testDocs := []struct {
		fileName string
		content  string
		fields   map[string]string
	}{
		{
			fileName: "policy1.txt",
			content:  "Government policy document about regulations",
			fields: map[string]string{
				"category": "policy",
				"tags":     "policy,government",
				"language": "en",
			},
		},
		{
			fileName: "strategy1.txt",
			content:  "Strategic planning document",
			fields: map[string]string{
				"category": "strategy",
				"tags":     "strategy,planning",
				"language": "en",
			},
		},
	}

	for _, doc := range testDocs {
		req, err := createMultipartRequest("POST", "/api/documents/upload", "file", doc.fileName, doc.content, doc.fields)
		if err != nil {
			t.Fatalf("Failed to create upload request: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Upload failed for %s: %s", doc.fileName, w.Body.String())
		}
	}

	// Wait a moment for documents to be indexed and processed
	time.Sleep(500 * time.Millisecond)

	// Test search
	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectResults  bool
	}{
		{
			name:           "search by query",
			query:          "query=policy",
			expectedStatus: http.StatusOK,
			expectResults:  true,
		},
		{
			name:           "search by category",
			query:          "category=strategy",
			expectedStatus: http.StatusOK,
			expectResults:  true,
		},
		{
			name:           "search with no results",
			query:          "query=nonexistent",
			expectedStatus: http.StatusOK,
			expectResults:  false,
		},
		{
			name:           "search with pagination",
			query:          "limit=1&skip=0",
			expectedStatus: http.StatusOK,
			expectResults:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/documents/search?"+tt.query, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var response SearchResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if tt.expectResults && response.Total == 0 {
					t.Errorf("Expected search results, but got 0 total")
				}

				if !tt.expectResults && response.Total > 0 {
					t.Errorf("Expected no search results, but got %d total", response.Total)
				}
			}
		})
	}
}

func TestDocumentHandler_ProcessDocument(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	// First upload a document
	req, err := createMultipartRequest("POST", "/api/documents/upload", "file", "test.txt", "Test content for processing", map[string]string{"language": "en"})
	if err != nil {
		t.Fatalf("Failed to create upload request: %v", err)
	}

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

	// Test processing the document
	documentID := uploadResult.DocumentID.Hex()
	req, err = http.NewRequest("POST", "/api/documents/"+documentID+"/process", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var processedDoc models.Document
	err = json.Unmarshal(w.Body.Bytes(), &processedDoc)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// The document should be processed, processing, or pending (since ProcessDocument forces processing)
	validStatuses := []models.ProcessingStatus{
		models.ProcessingStatusCompleted,
		models.ProcessingStatusProcessing,
		models.ProcessingStatusPending,
	}

	statusValid := false
	for _, validStatus := range validStatuses {
		if processedDoc.ProcessingStatus == validStatus {
			statusValid = true
			break
		}
	}

	if !statusValid {
		t.Errorf("Expected processing status to be completed, processing, or pending, got %s", processedDoc.ProcessingStatus)
	}
}

func TestDocumentHandler_MissingFile(t *testing.T) {
	db := setupAPITestDB(t)
	service := document.NewService(db)
	handler := NewDocumentHandler(service)
	router, _ := createTestRouter(handler)

	// Test upload without file
	req, err := http.NewRequest("POST", "/api/documents/upload", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResponse.Code != "MISSING_FILE" {
		t.Errorf("Expected error code 'MISSING_FILE', got '%s'", errorResponse.Code)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"tag1", []string{"tag1"}},
		{"tag1,tag2", []string{"tag1", "tag2"}},
		{"tag1, tag2, tag3", []string{"tag1", "tag2", "tag3"}},
		{" tag1 , tag2 , ", []string{"tag1", "tag2"}},
		{"tag1,,tag2", []string{"tag1", "tag2"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected tag '%s', got '%s'", expected, result[i])
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input     string
		expectErr bool
	}{
		{"2023-12-25T10:30:00Z", false},
		{"2023-12-25T10:30:00+05:00", false},
		{"invalid-date", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for input '%s', but got none", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for input '%s', but got: %v", tt.input, err)
			}
		})
	}
}
