package consultation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Integration test setup
func setupIntegrationTest(t *testing.T) (*Service, *SessionManager, *mongo.Database, func()) {
	// Skip integration tests if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TEST=true to run)")
	}

	// Connect to test MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_TEST_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Use a test database
	testDB := client.Database("ai_government_consultant_test")

	// Connect to test Redis
	redisAddr := os.Getenv("REDIS_TEST_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_TEST_PASSWORD")
	if redisPassword == "" {
		redisPassword = "testpassword"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       1, // Use test database
	})

	// Test Redis connection
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create mock Gemini server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic response time
		time.Sleep(100 * time.Millisecond)

		response := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: generateRealisticConsultationResponse()},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     150,
				CandidatesTokenCount: 300,
				TotalTokenCount:      450,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	// Create embedding service
	embeddingConfig := &embedding.Config{
		GeminiAPIKey: "test-api-key",
		GeminiURL:    server.URL + "/embedding",
		MongoDB:      testDB,
		Redis:        redisClient,
		Logger:       &MockLogger{},
	}

	embeddingService, err := embedding.NewService(embeddingConfig)
	if err != nil {
		t.Fatalf("Failed to create embedding service: %v", err)
	}

	// Create consultation service
	consultationConfig := &Config{
		GeminiAPIKey:     "test-api-key",
		GeminiURL:        server.URL,
		MongoDB:          testDB,
		Redis:            redisClient,
		EmbeddingService: embeddingService,
		Logger:           &MockLogger{},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
	}

	consultationService, err := NewService(consultationConfig)
	if err != nil {
		t.Fatalf("Failed to create consultation service: %v", err)
	}

	// Create session manager
	sessionManager := NewSessionManager(testDB, consultationService)

	// Cleanup function
	cleanup := func() {
		server.Close()
		
		// Clean up test data
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		testDB.Drop(ctx)
		redisClient.FlushDB(ctx)
		redisClient.Close()
		client.Disconnect(ctx)
	}

	return consultationService, sessionManager, testDB, cleanup
}

func generateRealisticConsultationResponse() string {
	return `
Executive Summary:
Based on comprehensive analysis of current government operations and best practices, this consultation provides strategic recommendations for improving organizational effectiveness and citizen service delivery.

Policy Analysis:
The current operational framework shows several areas for improvement:
- Process standardization opportunities identified
- Technology integration gaps present
- Stakeholder engagement mechanisms need enhancement
- Compliance monitoring requires strengthening

Key Findings:
- Current processes lack standardization across departments
- Manual workflows create bottlenecks and inefficiencies
- Limited data sharing between systems impacts decision-making
- Citizen feedback mechanisms are underutilized

Recommendations:

1. Implement Standardized Process Framework
Develop and deploy a comprehensive process standardization framework that establishes consistent procedures across all departments. This framework should include clear documentation, training materials, and compliance monitoring mechanisms to ensure uniform implementation.

2. Deploy Integrated Technology Platform
Establish a unified technology platform that enables seamless data sharing and workflow automation across departments. This platform should include modern APIs, secure data exchange protocols, and user-friendly interfaces for staff and citizens.

3. Enhance Stakeholder Engagement Program
Create a robust stakeholder engagement program that includes regular consultation sessions, feedback collection mechanisms, and transparent communication channels. This program should ensure all stakeholders have meaningful input into policy development and implementation.

4. Establish Performance Monitoring System
Implement a comprehensive performance monitoring and evaluation system that tracks key metrics, identifies improvement opportunities, and provides real-time insights for decision-making. This system should include automated reporting and alert mechanisms.

Risk Assessment:
Overall Risk Level: Medium

Key Risk Factors:
- Implementation complexity may cause delays
- Staff resistance to change could impact adoption
- Budget constraints might limit scope
- Technical integration challenges possible

Mitigation Strategies:
- Phased implementation approach to manage complexity
- Comprehensive change management and training programs
- Stakeholder engagement to build support
- Technical pilot programs to validate approaches

Implementation Plan:

Phase 1: Planning and Preparation (4-6 weeks)
- Establish project governance structure
- Conduct detailed stakeholder analysis
- Develop comprehensive project plan
- Secure necessary resources and approvals

Phase 2: Framework Development (8-12 weeks)
- Design standardized process framework
- Develop technology platform specifications
- Create stakeholder engagement protocols
- Build performance monitoring system design

Phase 3: Pilot Implementation (6-8 weeks)
- Deploy pilot programs in selected departments
- Test technology platform integration
- Validate stakeholder engagement approaches
- Refine monitoring and evaluation mechanisms

Phase 4: Full Deployment (12-16 weeks)
- Roll out standardized processes organization-wide
- Deploy complete technology platform
- Launch comprehensive stakeholder engagement program
- Activate full performance monitoring system

Phase 5: Optimization and Continuous Improvement (Ongoing)
- Monitor performance metrics and outcomes
- Collect feedback and identify improvement opportunities
- Implement refinements and enhancements
- Maintain stakeholder engagement and communication

Next Steps:
1. Establish executive sponsorship and project governance
2. Conduct detailed current state assessment
3. Develop detailed implementation roadmap
4. Secure budget approval and resource allocation
5. Begin stakeholder engagement and communication planning

Success Metrics:
- Process standardization completion rate: 95%
- Technology platform adoption rate: 90%
- Stakeholder satisfaction score: 4.0/5.0
- Performance improvement targets: 20% efficiency gain
- Compliance monitoring coverage: 100%

This comprehensive approach ensures systematic improvement while managing risks and maintaining operational continuity throughout the transformation process.
`
}

func TestFullConsultationWorkflow(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test user
	userID := primitive.NewObjectID()

	// Test different consultation types
	consultationTypes := []models.ConsultationType{
		models.ConsultationTypePolicy,
		models.ConsultationTypeStrategy,
		models.ConsultationTypeOperations,
		models.ConsultationTypeTechnology,
	}

	queries := map[models.ConsultationType]string{
		models.ConsultationTypePolicy:     "How should we implement a new data privacy policy that complies with government regulations?",
		models.ConsultationTypeStrategy:   "What strategic approach should we take for digital transformation in our agency?",
		models.ConsultationTypeOperations: "How can we improve operational efficiency in our document processing workflows?",
		models.ConsultationTypeTechnology: "What technology stack should we use for our new citizen services platform?",
	}

	for _, consultationType := range consultationTypes {
		t.Run(string(consultationType), func(t *testing.T) {
			// Create consultation request
			request := &ConsultationRequest{
				Query:  queries[consultationType],
				Type:   consultationType,
				UserID: userID,
				Context: models.ConsultationContext{
					RelatedDocuments: []primitive.ObjectID{},
					UserContext:      map[string]interface{}{"department": "IT"},
				},
				MaxSources:          5,
				ConfidenceThreshold: 0.7,
			}

			// Create session
			session, err := sessionManager.CreateSession(ctx, request)
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			if session.ID.IsZero() {
				t.Error("Session ID should not be zero")
			}

			if session.Status != models.SessionStatusActive {
				t.Errorf("Expected session status to be active, got %s", session.Status)
			}

			// Process session
			processedSession, err := sessionManager.ProcessSession(ctx, session.ID)
			if err != nil {
				t.Fatalf("Failed to process session: %v", err)
			}

			// Validate processed session
			if processedSession.Status != models.SessionStatusCompleted {
				t.Errorf("Expected session status to be completed, got %s", processedSession.Status)
			}

			if processedSession.Response == nil {
				t.Fatal("Session response should not be nil")
			}

			response := processedSession.Response

			// Validate response structure
			if len(response.Recommendations) == 0 {
				t.Error("Response should contain recommendations")
			}

			if response.Analysis.Summary == "" {
				t.Error("Response should contain analysis summary")
			}

			if response.ConfidenceScore <= 0 {
				t.Error("Confidence score should be positive")
			}

			if response.ProcessingTime <= 0 {
				t.Error("Processing time should be positive")
			}

			// Validate recommendations
			for i, rec := range response.Recommendations {
				if rec.Title == "" {
					t.Errorf("Recommendation %d should have a title", i)
				}

				if rec.Description == "" {
					t.Errorf("Recommendation %d should have a description", i)
				}

				if len(rec.Implementation.Steps) == 0 {
					t.Errorf("Recommendation %d should have implementation steps", i)
				}

				if rec.ConfidenceScore <= 0 {
					t.Errorf("Recommendation %d should have positive confidence score", i)
				}
			}

			// Test session retrieval
			retrievedSession, err := sessionManager.GetSession(ctx, session.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve session: %v", err)
			}

			if retrievedSession.ID != session.ID {
				t.Error("Retrieved session ID should match original")
			}

			if retrievedSession.Response == nil {
				t.Error("Retrieved session should have response")
			}
		})
	}
}

func TestSessionManagement(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Create multiple sessions
	sessionIDs := make([]primitive.ObjectID, 3)
	for i := 0; i < 3; i++ {
		request := &ConsultationRequest{
			Query:  "Test query " + string(rune(i+'1')),
			Type:   models.ConsultationTypePolicy,
			UserID: userID,
			Context: models.ConsultationContext{
				RelatedDocuments: []primitive.ObjectID{},
				UserContext:      make(map[string]interface{}),
			},
		}

		session, err := sessionManager.CreateSession(ctx, request)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}

		sessionIDs[i] = session.ID

		// Process some sessions
		if i < 2 {
			_, err = sessionManager.ProcessSession(ctx, session.ID)
			if err != nil {
				t.Fatalf("Failed to process session %d: %v", i, err)
			}
		}
	}

	// Test getting user sessions
	userSessions, err := sessionManager.GetUserSessions(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(userSessions) != 3 {
		t.Errorf("Expected 3 user sessions, got %d", len(userSessions))
	}

	// Test session search
	searchResults, err := sessionManager.SearchSessions(ctx, "Test query", &userID, nil, 10)
	if err != nil {
		t.Fatalf("Failed to search sessions: %v", err)
	}

	if len(searchResults) != 3 {
		t.Errorf("Expected 3 search results, got %d", len(searchResults))
	}

	// Test getting sessions by type
	typeSessions, err := sessionManager.GetSessionsByType(ctx, models.ConsultationTypePolicy, 10)
	if err != nil {
		t.Fatalf("Failed to get sessions by type: %v", err)
	}

	if len(typeSessions) != 3 {
		t.Errorf("Expected 3 sessions of policy type, got %d", len(typeSessions))
	}

	// Test session statistics
	stats, err := sessionManager.GetSessionStats(ctx, &userID)
	if err != nil {
		t.Fatalf("Failed to get session stats: %v", err)
	}

	if stats.TotalSessions != 3 {
		t.Errorf("Expected 3 total sessions in stats, got %d", stats.TotalSessions)
	}

	if stats.CompletedSessions != 2 {
		t.Errorf("Expected 2 completed sessions in stats, got %d", stats.CompletedSessions)
	}

	// Test session deletion
	err = sessionManager.DeleteSession(ctx, sessionIDs[0], userID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify session was deleted
	_, err = sessionManager.GetSession(ctx, sessionIDs[0])
	if err == nil {
		t.Error("Expected error when getting deleted session")
	}
}

func TestConcurrentConsultations(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Test concurrent consultations
	concurrency := 5
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			request := &ConsultationRequest{
				Query:  "Concurrent test query " + string(rune(index+'1')),
				Type:   models.ConsultationTypePolicy,
				UserID: userID,
				Context: models.ConsultationContext{
					RelatedDocuments: []primitive.ObjectID{},
					UserContext:      make(map[string]interface{}),
				},
			}

			session, err := sessionManager.CreateSession(ctx, request)
			if err != nil {
				results <- err
				return
			}

			_, err = sessionManager.ProcessSession(ctx, session.ID)
			results <- err
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent consultation %d failed: %v", i, err)
		}
	}

	// Verify all sessions were created
	userSessions, err := sessionManager.GetUserSessions(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(userSessions) != concurrency {
		t.Errorf("Expected %d concurrent sessions, got %d", concurrency, len(userSessions))
	}
}

func TestErrorHandling(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Test invalid session processing
	invalidSessionID := primitive.NewObjectID()
	_, err := sessionManager.ProcessSession(ctx, invalidSessionID)
	if err == nil {
		t.Error("Expected error when processing non-existent session")
	}

	// Test invalid user session retrieval
	invalidUserID := primitive.NewObjectID()
	sessions, err := sessionManager.GetUserSessions(ctx, invalidUserID, 10, 0)
	if err != nil {
		t.Fatalf("GetUserSessions should not error for non-existent user: %v", err)
	}
	if len(sessions) != 0 {
		t.Error("Should return empty sessions for non-existent user")
	}

	// Test session deletion with wrong user
	request := &ConsultationRequest{
		Query:  "Test query for error handling",
		Type:   models.ConsultationTypePolicy,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	session, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	wrongUserID := primitive.NewObjectID()
	err = sessionManager.DeleteSession(ctx, session.ID, wrongUserID)
	if err == nil {
		t.Error("Expected error when deleting session with wrong user ID")
	}
}

func TestPerformanceMetrics(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	request := &ConsultationRequest{
		Query:  "Performance test query with sufficient length to test processing time",
		Type:   models.ConsultationTypePolicy,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	// Measure session creation time
	startTime := time.Now()
	session, err := sessionManager.CreateSession(ctx, request)
	creationTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Logf("Session creation time: %v", creationTime)

	// Measure processing time
	startTime = time.Now()
	processedSession, err := sessionManager.ProcessSession(ctx, session.ID)
	totalProcessingTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to process session: %v", err)
	}

	t.Logf("Total processing time: %v", totalProcessingTime)
	t.Logf("Reported processing time: %v", processedSession.Response.ProcessingTime)

	// Validate performance expectations
	if creationTime > 1*time.Second {
		t.Errorf("Session creation took too long: %v", creationTime)
	}

	if totalProcessingTime > 30*time.Second {
		t.Errorf("Session processing took too long: %v", totalProcessingTime)
	}

	// Validate response quality metrics
	response := processedSession.Response
	if response.ConfidenceScore < 0.5 {
		t.Errorf("Confidence score too low: %f", response.ConfidenceScore)
	}

	if len(response.Recommendations) < 1 {
		t.Error("Should have at least 1 recommendation")
	}

	if len(response.Analysis.Summary) < 100 {
		t.Error("Analysis summary should be substantial")
	}
}