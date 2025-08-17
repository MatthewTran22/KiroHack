package research

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Integration tests for MongoDB repository
// These tests require a running MongoDB instance

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	// Connect to test database
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	// Use a test database
	db := client.Database("ai_government_consultant_test")
	
	// Cleanup function
	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}
	
	return db, cleanup
}

func TestMongoResearchRepository_SaveAndGetResearchResult(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create test research result
	result := &models.ResearchResult{
		DocumentID:    primitive.NewObjectID(),
		ResearchQuery: "test query",
		Status:        models.ResearchStatusCompleted,
		CurrentEvents: []models.CurrentEvent{
			{
				ID:          primitive.NewObjectID(),
				Title:       "Test Event",
				Description: "Test Description",
				Source:      "Test Source",
				URL:         "https://example.com",
				PublishedAt: time.Now(),
				Relevance:   0.8,
				Category:    "test",
				Tags:        []string{"test", "event"},
			},
		},
		PolicyImpacts: []models.PolicyImpact{
			{
				Area:         "test area",
				Impact:       "test impact",
				Severity:     "medium",
				Timeframe:    "short-term",
				Stakeholders: []string{"stakeholder1"},
				Mitigation:   []string{"mitigation1"},
				Confidence:   0.7,
				Evidence:     []string{"evidence1"},
			},
		},
		Sources: []models.ResearchSource{
			{
				ID:          primitive.NewObjectID(),
				Type:        models.ResearchSourceTypeNews,
				Title:       "Test Source",
				URL:         "https://example.com/source",
				Author:      "Test Author",
				PublishedAt: time.Now(),
				Credibility: 0.9,
				Relevance:   0.8,
				Content:     "Test content",
				Summary:     "Test summary",
				Keywords:    []string{"test", "source"},
				Language:    "en",
			},
		},
		Confidence:     0.8,
		ProcessingTime: 5 * time.Second,
		Metadata:       map[string]interface{}{"test": "value"},
	}
	
	// Save research result
	err := repo.SaveResearchResult(ctx, result)
	assert.NoError(t, err)
	assert.False(t, result.ID.IsZero())
	assert.False(t, result.GeneratedAt.IsZero())
	
	// Get research result
	retrieved, err := repo.GetResearchResult(ctx, result.ID.Hex())
	assert.NoError(t, err)
	assert.Equal(t, result.ID, retrieved.ID)
	assert.Equal(t, result.DocumentID, retrieved.DocumentID)
	assert.Equal(t, result.ResearchQuery, retrieved.ResearchQuery)
	assert.Equal(t, result.Status, retrieved.Status)
	assert.Equal(t, result.Confidence, retrieved.Confidence)
	assert.Len(t, retrieved.CurrentEvents, 1)
	assert.Len(t, retrieved.PolicyImpacts, 1)
	assert.Len(t, retrieved.Sources, 1)
}

func TestMongoResearchRepository_GetResearchResultsByDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	documentID := primitive.NewObjectID()
	
	// Create multiple research results for the same document
	results := []*models.ResearchResult{
		{
			DocumentID:    documentID,
			ResearchQuery: "query 1",
			Status:        models.ResearchStatusCompleted,
			Confidence:    0.8,
			Metadata:      make(map[string]interface{}),
		},
		{
			DocumentID:    documentID,
			ResearchQuery: "query 2",
			Status:        models.ResearchStatusCompleted,
			Confidence:    0.7,
			Metadata:      make(map[string]interface{}),
		},
		{
			DocumentID:    primitive.NewObjectID(), // Different document
			ResearchQuery: "query 3",
			Status:        models.ResearchStatusCompleted,
			Confidence:    0.9,
			Metadata:      make(map[string]interface{}),
		},
	}
	
	// Save all results
	for _, result := range results {
		err := repo.SaveResearchResult(ctx, result)
		assert.NoError(t, err)
	}
	
	// Get results by document
	retrieved, err := repo.GetResearchResultsByDocument(ctx, documentID.Hex())
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2) // Only results for the specified document
	
	// Results should be sorted by generated_at descending
	assert.True(t, retrieved[0].GeneratedAt.After(retrieved[1].GeneratedAt) || 
		retrieved[0].GeneratedAt.Equal(retrieved[1].GeneratedAt))
}

func TestMongoResearchRepository_SaveAndGetPolicySuggestion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create test policy suggestion
	suggestion := &models.PolicySuggestion{
		Title:       "Test Policy Suggestion",
		Description: "Test description",
		Rationale:   "Test rationale",
		Priority:    models.PolicyPriorityHigh,
		Category:    models.DocumentCategoryPolicy,
		Tags:        []string{"test", "policy"},
		Confidence:  0.8,
		CreatedBy:   primitive.NewObjectID(),
		CurrentContext: []models.CurrentEvent{
			{
				ID:          primitive.NewObjectID(),
				Title:       "Context Event",
				Description: "Context description",
				Source:      "Test Source",
				URL:         "https://example.com",
				PublishedAt: time.Now(),
				Relevance:   0.7,
			},
		},
		Implementation: models.ImplementationPlan{
			Steps:          []string{"step1", "step2"},
			Timeline:       "6 months",
			Resources:      []string{"resource1", "resource2"},
			Stakeholders:   []string{"stakeholder1", "stakeholder2"},
			Dependencies:   []string{"dependency1"},
			Milestones:     []string{"milestone1"},
			RiskFactors:    []string{"risk1"},
			SuccessMetrics: []string{"metric1"},
		},
		RiskAssessment: models.PolicyRiskAssessment{
			OverallRisk: "medium",
			RiskFactors: []models.PolicyRiskFactor{
				{
					Description: "Test risk",
					Probability: 0.3,
					Impact:      "medium",
					Mitigation:  "Test mitigation",
					Category:    "operational",
				},
			},
			AssessedAt: time.Now(),
			AssessedBy: "system",
			Confidence: 0.7,
		},
	}
	
	// Save policy suggestion
	err := repo.SavePolicySuggestion(ctx, suggestion)
	assert.NoError(t, err)
	assert.False(t, suggestion.ID.IsZero())
	assert.False(t, suggestion.CreatedAt.IsZero())
	assert.False(t, suggestion.UpdatedAt.IsZero())
	assert.Equal(t, "draft", suggestion.Status)
	
	// Get policy suggestion
	retrieved, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
	assert.NoError(t, err)
	assert.Equal(t, suggestion.ID, retrieved.ID)
	assert.Equal(t, suggestion.Title, retrieved.Title)
	assert.Equal(t, suggestion.Description, retrieved.Description)
	assert.Equal(t, suggestion.Priority, retrieved.Priority)
	assert.Equal(t, suggestion.Category, retrieved.Category)
	assert.Equal(t, suggestion.Confidence, retrieved.Confidence)
	assert.Len(t, retrieved.CurrentContext, 1)
}

func TestMongoResearchRepository_GetPolicySuggestionsByCategory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create policy suggestions with different categories
	suggestions := []*models.PolicySuggestion{
		{
			Title:       "Policy Suggestion 1",
			Description: "Description 1",
			Category:    models.DocumentCategoryPolicy,
			Priority:    models.PolicyPriorityHigh,
			Confidence:  0.8,
			CreatedBy:   primitive.NewObjectID(),
		},
		{
			Title:       "Policy Suggestion 2",
			Description: "Description 2",
			Category:    models.DocumentCategoryPolicy,
			Priority:    models.PolicyPriorityMedium,
			Confidence:  0.7,
			CreatedBy:   primitive.NewObjectID(),
		},
		{
			Title:       "Strategy Suggestion",
			Description: "Strategy description",
			Category:    models.DocumentCategoryStrategy,
			Priority:    models.PolicyPriorityLow,
			Confidence:  0.6,
			CreatedBy:   primitive.NewObjectID(),
		},
	}
	
	// Save all suggestions
	for _, suggestion := range suggestions {
		err := repo.SavePolicySuggestion(ctx, suggestion)
		assert.NoError(t, err)
	}
	
	// Get suggestions by category
	policyResults, err := repo.GetPolicySuggestionsByCategory(ctx, models.DocumentCategoryPolicy)
	assert.NoError(t, err)
	assert.Len(t, policyResults, 2)
	
	strategyResults, err := repo.GetPolicySuggestionsByCategory(ctx, models.DocumentCategoryStrategy)
	assert.NoError(t, err)
	assert.Len(t, strategyResults, 1)
}

func TestMongoResearchRepository_SaveAndGetCurrentEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create test current event
	event := &models.CurrentEvent{
		Title:       "Test Current Event",
		Description: "Test description",
		Source:      "Test Source",
		URL:         "https://example.com/unique-url",
		PublishedAt: time.Now().Add(-24 * time.Hour),
		Relevance:   0.8,
		Category:    "test",
		Tags:        []string{"test", "event"},
		Content:     "Test content",
		Author:      "Test Author",
		Language:    "en",
	}
	
	// Save current event
	err := repo.SaveCurrentEvent(ctx, event)
	assert.NoError(t, err)
	assert.False(t, event.ID.IsZero())
	assert.False(t, event.CreatedAt.IsZero())
	assert.False(t, event.UpdatedAt.IsZero())
	
	// Get current events with filters
	filters := CurrentEventFilters{
		Category: &event.Category,
		Limit:    10,
		Offset:   0,
	}
	
	events, err := repo.GetCurrentEvents(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, event.Title, events[0].Title)
	assert.Equal(t, event.URL, events[0].URL)
}

func TestMongoResearchRepository_SaveAndGetResearchSource(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create test research source
	source := &models.ResearchSource{
		Type:        models.ResearchSourceTypeNews,
		Title:       "Test Research Source",
		URL:         "https://example.com/unique-source",
		Author:      "Test Author",
		PublishedAt: time.Now().Add(-48 * time.Hour),
		Credibility: 0.9,
		Relevance:   0.8,
		Content:     "Test content",
		Summary:     "Test summary",
		Keywords:    []string{"test", "research"},
		Language:    "en",
	}
	
	// Save research source
	err := repo.SaveResearchSource(ctx, source)
	assert.NoError(t, err)
	assert.False(t, source.ID.IsZero())
	assert.False(t, source.CreatedAt.IsZero())
	assert.False(t, source.UpdatedAt.IsZero())
	
	// Get research sources with filters
	filters := ResearchSourceFilters{
		Type:           &source.Type,
		MinCredibility: &[]float64{0.8}[0],
		Limit:          10,
		Offset:         0,
	}
	
	sources, err := repo.GetResearchSources(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, sources, 1)
	assert.Equal(t, source.Title, sources[0].Title)
	assert.Equal(t, source.URL, sources[0].URL)
	assert.Equal(t, source.Credibility, sources[0].Credibility)
}

func TestMongoResearchRepository_UpdatePolicySuggestionStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create and save policy suggestion
	suggestion := &models.PolicySuggestion{
		Title:       "Test Policy Suggestion",
		Description: "Test description",
		Category:    models.DocumentCategoryPolicy,
		Priority:    models.PolicyPriorityMedium,
		Confidence:  0.7,
		CreatedBy:   primitive.NewObjectID(),
	}
	
	err := repo.SavePolicySuggestion(ctx, suggestion)
	assert.NoError(t, err)
	
	// Update status
	reviewNotes := "Approved after review"
	err = repo.UpdatePolicySuggestionStatus(ctx, suggestion.ID.Hex(), "approved", &reviewNotes)
	assert.NoError(t, err)
	
	// Verify update
	updated, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
	assert.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
	assert.Equal(t, reviewNotes, *updated.ReviewNotes)
	assert.NotNil(t, updated.ApprovedAt)
}

func TestMongoResearchRepository_DuplicateHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Test duplicate current event handling
	event1 := &models.CurrentEvent{
		Title:       "Original Event",
		Description: "Original description",
		Source:      "Test Source",
		URL:         "https://example.com/duplicate-test",
		PublishedAt: time.Now(),
		Relevance:   0.7,
		Category:    "test",
	}
	
	event2 := &models.CurrentEvent{
		Title:       "Updated Event",
		Description: "Updated description",
		Source:      "Test Source",
		URL:         "https://example.com/duplicate-test", // Same URL
		PublishedAt: time.Now(),
		Relevance:   0.8,
		Category:    "test",
	}
	
	// Save first event
	err := repo.SaveCurrentEvent(ctx, event1)
	assert.NoError(t, err)
	
	// Save second event with same URL (should update)
	err = repo.SaveCurrentEvent(ctx, event2)
	assert.NoError(t, err)
	
	// Verify only one event exists with updated information
	filters := CurrentEventFilters{Limit: 10}
	events, err := repo.GetCurrentEvents(ctx, filters)
	assert.NoError(t, err)
	
	// Find our test event
	var foundEvent *models.CurrentEvent
	for _, e := range events {
		if e.URL == "https://example.com/duplicate-test" {
			foundEvent = &e
			break
		}
	}
	
	assert.NotNil(t, foundEvent)
	assert.Equal(t, "Updated Event", foundEvent.Title)
	assert.Equal(t, "Updated description", foundEvent.Description)
	assert.Equal(t, 0.8, foundEvent.Relevance)
}

func TestMongoResearchRepository_CreateIndexes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	repo := NewMongoResearchRepository(db)
	ctx := context.Background()
	
	// Create indexes
	err := repo.CreateIndexes(ctx)
	assert.NoError(t, err)
	
	// Verify indexes were created by checking collections exist
	// (In a real test, you might want to verify specific indexes)
	collections, err := db.ListCollectionNames(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	
	// After creating indexes and inserting test data, collections should exist
	// Insert some test data to trigger collection creation
	testResult := &models.ResearchResult{
		DocumentID:    primitive.NewObjectID(),
		ResearchQuery: "test",
		Status:        models.ResearchStatusCompleted,
		Confidence:    0.5,
		Metadata:      make(map[string]interface{}),
	}
	
	err = repo.SaveResearchResult(ctx, testResult)
	assert.NoError(t, err)
	
	// Check that collections were created
	collections, err = db.ListCollectionNames(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Contains(t, collections, "research_results")
}