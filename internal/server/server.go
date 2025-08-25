package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-government-consultant/internal/api"
	"ai-government-consultant/internal/auth"
	"ai-government-consultant/internal/config"
	"ai-government-consultant/internal/consultation"
	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/embedding"
	"ai-government-consultant/internal/websocket"
	"ai-government-consultant/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Server represents the HTTP server
type Server struct {
	config              *config.Config
	router              *gin.Engine
	logger              logger.Logger
	authService         *auth.AuthService
	documentService     *document.Service
	consultationService *consultation.Service
	knowledgeService    api.KnowledgeServiceInterface
	auditService        api.AuditServiceInterface
	wsHub               *websocket.Hub
	wsHandler           *websocket.Handler
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Initialize logger
	log := logger.New(cfg.Logging)

	// Set Gin mode based on environment
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Add basic middleware
	router.Use(gin.Recovery())
	router.Use(logger.GinMiddleware(log))

	return &Server{
		config: cfg,
		router: router,
		logger: log,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Initialize services
	if err := s.initializeServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting server", map[string]interface{}{
			"host": s.config.Server.Host,
			"port": s.config.Server.Port,
		})

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server failed to start", err, nil)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...", nil)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", err, nil)
		return err
	}

	s.logger.Info("Server exited", nil)
	return nil
}

// initializeServices initializes all the services
func (s *Server) initializeServices() error {
	// Initialize MongoDB
	mongoClient, err := database.NewMongoClient(s.config.Database.MongoURI)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	db := mongoClient.Database(s.config.Database.Database)

	// Initialize database (create indexes, default admin user, etc.)
	mongodb := &database.MongoDB{
		Client:   mongoClient,
		Database: db,
	}
	if err := database.InitializeDatabase(mongodb); err != nil {
		s.logger.Error("Failed to initialize database", err, nil)
		// Don't fail startup, just log the error
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", s.config.Redis.Host, s.config.Redis.Port),
		Password: s.config.Redis.Password,
		DB:       s.config.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize services
	s.documentService = document.NewService(db)
	s.knowledgeService = api.NewSimpleKnowledgeService(db)
	s.auditService = api.NewSimpleAuditService(db)

	// Initialize auth service
	jwtConfig := auth.JWTConfig{
		AccessSecret:  s.config.Security.JWTSecret,
		RefreshSecret: s.config.Security.JWTSecret + "_refresh",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
		SessionTTL:    24 * time.Hour,
		BlacklistTTL:  7 * 24 * time.Hour,
		Issuer:        "ai-government-consultant",
	}
	s.authService = auth.NewAuthService(db.Collection("users"), redisClient, jwtConfig)

	// Initialize embedding service (placeholder)
	embeddingService := &embedding.Service{} // This would be properly initialized

	// Initialize consultation service
	consultationConfig := &consultation.Config{
		GeminiAPIKey:     s.config.AI.LLMAPIKey,
		MongoDB:          db,
		Redis:            redisClient,
		EmbeddingService: embeddingService,
		Logger:           s.logger,
		RateLimit: consultation.RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
	}
	s.consultationService, err = consultation.NewService(consultationConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize consultation service: %w", err)
	}

	// Initialize WebSocket hub and handler
	s.wsHub = websocket.NewHub()
	s.wsHandler = websocket.NewHandler(s.wsHub, s.authService)

	// Start WebSocket hub in a goroutine
	go s.wsHub.Run()

	s.logger.Info("All services initialized successfully", nil)
	return nil
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Define allowed origins
	allowedOrigins := []string{
		"http://localhost:3000",
		"https://localhost:3000",
		"http://localhost:8080",
		"https://localhost:8080",
	}

	// Setup WebSocket route with minimal middleware
	wsGroup := s.router.Group("/")
	wsGroup.Use(func(c *gin.Context) {
		// Add CORS headers for WebSocket
		origin := c.GetHeader("Origin")
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				break
			}
		}
		c.Next()
	})
	
	// Add a test endpoint to verify routing works
	wsGroup.GET("/ws-test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "WebSocket routing works"})
	})
	
	wsGroup.GET("/ws", s.wsHandler.HandleWebSocket)
	s.logger.Info("WebSocket route registered at /ws", nil)

	// Setup API routes
	routerConfig := &api.RouterConfig{
		AuthService:         s.authService,
		DocumentService:     s.documentService,
		ConsultationService: s.consultationService,
		KnowledgeService:    s.knowledgeService,
		AuditService:        s.auditService,
		SpeechService:       nil, // Speech service is optional
		AllowedOrigins:      allowedOrigins,
	}

	api.SetupRoutes(s.router, routerConfig)

	s.logger.Info("API routes and WebSocket configured successfully", nil)
}
