package test

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/internal/research"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestDockerMongoDB tests MongoDB connection in Docker environment
func TestDockerMongoDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	// Connect directly to MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:password@localhost:27017"))
	require.NoError(t, err, "Failed to connect to MongoDB")
	defer client.Disconnect(ctx)
	
	// Test ping
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Failed to ping MongoDB")
	
	// Test database operations
	db := client.Database("test_research_db")
	collection := db.Collection("test_collection")
	
	// Insert test document
	testDoc := map[string]interface{}{
		"title":      "Docker Test Document",
		"content":    "Testing MongoDB in Docker",
		"created_at": time.Now(),
	}
	
	result, err := collection.InsertOne(ctx, testDoc)
	require.NoError(t, err, "Failed to insert test document")
	assert.NotNil(t, result.InsertedID, "Insert should return an ID")
	
	// Query test document
	var retrieved map[string]interface{}
	err = collection.FindOne(ctx, map[string]interface{}{"title": "Docker Test Document"}).Decode(&retrieved)
	require.NoError(t, err, "Failed to retrieve test document")
	assert.Equal(t, "Docker Test Document", retrieved["title"])
	
	// Clean up
	_, err = collection.DeleteOne(ctx, map[string]interface{}{"title": "Docker Test Document"})
	require.NoError(t, err, "Failed to delete test document")
	
	t.Log("✅ MongoDB Docker connection test passed")
}

// TestDockerResearchRepository tests the research repository with Docker MongoDB
func TestDockerResearchRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	// Connect to MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:password@localhost:27017"))
	require.NoError(t, err, "Failed to connect to MongoDB")
	defer client.Disconnect(ctx)
	
	// Use test database
	db := client.Database("test_research_repository")
	
	// Create research repository
	repo := research.NewMongoResearchRepository(db)
	
	// Create indexes
	err = repo.CreateIndexes(ctx)
	require.NoError(t, err, "Failed to create indexes")
	
	t.Run("Test Research Result Operations", func(t *testing.T) {
		// Create test research result
		researchResult := &models.ResearchResult{
			DocumentID:    primitive.NewObjectID(),
			ResearchQuery: "Docker repository test",
			Status:        models.ResearchStatusCompleted,
			CurrentEvents: []models.CurrentEvent{
				{
					ID:          primitive.NewObjectID(),
					Title:       "Test Event",
					Description: "Docker test event",
					Source:      "Test Source",
					URL:         "https://example.com/test",
					PublishedAt: time.Now(),
					Relevance:   0.8,
					Category:    "test",
				},
			},
			Sources: []models.ResearchSource{
				{
					ID:          primitive.NewObjectID(),
					Type:        models.ResearchSourceTypeNews,
					Title:       "Test Source",
					URL:         "https://example.com/source",
					Credibility: 0.9,
					Relevance:   0.8,
					PublishedAt: time.Now(),
				},
			},
			Confidence: 0.85,
			Metadata:   map[string]interface{}{"test": "docker"},
		}
		
		// Save research result
		err := repo.SaveResearchResult(ctx, researchResult)
		require.NoError(t, err, "Failed to save research result")
		assert.False(t, researchResult.ID.IsZero(), "Research result ID should be set")
		
		// Retrieve research result
		retrieved, err := repo.GetResearchResult(ctx, researchResult.ID.Hex())
		require.NoError(t, err, "Failed to retrieve research result")
		
		// Verify data
		assert.Equal(t, researchResult.DocumentID, retrieved.DocumentID)
		assert.Equal(t, researchResult.ResearchQuery, retrieved.ResearchQuery)
		assert.Equal(t, researchResult.Status, retrieved.Status)
		assert.Equal(t, researchResult.Confidence, retrieved.Confidence)
		assert.Len(t, retrieved.CurrentEvents, 1)
		assert.Len(t, retrieved.Sources, 1)
		
		t.Log("✅ Research result operations test passed")
	})
	
	t.Run("Test Policy Suggestion Operations", func(t *testing.T) {
		// Create test policy suggestion
		suggestion := &models.PolicySuggestion{
			Title:       "Docker Test Policy",
			Description: "Test policy for Docker integration",
			Priority:    models.PolicyPriorityMedium,
			Category:    models.DocumentCategoryTechnology,
			Confidence:  0.8,
			CreatedBy:   primitive.NewObjectID(),
		}
		
		// Save policy suggestion
		err := repo.SavePolicySuggestion(ctx, suggestion)
		require.NoError(t, err, "Failed to save policy suggestion")
		assert.False(t, suggestion.ID.IsZero(), "Policy suggestion ID should be set")
		
		// Retrieve policy suggestion
		retrieved, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
		require.NoError(t, err, "Failed to retrieve policy suggestion")
		
		// Verify data
		assert.Equal(t, suggestion.Title, retrieved.Title)
		assert.Equal(t, suggestion.Description, retrieved.Description)
		assert.Equal(t, suggestion.Priority, retrieved.Priority)
		assert.Equal(t, suggestion.Category, retrieved.Category)
		
		t.Log("✅ Policy suggestion operations test passed")
	})
	
	t.Run("Test Current Event Operations", func(t *testing.T) {
		// Create test current event
		event := &models.CurrentEvent{
			Title:       "Docker Test Event",
			Description: "Test event for Docker integration",
			Source:      "Docker Test Source",
			URL:         "https://example.com/docker-event",
			PublishedAt: time.Now(),
			Relevance:   0.9,
			Category:    "docker",
			Language:    "en",
		}
		
		// Save current event
		err := repo.SaveCurrentEvent(ctx, event)
		require.NoError(t, err, "Failed to save current event")
		assert.False(t, event.ID.IsZero(), "Current event ID should be set")
		
		// Query current events
		filters := research.CurrentEventFilters{
			Category: stringPtr("docker"),
			Limit:    10,
		}
		
		events, err := repo.GetCurrentEvents(ctx, filters)
		require.NoError(t, err, "Failed to query current events")
		assert.GreaterOrEqual(t, len(events), 1, "Should find at least one event")
		
		t.Log("✅ Current event operations test passed")
	})
	
	// Clean up test database
	err = db.Drop(ctx)
	require.NoError(t, err, "Failed to clean up test database")
	
	t.Log("✅ Docker research repository test completed successfully")
}

// TestDockerResearchServiceComponents tests individual research service components
func TestDockerResearchServiceComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	t.Run("Test News API Client", func(t *testing.T) {
		// Test news client creation
		client := research.NewHTTPNewsAPIClient("test-key", "")
		assert.NotNil(t, client, "News client should be created")
		
		// Test API key validation (will fail with test key, but should not panic)
		ctx := context.Background()
		err := client.ValidateAPIKey(ctx)
		assert.Error(t, err, "Should fail with test API key")
		assert.Contains(t, err.Error(), "invalid API key", "Should indicate invalid API key")
		
		t.Log("✅ News API client test passed")
	})
	
	t.Run("Test Gemini LLM Client", func(t *testing.T) {
		// Test LLM client creation
		client := research.NewGeminiLLMClient("test-key", "gemini-1.5-flash")
		assert.NotNil(t, client, "LLM client should be created")
		
		// Test text generation (will fail with test key, but should not panic)
		ctx := context.Background()
		_, err := client.GenerateText(ctx, "test prompt", research.LLMOptions{})
		assert.Error(t, err, "Should fail with test API key")
		
		t.Log("✅ Gemini LLM client test passed")
	})
	
	t.Run("Test Research Service Configuration", func(t *testing.T) {
		// Test default configuration
		service := research.NewLangChainResearchService(nil, nil, nil, nil)
		assert.NotNil(t, service, "Research service should be created with default config")
		
		// Test custom configuration
		config := &research.ResearchConfig{
			MaxConcurrentRequests: 10,
			DefaultLanguage:       "en",
			MaxSourcesPerQuery:    50,
			MinCredibilityScore:   0.8,
			MinRelevanceScore:     0.7,
		}
		
		customService := research.NewLangChainResearchService(nil, nil, nil, config)
		assert.NotNil(t, customService, "Research service should be created with custom config")
		
		t.Log("✅ Research service configuration test passed")
	})
}

// Helper function (if not already defined)
// func stringPtr(s string) *string {
// 	return &s
// }