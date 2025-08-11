package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SessionData represents the data stored in a user session
type SessionData struct {
	UserID            primitive.ObjectID `json:"user_id"`
	Email             string             `json:"email"`
	Role              string             `json:"role"`
	SecurityClearance string             `json:"security_clearance"`
	Permissions       []string           `json:"permissions"`
	LoginTime         time.Time          `json:"login_time"`
	LastActivity      time.Time          `json:"last_activity"`
	IPAddress         string             `json:"ip_address"`
	UserAgent         string             `json:"user_agent"`
	MFAVerified       bool               `json:"mfa_verified"`
}

// SessionService handles session management with Redis
type SessionService struct {
	client       *redis.Client
	sessionTTL   time.Duration
	blacklistTTL time.Duration
	keyPrefix    string
}

// NewSessionService creates a new session service
func NewSessionService(client *redis.Client, sessionTTL, blacklistTTL time.Duration) *SessionService {
	return &SessionService{
		client:       client,
		sessionTTL:   sessionTTL,
		blacklistTTL: blacklistTTL,
		keyPrefix:    "session:",
	}
}

// CreateSession creates a new session for a user
func (s *SessionService) CreateSession(ctx context.Context, sessionID string, data *SessionData) error {
	key := s.sessionKey(sessionID)

	sessionJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	err = s.client.Set(ctx, key, sessionJSON, s.sessionTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves session data by session ID
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	key := s.sessionKey(sessionID)

	sessionJSON, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var data SessionData
	err = json.Unmarshal([]byte(sessionJSON), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &data, nil
}

// UpdateSession updates session data
func (s *SessionService) UpdateSession(ctx context.Context, sessionID string, data *SessionData) error {
	key := s.sessionKey(sessionID)

	// Check if session exists
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("session not found")
	}

	sessionJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Update with the remaining TTL
	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get session TTL: %w", err)
	}

	err = s.client.Set(ctx, key, sessionJSON, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// RefreshSession extends the session TTL
func (s *SessionService) RefreshSession(ctx context.Context, sessionID string) error {
	key := s.sessionKey(sessionID)

	err := s.client.Expire(ctx, key, s.sessionTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	return nil
}

// UpdateLastActivity updates the last activity timestamp for a session
func (s *SessionService) UpdateLastActivity(ctx context.Context, sessionID string) error {
	data, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	data.LastActivity = time.Now()
	return s.UpdateSession(ctx, sessionID, data)
}

// DeleteSession removes a session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	key := s.sessionKey(sessionID)

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// BlacklistToken adds a token to the blacklist
func (s *SessionService) BlacklistToken(ctx context.Context, tokenID string) error {
	key := s.blacklistKey(tokenID)

	err := s.client.Set(ctx, key, "blacklisted", s.blacklistTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (s *SessionService) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := s.blacklistKey(tokenID)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID primitive.ObjectID) ([]*SessionData, error) {
	pattern := s.sessionKey("*")

	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session keys: %w", err)
	}

	var userSessions []*SessionData
	for _, key := range keys {
		sessionJSON, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue // Skip invalid sessions
		}

		var data SessionData
		err = json.Unmarshal([]byte(sessionJSON), &data)
		if err != nil {
			continue // Skip invalid session data
		}

		if data.UserID == userID {
			userSessions = append(userSessions, &data)
		}
	}

	return userSessions, nil
}

// InvalidateUserSessions invalidates all sessions for a user
func (s *SessionService) InvalidateUserSessions(ctx context.Context, userID primitive.ObjectID) error {
	// Extract session ID from the session data (this is a simplified approach)
	// In a real implementation, you might store the session ID in the session data
	// or use a different approach to map sessions to IDs
	pattern := s.sessionKey("*")
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get session keys: %w", err)
	}

	for _, key := range keys {
		sessionJSON, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var data SessionData
		err = json.Unmarshal([]byte(sessionJSON), &data)
		if err != nil {
			continue
		}

		if data.UserID == userID {
			s.client.Del(ctx, key)
		}
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions (Redis handles this automatically, but this can be used for manual cleanup)
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	// Redis automatically handles TTL expiration, but we can implement additional cleanup logic here
	// For example, logging or metrics collection
	return nil
}

// GetSessionStats returns statistics about active sessions
func (s *SessionService) GetSessionStats(ctx context.Context) (*SessionStats, error) {
	pattern := s.sessionKey("*")

	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session keys: %w", err)
	}

	stats := &SessionStats{
		TotalSessions: len(keys),
		UserSessions:  make(map[string]int),
	}

	for _, key := range keys {
		sessionJSON, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var data SessionData
		err = json.Unmarshal([]byte(sessionJSON), &data)
		if err != nil {
			continue
		}

		userIDStr := data.UserID.Hex()
		stats.UserSessions[userIDStr]++
	}

	return stats, nil
}

// sessionKey generates a Redis key for a session
func (s *SessionService) sessionKey(sessionID string) string {
	return s.keyPrefix + sessionID
}

// blacklistKey generates a Redis key for a blacklisted token
func (s *SessionService) blacklistKey(tokenID string) string {
	return "blacklist:" + tokenID
}

// SessionStats represents session statistics
type SessionStats struct {
	TotalSessions int            `json:"total_sessions"`
	UserSessions  map[string]int `json:"user_sessions"` // userID -> session count
}
