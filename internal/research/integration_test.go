package research

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Integration test for the complete research workflow
// This test requires a running MongoDB instance and demonstrates the full research pipeline

func TestResearchWorkflowIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	// Setup test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(context.Background())
	
	db := client.Database("ai_government_consultant_integration_test")
	defer db.Drop(context.Background())
	
	// Create repository
	repo := NewMongoResearchRepository(db)
	err = repo.CreateIndexes(context.Background())
	require.NoError(t, err)
	
	// Create mock clients for testing
	mockNews := &MockNewsAPIClient{}
	mockLLM := &MockLLMClient{}
	
	// Setup mock expectations
	setupMockExpectations(mockNews, mockLLM)
	
	// Create research service
	config := &ResearchConfig{
		MaxConcurrentRequests: 5,
		RequestTimeout:        30 * time.Second,
		DefaultLanguage:       "en",
		MaxSourcesPerQuery:    10,
		MinCredibilityScore:   0.6,
		MinRelevanceScore:     0.5,
	}
	
	service := NewLangChainResearchService(repo, mockNews, mockLLM, config)
	
	// Create test document
	document := &models.Document{
		ID:      primitive.NewObjectID(),
		Name:    "Healthcare Policy Reform Document",
		Content: "This document outlines comprehensive healthcare policy reforms aimed at improving patient care, reducing costs, and enhancing accessibility. The proposed reforms include expanding coverage, implementing digital health records, and establishing quality metrics for healthcare providers.",
		Metadata: models.DocumentMetadata{
			Category: models.DocumentCategoryPolicy,
			Tags:     []string{"healthcare", "policy", "reform"},
		},
	}
	
	ctx := context.Background()
	
	// Step 1: Research policy context
	t.Run("Research Policy Context", func(t *testing.T) {
		result, err := service.ResearchPolicyContext(ctx, document)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.ID, result.DocumentID)
		assert.Equal(t, models.ResearchStatusCompleted, result.Status)
		assert.NotEmpty(t, result.ResearchQuery)
		assert.True(t, result.Confidence > 0)
		assert.Len(t, result.CurrentEvents, 2) // Based on mock setup
		assert.Len(t, result.PolicyImpacts, 1) // Based on mock setup
		assert.Len(t, result.Sources, 2) // Converted from events
		
		// Verify data was saved to database
		savedResult, err := repo.GetResearchResult(ctx, result.ID.Hex())
		require.NoError(t, err)
		assert.Equal(t, result.ID, savedResult.ID)
		assert.Equal(t, result.ResearchQuery, savedResult.ResearchQuery)
	})
	
	// Step 2: Generate policy suggestions
	t.Run("Generate Policy Suggestions", func(t *testing.T) {
		// Get the research result from the database
		results, err := repo.GetResearchResultsByDocument(ctx, document.ID.Hex())
		require.NoError(t, err)
		require.Len(t, results, 1)
		
		researchResult := &results[0]
		
		suggestions, err := service.GeneratePolicySuggestions(ctx, researchResult)
		
		require.NoError(t, err)
		assert.Len(t, suggestions, 1) // Based on simplified parser
		
		suggestion := suggestions[0]
		assert.NotEmpty(t, suggestion.Title)
		assert.NotEmpty(t, suggestion.Description)
		assert.NotEmpty(t, suggestion.Rationale)
		assert.True(t, suggestion.Confidence > 0)
		assert.Equal(t, models.DocumentCategoryPolicy, suggestion.Category)
		assert.Equal(t, "draft", suggestion.Status)
		
		// Verify suggestion was saved to database
		savedSuggestion, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
		require.NoError(t, err)
		assert.Equal(t, suggestion.ID, savedSuggestion.ID)
		assert.Equal(t, suggestion.Title, savedSuggestion.Title)
	})
	
	// Step 3: Validate research sources
	t.Run("Validate Research Sources", func(t *testing.T) {
		// Get research sources from database
		filters := ResearchSourceFilters{
			Limit: 10,
		}
		sources, err := repo.GetResearchSources(ctx, filters)
		require.NoError(t, err)
		require.NotEmpty(t, sources)
		
		validation, err := service.ValidateResearchSources(ctx, sources)
		
		require.NoError(t, err)
		assert.NotNil(t, validation)
		assert.True(t, validation.CredibilityScore > 0)
		assert.NotNil(t, validation.Issues)
		assert.NotNil(t, validation.Recommendations)
		assert.False(t, validation.ValidatedAt.IsZero())
		assert.Equal(t, "system", validation.ValidatedBy)
	})
	
	// Step 4: Query current events
	t.Run("Query Current Events", func(t *testing.T) {
		events, err := service.GetCurrentEvents(ctx, "healthcare policy", 7*24*time.Hour)
		
		require.NoError(t, err)
		assert.Len(t, events, 2) // Based on mock setup
		
		for _, event := range events {
			assert.NotEmpty(t, event.Title)
			assert.NotEmpty(t, event.Source)
			assert.NotEmpty(t, event.URL)
			assert.True(t, event.Relevance >= config.MinRelevanceScore)
		}
	})
	
	// Step 5: Analyze policy impact
	t.Run("Analyze Policy Impact", func(t *testing.T) {
		// Get current events from database
		filters := CurrentEventFilters{
			Limit: 10,
		}
		events, err := repo.GetCurrentEvents(ctx, filters)
		require.NoError(t, err)
		require.NotEmpty(t, events)
		
		impacts, err := service.AnalyzePolicyImpact(ctx, events, "healthcare policy")
		
		require.NoError(t, err)
		assert.Len(t, impacts, 1) // Based on simplified parser
		
		impact := impacts[0]
		assert.NotEmpty(t, impact.Area)
		assert.NotEmpty(t, impact.Impact)
		assert.NotEmpty(t, impact.Severity)
		assert.True(t, impact.Confidence > 0)
		assert.NotEmpty(t, impact.Stakeholders)
		assert.NotEmpty(t, impact.Mitigation)
	})
	
	// Step 6: Update policy suggestion status
	t.Run("Update Policy Suggestion Status", func(t *testing.T) {
		// Get policy suggestions from database
		suggestions, err := repo.GetPolicySuggestionsByCategory(ctx, models.DocumentCategoryPolicy)
		require.NoError(t, err)
		require.NotEmpty(t, suggestions)
		
		suggestion := suggestions[0]
		reviewNotes := "Approved after thorough review and stakeholder consultation"
		
		err = repo.UpdatePolicySuggestionStatus(ctx, suggestion.ID.Hex(), "approved", &reviewNotes)
		require.NoError(t, err)
		
		// Verify update
		updated, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
		require.NoError(t, err)
		assert.Equal(t, "approved", updated.Status)
		assert.Equal(t, reviewNotes, *updated.ReviewNotes)
		assert.NotNil(t, updated.ApprovedAt)
	})
	
	// Verify all mock expectations were met
	mockNews.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
}

func setupMockExpectations(mockNews *MockNewsAPIClient, mockLLM *MockLLMClient) {
	// Mock research query generation
	mockLLM.On("GenerateText", 
		mock.Anything, 
		mock.MatchedBy(func(prompt string) bool {
			return contains(prompt, "optimized search query")
		}), 
		mock.AnythingOfType("LLMOptions")).
		Return("healthcare policy reform patient care cost reduction", nil).Once()
	
	// Mock news search
	mockEvents := []models.CurrentEvent{
		{
			ID:          primitive.NewObjectID(),
			Title:       "Healthcare Reform Bill Passes Committee",
			Description: "New healthcare legislation advances through congressional committee",
			Source:      "Reuters",
			URL:         "https://example.com/news1",
			PublishedAt: time.Now().Add(-24 * time.Hour),
			Relevance:   0.9,
			Category:    "healthcare",
			Tags:        []string{"healthcare", "policy", "reform"},
			Content:     "Detailed coverage of healthcare reform progress",
			Author:      "Health Reporter",
			Language:    "en",
		},
		{
			ID:          primitive.NewObjectID(),
			Title:       "Cost Reduction Strategies in Healthcare",
			Description: "Analysis of cost-saving measures in healthcare delivery",
			Source:      "Associated Press",
			URL:         "https://example.com/news2",
			PublishedAt: time.Now().Add(-48 * time.Hour),
			Relevance:   0.8,
			Category:    "healthcare",
			Tags:        []string{"healthcare", "cost", "efficiency"},
			Content:     "In-depth analysis of healthcare cost reduction strategies",
			Author:      "Policy Analyst",
			Language:    "en",
		},
	}
	
	mockNews.On("SearchNews", 
		mock.Anything, 
		"healthcare policy reform patient care cost reduction", 
		mock.AnythingOfType("NewsSearchOptions")).
		Return(mockEvents, nil).Once()
	
	// Mock policy impact analysis
	mockLLM.On("GenerateText", 
		mock.Anything, 
		mock.MatchedBy(func(prompt string) bool {
			return contains(prompt, "policy impacts") && contains(prompt, "current events")
		}), 
		mock.AnythingOfType("LLMOptions")).
		Return(`[{"area": "healthcare policy", "impact": "Improved patient outcomes", "severity": "medium"}]`, nil).Once()
	
	// Mock policy suggestion generation
	mockLLM.On("GenerateText", 
		mock.Anything, 
		mock.MatchedBy(func(prompt string) bool {
			return contains(prompt, "policy suggestions") && contains(prompt, "research findings")
		}), 
		mock.AnythingOfType("LLMOptions")).
		Return(`[{"title": "Healthcare Access Expansion", "description": "Expand healthcare access", "priority": "high"}]`, nil).Once()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test research service configuration
func TestResearchServiceConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	mockRepo := &MockResearchRepository{}
	mockNews := &MockNewsAPIClient{}
	mockLLM := &MockLLMClient{}
	
	// Test with nil config (should use defaults)
	service1 := NewLangChainResearchService(mockRepo, mockNews, mockLLM, nil)
	assert.NotNil(t, service1.config)
	assert.Equal(t, 5, service1.config.MaxConcurrentRequests)
	assert.Equal(t, "en", service1.config.DefaultLanguage)
	assert.Equal(t, 20, service1.config.MaxSourcesPerQuery)
	
	// Test with custom config
	customConfig := &ResearchConfig{
		MaxConcurrentRequests: 10,
		DefaultLanguage:       "es",
		MaxSourcesPerQuery:    50,
		MinCredibilityScore:   0.8,
		MinRelevanceScore:     0.7,
	}
	
	service2 := NewLangChainResearchService(mockRepo, mockNews, mockLLM, customConfig)
	assert.Equal(t, customConfig, service2.config)
	assert.Equal(t, 10, service2.config.MaxConcurrentRequests)
	assert.Equal(t, "es", service2.config.DefaultLanguage)
	assert.Equal(t, 50, service2.config.MaxSourcesPerQuery)
	assert.Equal(t, 0.8, service2.config.MinCredibilityScore)
	assert.Equal(t, 0.7, service2.config.MinRelevanceScore)
}

// Test error handling in research workflow
func TestResearchWorkflowErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	mockRepo := &MockResearchRepository{}
	mockNews := &MockNewsAPIClient{}
	mockLLM := &MockLLMClient{}
	
	service := NewLangChainResearchService(mockRepo, mockNews, mockLLM, nil)
	
	document := &models.Document{
		ID:      primitive.NewObjectID(),
		Name:    "Test Document",
		Content: "Test content",
		Metadata: models.DocumentMetadata{
			Category: models.DocumentCategoryPolicy,
		},
	}
	
	ctx := context.Background()
	
	// Test research query generation failure
	t.Run("Research Query Generation Failure", func(t *testing.T) {
		mockLLM.On("GenerateText", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("LLMOptions")).
			Return("", assert.AnError).Once()
		
		mockRepo.On("SaveResearchResult", mock.Anything, mock.AnythingOfType("*models.ResearchResult")).
			Return(nil).Maybe()
		
		_, err := service.ResearchPolicyContext(ctx, document)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate research query")
	})
	
	// Test news search failure
	t.Run("News Search Failure", func(t *testing.T) {
		mockLLM.On("GenerateText", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("LLMOptions")).
			Return("test query", nil).Once()
		
		mockNews.On("SearchNews", mock.Anything, "test query", mock.AnythingOfType("NewsSearchOptions")).
			Return([]models.CurrentEvent{}, assert.AnError).Once()
		
		mockRepo.On("SaveResearchResult", mock.Anything, mock.AnythingOfType("*models.ResearchResult")).
			Return(nil).Twice()
		
		_, err := service.ResearchPolicyContext(ctx, document)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current events")
	})
	
	// Test incomplete research result
	t.Run("Incomplete Research Result", func(t *testing.T) {
		incompleteResult := &models.ResearchResult{
			Status: models.ResearchStatusPending,
		}
		
		_, err := service.GeneratePolicySuggestions(ctx, incompleteResult)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "research result is not completed")
	})
	
	mockRepo.AssertExpectations(t)
	mockNews.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
}