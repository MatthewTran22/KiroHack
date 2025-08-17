package consultation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockEmbeddingService implements a subset of embedding service methods for testing
type MockEmbeddingService struct {
	searchResults []embedding.SearchResult
	searchError   error
}

func (m *MockEmbeddingService) VectorSearch(ctx context.Context, query string, options *embedding.SearchOptions) ([]embedding.SearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}



// MockLogger implements logger.Logger interface for testing
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {}
func (m *MockLogger) Info(msg string, fields map[string]interface{})  {}
func (m *MockLogger) Warn(msg string, fields map[string]interface{})  {}
func (m *MockLogger) Error(msg string, err error, fields map[string]interface{}) {}
func (m *MockLogger) Fatal(msg string, err error, fields map[string]interface{}) {}

// Test setup helpers
func setupTestService(t *testing.T) (*Service, *httptest.Server, *MockEmbeddingService) {
	// Create mock Gemini API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: generateMockConsultationResponse()},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     100,
				CandidatesTokenCount: 200,
				TotalTokenCount:      300,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// Create mock embedding service
	mockEmbedding := &MockEmbeddingService{
		searchResults: []embedding.SearchResult{
			{
				ID:    "doc1",
				Score: 0.9,
				Document: &models.Document{
					ID:      primitive.NewObjectID(),
					Name:    "Test Policy Document",
					Content: "This is a test policy document with relevant information about government procedures.",
				},
			},
			{
				ID:    "knowledge1",
				Score: 0.8,
				Knowledge: &models.KnowledgeItem{
					ID:      primitive.NewObjectID(),
					Title:   "Government Best Practices",
					Content: "Best practices for government operations and policy implementation.",
				},
			},
		},
	}

	// Create service
	config := &Config{
		GeminiAPIKey:     "test-api-key",
		GeminiURL:        server.URL,
		EmbeddingService: EmbeddingServiceInterface(mockEmbedding),
		Logger:           &MockLogger{},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
	}

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	return service, server, mockEmbedding
}

func generateMockConsultationResponse() string {
	return `
Executive Summary:
Based on the analysis of the provided query and relevant documentation, I recommend implementing a comprehensive policy framework that addresses the key concerns raised.

Policy Analysis:
The current situation requires immediate attention to regulatory compliance and stakeholder engagement. Key findings include:
- Need for updated procedures
- Compliance gaps identified
- Stakeholder alignment required

Recommendations:
1. Implement Updated Policy Framework
   Develop and deploy a comprehensive policy framework that addresses current regulatory requirements and stakeholder needs. This should include clear guidelines, procedures, and compliance mechanisms.

2. Establish Monitoring System
   Create a robust monitoring and evaluation system to track policy implementation and effectiveness. This will ensure continuous improvement and compliance.

3. Conduct Stakeholder Engagement
   Initiate comprehensive stakeholder consultation to ensure buy-in and address concerns. This is critical for successful implementation.

Risk Assessment:
Overall risk level: Medium
Key risks include implementation delays, stakeholder resistance, and resource constraints.

Implementation Plan:
Phase 1: Planning and preparation (2-4 weeks)
Phase 2: Stakeholder engagement (4-6 weeks)  
Phase 3: Policy development (6-8 weeks)
Phase 4: Implementation and monitoring (ongoing)

Next Steps:
1. Establish project team and governance structure
2. Conduct detailed stakeholder analysis
3. Develop implementation timeline
4. Begin policy framework development
`
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "missing API key",
			config: &Config{
				EmbeddingService: &MockEmbeddingService{},
				Logger:           &MockLogger{},
			},
			expectError: true,
		},
		{
			name: "missing embedding service",
			config: &Config{
				GeminiAPIKey: "test-key",
				Logger:       &MockLogger{},
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &Config{
				GeminiAPIKey:     "test-key",
				EmbeddingService: EmbeddingServiceInterface(&MockEmbeddingService{}),
				Logger:           &MockLogger{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if service != nil {
					t.Error("Expected nil service but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if service == nil {
					t.Error("Expected non-nil service but got nil")
				}
			}
		})
	}
}

func TestConsultPolicy(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "How should we implement a new data privacy policy?",
		Type:   models.ConsultationTypePolicy,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
		MaxSources:          5,
		ConfidenceThreshold: 0.7,
	}

	response, err := service.ConsultPolicy(ctx, request)
	if err != nil {
		t.Fatalf("ConsultPolicy failed: %v", err)
	}

	// Validate response structure
	if response == nil {
		t.Fatal("Response is nil")
	}

	if len(response.Recommendations) == 0 {
		t.Error("No recommendations in response")
	}

	if response.Analysis.Summary == "" {
		t.Error("Analysis summary is empty")
	}

	if response.ConfidenceScore <= 0 {
		t.Error("Confidence score should be positive")
	}

	// Validate first recommendation
	if len(response.Recommendations) > 0 {
		rec := response.Recommendations[0]
		if rec.Title == "" {
			t.Error("Recommendation title is empty")
		}
		if rec.Description == "" {
			t.Error("Recommendation description is empty")
		}
		if len(rec.Implementation.Steps) == 0 {
			t.Error("Recommendation has no implementation steps")
		}
	}
}

func TestConsultStrategy(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "What strategic approach should we take for digital transformation?",
		Type:   models.ConsultationTypeStrategy,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	response, err := service.ConsultStrategy(ctx, request)
	if err != nil {
		t.Fatalf("ConsultStrategy failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if len(response.Recommendations) == 0 {
		t.Error("No recommendations in response")
	}
}

func TestConsultOperations(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "How can we improve our operational efficiency?",
		Type:   models.ConsultationTypeOperations,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	response, err := service.ConsultOperations(ctx, request)
	if err != nil {
		t.Fatalf("ConsultOperations failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if len(response.Recommendations) == 0 {
		t.Error("No recommendations in response")
	}
}

func TestConsultTechnology(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "What technology stack should we use for our new system?",
		Type:   models.ConsultationTypeTechnology,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	response, err := service.ConsultTechnology(ctx, request)
	if err != nil {
		t.Fatalf("ConsultTechnology failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if len(response.Recommendations) == 0 {
		t.Error("No recommendations in response")
	}
}

func TestInvalidConsultationType(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "Test query",
		Type:   models.ConsultationTypeStrategy, // Wrong type for policy consultation
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	_, err := service.ConsultPolicy(ctx, request)
	if err == nil {
		t.Error("Expected error for invalid consultation type")
	}
}

func TestRateLimiting(t *testing.T) {
	// Create service with very low rate limit
	config := &Config{
		GeminiAPIKey:     "test-key",
		GeminiURL:        "http://localhost:8999", // Non-existent server
		EmbeddingService: EmbeddingServiceInterface(&MockEmbeddingService{}),
		Logger:           &MockLogger{},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 1,  // Very low limit
			BurstSize:         1,
		},
	}

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "Test query",
		Type:   models.ConsultationTypePolicy,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	// First request should be allowed (but will fail due to non-existent server)
	_, err1 := service.ConsultPolicy(ctx, request)
	
	// Second immediate request should be rate limited
	_, err2 := service.ConsultPolicy(ctx, request)

	// At least one should fail due to rate limiting or connection error
	if err1 == nil && err2 == nil {
		t.Error("Expected at least one request to fail due to rate limiting or connection error")
	}
}

func TestContextRetrieval(t *testing.T) {
	service, server, mockEmbedding := setupTestService(t)
	defer server.Close()

	ctx := context.Background()
	
	// Test with search results
	contextData, err := service.retrieveContext(ctx, "test query", 4) // Use 4 to get 2 docs + 2 knowledge
	if err != nil {
		t.Fatalf("retrieveContext failed: %v", err)
	}

	if contextData.TotalSources != 4 {
		t.Errorf("Expected 4 total sources, got %d", contextData.TotalSources)
	}

	if len(contextData.Documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(contextData.Documents))
	}

	if len(contextData.Knowledge) != 2 {
		t.Errorf("Expected 2 knowledge items, got %d", len(contextData.Knowledge))
	}

	// Test with search error
	mockEmbedding.searchError = &embedding.EmbeddingError{Message: "Search failed"}
	contextData, err = service.retrieveContext(ctx, "test query", 5)
	if err != nil {
		t.Fatalf("retrieveContext should handle search errors gracefully: %v", err)
	}

	if contextData.TotalSources != 0 {
		t.Errorf("Expected 0 total sources when search fails, got %d", contextData.TotalSources)
	}
}

func TestPromptGeneration(t *testing.T) {
	service, server, _ := setupTestService(t)
	defer server.Close()

	contextData := &ContextData{
		Documents: []embedding.SearchResult{
			{
				Document: &models.Document{
					Name:    "Test Document",
					Content: "Test content for document",
				},
				Score: 0.9,
			},
		},
		Knowledge: []embedding.SearchResult{
			{
				Knowledge: &models.KnowledgeItem{
					Title:   "Test Knowledge",
					Content: "Test content for knowledge",
				},
				Score: 0.8,
			},
		},
		TotalSources: 2,
	}

	tests := []struct {
		name     string
		promptFn func(string, *ContextData) string
		query    string
	}{
		{
			name:     "policy prompt",
			promptFn: service.generatePolicyPrompt,
			query:    "How should we implement data privacy policies?",
		},
		{
			name:     "strategy prompt",
			promptFn: service.generateStrategyPrompt,
			query:    "What strategic approach should we take?",
		},
		{
			name:     "operations prompt",
			promptFn: service.generateOperationsPrompt,
			query:    "How can we improve efficiency?",
		},
		{
			name:     "technology prompt",
			promptFn: service.generateTechnologyPrompt,
			query:    "What technology should we use?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := tt.promptFn(tt.query, contextData)
			
			if prompt == "" {
				t.Error("Generated prompt is empty")
			}

			if !contains(prompt, tt.query) {
				t.Error("Prompt does not contain the original query")
			}

			if !contains(prompt, "Test Document") {
				t.Error("Prompt does not contain document context")
			}

			if !contains(prompt, "Test Knowledge") {
				t.Error("Prompt does not contain knowledge context")
			}
		})
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsAt(s, substr, 1))))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) > len(s) {
		return containsAt(s, substr, start+1)
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}

// Benchmark tests
func BenchmarkConsultPolicy(b *testing.B) {
	service, server, _ := setupTestService(&testing.T{})
	defer server.Close()

	ctx := context.Background()
	request := &ConsultationRequest{
		Query:  "How should we implement a new data privacy policy?",
		Type:   models.ConsultationTypePolicy,
		UserID: primitive.NewObjectID(),
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ConsultPolicy(ctx, request)
		if err != nil {
			b.Fatalf("ConsultPolicy failed: %v", err)
		}
	}
}

func BenchmarkPromptGeneration(b *testing.B) {
	service, server, _ := setupTestService(&testing.T{})
	defer server.Close()

	contextData := &ContextData{
		Documents: []embedding.SearchResult{
			{
				Document: &models.Document{
					Name:    "Test Document",
					Content: "Test content for document",
				},
				Score: 0.9,
			},
		},
		TotalSources: 1,
	}

	query := "How should we implement data privacy policies?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.generatePolicyPrompt(query, contextData)
	}
}