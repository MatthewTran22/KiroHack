package consultation

import (
	"context"
	"fmt"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SessionManager manages consultation sessions
type SessionManager struct {
	mongodb   *mongo.Database
	service   *Service
	cache     *ConsultationCache
	analytics *AnalyticsService
}

// NewSessionManager creates a new session manager
func NewSessionManager(mongodb *mongo.Database, service *Service) *SessionManager {
	cache := NewConsultationCache(service.redis, service.logger)
	analytics := NewAnalyticsService(mongodb, cache, service.logger)
	return &SessionManager{
		mongodb:   mongodb,
		service:   service,
		cache:     cache,
		analytics: analytics,
	}
}

// CreateSession creates a new consultation session
func (sm *SessionManager) CreateSession(ctx context.Context, request *ConsultationRequest) (*models.ConsultationSession, error) {
	// Validate request
	validator := NewResponseValidator()
	if err := validator.ValidateRequest(request); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Create session
	session := &models.ConsultationSession{
		ID:        primitive.NewObjectID(),
		UserID:    request.UserID,
		Type:      request.Type,
		Query:     request.Query,
		Context:   request.Context,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    models.SessionStatusActive,
		Tags:      []string{},
		Metadata:  make(map[string]interface{}),
	}

	// Store session in database
	collection := sm.mongodb.Collection("consultations")
	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sm.service.logger.Info("Created consultation session", map[string]interface{}{
		"session_id": session.ID.Hex(),
		"user_id":    session.UserID.Hex(),
		"type":       session.Type,
	})

	return session, nil
}

// ProcessSession processes a consultation session and generates a response
func (sm *SessionManager) ProcessSession(ctx context.Context, sessionID primitive.ObjectID) (*models.ConsultationSession, error) {
	// Retrieve session
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	if session.Status != models.SessionStatusActive {
		return nil, fmt.Errorf("session is not active (status: %s)", session.Status)
	}

	// Check cache for similar query first
	if cachedResponse, err := sm.cache.GetCachedQueryResponse(ctx, session.Query, session.Type, session.UserID); err == nil && cachedResponse != nil {
		sm.service.logger.Info("Using cached response for similar query", map[string]interface{}{
			"session_id": sessionID.Hex(),
		})

		// Update session with cached response
		session.Response = cachedResponse
		session.Status = models.SessionStatusCompleted
		session.UpdatedAt = time.Now()
		session.Metadata["cached"] = true
		session.Metadata["confidence_score"] = cachedResponse.ConfidenceScore
		session.Metadata["recommendations_count"] = len(cachedResponse.Recommendations)
		session.Metadata["sources_count"] = len(cachedResponse.Sources)

		if err := sm.updateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session with cached response: %w", err)
		}

		return session, nil
	}

	// Update session status to processing
	if err := sm.updateSessionStatus(ctx, sessionID, models.SessionStatusActive); err != nil {
		sm.service.logger.Error("Failed to update session status", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
	}

	startTime := time.Now()

	// Create consultation request
	consultationRequest := &ConsultationRequest{
		Query:               session.Query,
		Type:                session.Type,
		UserID:              session.UserID,
		Context:             session.Context,
		MaxSources:          10,
		ConfidenceThreshold: 0.7,
	}

	// Process consultation based on type
	var response *models.ConsultationResponse
	switch session.Type {
	case models.ConsultationTypePolicy:
		response, err = sm.service.ConsultPolicy(ctx, consultationRequest)
	case models.ConsultationTypeStrategy:
		response, err = sm.service.ConsultStrategy(ctx, consultationRequest)
	case models.ConsultationTypeOperations:
		response, err = sm.service.ConsultOperations(ctx, consultationRequest)
	case models.ConsultationTypeTechnology:
		response, err = sm.service.ConsultTechnology(ctx, consultationRequest)
	default:
		err = fmt.Errorf("unsupported consultation type: %s", session.Type)
	}

	processingTime := time.Since(startTime)

	if err != nil {
		// Update session with error status
		session.Status = models.SessionStatusFailed
		session.UpdatedAt = time.Now()
		session.Metadata["error"] = err.Error()
		session.Metadata["processing_time"] = processingTime.String()

		if updateErr := sm.updateSession(ctx, session); updateErr != nil {
			sm.service.logger.Error("Failed to update failed session", updateErr, map[string]interface{}{
				"session_id": sessionID.Hex(),
			})
		}

		return nil, fmt.Errorf("consultation processing failed: %w", err)
	}

	// Update response with actual processing time
	response.ProcessingTime = processingTime

	// Cache the response for future similar queries
	if err := sm.cache.CacheQueryResponse(ctx, session.Query, session.Type, session.UserID, response); err != nil {
		sm.service.logger.Error("Failed to cache query response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
	}

	// Cache the session response
	if err := sm.cache.CacheResponse(ctx, sessionID, response); err != nil {
		sm.service.logger.Error("Failed to cache session response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
	}

	// Update session with response
	session.Response = response
	session.Status = models.SessionStatusCompleted
	session.UpdatedAt = time.Now()
	session.Metadata["processing_time"] = processingTime.String()
	session.Metadata["confidence_score"] = response.ConfidenceScore
	session.Metadata["recommendations_count"] = len(response.Recommendations)
	session.Metadata["sources_count"] = len(response.Sources)
	session.Metadata["cached"] = false

	if err := sm.updateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session with response: %w", err)
	}

	// Track usage analytics
	if err := sm.analytics.TrackConsultationUsage(ctx, session); err != nil {
		sm.service.logger.Error("Failed to track consultation usage", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
	}

	sm.service.logger.Info("Completed consultation session", map[string]interface{}{
		"session_id":            sessionID.Hex(),
		"processing_time":       processingTime.String(),
		"confidence_score":      response.ConfidenceScore,
		"recommendations_count": len(response.Recommendations),
	})

	return session, nil
}

// GetSession retrieves a consultation session by ID
func (sm *SessionManager) GetSession(ctx context.Context, sessionID primitive.ObjectID) (*models.ConsultationSession, error) {
	collection := sm.mongodb.Collection("consultations")

	var session models.ConsultationSession
	err := collection.FindOne(ctx, bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	return &session, nil
}

// GetUserSessions retrieves all sessions for a user
func (sm *SessionManager) GetUserSessions(ctx context.Context, userID primitive.ObjectID, limit int, offset int) ([]*models.ConsultationSession, error) {
	collection := sm.mongodb.Collection("consultations")

	// Build query options
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by creation time, newest first
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.ConsultationSession
	for cursor.Next(ctx) {
		var session models.ConsultationSession
		if err := cursor.Decode(&session); err != nil {
			sm.service.logger.Error("Failed to decode session", err, map[string]interface{}{
				"user_id": userID.Hex(),
			})
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// SearchSessions searches for sessions based on criteria
func (sm *SessionManager) SearchSessions(ctx context.Context, query string, userID *primitive.ObjectID, consultationType *models.ConsultationType, limit int) ([]*models.ConsultationSession, error) {
	collection := sm.mongodb.Collection("consultations")

	// Build search filter
	filter := bson.M{}

	if userID != nil {
		filter["user_id"] = *userID
	}

	if consultationType != nil {
		filter["type"] = *consultationType
	}

	if query != "" {
		// Text search on query and response content
		filter["$or"] = []bson.M{
			{"query": bson.M{"$regex": query, "$options": "i"}},
			{"response.analysis.summary": bson.M{"$regex": query, "$options": "i"}},
			{"response.recommendations.title": bson.M{"$regex": query, "$options": "i"}},
			{"response.recommendations.description": bson.M{"$regex": query, "$options": "i"}},
		}
	}

	// Build query options
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.ConsultationSession
	for cursor.Next(ctx) {
		var session models.ConsultationSession
		if err := cursor.Decode(&session); err != nil {
			sm.service.logger.Error("Failed to decode session in search", err, nil)
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetSessionsByType retrieves sessions by consultation type
func (sm *SessionManager) GetSessionsByType(ctx context.Context, consultationType models.ConsultationType, limit int) ([]*models.ConsultationSession, error) {
	collection := sm.mongodb.Collection("consultations")

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(ctx, bson.M{"type": consultationType}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions by type: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.ConsultationSession
	for cursor.Next(ctx) {
		var session models.ConsultationSession
		if err := cursor.Decode(&session); err != nil {
			sm.service.logger.Error("Failed to decode session by type", err, map[string]interface{}{
				"type": consultationType,
			})
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// DeleteSession deletes a consultation session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID primitive.ObjectID, userID primitive.ObjectID) error {
	collection := sm.mongodb.Collection("consultations")

	// Ensure user can only delete their own sessions
	filter := bson.M{
		"_id":     sessionID,
		"user_id": userID,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("session not found or access denied")
	}

	sm.service.logger.Info("Deleted consultation session", map[string]interface{}{
		"session_id": sessionID.Hex(),
		"user_id":    userID.Hex(),
	})

	return nil
}

// GetSessionStats retrieves statistics about consultation sessions
func (sm *SessionManager) GetSessionStats(ctx context.Context, userID *primitive.ObjectID) (*SessionStats, error) {
	collection := sm.mongodb.Collection("consultations")

	// Build match stage
	matchStage := bson.M{}
	if userID != nil {
		matchStage["user_id"] = *userID
	}

	pipeline := []bson.M{
		{"$match": matchStage},
		{
			"$group": bson.M{
				"_id":            nil,
				"total_sessions": bson.M{"$sum": 1},
				"completed_sessions": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", "completed"}},
							1,
							0,
						},
					},
				},
				"avg_confidence": bson.M{
					"$avg": "$response.confidence_score",
				},
				"types": bson.M{"$addToSet": "$type"},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalSessions     int                       `bson:"total_sessions"`
		CompletedSessions int                       `bson:"completed_sessions"`
		AvgConfidence     float64                   `bson:"avg_confidence"`
		Types             []models.ConsultationType `bson:"types"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
	}

	stats := &SessionStats{
		TotalSessions:     result.TotalSessions,
		CompletedSessions: result.CompletedSessions,
		FailedSessions:    result.TotalSessions - result.CompletedSessions,
		AverageConfidence: result.AvgConfidence,
		ConsultationTypes: result.Types,
	}

	return stats, nil
}

// SessionStats represents consultation session statistics
type SessionStats struct {
	TotalSessions     int                       `json:"total_sessions"`
	CompletedSessions int                       `json:"completed_sessions"`
	FailedSessions    int                       `json:"failed_sessions"`
	AverageConfidence float64                   `json:"average_confidence"`
	ConsultationTypes []models.ConsultationType `json:"consultation_types"`
}

// AddConversationTurn adds a new turn to a multi-turn conversation session
func (sm *SessionManager) AddConversationTurn(ctx context.Context, sessionID primitive.ObjectID, query string) (*models.ConsultationSession, error) {
	// Retrieve existing session
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Ensure session is completed (can't add turns to active/failed sessions)
	if session.Status != models.SessionStatusCompleted {
		return nil, fmt.Errorf("can only add turns to completed sessions (current status: %s)", session.Status)
	}

	// Mark session as multi-turn
	session.IsMultiTurn = true

	// Create new conversation turn
	turnIndex := len(session.ConversationTurns)
	newTurn := models.ConversationTurn{
		ID:        primitive.NewObjectID(),
		Query:     query,
		Timestamp: time.Now(),
		TurnIndex: turnIndex,
	}

	// Build enhanced context for multi-turn conversation
	enhancedContext := session.Context

	// Add previous conversation context
	if enhancedContext.SystemContext == nil {
		enhancedContext.SystemContext = make(map[string]interface{})
	}

	enhancedContext.SystemContext["conversation_history"] = session.ConversationTurns
	enhancedContext.SystemContext["previous_query"] = session.Query
	if session.Response != nil {
		enhancedContext.SystemContext["previous_response_summary"] = session.Response.Analysis.Summary
		enhancedContext.SystemContext["previous_recommendations"] = session.Response.Recommendations
	}

	// Create consultation request for the new turn
	consultationRequest := &ConsultationRequest{
		Query:               query,
		Type:                session.Type,
		UserID:              session.UserID,
		Context:             enhancedContext,
		MaxSources:          10,
		ConfidenceThreshold: 0.7,
	}

	// Check cache for similar query in conversation context
	conversationCacheKey := fmt.Sprintf("%s:%d", sessionID.Hex(), turnIndex)
	if cachedResponse, err := sm.cache.GetCachedQueryResponse(ctx, conversationCacheKey, session.Type, session.UserID); err == nil && cachedResponse != nil {
		newTurn.Response = cachedResponse
		session.ConversationTurns = append(session.ConversationTurns, newTurn)
		session.UpdatedAt = time.Now()

		if err := sm.updateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session with cached turn: %w", err)
		}

		return session, nil
	}

	startTime := time.Now()

	// Process consultation for the new turn
	var response *models.ConsultationResponse
	switch session.Type {
	case models.ConsultationTypePolicy:
		response, err = sm.service.ConsultPolicy(ctx, consultationRequest)
	case models.ConsultationTypeStrategy:
		response, err = sm.service.ConsultStrategy(ctx, consultationRequest)
	case models.ConsultationTypeOperations:
		response, err = sm.service.ConsultOperations(ctx, consultationRequest)
	case models.ConsultationTypeTechnology:
		response, err = sm.service.ConsultTechnology(ctx, consultationRequest)
	default:
		return nil, fmt.Errorf("unsupported consultation type: %s", session.Type)
	}

	processingTime := time.Since(startTime)

	if err != nil {
		sm.service.logger.Error("Failed to process conversation turn", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
			"turn_index": turnIndex,
		})
		return nil, fmt.Errorf("failed to process conversation turn: %w", err)
	}

	// Update response with processing time
	response.ProcessingTime = processingTime

	// Cache the turn response
	if err := sm.cache.CacheQueryResponse(ctx, conversationCacheKey, session.Type, session.UserID, response); err != nil {
		sm.service.logger.Error("Failed to cache turn response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
			"turn_index": turnIndex,
		})
	}

	// Add response to turn
	newTurn.Response = response

	// Add turn to session
	session.ConversationTurns = append(session.ConversationTurns, newTurn)
	session.UpdatedAt = time.Now()

	// Update session metadata
	session.Metadata["conversation_turns"] = len(session.ConversationTurns)
	session.Metadata["last_turn_confidence"] = response.ConfidenceScore
	session.Metadata["last_turn_processing_time"] = processingTime.String()

	// Update session in database
	if err := sm.updateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session with new turn: %w", err)
	}

	sm.service.logger.Info("Added conversation turn", map[string]interface{}{
		"session_id":      sessionID.Hex(),
		"turn_index":      turnIndex,
		"processing_time": processingTime.String(),
		"confidence":      response.ConfidenceScore,
	})

	return session, nil
}

// GetConversationHistory retrieves the conversation history for a session
func (sm *SessionManager) GetConversationHistory(ctx context.Context, sessionID primitive.ObjectID) ([]models.ConversationTurn, error) {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	return session.ConversationTurns, nil
}

// ContinueConversation creates a new session that continues from a previous session
func (sm *SessionManager) ContinueConversation(ctx context.Context, previousSessionID primitive.ObjectID, query string) (*models.ConsultationSession, error) {
	// Get previous session for context
	previousSession, err := sm.GetSession(ctx, previousSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve previous session: %w", err)
	}

	// Create enhanced context with conversation history
	enhancedContext := previousSession.Context
	if enhancedContext.PreviousSessions == nil {
		enhancedContext.PreviousSessions = []primitive.ObjectID{}
	}
	enhancedContext.PreviousSessions = append(enhancedContext.PreviousSessions, previousSessionID)

	// Add conversation context
	if enhancedContext.SystemContext == nil {
		enhancedContext.SystemContext = make(map[string]interface{})
	}
	enhancedContext.SystemContext["previous_session_id"] = previousSessionID.Hex()
	enhancedContext.SystemContext["previous_query"] = previousSession.Query
	if previousSession.Response != nil {
		enhancedContext.SystemContext["previous_response_summary"] = previousSession.Response.Analysis.Summary
	}
	if len(previousSession.ConversationTurns) > 0 {
		enhancedContext.SystemContext["conversation_history"] = previousSession.ConversationTurns
	}

	// Create new session request
	request := &ConsultationRequest{
		Query:               query,
		Type:                previousSession.Type,
		UserID:              previousSession.UserID,
		Context:             enhancedContext,
		MaxSources:          10,
		ConfidenceThreshold: 0.7,
	}

	// Create new session
	newSession, err := sm.CreateSession(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create continuation session: %w", err)
	}

	// Mark as continuation
	newSession.Metadata["is_continuation"] = true
	newSession.Metadata["previous_session_id"] = previousSessionID.Hex()

	// Update session in database
	if err := sm.updateSession(ctx, newSession); err != nil {
		sm.service.logger.Error("Failed to update continuation session metadata", err, map[string]interface{}{
			"session_id": newSession.ID.Hex(),
		})
	}

	sm.service.logger.Info("Created continuation session", map[string]interface{}{
		"new_session_id":      newSession.ID.Hex(),
		"previous_session_id": previousSessionID.Hex(),
	})

	return newSession, nil
}

// GetRelatedSessions retrieves sessions related to a given session through conversation history
func (sm *SessionManager) GetRelatedSessions(ctx context.Context, sessionID primitive.ObjectID) ([]*models.ConsultationSession, error) {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	var relatedSessionIDs []primitive.ObjectID

	// Add previous sessions from context
	relatedSessionIDs = append(relatedSessionIDs, session.Context.PreviousSessions...)

	// Find sessions that reference this session
	collection := sm.mongodb.Collection("consultations")
	cursor, err := collection.Find(ctx, bson.M{
		"context.previous_sessions": sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find related sessions: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var relatedSession models.ConsultationSession
		if err := cursor.Decode(&relatedSession); err != nil {
			continue
		}
		relatedSessionIDs = append(relatedSessionIDs, relatedSession.ID)
	}

	// Retrieve all related sessions
	var relatedSessions []*models.ConsultationSession
	for _, id := range relatedSessionIDs {
		if id == sessionID {
			continue // Skip self
		}

		relatedSession, err := sm.GetSession(ctx, id)
		if err != nil {
			sm.service.logger.Error("Failed to retrieve related session", err, map[string]interface{}{
				"related_session_id": id.Hex(),
			})
			continue
		}
		relatedSessions = append(relatedSessions, relatedSession)
	}

	return relatedSessions, nil
}

// GetUsageAnalytics retrieves usage analytics for consultations
func (sm *SessionManager) GetUsageAnalytics(ctx context.Context, startDate, endDate time.Time) (*UsageMetrics, error) {
	return sm.analytics.GetUsageMetrics(ctx, startDate, endDate)
}

// GetUserAnalytics retrieves analytics for a specific user
func (sm *SessionManager) GetUserAnalytics(ctx context.Context, userID primitive.ObjectID, startDate, endDate time.Time) (*UserUsageStats, error) {
	return sm.analytics.GetUserAnalytics(ctx, userID, startDate, endDate)
}

// GetCacheStats retrieves cache statistics
func (sm *SessionManager) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	return sm.cache.GetCacheStats(ctx)
}

// InvalidateCache invalidates cached data for a session
func (sm *SessionManager) InvalidateCache(ctx context.Context, sessionID primitive.ObjectID) error {
	return sm.cache.InvalidateSessionCache(ctx, sessionID)
}

// Helper methods

func (sm *SessionManager) updateSession(ctx context.Context, session *models.ConsultationSession) error {
	collection := sm.mongodb.Collection("consultations")

	session.UpdatedAt = time.Now()

	_, err := collection.ReplaceOne(ctx, bson.M{"_id": session.ID}, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (sm *SessionManager) updateSessionStatus(ctx context.Context, sessionID primitive.ObjectID, status models.SessionStatus) error {
	collection := sm.mongodb.Collection("consultations")

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": sessionID}, update)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	return nil
}
