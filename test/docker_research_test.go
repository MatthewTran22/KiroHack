package test

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/config"
	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/internal/research"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDockerResearchService tests the research service in a Docker environment
func TestDockerResearchService(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	// Set required environment variables for testing
	t.Setenv("JWT_SECRET", "test-jwt-secret-for-docker-integration-testing")
	t.Setenv("LLM_API_KEY", "test-llm-api-key")
	t.Setenv("MONGO_URI", "mongodb://admin:password@localhost:27017")
	t.Setenv("MONGO_DATABASE", "ai_government_consultant_test")
	
	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load configuration")
	
	// Connect to MongoDB (should be running in Docker)
	dbConfig := &database.Config{
		URI:          cfg.Database.MongoURI,
		DatabaseName: cfg.Database.Database + "_test",
	}
	
	mongodb, err := database.NewMongoDB(dbConfig)
	require.NoError(t, err, "Failed to connect to MongoDB")
	defer mongodb.Close(context.Background())
	
	// Test MongoDB connection
	ctx := context.Background()
	err = mongodb.Ping(ctx)
	require.NoError(t, err, "Failed to ping MongoDB")
	
	// Create research repository
	repo := research.NewMongoResearchRepository(mongodb.Database)
	
	// Create indexes
	err = repo.CreateIndexes(ctx)
	require.NoError(t, err, "Failed to create indexes")
	
	// Test saving and retrieving research data
	t.Run("Test Research Data Persistence", func(t *testing.T) {
		// Create test research result
		researchResult := &models.ResearchResult{
			DocumentID:    primitive.NewObjectID(),
			ResearchQuery: "Docker test query",
			Status:        models.ResearchStatusCompleted,
			CurrentEvents: []models.CurrentEvent{
				{
					ID:          primitive.NewObjectID(),
					Title:       "Docker Test Event",
					Description: "Test event for Docker integration",
					Source:      "Test Source",
					URL:         "https://example.com/docker-test",
					PublishedAt: time.Now(),
					Relevance:   0.8,
					Category:    "test",
					Tags:        []string{"docker", "test"},
				},
			},
			PolicyImpacts: []models.PolicyImpact{
				{
					Area:         "Docker Testing",
					Impact:       "Successful integration test",
					Severity:     "low",
					Timeframe:    "immediate",
					Stakeholders: []string{"developers"},
					Mitigation:   []string{"continue testing"},
					Confidence:   0.9,
					Evidence:     []string{"successful execution"},
				},
			},
			Sources: []models.ResearchSource{
				{
					ID:          primitive.NewObjectID(),
					Type:        models.ResearchSourceTypeNews,
					Title:       "Docker Test Source",
					URL:         "https://example.com/docker-source",
					Author:      "Test Author",
					PublishedAt: time.Now(),
					Credibility: 0.8,
					Relevance:   0.7,
					Content:     "Docker integration test content",
					Summary:     "Test summary",
					Keywords:    []string{"docker", "integration"},
					Language:    "en",
				},
			},
			Confidence:     0.85,
			ProcessingTime: 2 * time.Second,
			Metadata:       map[string]interface{}{"test": "docker"},
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
		assert.Len(t, retrieved.PolicyImpacts, 1)
		assert.Len(t, retrieved.Sources, 1)
		
		// Test current event persistence
		event := &researchResult.CurrentEvents[0]
		err = repo.SaveCurrentEvent(ctx, event)
		require.NoError(t, err, "Failed to save current event")
		
		// Test research source persistence
		source := &researchResult.Sources[0]
		err = repo.SaveResearchSource(ctx, source)
		require.NoError(t, err, "Failed to save research source")
		
		// Query current events
		eventFilters := research.CurrentEventFilters{
			Category: stringPtr("test"),
			Limit:    10,
		}
		events, err := repo.GetCurrentEvents(ctx, eventFilters)
		require.NoError(t, err, "Failed to query current events")
		assert.GreaterOrEqual(t, len(events), 1, "Should find at least one event")
		
		// Query research sources
		sourceFilters := research.ResearchSourceFilters{
			Type:  &[]models.ResearchSourceType{models.ResearchSourceTypeNews}[0],
			Limit: 10,
		}
		sources, err := repo.GetResearchSources(ctx, sourceFilters)
		require.NoError(t, err, "Failed to query research sources")
		assert.GreaterOrEqual(t, len(sources), 1, "Should find at least one source")
	})
	
	// Test policy suggestion persistence
	t.Run("Test Policy Suggestion Persistence", func(t *testing.T) {
		suggestion := &models.PolicySuggestion{
			Title:       "Docker Test Policy",
			Description: "Test policy suggestion for Docker integration",
			Rationale:   "Testing Docker environment",
			Priority:    models.PolicyPriorityMedium,
			Category:    models.DocumentCategoryTechnology,
			Tags:        []string{"docker", "testing"},
			Confidence:  0.8,
			CreatedBy:   primitive.NewObjectID(),
			Implementation: models.ImplementationPlan{
				Steps:          []string{"step1", "step2"},
				Timeline:       "1 week",
				Resources:      []string{"docker", "testing"},
				Stakeholders:   []string{"developers"},
				Dependencies:   []string{"mongodb"},
				Milestones:     []string{"successful test"},
				RiskFactors:    []string{"integration issues"},
				SuccessMetrics: []string{"tests pass"},
			},
		}
		
		// Save policy suggestion
		err := repo.SavePolicySuggestion(ctx, suggestion)
		require.NoError(t, err, "Failed to save policy suggestion")
		assert.False(t, suggestion.ID.IsZero(), "Policy suggestion ID should be set")
		assert.Equal(t, "draft", suggestion.Status, "Default status should be draft")
		
		// Retrieve policy suggestion
		retrieved, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
		require.NoError(t, err, "Failed to retrieve policy suggestion")
		
		// Verify data
		assert.Equal(t, suggestion.Title, retrieved.Title)
		assert.Equal(t, suggestion.Description, retrieved.Description)
		assert.Equal(t, suggestion.Priority, retrieved.Priority)
		assert.Equal(t, suggestion.Category, retrieved.Category)
		assert.Equal(t, suggestion.Confidence, retrieved.Confidence)
		
		// Test status update
		reviewNotes := "Docker integration test approved"
		err = repo.UpdatePolicySuggestionStatus(ctx, suggestion.ID.Hex(), "approved", &reviewNotes)
		require.NoError(t, err, "Failed to update policy suggestion status")
		
		// Verify status update
		updated, err := repo.GetPolicySuggestion(ctx, suggestion.ID.Hex())
		require.NoError(t, err, "Failed to retrieve updated policy suggestion")
		assert.Equal(t, "approved", updated.Status)
		assert.Equal(t, reviewNotes, *updated.ReviewNotes)
		assert.NotNil(t, updated.ApprovedAt)
		
		// Query by category
		suggestions, err := repo.GetPolicySuggestionsByCategory(ctx, models.DocumentCategoryTechnology)
		require.NoError(t, err, "Failed to query policy suggestions by category")
		assert.GreaterOrEqual(t, len(suggestions), 1, "Should find at least one suggestion")
	})
	
	// Test configuration loading
	t.Run("Test Research Configuration", func(t *testing.T) {
		// Verify research configuration is loaded correctly
		assert.NotEmpty(t, cfg.Research.NewsAPIBaseURL, "News API base URL should be set")
		assert.NotEmpty(t, cfg.Research.LLMModel, "LLM model should be set")
		assert.Greater(t, cfg.Research.MaxConcurrentRequests, 0, "Max concurrent requests should be positive")
		assert.Greater(t, cfg.Research.RequestTimeout, 0, "Request timeout should be positive")
		assert.NotEmpty(t, cfg.Research.DefaultLanguage, "Default language should be set")
		assert.Greater(t, cfg.Research.MaxSourcesPerQuery, 0, "Max sources per query should be positive")
		assert.GreaterOrEqual(t, cfg.Research.MinCredibilityScore, 0.0, "Min credibility score should be non-negative")
		assert.GreaterOrEqual(t, cfg.Research.MinRelevanceScore, 0.0, "Min relevance score should be non-negative")
		
		t.Logf("Research Configuration:")
		t.Logf("  News API Base URL: %s", cfg.Research.NewsAPIBaseURL)
		t.Logf("  LLM Model: %s", cfg.Research.LLMModel)
		t.Logf("  Max Concurrent Requests: %d", cfg.Research.MaxConcurrentRequests)
		t.Logf("  Request Timeout: %d seconds", cfg.Research.RequestTimeout)
		t.Logf("  Default Language: %s", cfg.Research.DefaultLanguage)
		t.Logf("  Max Sources Per Query: %d", cfg.Research.MaxSourcesPerQuery)
		t.Logf("  Min Credibility Score: %.2f", cfg.Research.MinCredibilityScore)
		t.Logf("  Min Relevance Score: %.2f", cfg.Research.MinRelevanceScore)
	})
}

// TestDockerHealthCheck tests the application health in Docker environment
func TestDockerHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	// Set required environment variables for testing
	t.Setenv("JWT_SECRET", "test-jwt-secret-for-docker-health-testing")
	t.Setenv("LLM_API_KEY", "test-llm-api-key")
	t.Setenv("MONGO_URI", "mongodb://admin:password@localhost:27017")
	t.Setenv("MONGO_DATABASE", "ai_government_consultant_health_test")
	
	// Load configuration
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load configuration")
	
	// Test database connections
	t.Run("Test MongoDB Connection", func(t *testing.T) {
		dbConfig := &database.Config{
			URI:          cfg.Database.MongoURI,
			DatabaseName: cfg.Database.Database + "_health_test",
		}
		
		mongodb, err := database.NewMongoDB(dbConfig)
		require.NoError(t, err, "Failed to connect to MongoDB")
		defer mongodb.Close(context.Background())
		
		ctx := context.Background()
		err = mongodb.HealthCheck(ctx)
		assert.NoError(t, err, "MongoDB health check should pass")
		
		t.Log("✅ MongoDB connection healthy")
	})
	
	// Note: Redis health check would require Redis client setup
	// For now, we'll just verify the configuration
	t.Run("Test Redis Configuration", func(t *testing.T) {
		assert.NotEmpty(t, cfg.Redis.Host, "Redis host should be configured")
		assert.NotEmpty(t, cfg.Redis.Port, "Redis port should be configured")
		
		t.Logf("✅ Redis configuration: %s:%s", cfg.Redis.Host, cfg.Redis.Port)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}