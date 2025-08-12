package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-government-consultant/internal/embedding"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock embedding service for testing
type mockEmbeddingService struct {
	generateEmbeddingFunc          func(ctx context.Context, text string) ([]float64, error)
	generateDocumentEmbeddingFunc  func(ctx context.Context, documentID primitive.ObjectID) error
	generateKnowledgeEmbeddingFunc func(ctx context.Context, knowledgeID primitive.ObjectID) error
	vectorSearchFunc               func(ctx context.Context, query string, options *embedding.SearchOptions) ([]embedding.SearchResult, error)
	getSimilarDocumentsFunc        func(ctx context.Context, documentID primitive.ObjectID, limit int) ([]embedding.SearchResult, error)
	getSimilarKnowledgeFunc        func(ctx context.Context, knowledgeID primitive.ObjectID, limit int) ([]embedding.SearchResult, error)
	clearCacheFunc                 func(ctx context.Context) error
}

func (m *mockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text)
	}
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *mockEmbeddingService) GenerateDocumentEmbedding(ctx context.Context, documentID primitive.ObjectID) error {
	if m.generateDocumentEmbeddingFunc != nil {
		return m.generateDocumentEmbeddingFunc(ctx, documentID)
	}
	return nil
}

func (m *mockEmbeddingService) GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID primitive.ObjectID) error {
	if m.generateKnowledgeEmbeddingFunc != nil {
		return m.generateKnowledgeEmbeddingFunc(ctx, knowledgeID)
	}
	return nil
}

func (m *mockEmbeddingService) VectorSearch(ctx context.Context, query string, options *embedding.SearchOptions) ([]embedding.SearchResult, error) {
	if m.vectorSearchFunc != nil {
		return m.vectorSearchFunc(ctx, query, options)
	}
	return []embedding.SearchResult{
		{
			ID:    "test-id-1",
			Score: 0.85,
			Metadata: map[string]interface{}{
				"type": "document",
			},
		},
	}, nil
}

func (m *mockEmbeddingService) GetSimilarDocuments(ctx context.Context, documentID primitive.ObjectID, limit int) ([]embedding.SearchResult, error) {
	if m.getSimilarDocumentsFunc != nil {
		return m.getSimilarDocumentsFunc(ctx, documentID, limit)
	}
	return []embedding.SearchResult{
		{
			ID:    "similar-doc-1",
			Score: 0.75,
			Metadata: map[string]interface{}{
				"type": "document",
			},
		},
	}, nil
}

func (m *mockEmbeddingService) GetSimilarKnowledge(ctx context.Context, knowledgeID primitive.ObjectID, limit int) ([]embedding.SearchResult, error) {
	if m.getSimilarKnowledgeFunc != nil {
		return m.getSimilarKnowledgeFunc(ctx, knowledgeID, limit)
	}
	return []embedding.SearchResult{
		{
			ID:    "similar-knowledge-1",
			Score: 0.80,
			Metadata: map[string]interface{}{
				"type": "knowledge",
			},
		},
	}, nil
}

func (m *mockEmbeddingService) ClearCache(ctx context.Context) error {
	if m.clearCacheFunc != nil {
		return m.clearCacheFunc(ctx)
	}
	return nil
}

// Mock embedding repository for testing
type mockEmbeddingRepository struct {
	getEmbeddingStatsFunc func(ctx context.Context) (*embedding.EmbeddingStats, error)
}

func (m *mockEmbeddingRepository) GetEmbeddingStats(ctx context.Context) (*embedding.EmbeddingStats, error) {
	if m.getEmbeddingStatsFunc != nil {
		return m.getEmbeddingStatsFunc(ctx)
	}
	return &embedding.EmbeddingStats{
		DocumentsWithEmbeddings: 5,
		TotalDocuments:          10,
		KnowledgeWithEmbeddings: 3,
		TotalKnowledgeItems:     8,
	}, nil
}

// Mock embedding pipeline for testing
type mockEmbeddingPipeline struct {
	processAllDocumentsFunc           func(ctx context.Context) (*embedding.ProcessResult, error)
	processAllKnowledgeItemsFunc      func(ctx context.Context) (*embedding.ProcessResult, error)
	processSpecificDocumentsFunc      func(ctx context.Context, documentIDs []primitive.ObjectID) (*embedding.ProcessResult, error)
	processSpecificKnowledgeItemsFunc func(ctx context.Context, knowledgeIDs []primitive.ObjectID) (*embedding.ProcessResult, error)
}

func (m *mockEmbeddingPipeline) ProcessAllDocuments(ctx context.Context) (*embedding.ProcessResult, error) {
	if m.processAllDocumentsFunc != nil {
		return m.processAllDocumentsFunc(ctx)
	}
	return &embedding.ProcessResult{
		TotalProcessed: 5,
		Successful:     4,
		Failed:         1,
		Duration:       time.Second * 10,
		Errors:         []string{"one error"},
	}, nil
}

func (m *mockEmbeddingPipeline) ProcessAllKnowledgeItems(ctx context.Context) (*embedding.ProcessResult, error) {
	if m.processAllKnowledgeItemsFunc != nil {
		return m.processAllKnowledgeItemsFunc(ctx)
	}
	return &embedding.ProcessResult{
		TotalProcessed: 3,
		Successful:     3,
		Failed:         0,
		Duration:       time.Second * 5,
		Errors:         []string{},
	}, nil
}

func (m *mockEmbeddingPipeline) ProcessSpecificDocuments(ctx context.Context, documentIDs []primitive.ObjectID) (*embedding.ProcessResult, error) {
	if m.processSpecificDocumentsFunc != nil {
		return m.processSpecificDocumentsFunc(ctx, documentIDs)
	}
	return &embedding.ProcessResult{
		TotalProcessed: len(documentIDs),
		Successful:     len(documentIDs),
		Failed:         0,
		Duration:       time.Second * 2,
		Errors:         []string{},
	}, nil
}

func (m *mockEmbeddingPipeline) ProcessSpecificKnowledgeItems(ctx context.Context, knowledgeIDs []primitive.ObjectID) (*embedding.ProcessResult, error) {
	if m.processSpecificKnowledgeItemsFunc != nil {
		return m.processSpecificKnowledgeItemsFunc(ctx, knowledgeIDs)
	}
	return &embedding.ProcessResult{
		TotalProcessed: len(knowledgeIDs),
		Successful:     len(knowledgeIDs),
		Failed:         0,
		Duration:       time.Second * 2,
		Errors:         []string{},
	}, nil
}

func setupEmbeddingHandler() (*EmbeddingHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	service := &mockEmbeddingService{}
	repository := &mockEmbeddingRepository{}
	pipeline := &mockEmbeddingPipeline{}

	handler := NewEmbeddingHandler(service, repository, pipeline)

	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, router
}

func TestGenerateEmbedding(t *testing.T) {
	_, router := setupEmbeddingHandler()

	tests := []struct {
		name           string
		requestBody    GenerateEmbeddingRequest
		expectedStatus int
		expectedDims   int
	}{
		{
			name: "successful embedding generation",
			requestBody: GenerateEmbeddingRequest{
				Text: "This is a test document",
			},
			expectedStatus: http.StatusOK,
			expectedDims:   5,
		},
		{
			name: "empty text",
			requestBody: GenerateEmbeddingRequest{
				Text: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/embeddings/generate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response GenerateEmbeddingResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				if response.Dimensions != tt.expectedDims {
					t.Errorf("expected %d dimensions, got %d", tt.expectedDims, response.Dimensions)
				}

				if len(response.Embeddings) != tt.expectedDims {
					t.Errorf("expected %d embeddings, got %d", tt.expectedDims, len(response.Embeddings))
				}
			}
		})
	}
}

func TestVectorSearch(t *testing.T) {
	_, router := setupEmbeddingHandler()

	tests := []struct {
		name           string
		requestBody    VectorSearchRequest
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "successful search",
			requestBody: VectorSearchRequest{
				Query:     "test query",
				Limit:     10,
				Threshold: 0.7,
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name: "search with filters",
			requestBody: VectorSearchRequest{
				Query:      "test query",
				Collection: "documents",
				Filters: map[string]interface{}{
					"category": "policy",
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name: "empty query",
			requestBody: VectorSearchRequest{
				Query: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/embeddings/search", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response VectorSearchResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				if response.Count != tt.expectedCount {
					t.Errorf("expected %d results, got %d", tt.expectedCount, response.Count)
				}

				if len(response.Results) != tt.expectedCount {
					t.Errorf("expected %d results, got %d", tt.expectedCount, len(response.Results))
				}
			}
		})
	}
}

func TestProcessEmbeddings(t *testing.T) {
	_, router := setupEmbeddingHandler()

	tests := []struct {
		name           string
		requestBody    ProcessEmbeddingsRequest
		expectedStatus int
	}{
		{
			name: "process all",
			requestBody: ProcessEmbeddingsRequest{
				ProcessAll: true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "process specific documents",
			requestBody: ProcessEmbeddingsRequest{
				DocumentIDs: []string{primitive.NewObjectID().Hex()},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "process specific knowledge items",
			requestBody: ProcessEmbeddingsRequest{
				KnowledgeIDs: []string{primitive.NewObjectID().Hex()},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid document ID",
			requestBody: ProcessEmbeddingsRequest{
				DocumentIDs: []string{"invalid-id"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no parameters",
			requestBody:    ProcessEmbeddingsRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/embeddings/process", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response ProcessEmbeddingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				if response.TotalProcessed <= 0 {
					t.Error("expected positive total processed count")
				}
			}
		})
	}
}

func TestGetEmbeddingStats(t *testing.T) {
	_, router := setupEmbeddingHandler()

	req := httptest.NewRequest("GET", "/api/v1/embeddings/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var stats embedding.EmbeddingStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	if err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	if stats.TotalDocuments != 10 {
		t.Errorf("expected 10 total documents, got %d", stats.TotalDocuments)
	}
	if stats.DocumentsWithEmbeddings != 5 {
		t.Errorf("expected 5 documents with embeddings, got %d", stats.DocumentsWithEmbeddings)
	}
}

func TestGenerateDocumentEmbedding(t *testing.T) {
	_, router := setupEmbeddingHandler()

	tests := []struct {
		name           string
		documentID     string
		expectedStatus int
	}{
		{
			name:           "valid document ID",
			documentID:     primitive.NewObjectID().Hex(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid document ID",
			documentID:     "invalid-id",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/embeddings/documents/"+tt.documentID+"/embedding", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetSimilarDocuments(t *testing.T) {
	_, router := setupEmbeddingHandler()

	tests := []struct {
		name           string
		documentID     string
		limit          string
		expectedStatus int
	}{
		{
			name:           "valid document ID",
			documentID:     primitive.NewObjectID().Hex(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid document ID with limit",
			documentID:     primitive.NewObjectID().Hex(),
			limit:          "3",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid document ID",
			documentID:     "invalid-id",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/embeddings/documents/" + tt.documentID + "/similar"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response VectorSearchResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				if response.Count != 1 {
					t.Errorf("expected 1 result, got %d", response.Count)
				}
			}
		})
	}
}

func TestClearCache(t *testing.T) {
	_, router := setupEmbeddingHandler()

	req := httptest.NewRequest("DELETE", "/api/v1/embeddings/cache", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Message == "" {
		t.Error("expected success message")
	}
}
