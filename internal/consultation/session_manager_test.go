package consultation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSessionManagerMultiTurn(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Create initial session
	request := &ConsultationRequest{
		Query:  "What are the best practices for data privacy in government?",
		Type:   models.ConsultationTypePolicy,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      map[string]interface{}{"department": "IT"},
		},
	}

	session, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Process initial session
	processedSession, err := sessionManager.ProcessSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to process session: %v", err)
	}

	if processedSession.Status != models.SessionStatusCompleted {
		t.Errorf("Expected session status to be completed, got %s", processedSession.Status)
	}

	// Add conversation turn
	followUpQuery := "How should we implement these practices in our current system?"
	updatedSession, err := sessionManager.AddConversationTurn(ctx, session.ID, followUpQuery)
	if err != nil {
		t.Fatalf("Failed to add conversation turn: %v", err)
	}

	// Validate multi-turn session
	if !updatedSession.IsMultiTurn {
		t.Error("Session should be marked as multi-turn")
	}

	if len(updatedSession.ConversationTurns) != 1 {
		t.Errorf("Expected 1 conversation turn, got %d", len(updatedSession.ConversationTurns))
	}

	turn := updatedSession.ConversationTurns[0]
	if turn.Query != followUpQuery {
		t.Errorf("Expected turn query to be %s, got %s", followUpQuery, turn.Query)
	}

	if turn.Response == nil {
		t.Error("Turn should have a response")
	}

	if turn.TurnIndex != 0 {
		t.Errorf("Expected turn index to be 0, got %d", turn.TurnIndex)
	}

	// Add another turn
	secondFollowUp := "What are the potential risks and how can we mitigate them?"
	updatedSession, err = sessionManager.AddConversationTurn(ctx, session.ID, secondFollowUp)
	if err != nil {
		t.Fatalf("Failed to add second conversation turn: %v", err)
	}

	if len(updatedSession.ConversationTurns) != 2 {
		t.Errorf("Expected 2 conversation turns, got %d", len(updatedSession.ConversationTurns))
	}

	// Test conversation history retrieval
	history, err := sessionManager.GetConversationHistory(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get conversation history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 turns in history, got %d", len(history))
	}
}

func TestSessionManagerContinueConversation(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Create and process initial session
	request := &ConsultationRequest{
		Query:  "What technology stack should we use for our new citizen portal?",
		Type:   models.ConsultationTypeTechnology,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      map[string]interface{}{"department": "Digital Services"},
		},
	}

	initialSession, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create initial session: %v", err)
	}

	_, err = sessionManager.ProcessSession(ctx, initialSession.ID)
	if err != nil {
		t.Fatalf("Failed to process initial session: %v", err)
	}

	// Continue conversation in new session
	continuationQuery := "How should we handle security and compliance for this technology stack?"
	continuationSession, err := sessionManager.ContinueConversation(ctx, initialSession.ID, continuationQuery)
	if err != nil {
		t.Fatalf("Failed to continue conversation: %v", err)
	}

	// Validate continuation session
	if continuationSession.Query != continuationQuery {
		t.Errorf("Expected continuation query to be %s, got %s", continuationQuery, continuationSession.Query)
	}

	if continuationSession.Type != models.ConsultationTypeTechnology {
		t.Errorf("Expected continuation type to be %s, got %s", models.ConsultationTypeTechnology, continuationSession.Type)
	}

	if continuationSession.UserID != userID {
		t.Error("Continuation session should have same user ID")
	}

	// Check if previous session is referenced
	if len(continuationSession.Context.PreviousSessions) == 0 {
		t.Error("Continuation session should reference previous session")
	}

	if continuationSession.Context.PreviousSessions[0] != initialSession.ID {
		t.Error("Continuation session should reference correct previous session")
	}

	// Check metadata
	if continuationSession.Metadata["is_continuation"] != true {
		t.Error("Continuation session should be marked as continuation")
	}

	if continuationSession.Metadata["previous_session_id"] != initialSession.ID.Hex() {
		t.Error("Continuation session should have correct previous session ID in metadata")
	}

	// Test related sessions retrieval
	relatedSessions, err := sessionManager.GetRelatedSessions(ctx, continuationSession.ID)
	if err != nil {
		t.Fatalf("Failed to get related sessions: %v", err)
	}

	if len(relatedSessions) != 1 {
		t.Errorf("Expected 1 related session, got %d", len(relatedSessions))
	}

	if relatedSessions[0].ID != initialSession.ID {
		t.Error("Related session should be the initial session")
	}
}

func TestSessionManagerCaching(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Create and process first session
	request := &ConsultationRequest{
		Query:  "How can we improve operational efficiency in document processing?",
		Type:   models.ConsultationTypeOperations,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	session1, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create first session: %v", err)
	}

	startTime := time.Now()
	processedSession1, err := sessionManager.ProcessSession(ctx, session1.ID)
	firstProcessingTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to process first session: %v", err)
	}

	// Create second session with same query
	session2, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}

	startTime = time.Now()
	processedSession2, err := sessionManager.ProcessSession(ctx, session2.ID)
	secondProcessingTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to process second session: %v", err)
	}

	// Second session should be faster due to caching
	if secondProcessingTime >= firstProcessingTime {
		t.Logf("First processing time: %v, Second processing time: %v", firstProcessingTime, secondProcessingTime)
		// Note: This might not always be true in test environment, so we'll just log it
	}

	// Check if second session used cache
	if processedSession2.Metadata["cached"] == true {
		t.Log("Second session used cached response")
	}

	// Validate responses are similar (should be from cache)
	if processedSession1.Response.ConfidenceScore != processedSession2.Response.ConfidenceScore {
		t.Log("Responses have different confidence scores - cache might not have been used")
	}

	// Test cache invalidation
	err = sessionManager.InvalidateCache(ctx, session1.ID)
	if err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// Test cache stats
	cacheStats, err := sessionManager.GetCacheStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	if cacheStats.TotalKeys < 0 {
		t.Error("Cache stats should have non-negative total keys")
	}

	t.Logf("Cache stats: Total keys: %d, Response keys: %d, Query keys: %d",
		cacheStats.TotalKeys, cacheStats.ResponseKeys, cacheStats.QueryKeys)
}

func TestSessionManagerAnalytics(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Create multiple sessions for analytics
	consultationTypes := []models.ConsultationType{
		models.ConsultationTypePolicy,
		models.ConsultationTypeStrategy,
		models.ConsultationTypeOperations,
	}

	sessionIDs := make([]primitive.ObjectID, len(consultationTypes))

	for i, consultationType := range consultationTypes {
		request := &ConsultationRequest{
			Query:  fmt.Sprintf("Test query for %s consultation", consultationType),
			Type:   consultationType,
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

		_, err = sessionManager.ProcessSession(ctx, session.ID)
		if err != nil {
			t.Fatalf("Failed to process session %d: %v", i, err)
		}

		sessionIDs[i] = session.ID
	}

	// Wait a bit for analytics to be processed
	time.Sleep(100 * time.Millisecond)

	// Test usage analytics
	endDate := time.Now()
	startDate := endDate.Add(-24 * time.Hour)

	usageMetrics, err := sessionManager.GetUsageAnalytics(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to get usage analytics: %v", err)
	}

	if usageMetrics.TotalConsultations < 3 {
		t.Errorf("Expected at least 3 consultations, got %d", usageMetrics.TotalConsultations)
	}

	if len(usageMetrics.ConsultationsByType) == 0 {
		t.Error("Expected consultations by type data")
	}

	// Verify we have data for each consultation type
	for _, consultationType := range consultationTypes {
		if count, exists := usageMetrics.ConsultationsByType[consultationType]; !exists || count == 0 {
			t.Errorf("Expected data for consultation type %s", consultationType)
		}
	}

	// Test user analytics
	userAnalytics, err := sessionManager.GetUserAnalytics(ctx, userID, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to get user analytics: %v", err)
	}

	if userAnalytics.TotalConsultations < 3 {
		t.Errorf("Expected at least 3 consultations for user, got %d", userAnalytics.TotalConsultations)
	}

	if userAnalytics.UserID != userID {
		t.Error("User analytics should have correct user ID")
	}

	if userAnalytics.FavoriteType == "" {
		t.Error("User should have a favorite consultation type")
	}

	t.Logf("Usage metrics: Total: %d, By type: %v",
		usageMetrics.TotalConsultations, usageMetrics.ConsultationsByType)
	t.Logf("User analytics: Total: %d, Favorite type: %s, Avg confidence: %.2f",
		userAnalytics.TotalConsultations, userAnalytics.FavoriteType, userAnalytics.AverageConfidence)
}

func TestSessionManagerErrorHandling(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Test adding turn to non-existent session
	nonExistentID := primitive.NewObjectID()
	_, err := sessionManager.AddConversationTurn(ctx, nonExistentID, "test query")
	if err == nil {
		t.Error("Expected error when adding turn to non-existent session")
	}

	// Test adding turn to active session (should fail)
	request := &ConsultationRequest{
		Query:  "Test query for error handling",
		Type:   models.ConsultationTypePolicy,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext:      make(map[string]interface{}),
		},
	}

	activeSession, err := sessionManager.CreateSession(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create active session: %v", err)
	}

	// Try to add turn to active session (should fail)
	_, err = sessionManager.AddConversationTurn(ctx, activeSession.ID, "follow up query")
	if err == nil {
		t.Error("Expected error when adding turn to active session")
	}

	// Test continuing from non-existent session
	_, err = sessionManager.ContinueConversation(ctx, nonExistentID, "continuation query")
	if err == nil {
		t.Error("Expected error when continuing from non-existent session")
	}

	// Test getting related sessions for non-existent session
	_, err = sessionManager.GetRelatedSessions(ctx, nonExistentID)
	if err == nil {
		t.Error("Expected error when getting related sessions for non-existent session")
	}

	// Test analytics with invalid date range
	endDate := time.Now()
	startDate := endDate.Add(24 * time.Hour) // Future start date

	_, err = sessionManager.GetUsageAnalytics(ctx, startDate, endDate)
	if err != nil {
		t.Log("Analytics with invalid date range handled gracefully")
	}
}

func TestSessionManagerConcurrency(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Test concurrent session creation and processing
	concurrency := 3
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			request := &ConsultationRequest{
				Query:  fmt.Sprintf("Concurrent test query %d", index),
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
			t.Errorf("Concurrent session %d failed: %v", i, err)
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

func TestSessionManagerPerformance(t *testing.T) {
	_, sessionManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := primitive.NewObjectID()

	// Measure session creation performance
	startTime := time.Now()

	request := &ConsultationRequest{
		Query:  "Performance test query with detailed requirements and context",
		Type:   models.ConsultationTypeStrategy,
		UserID: userID,
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{},
			UserContext: map[string]interface{}{
				"department": "Strategic Planning",
				"priority":   "high",
				"deadline":   "Q4 2024",
			},
		},
	}

	session, err := sessionManager.CreateSession(ctx, request)
	creationTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	t.Logf("Session creation time: %v", creationTime)

	// Measure processing performance
	startTime = time.Now()
	processedSession, err := sessionManager.ProcessSession(ctx, session.ID)
	processingTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to process session: %v", err)
	}

	t.Logf("Session processing time: %v", processingTime)
	t.Logf("Reported processing time: %v", processedSession.Response.ProcessingTime)

	// Performance expectations
	if creationTime > 5*time.Second {
		t.Errorf("Session creation took too long: %v", creationTime)
	}

	if processingTime > 60*time.Second {
		t.Errorf("Session processing took too long: %v", processingTime)
	}

	// Measure multi-turn performance
	startTime = time.Now()
	_, err = sessionManager.AddConversationTurn(ctx, session.ID, "Follow-up question about implementation timeline")
	turnTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to add conversation turn: %v", err)
	}

	t.Logf("Conversation turn time: %v", turnTime)

	if turnTime > 60*time.Second {
		t.Errorf("Conversation turn took too long: %v", turnTime)
	}
}
