package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-government-consultant/internal/config"
	"ai-government-consultant/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	router *gin.Engine
	logger logger.Logger
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

	// Add middleware
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

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// API version 1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Placeholder routes - will be implemented in later tasks
		v1.GET("/status", s.statusCheck)
	}
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// statusCheck handles status check requests
func (s *Server) statusCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "operational",
		"service": "ai-government-consultant",
	})
}
