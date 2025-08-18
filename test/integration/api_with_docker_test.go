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
	"ai-government-consultant/internal/auth"
	"ai-government-consultant/internal/config"
	"ai-government-consultant/internal/consultation"
	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// TestAPIWithDockerContainers tests API endpoints using Docker containers
func TestAPIWithDockerContainers(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping Docker-based API tests in short mode")
	}

	// Setup test server with Docker containers
	ts := setupTestServerWithDocker(t)
	defer ts.teardown(t)

	t.Run("Health Endpoints with Docker", func(t *testing.T) {
		// Test health endpoint
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	t.Run("Authentication Flow with Docker", func(t *testing.T) {
		// Test user registration
		registerReq := map[string]interface{}{
			"email":              "dockertest@example.com",
			"name":               "Docker Test User",
			"department":         "Test Department",
			"role":               "analyst",
			"security_clearance": "internal",
			"password":           "dockertest123",
		}

		reqBody, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User registered successfully", response["message"])

		// Test user login
		loginReq := map[string]interface{}{
			"email":    "dockertest@example.com",
			"password": "dockertest123",
		}

		reqBody, _ = json.Marshal(loginReq)
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Login successful", response["message"])
		assert.Contains(t, response, "tokens")
	})

	t.Run("Knowledge Management with Docker", func(t *testing.T) {
		// Test creating knowledge item
		createReq := map[string]interface{}{
			"title":    "Docker Test Knowledge",
			"content":  "This is a test knowledge item created during Docker testing",
			"type":     "fact",
			"category": "test",
			"tags":     []string{"docker", "test", "integration"},
		}

		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/knowledge", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+ts.testToken)

		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Knowledge item created successfully", response["message"])

		// Test listing knowledge items
		req, _ = http.NewRequest("GET", "/api/v1/knowledge", nil)
		req.Header.Set("Authorization", "Bearer "+ts.testToken)

		w = httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "knowledge_items")
	})

	t.Run("Document Management with Docker", func(t *testing.T) {
		// Test listing documents
		req, _ := http.NewRequest("GET", "/api/v1/documents", nil)
		req.Header.Set("Authorization", "Bearer "+ts.testToken)

		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "documents")
		assert.Contains(t, response, "total")
	})

	t.Run("Audit Endpoints with Docker", func(t *testing.T) {
		// Test getting audit logs
		req, _ := http.NewRequest("GET", "/api/v1/audit/logs", nil)
		req.Header.Set("Authorization", "Bearer "+ts.testToken)

		w := httptest.NewRecorder()
		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "audit_logs")
	})
}

// dockerTestServer holds the test server setup for Docker-based tests
type dockerTestServer struct {
	router      *gin.Engine
	authService *auth.AuthService
	jwtService  *auth.JWTService
	mongoClient *mongo.Client
	redisClient *redis.Client
	testUser    *models.User
	testToken   string
}

// setupTestServerWithDocker creates a test server using Docker containers
func setupTestServerWithDocker(t *testing.T) *dockerTestServer {
	// Configuration for Docker containers
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MongoURI: "mongodb://testadmin:testpassword@localhost:27018/ai_government_consultant_test?authSource=admin",
			Database: "ai_government_consultant_test",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6380",
			Password: "testpassword",
			DB:       0,
		},
		AI: config.AIConfig{
			LLMAPIKey: "test-key-for-docker-tests",
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-jwt-secret-for-docker-tests",
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	// Initialize MongoDB
	mongoClient, err := database.NewMongoClient(cfg.Database.MongoURI)
	require.NoError(t, err, "Failed to connect to Docker MongoDB")

	db := mongoClient.Database(cfg.Database.Database)

	// Clean up test database
	err = db.Drop(context.Background())
	require.NoError(t, err, "Failed to drop test database")
	
	// Wait a moment for the drop to complete
	time.Sleep(100 * time.Millisecond)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = redisClient.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Docker Redis")

	// Initialize logger with test-friendly configuration
	testLoggingConfig := config.LoggingConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout", // Use stdout for tests
	}
	log := logger.New(testLoggingConfig)

	// Initialize services
	documentService := document.NewService(db)
	knowledgeService := api.NewSimpleKnowledgeService(db)
	auditService := api.NewSimpleAuditService(db)

	// Initialize auth service
	jwtConfig := auth.JWTConfig{
		AccessSecret:  cfg.Security.JWTSecret,
		RefreshSecret: cfg.Security.JWTSecret + "_refresh",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
		SessionTTL:    24 * time.Hour,
		BlacklistTTL:  7 * 24 * time.Hour,
		Issuer:        "ai-government-consultant-docker-test",
	}
	authService := auth.NewAuthService(db.Collection("users"), redisClient, jwtConfig)
	
	// Create JWT service for token generation
	jwtService := auth.NewJWTService(
		cfg.Security.JWTSecret,
		cfg.Security.JWTSecret+"_refresh",
		15*time.Minute,
		7*24*time.Hour,
		"ai-government-consultant-docker-test",
	)

	// Initialize consultation service
	embeddingService := &embedding.Service{} // Mock service
	consultationConfig := &consultation.Config{
		GeminiAPIKey:     cfg.AI.LLMAPIKey,
		MongoDB:          db,
		Redis:            redisClient,
		EmbeddingService: embeddingService,
		Logger:           log,
		RateLimit: consultation.RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
	}
	consultationService, err := consultation.NewService(consultationConfig)
	require.NoError(t, err, "Failed to initialize consultation service")

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup API routes
	routerConfig := &api.RouterConfig{
		AuthService:         authService,
		DocumentService:     documentService,
		ConsultationService: consultationService,
		KnowledgeService:    knowledgeService,
		AuditService:        auditService,
		AllowedOrigins:      []string{"http://localhost:3000"},
	}
	api.SetupRoutes(router, routerConfig)

	// Create test user with unique email
	testEmail := fmt.Sprintf("dockertest-%d@example.com", time.Now().UnixNano())
	testUser := &models.User{
		Email:             testEmail,
		Name:              "Docker Test User",
		Department:        "Test Department",
		Role:              models.UserRoleAdmin,
		SecurityClearance: models.SecurityClearanceSecret,
		Permissions: []models.Permission{
			{Resource: "documents", Actions: []string{"read", "write", "delete"}},
			{Resource: "consultations", Actions: []string{"read", "write", "delete"}},
			{Resource: "knowledge", Actions: []string{"read", "write", "delete"}},
			{Resource: "audit", Actions: []string{"read", "write", "export", "report"}},
		},
	}

	err = authService.RegisterUser(context.Background(), testUser, "dockertest123")
	require.NoError(t, err, "Failed to create test user")

	// Generate test token
	tokenPair, err := jwtService.GenerateTokenPair(
		testUser.ID,
		testUser.Email,
		string(testUser.Role),
		string(testUser.SecurityClearance),
		[]string{"documents:read", "documents:write", "consultations:read", "consultations:write"},
	)
	require.NoError(t, err, "Failed to generate test token")

	return &dockerTestServer{
		router:      router,
		authService: authService,
		jwtService:  jwtService,
		mongoClient: mongoClient,
		redisClient: redisClient,
		testUser:    testUser,
		testToken:   tokenPair.AccessToken,
	}
}

// teardown cleans up the Docker test server
func (ts *dockerTestServer) teardown(t *testing.T) {
	// Clean up test database
	if ts.mongoClient != nil {
		db := ts.mongoClient.Database("ai_government_consultant_test")
		err := db.Drop(context.Background())
		require.NoError(t, err, "Failed to drop test database")
		ts.mongoClient.Disconnect(context.Background())
	}

	// Close Redis connection
	if ts.redisClient != nil {
		err := ts.redisClient.Close()
		require.NoError(t, err, "Failed to close Redis connection")
	}
}