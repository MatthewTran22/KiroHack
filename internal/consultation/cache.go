package consultation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConsultationCache handles caching of consultation results
type ConsultationCache struct {
	redis  *redis.Client
	logger logger.Logger
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	DefaultTTL   time.Duration
	ResponseTTL  time.Duration
	ContextTTL   time.Duration
	AnalyticsTTL time.Duration
}

// NewConsultationCache creates a new consultation cache
func NewConsultationCache(redis *redis.Client, logger logger.Logger) *ConsultationCache {
	return &ConsultationCache{
		redis:  redis,
		logger: logger,
	}
}

// GetDefaultConfig returns default cache configuration
func GetDefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		DefaultTTL:   24 * time.Hour,
		ResponseTTL:  6 * time.Hour,
		ContextTTL:   12 * time.Hour,
		AnalyticsTTL: 7 * 24 * time.Hour, // 1 week
	}
}

// CacheResponse caches a consultation response
func (c *ConsultationCache) CacheResponse(ctx context.Context, sessionID primitive.ObjectID, response *models.ConsultationResponse) error {
	if c.redis == nil {
		return nil // Cache is optional
	}

	key := fmt.Sprintf("consultation:response:%s", sessionID.Hex())

	data, err := json.Marshal(response)
	if err != nil {
		c.logger.Error("Failed to marshal response for caching", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
		return err
	}

	config := GetDefaultCacheConfig()
	err = c.redis.Set(ctx, key, data, config.ResponseTTL).Err()
	if err != nil {
		c.logger.Error("Failed to cache response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
		return err
	}

	c.logger.Debug("Cached consultation response", map[string]interface{}{
		"session_id": sessionID.Hex(),
		"ttl":        config.ResponseTTL.String(),
	})

	return nil
}

// GetCachedResponse retrieves a cached consultation response
func (c *ConsultationCache) GetCachedResponse(ctx context.Context, sessionID primitive.ObjectID) (*models.ConsultationResponse, error) {
	if c.redis == nil {
		return nil, nil // Cache is optional
	}

	key := fmt.Sprintf("consultation:response:%s", sessionID.Hex())

	data, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		c.logger.Error("Failed to get cached response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
		return nil, err
	}

	var response models.ConsultationResponse
	err = json.Unmarshal([]byte(data), &response)
	if err != nil {
		c.logger.Error("Failed to unmarshal cached response", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
		return nil, err
	}

	c.logger.Debug("Retrieved cached consultation response", map[string]interface{}{
		"session_id": sessionID.Hex(),
	})

	return &response, nil
}

// CacheQueryResponse caches a response by query hash for similar queries
func (c *ConsultationCache) CacheQueryResponse(ctx context.Context, query string, consultationType models.ConsultationType, userID primitive.ObjectID, response *models.ConsultationResponse) error {
	if c.redis == nil {
		return nil
	}

	queryHash := c.generateQueryHash(query, consultationType, userID)
	key := fmt.Sprintf("consultation:query:%s", queryHash)

	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	config := GetDefaultCacheConfig()
	err = c.redis.Set(ctx, key, data, config.ResponseTTL).Err()
	if err != nil {
		c.logger.Error("Failed to cache query response", err, map[string]interface{}{
			"query_hash": queryHash,
		})
		return err
	}

	return nil
}

// GetCachedQueryResponse retrieves a cached response by query hash
func (c *ConsultationCache) GetCachedQueryResponse(ctx context.Context, query string, consultationType models.ConsultationType, userID primitive.ObjectID) (*models.ConsultationResponse, error) {
	if c.redis == nil {
		return nil, nil
	}

	queryHash := c.generateQueryHash(query, consultationType, userID)
	key := fmt.Sprintf("consultation:query:%s", queryHash)

	data, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var response models.ConsultationResponse
	err = json.Unmarshal([]byte(data), &response)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Retrieved cached query response", map[string]interface{}{
		"query_hash": queryHash,
	})

	return &response, nil
}

// CacheSessionContext caches session context for multi-turn conversations
func (c *ConsultationCache) CacheSessionContext(ctx context.Context, sessionID primitive.ObjectID, context *models.ConsultationContext) error {
	if c.redis == nil {
		return nil
	}

	key := fmt.Sprintf("consultation:context:%s", sessionID.Hex())

	data, err := json.Marshal(context)
	if err != nil {
		return err
	}

	config := GetDefaultCacheConfig()
	err = c.redis.Set(ctx, key, data, config.ContextTTL).Err()
	if err != nil {
		c.logger.Error("Failed to cache session context", err, map[string]interface{}{
			"session_id": sessionID.Hex(),
		})
		return err
	}

	return nil
}

// GetCachedSessionContext retrieves cached session context
func (c *ConsultationCache) GetCachedSessionContext(ctx context.Context, sessionID primitive.ObjectID) (*models.ConsultationContext, error) {
	if c.redis == nil {
		return nil, nil
	}

	key := fmt.Sprintf("consultation:context:%s", sessionID.Hex())

	data, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var context models.ConsultationContext
	err = json.Unmarshal([]byte(data), &context)
	if err != nil {
		return nil, err
	}

	return &context, nil
}

// InvalidateSessionCache invalidates all cached data for a session
func (c *ConsultationCache) InvalidateSessionCache(ctx context.Context, sessionID primitive.ObjectID) error {
	if c.redis == nil {
		return nil
	}

	keys := []string{
		fmt.Sprintf("consultation:response:%s", sessionID.Hex()),
		fmt.Sprintf("consultation:context:%s", sessionID.Hex()),
	}

	for _, key := range keys {
		err := c.redis.Del(ctx, key).Err()
		if err != nil {
			c.logger.Error("Failed to invalidate cache key", err, map[string]interface{}{
				"key": key,
			})
		}
	}

	return nil
}

// CacheAnalytics caches analytics data
func (c *ConsultationCache) CacheAnalytics(ctx context.Context, key string, data interface{}) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("consultation:analytics:%s", key)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	config := GetDefaultCacheConfig()
	err = c.redis.Set(ctx, cacheKey, jsonData, config.AnalyticsTTL).Err()
	if err != nil {
		c.logger.Error("Failed to cache analytics", err, map[string]interface{}{
			"key": key,
		})
		return err
	}

	return nil
}

// GetCachedAnalytics retrieves cached analytics data
func (c *ConsultationCache) GetCachedAnalytics(ctx context.Context, key string, result interface{}) error {
	if c.redis == nil {
		return redis.Nil
	}

	cacheKey := fmt.Sprintf("consultation:analytics:%s", key)

	data, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), result)
	if err != nil {
		c.logger.Error("Failed to unmarshal cached analytics", err, map[string]interface{}{
			"key": key,
		})
		return err
	}

	return nil
}

// generateQueryHash generates a hash for query caching
func (c *ConsultationCache) generateQueryHash(query string, consultationType models.ConsultationType, userID primitive.ObjectID) string {
	data := fmt.Sprintf("%s:%s:%s", query, consultationType, userID.Hex())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// GetCacheStats returns cache statistics
func (c *ConsultationCache) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	if c.redis == nil {
		return &CacheStats{}, nil
	}

	info, err := c.redis.Info(ctx, "memory").Result()
	if err != nil {
		return nil, err
	}

	// Parse Redis info for memory usage
	stats := &CacheStats{
		RedisInfo: info,
	}

	// Count consultation-related keys
	keys, err := c.redis.Keys(ctx, "consultation:*").Result()
	if err != nil {
		return stats, err
	}

	stats.TotalKeys = len(keys)

	// Count by type
	for _, key := range keys {
		if strings.Contains(key, ":response:") {
			stats.ResponseKeys++
		} else if strings.Contains(key, ":context:") {
			stats.ContextKeys++
		} else if strings.Contains(key, ":analytics:") {
			stats.AnalyticsKeys++
		} else if strings.Contains(key, ":query:") {
			stats.QueryKeys++
		}
	}

	return stats, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalKeys     int    `json:"total_keys"`
	ResponseKeys  int    `json:"response_keys"`
	ContextKeys   int    `json:"context_keys"`
	AnalyticsKeys int    `json:"analytics_keys"`
	QueryKeys     int    `json:"query_keys"`
	RedisInfo     string `json:"redis_info"`
}
