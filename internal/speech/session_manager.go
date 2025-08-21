package speech

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoSpeechSessionManager implements SpeechSessionManager using MongoDB
type MongoSpeechSessionManager struct {
	collection *mongo.Collection
	mutex      sync.RWMutex
	sessions   map[string]*SpeechSession // In-memory cache for active sessions
}

// NewMongoSpeechSessionManager creates a new MongoDB-based session manager
func NewMongoSpeechSessionManager(db *mongo.Database) *MongoSpeechSessionManager {
	return &MongoSpeechSessionManager{
		collection: db.Collection("speech_sessions"),
		sessions:   make(map[string]*SpeechSession),
	}
}

// CreateSession creates a new speech session
func (m *MongoSpeechSessionManager) CreateSession(ctx context.Context, userID string, sessionType SessionType) (*SpeechSession, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	
	session := &SpeechSession{
		ID:           sessionID,
		UserID:       userID,
		Type:         sessionType,
		Status:       SessionStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(m.getSessionDuration(sessionType)),
		Metadata:     make(map[string]interface{}),
		Interactions: []SpeechInteraction{},
	}

	// Store in MongoDB
	_, err := m.collection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}

	// Cache in memory
	m.mutex.Lock()
	m.sessions[sessionID] = session
	m.mutex.Unlock()

	return session, nil
}

// GetSession retrieves a speech session by ID
func (m *MongoSpeechSessionManager) GetSession(ctx context.Context, sessionID string) (*SpeechSession, error) {
	// Check in-memory cache first
	m.mutex.RLock()
	if session, exists := m.sessions[sessionID]; exists {
		m.mutex.RUnlock()
		
		// Check if session is expired
		if time.Now().After(session.ExpiresAt) {
			session.Status = SessionStatusExpired
			m.UpdateSession(ctx, sessionID, map[string]interface{}{"status": SessionStatusExpired})
			return nil, fmt.Errorf("session expired")
		}
		
		return session, nil
	}
	m.mutex.RUnlock()

	// Fetch from database
	var session SpeechSession
	err := m.collection.FindOne(ctx, bson.M{"_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		session.Status = SessionStatusExpired
		m.UpdateSession(ctx, sessionID, map[string]interface{}{"status": SessionStatusExpired})
		return nil, fmt.Errorf("session expired")
	}

	// Cache in memory
	m.mutex.Lock()
	m.sessions[sessionID] = &session
	m.mutex.Unlock()

	return &session, nil
}

// UpdateSession updates a speech session
func (m *MongoSpeechSessionManager) UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	// Add updated timestamp
	updates["updated_at"] = time.Now()

	// Update in database
	filter := bson.M{"_id": sessionID}
	update := bson.M{"$set": updates}
	
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update session in database: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("session not found")
	}

	// Update in-memory cache
	m.mutex.Lock()
	if session, exists := m.sessions[sessionID]; exists {
		for key, value := range updates {
			switch key {
			case "status":
				if status, ok := value.(SessionStatus); ok {
					session.Status = status
				}
			case "updated_at":
				if timestamp, ok := value.(time.Time); ok {
					session.UpdatedAt = timestamp
				}
			case "expires_at":
				if timestamp, ok := value.(time.Time); ok {
					session.ExpiresAt = timestamp
				}
			default:
				session.Metadata[key] = value
			}
		}
	}
	m.mutex.Unlock()

	return nil
}

// EndSession ends a speech session
func (m *MongoSpeechSessionManager) EndSession(ctx context.Context, sessionID string) error {
	updates := map[string]interface{}{
		"status":    SessionStatusEnded,
		"ended_at":  time.Now(),
	}

	err := m.UpdateSession(ctx, sessionID, updates)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Remove from in-memory cache
	m.mutex.Lock()
	delete(m.sessions, sessionID)
	m.mutex.Unlock()

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (m *MongoSpeechSessionManager) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now()
	
	// Update expired sessions in database
	filter := bson.M{
		"expires_at": bson.M{"$lt": now},
		"status":     bson.M{"$ne": SessionStatusExpired},
	}
	update := bson.M{
		"$set": bson.M{
			"status":     SessionStatusExpired,
			"updated_at": now,
		},
	}
	
	_, err := m.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update expired sessions: %w", err)
	}

	// Clean up in-memory cache
	m.mutex.Lock()
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			session.Status = SessionStatusExpired
			delete(m.sessions, sessionID)
		}
	}
	m.mutex.Unlock()

	// Delete old expired sessions (older than 24 hours)
	deleteThreshold := now.Add(-24 * time.Hour)
	deleteFilter := bson.M{
		"status":     SessionStatusExpired,
		"updated_at": bson.M{"$lt": deleteThreshold},
	}
	
	_, err = m.collection.DeleteMany(ctx, deleteFilter)
	if err != nil {
		return fmt.Errorf("failed to delete old expired sessions: %w", err)
	}

	return nil
}

// AddInteraction adds an interaction to a session
func (m *MongoSpeechSessionManager) AddInteraction(ctx context.Context, sessionID string, interaction SpeechInteraction) error {
	interaction.ID = uuid.New().String()
	interaction.Timestamp = time.Now()

	// Update in database
	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$push": bson.M{"interactions": interaction},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add interaction to database: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("session not found")
	}

	// Update in-memory cache
	m.mutex.Lock()
	if session, exists := m.sessions[sessionID]; exists {
		session.Interactions = append(session.Interactions, interaction)
		session.UpdatedAt = time.Now()
	}
	m.mutex.Unlock()

	return nil
}

// GetSessionInteractions retrieves interactions for a session
func (m *MongoSpeechSessionManager) GetSessionInteractions(ctx context.Context, sessionID string) ([]SpeechInteraction, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return session.Interactions, nil
}

// GetUserSessions retrieves all sessions for a user
func (m *MongoSpeechSessionManager) GetUserSessions(ctx context.Context, userID string, limit int) ([]SpeechSession, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []SpeechSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// GetSessionStats retrieves statistics for sessions
func (m *MongoSpeechSessionManager) GetSessionStats(ctx context.Context, userID string, timeRange time.Duration) (*SessionStats, error) {
	since := time.Now().Add(-timeRange)
	
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"user_id":    userID,
				"created_at": bson.M{"$gte": since},
			},
		},
		{
			"$group": bson.M{
				"_id": "$type",
				"count": bson.M{"$sum": 1},
				"total_interactions": bson.M{"$sum": bson.M{"$size": "$interactions"}},
				"avg_duration": bson.M{
					"$avg": bson.M{
						"$divide": []interface{}{
							bson.M{"$subtract": []interface{}{"$updated_at", "$created_at"}},
							1000, // Convert to seconds
						},
					},
				},
			},
		},
	}

	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate session stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := &SessionStats{
		UserID:    userID,
		TimeRange: timeRange,
		TypeStats: make(map[SessionType]TypeStats),
	}

	for cursor.Next(ctx) {
		var result struct {
			ID                string      `bson:"_id"`
			Count             int         `bson:"count"`
			TotalInteractions int         `bson:"total_interactions"`
			AvgDuration       float64     `bson:"avg_duration"`
		}
		
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		sessionType := SessionType(result.ID)
		stats.TypeStats[sessionType] = TypeStats{
			SessionCount:      result.Count,
			TotalInteractions: result.TotalInteractions,
			AvgDuration:       result.AvgDuration,
		}
		
		stats.TotalSessions += result.Count
		stats.TotalInteractions += result.TotalInteractions
	}

	return stats, nil
}

// getSessionDuration returns the duration for different session types
func (m *MongoSpeechSessionManager) getSessionDuration(sessionType SessionType) time.Duration {
	switch sessionType {
	case SessionTypeConsultation:
		return 2 * time.Hour
	case SessionTypeTranscription:
		return 1 * time.Hour
	case SessionTypeVoiceAuth:
		return 15 * time.Minute
	default:
		return 30 * time.Minute
	}
}

// SessionStats represents session statistics
type SessionStats struct {
	UserID            string                    `json:"user_id"`
	TimeRange         time.Duration             `json:"time_range"`
	TotalSessions     int                       `json:"total_sessions"`
	TotalInteractions int                       `json:"total_interactions"`
	TypeStats         map[SessionType]TypeStats `json:"type_stats"`
}

// TypeStats represents statistics for a specific session type
type TypeStats struct {
	SessionCount      int     `json:"session_count"`
	TotalInteractions int     `json:"total_interactions"`
	AvgDuration       float64 `json:"avg_duration"`
}

// InMemorySpeechSessionManager provides an in-memory implementation for testing
type InMemorySpeechSessionManager struct {
	sessions map[string]*SpeechSession
	mutex    sync.RWMutex
}

// NewInMemorySpeechSessionManager creates a new in-memory session manager
func NewInMemorySpeechSessionManager() *InMemorySpeechSessionManager {
	return &InMemorySpeechSessionManager{
		sessions: make(map[string]*SpeechSession),
	}
}

// CreateSession creates a new speech session in memory
func (m *InMemorySpeechSessionManager) CreateSession(ctx context.Context, userID string, sessionType SessionType) (*SpeechSession, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	
	session := &SpeechSession{
		ID:           sessionID,
		UserID:       userID,
		Type:         sessionType,
		Status:       SessionStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(30 * time.Minute), // Default 30 minutes
		Metadata:     make(map[string]interface{}),
		Interactions: []SpeechInteraction{},
	}

	m.mutex.Lock()
	m.sessions[sessionID] = session
	m.mutex.Unlock()

	return session, nil
}

// GetSession retrieves a speech session by ID from memory
func (m *InMemorySpeechSessionManager) GetSession(ctx context.Context, sessionID string) (*SpeechSession, error) {
	m.mutex.RLock()
	session, exists := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		session.Status = SessionStatusExpired
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// UpdateSession updates a speech session in memory
func (m *InMemorySpeechSessionManager) UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "status":
			if status, ok := value.(SessionStatus); ok {
				session.Status = status
			}
		case "expires_at":
			if timestamp, ok := value.(time.Time); ok {
				session.ExpiresAt = timestamp
			}
		default:
			session.Metadata[key] = value
		}
	}

	session.UpdatedAt = time.Now()
	return nil
}

// EndSession ends a speech session in memory
func (m *InMemorySpeechSessionManager) EndSession(ctx context.Context, sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Status = SessionStatusEnded
	session.UpdatedAt = time.Now()
	
	return nil
}

// CleanupExpiredSessions removes expired sessions from memory
func (m *InMemorySpeechSessionManager) CleanupExpiredSessions(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			session.Status = SessionStatusExpired
			delete(m.sessions, sessionID)
		}
	}

	return nil
}