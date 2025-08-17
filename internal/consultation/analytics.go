package consultation

import (
	"context"
	"fmt"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AnalyticsService handles consultation analytics and usage tracking
type AnalyticsService struct {
	mongodb *mongo.Database
	cache   *ConsultationCache
	logger  logger.Logger
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(mongodb *mongo.Database, cache *ConsultationCache, logger logger.Logger) *AnalyticsService {
	return &AnalyticsService{
		mongodb: mongodb,
		cache:   cache,
		logger:  logger,
	}
}

// UsageMetrics represents usage metrics for consultations
type UsageMetrics struct {
	TotalConsultations    int                             `json:"total_consultations"`
	ConsultationsByType   map[models.ConsultationType]int `json:"consultations_by_type"`
	ConsultationsByStatus map[models.SessionStatus]int    `json:"consultations_by_status"`
	AverageProcessingTime time.Duration                   `json:"average_processing_time"`
	AverageConfidence     float64                         `json:"average_confidence"`
	TopUsers              []UserUsageStats                `json:"top_users"`
	DailyUsage            []DailyUsageStats               `json:"daily_usage"`
	PopularQueries        []QueryStats                    `json:"popular_queries"`
	PerformanceMetrics    PerformanceMetrics              `json:"performance_metrics"`
}

// UserUsageStats represents usage statistics for a user
type UserUsageStats struct {
	UserID             primitive.ObjectID      `json:"user_id"`
	TotalConsultations int                     `json:"total_consultations"`
	LastConsultation   time.Time               `json:"last_consultation"`
	FavoriteType       models.ConsultationType `json:"favorite_type"`
	AverageConfidence  float64                 `json:"average_confidence"`
}

// DailyUsageStats represents daily usage statistics
type DailyUsageStats struct {
	Date          time.Time `json:"date"`
	Consultations int       `json:"consultations"`
	UniqueUsers   int       `json:"unique_users"`
}

// QueryStats represents statistics for popular queries
type QueryStats struct {
	QueryHash string                  `json:"query_hash"`
	Query     string                  `json:"query"`
	Count     int                     `json:"count"`
	Type      models.ConsultationType `json:"type"`
	LastUsed  time.Time               `json:"last_used"`
}

// PerformanceMetrics represents performance-related metrics
type PerformanceMetrics struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	MedianResponseTime  time.Duration `json:"median_response_time"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	SuccessRate         float64       `json:"success_rate"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	ErrorRate           float64       `json:"error_rate"`
}

// TrackConsultationUsage tracks usage for a consultation session
func (a *AnalyticsService) TrackConsultationUsage(ctx context.Context, session *models.ConsultationSession) error {
	// Create usage tracking record
	usageRecord := bson.M{
		"session_id":        session.ID,
		"user_id":           session.UserID,
		"consultation_type": session.Type,
		"status":            session.Status,
		"created_at":        session.CreatedAt,
		"updated_at":        session.UpdatedAt,
		"processing_time":   session.GetDuration(),
		"has_response":      session.HasResponse(),
		"metadata":          session.Metadata,
	}

	if session.Response != nil {
		usageRecord["confidence_score"] = session.Response.ConfidenceScore
		usageRecord["recommendations_count"] = len(session.Response.Recommendations)
		usageRecord["sources_count"] = len(session.Response.Sources)
		usageRecord["response_processing_time"] = session.Response.ProcessingTime
	}

	// Store in analytics collection
	collection := a.mongodb.Collection("consultation_analytics")
	_, err := collection.InsertOne(ctx, usageRecord)
	if err != nil {
		a.logger.Error("Failed to track consultation usage", err, map[string]interface{}{
			"session_id": session.ID.Hex(),
		})
		return err
	}

	// Update cached metrics (async)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a.invalidateCachedMetrics(ctx)
	}()

	return nil
}

// GetUsageMetrics retrieves comprehensive usage metrics
func (a *AnalyticsService) GetUsageMetrics(ctx context.Context, startDate, endDate time.Time) (*UsageMetrics, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("usage_metrics:%s:%s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	var cachedMetrics UsageMetrics
	if err := a.cache.GetCachedAnalytics(ctx, cacheKey, &cachedMetrics); err == nil {
		return &cachedMetrics, nil
	}

	// Calculate metrics from database
	metrics := &UsageMetrics{
		ConsultationsByType:   make(map[models.ConsultationType]int),
		ConsultationsByStatus: make(map[models.SessionStatus]int),
	}

	collection := a.mongodb.Collection("consultations")

	// Build date filter
	dateFilter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	// Get total consultations
	totalCount, err := collection.CountDocuments(ctx, dateFilter)
	if err != nil {
		return nil, err
	}
	metrics.TotalConsultations = int(totalCount)

	// Get consultations by type and status
	pipeline := []bson.M{
		{"$match": dateFilter},
		{
			"$group": bson.M{
				"_id": bson.M{
					"type":   "$type",
					"status": "$status",
				},
				"count":          bson.M{"$sum": 1},
				"avg_confidence": bson.M{"$avg": "$response.confidence_score"},
				"avg_processing_time": bson.M{"$avg": bson.M{
					"$subtract": []interface{}{"$updated_at", "$created_at"},
				}},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var totalConfidence float64
	var totalProcessingTime int64
	var confidenceCount int
	var processingTimeCount int

	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Type   models.ConsultationType `bson:"type"`
				Status models.SessionStatus    `bson:"status"`
			} `bson:"_id"`
			Count             int     `bson:"count"`
			AvgConfidence     float64 `bson:"avg_confidence"`
			AvgProcessingTime int64   `bson:"avg_processing_time"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		metrics.ConsultationsByType[result.ID.Type] += result.Count
		metrics.ConsultationsByStatus[result.ID.Status] += result.Count

		if result.AvgConfidence > 0 {
			totalConfidence += result.AvgConfidence * float64(result.Count)
			confidenceCount += result.Count
		}

		if result.AvgProcessingTime > 0 {
			totalProcessingTime += result.AvgProcessingTime * int64(result.Count)
			processingTimeCount += result.Count
		}
	}

	// Calculate averages
	if confidenceCount > 0 {
		metrics.AverageConfidence = totalConfidence / float64(confidenceCount)
	}

	if processingTimeCount > 0 {
		metrics.AverageProcessingTime = time.Duration(totalProcessingTime / int64(processingTimeCount))
	}

	// Get top users
	topUsers, err := a.getTopUsers(ctx, dateFilter, 10)
	if err != nil {
		a.logger.Error("Failed to get top users", err, nil)
	} else {
		metrics.TopUsers = topUsers
	}

	// Get daily usage
	dailyUsage, err := a.getDailyUsage(ctx, startDate, endDate)
	if err != nil {
		a.logger.Error("Failed to get daily usage", err, nil)
	} else {
		metrics.DailyUsage = dailyUsage
	}

	// Get popular queries
	popularQueries, err := a.getPopularQueries(ctx, dateFilter, 10)
	if err != nil {
		a.logger.Error("Failed to get popular queries", err, nil)
	} else {
		metrics.PopularQueries = popularQueries
	}

	// Get performance metrics
	performanceMetrics, err := a.getPerformanceMetrics(ctx, dateFilter)
	if err != nil {
		a.logger.Error("Failed to get performance metrics", err, nil)
	} else {
		metrics.PerformanceMetrics = *performanceMetrics
	}

	// Cache the results
	if err := a.cache.CacheAnalytics(ctx, cacheKey, metrics); err != nil {
		a.logger.Error("Failed to cache usage metrics", err, nil)
	}

	return metrics, nil
}

// GetUserAnalytics retrieves analytics for a specific user
func (a *AnalyticsService) GetUserAnalytics(ctx context.Context, userID primitive.ObjectID, startDate, endDate time.Time) (*UserUsageStats, error) {
	collection := a.mongodb.Collection("consultations")

	filter := bson.M{
		"user_id": userID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":                 "$user_id",
				"total_consultations": bson.M{"$sum": 1},
				"last_consultation":   bson.M{"$max": "$created_at"},
				"avg_confidence":      bson.M{"$avg": "$response.confidence_score"},
				"types":               bson.M{"$push": "$type"},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		ID                 primitive.ObjectID        `bson:"_id"`
		TotalConsultations int                       `bson:"total_consultations"`
		LastConsultation   time.Time                 `bson:"last_consultation"`
		AvgConfidence      float64                   `bson:"avg_confidence"`
		Types              []models.ConsultationType `bson:"types"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	} else {
		return &UserUsageStats{UserID: userID}, nil
	}

	// Find favorite type
	typeCount := make(map[models.ConsultationType]int)
	for _, t := range result.Types {
		typeCount[t]++
	}

	var favoriteType models.ConsultationType
	var maxCount int
	for t, count := range typeCount {
		if count > maxCount {
			maxCount = count
			favoriteType = t
		}
	}

	return &UserUsageStats{
		UserID:             result.ID,
		TotalConsultations: result.TotalConsultations,
		LastConsultation:   result.LastConsultation,
		FavoriteType:       favoriteType,
		AverageConfidence:  result.AvgConfidence,
	}, nil
}

// Helper methods

func (a *AnalyticsService) getTopUsers(ctx context.Context, filter bson.M, limit int) ([]UserUsageStats, error) {
	collection := a.mongodb.Collection("consultations")

	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id":                 "$user_id",
				"total_consultations": bson.M{"$sum": 1},
				"last_consultation":   bson.M{"$max": "$created_at"},
				"avg_confidence":      bson.M{"$avg": "$response.confidence_score"},
				"types":               bson.M{"$push": "$type"},
			},
		},
		{"$sort": bson.M{"total_consultations": -1}},
		{"$limit": limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []UserUsageStats
	for cursor.Next(ctx) {
		var result struct {
			ID                 primitive.ObjectID        `bson:"_id"`
			TotalConsultations int                       `bson:"total_consultations"`
			LastConsultation   time.Time                 `bson:"last_consultation"`
			AvgConfidence      float64                   `bson:"avg_confidence"`
			Types              []models.ConsultationType `bson:"types"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		// Find favorite type
		typeCount := make(map[models.ConsultationType]int)
		for _, t := range result.Types {
			typeCount[t]++
		}

		var favoriteType models.ConsultationType
		var maxCount int
		for t, count := range typeCount {
			if count > maxCount {
				maxCount = count
				favoriteType = t
			}
		}

		users = append(users, UserUsageStats{
			UserID:             result.ID,
			TotalConsultations: result.TotalConsultations,
			LastConsultation:   result.LastConsultation,
			FavoriteType:       favoriteType,
			AverageConfidence:  result.AvgConfidence,
		})
	}

	return users, nil
}

func (a *AnalyticsService) getDailyUsage(ctx context.Context, startDate, endDate time.Time) ([]DailyUsageStats, error) {
	collection := a.mongodb.Collection("consultations")

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$created_at"},
					"month": bson.M{"$month": "$created_at"},
					"day":   bson.M{"$dayOfMonth": "$created_at"},
				},
				"consultations": bson.M{"$sum": 1},
				"unique_users":  bson.M{"$addToSet": "$user_id"},
			},
		},
		{
			"$project": bson.M{
				"_id":           1,
				"consultations": 1,
				"unique_users":  bson.M{"$size": "$unique_users"},
			},
		},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dailyStats []DailyUsageStats
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Year  int `bson:"year"`
				Month int `bson:"month"`
				Day   int `bson:"day"`
			} `bson:"_id"`
			Consultations int `bson:"consultations"`
			UniqueUsers   int `bson:"unique_users"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		date := time.Date(result.ID.Year, time.Month(result.ID.Month), result.ID.Day, 0, 0, 0, 0, time.UTC)
		dailyStats = append(dailyStats, DailyUsageStats{
			Date:          date,
			Consultations: result.Consultations,
			UniqueUsers:   result.UniqueUsers,
		})
	}

	return dailyStats, nil
}

func (a *AnalyticsService) getPopularQueries(ctx context.Context, filter bson.M, limit int) ([]QueryStats, error) {
	collection := a.mongodb.Collection("consultations")

	pipeline := []bson.M{
		{"$match": filter},
		{
			"$group": bson.M{
				"_id": bson.M{
					"query": "$query",
					"type":  "$type",
				},
				"count":     bson.M{"$sum": 1},
				"last_used": bson.M{"$max": "$created_at"},
			},
		},
		{"$sort": bson.M{"count": -1}},
		{"$limit": limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var queries []QueryStats
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Query string                  `bson:"query"`
				Type  models.ConsultationType `bson:"type"`
			} `bson:"_id"`
			Count    int       `bson:"count"`
			LastUsed time.Time `bson:"last_used"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		queries = append(queries, QueryStats{
			Query:    result.ID.Query,
			Type:     result.ID.Type,
			Count:    result.Count,
			LastUsed: result.LastUsed,
		})
	}

	return queries, nil
}

func (a *AnalyticsService) getPerformanceMetrics(ctx context.Context, filter bson.M) (*PerformanceMetrics, error) {
	collection := a.mongodb.Collection("consultations")

	pipeline := []bson.M{
		{"$match": filter},
		{
			"$project": bson.M{
				"processing_time": bson.M{
					"$subtract": []interface{}{"$updated_at", "$created_at"},
				},
				"status":       1,
				"has_response": bson.M{"$ne": []interface{}{"$response", nil}},
			},
		},
		{
			"$group": bson.M{
				"_id":              nil,
				"processing_times": bson.M{"$push": "$processing_time"},
				"total_sessions":   bson.M{"$sum": 1},
				"successful_sessions": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", "completed"}},
							1,
							0,
						},
					},
				},
				"failed_sessions": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", "failed"}},
							1,
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		ProcessingTimes    []int64 `bson:"processing_times"`
		TotalSessions      int     `bson:"total_sessions"`
		SuccessfulSessions int     `bson:"successful_sessions"`
		FailedSessions     int     `bson:"failed_sessions"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}

	metrics := &PerformanceMetrics{}

	if result.TotalSessions > 0 {
		metrics.SuccessRate = float64(result.SuccessfulSessions) / float64(result.TotalSessions)
		metrics.ErrorRate = float64(result.FailedSessions) / float64(result.TotalSessions)
	}

	// Calculate response time metrics
	if len(result.ProcessingTimes) > 0 {
		var total int64
		for _, t := range result.ProcessingTimes {
			total += t
		}
		metrics.AverageResponseTime = time.Duration(total / int64(len(result.ProcessingTimes)))

		// Calculate median and P95
		// Note: This is a simplified calculation. For production, consider using more sophisticated percentile calculations
		if len(result.ProcessingTimes) >= 2 {
			mid := len(result.ProcessingTimes) / 2
			metrics.MedianResponseTime = time.Duration(result.ProcessingTimes[mid])
		}

		if len(result.ProcessingTimes) >= 20 {
			p95Index := int(float64(len(result.ProcessingTimes)) * 0.95)
			metrics.P95ResponseTime = time.Duration(result.ProcessingTimes[p95Index])
		}
	}

	// Get cache hit rate from cache stats
	if cacheStats, err := a.cache.GetCacheStats(ctx); err == nil {
		// This is a simplified calculation - in production you'd track hits/misses more precisely
		if cacheStats.TotalKeys > 0 {
			metrics.CacheHitRate = 0.75 // Placeholder - implement proper cache hit tracking
		}
	}

	return metrics, nil
}

func (a *AnalyticsService) invalidateCachedMetrics(ctx context.Context) {
	// Invalidate cached usage metrics
	// This is a simple approach - in production, you might want more granular cache invalidation
	keys := []string{
		"usage_metrics:*",
		"user_analytics:*",
		"performance_metrics:*",
	}

	for _, pattern := range keys {
		// Note: Redis KEYS command is not recommended for production - use SCAN instead
		if keys, err := a.cache.redis.Keys(ctx, pattern).Result(); err == nil {
			for _, key := range keys {
				a.cache.redis.Del(ctx, key)
			}
		}
	}
}
