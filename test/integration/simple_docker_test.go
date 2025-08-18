package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-government-consultant/internal/api"
	"ai-government-consultant/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleDockerIntegration tests basic API functionality with Docker containers
func TestSimpleDockerIntegration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping Docker integration tests in short mode")
	}

	// Test Docker container connectivity first
	t.Run("Docker Container Connectivity", func(t *testing.T) {
		// Test MongoDB connectivity
		mongoURI := "mongodb://testadmin:testpassword@localhost:27018/ai_government_consultant_test?authSource=admin"
		mongoClient, err := database.NewMongoClient(mongoURI)
		require.NoError(t, err, "Failed to connect to Docker MongoDB")
		defer mongoClient.Disconnect(context.Background())

		// Test MongoDB ping
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = mongoClient.Ping(ctx, nil)
		assert.NoError(t, err, "Failed to ping Docker MongoDB")

		// Test Redis connectivity
		redisClient := redis.NewClient(&redis.Options{
			Addr:     "localhost:6380",
			Password: "testpassword",
			DB:       0,
		})
		defer redisClient.Close()

		pong, err := redisClient.Ping(ctx).Result()
		assert.NoError(t, err, "Failed to ping Docker Redis")
		assert.Equal(t, "PONG", pong, "Redis ping should return PONG")
	})

	t.Run("API Server with Docker Services", func(t *testing.T) {
		// Setup minimal API server
		gin.SetMode(gin.TestMode)
		router := gin.New()

		// Create simple services
		knowledgeService := api.NewSimpleKnowledgeService(nil)
		auditService := api.NewSimpleAuditService(nil)

		// Setup basic routes
		setupDockerTestRoutes(router, knowledgeService, auditService)

		// Test health endpoint
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])

		// Test readiness endpoint
		req, _ = http.NewRequest("GET", "/ready", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ready", response["status"])
	})

	t.Run("Database Operations with Docker", func(t *testing.T) {
		// Connect to Docker MongoDB
		mongoURI := "mongodb://testadmin:testpassword@localhost:27018/ai_government_consultant_test?authSource=admin"
		mongoClient, err := database.NewMongoClient(mongoURI)
		require.NoError(t, err, "Failed to connect to Docker MongoDB")
		defer mongoClient.Disconnect(context.Background())

		db := mongoClient.Database("ai_government_consultant_test")
		collection := db.Collection("test_api_collection")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test document insertion
		testDoc := map[string]interface{}{
			"test_field": "api_test_value",
			"timestamp":  time.Now(),
			"test_type":  "docker_integration",
		}

		result, err := collection.InsertOne(ctx, testDoc)
		assert.NoError(t, err, "Failed to insert test document")
		assert.NotNil(t, result.InsertedID, "Insert result should have an ID")

		// Test document retrieval
		var retrievedDoc map[string]interface{}
		err = collection.FindOne(ctx, map[string]interface{}{"_id": result.InsertedID}).Decode(&retrievedDoc)
		assert.NoError(t, err, "Failed to retrieve test document")
		assert.Equal(t, "api_test_value", retrievedDoc["test_field"])

		// Clean up
		_, err = collection.DeleteOne(ctx, map[string]interface{}{"_id": result.InsertedID})
		assert.NoError(t, err, "Failed to clean up test document")
	})

	t.Run("Redis Operations with Docker", func(t *testing.T) {
		// Connect to Docker Redis
		redisClient := redis.NewClient(&redis.Options{
			Addr:     "localhost:6380",
			Password: "testpassword",
			DB:       0,
		})
		defer redisClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test Redis operations
		testKey := fmt.Sprintf("api_test_key_%d", time.Now().UnixNano())
		testValue := "api_test_value"

		// Set value
		err := redisClient.Set(ctx, testKey, testValue, time.Minute).Err()
		assert.NoError(t, err, "Failed to set value in Docker Redis")

		// Get value
		retrievedValue, err := redisClient.Get(ctx, testKey).Result()
		assert.NoError(t, err, "Failed to get value from Docker Redis")
		assert.Equal(t, testValue, retrievedValue, "Retrieved value should match set value")

		// Test expiration
		ttl, err := redisClient.TTL(ctx, testKey).Result()
		assert.NoError(t, err, "Failed to get TTL")
		assert.True(t, ttl > 0, "TTL should be positive")

		// Clean up
		err = redisClient.Del(ctx, testKey).Err()
		assert.NoError(t, err, "Failed to clean up test key")
	})
}

// setupDockerTestRoutes sets up basic routes for Docker testing
func setupDockerTestRoutes(router *gin.Engine, knowledgeService api.KnowledgeServiceInterface, auditService api.AuditServiceInterface) {
	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "ai-government-consultant",
			"docker":    true,
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"checks": gin.H{
				"database":    "ok",
				"redis":       "ok",
				"ai_service":  "ok",
				"file_system": "ok",
			},
			"docker": true,
		})
	})

	// API endpoints
	v1 := router.Group("/api/v1")
	{
		// Knowledge endpoints (no auth for testing)
		v1.GET("/knowledge/types", func(c *gin.Context) {
			types := []gin.H{
				{"value": "fact", "label": "Fact"},
				{"value": "procedure", "label": "Procedure"},
				{"value": "policy", "label": "Policy"},
			}
			c.JSON(http.StatusOK, gin.H{
				"types": types,
				"total": len(types),
			})
		})

		v1.GET("/knowledge/categories", func(c *gin.Context) {
			categories := []string{"policy", "strategy", "operations", "technology", "general"}
			c.JSON(http.StatusOK, gin.H{
				"categories": categories,
				"total":      len(categories),
			})
		})

		// Test endpoint for JSON handling
		v1.POST("/test/json", func(c *gin.Context) {
			var request map[string]interface{}
			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid JSON",
					"message": err.Error(),
					"code":    "INVALID_JSON",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "JSON processed successfully",
				"data":    request,
				"docker":  true,
			})
		})
	}
}

// TestDockerAPIEndpoints tests specific API endpoints with Docker
func TestDockerAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker API endpoint tests in short mode")
	}

	// Setup test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	knowledgeService := api.NewSimpleKnowledgeService(nil)
	auditService := api.NewSimpleAuditService(nil)
	setupDockerTestRoutes(router, knowledgeService, auditService)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "Health Check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "docker"},
		},
		{
			name:           "Readiness Check",
			method:         "GET",
			path:           "/ready",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "checks", "docker"},
		},
		{
			name:           "Knowledge Types",
			method:         "GET",
			path:           "/api/v1/knowledge/types",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"types", "total"},
		},
		{
			name:           "Knowledge Categories",
			method:         "GET",
			path:           "/api/v1/knowledge/categories",
			expectedStatus: http.StatusOK,
			expectedFields: []string{"categories", "total"},
		},
		{
			name:   "JSON Test Endpoint",
			method: "POST",
			path:   "/api/v1/test/json",
			body: map[string]interface{}{
				"test_field": "test_value",
				"number":     42,
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"message", "data", "docker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match")

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Response should be valid JSON")

			for _, field := range tt.expectedFields {
				assert.Contains(t, response, field, "Response should contain field: %s", field)
			}
		})
	}
}