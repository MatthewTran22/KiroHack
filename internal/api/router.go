package api

import (
	"ai-government-consultant/internal/auth"
	"ai-government-consultant/internal/consultation"
	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/internal/speech"

	"github.com/gin-gonic/gin"
)

// RouterConfig holds the configuration for setting up API routes
type RouterConfig struct {
	AuthService         *auth.AuthService
	DocumentService     *document.Service
	ConsultationService *consultation.Service
	KnowledgeService    KnowledgeServiceInterface
	AuditService        AuditServiceInterface
	SpeechService       *speech.SpeechService
	AllowedOrigins      []string
}

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, config *RouterConfig) {
	// Create handlers
	authHandler := NewAuthHandler(config.AuthService)
	documentHandler := NewDocumentHandler(config.DocumentService)
	consultationHandler := NewConsultationHandler(config.ConsultationService)
	knowledgeHandler := NewKnowledgeHandler(config.KnowledgeService)
	auditHandler := NewAuditHandler(config.AuditService)
	var speechHandler *SpeechHandler
	if config.SpeechService != nil {
		speechHandler = NewSpeechHandler(config.SpeechService)
	}

	// Global middleware
	router.Use(SecurityHeadersMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(CORSMiddleware(config.AllowedOrigins))
	router.Use(ContentTypeMiddleware())
	router.Use(AuditMiddleware())

	// Health check endpoints (no auth required)
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// API version 1
	v1 := router.Group("/api/v1")
	{
		// Public authentication endpoints (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/validate", authHandler.ValidateToken)
		}

		// Protected authentication endpoints (auth required)
		authProtected := v1.Group("/auth")
		authProtected.Use(AuthMiddleware(config.AuthService))
		{
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.PUT("/profile", authHandler.UpdateProfile)
			authProtected.POST("/change-password", authHandler.ChangePassword)
			authProtected.POST("/mfa/setup", authHandler.SetupMFA)
			authProtected.POST("/mfa/enable", authHandler.EnableMFA)
			authProtected.POST("/mfa/disable", authHandler.DisableMFA)
		}

		// User management endpoints (admin only)
		users := v1.Group("/users")
		users.Use(AuthMiddleware(config.AuthService))
		users.Use(RequireRole(models.UserRoleAdmin))
		{
			users.GET("", authHandler.ListUsers)
			users.GET("/:id", authHandler.GetUser)
		}

		// Document management endpoints
		documents := v1.Group("/documents")
		documents.Use(AuthMiddleware(config.AuthService))
		{
			documents.POST("", documentHandler.UploadDocument)
			documents.GET("", documentHandler.ListDocuments)
			documents.POST("/search", documentHandler.SearchDocuments)
			documents.POST("/validate", documentHandler.ValidateDocument)

			// Document-specific endpoints
			documents.GET("/:id", documentHandler.GetDocument)
			documents.PUT("/:id", documentHandler.UpdateDocument)
			documents.DELETE("/:id", documentHandler.DeleteDocument)
			documents.POST("/:id/process", documentHandler.ProcessDocument)
			documents.GET("/:id/status", documentHandler.GetProcessingStatus)
			documents.GET("/:id/content", documentHandler.GetDocumentContent)
			documents.GET("/:id/file", documentHandler.GetDocumentFile)
		}

		// Consultation endpoints
		consultations := v1.Group("/consultations")
		consultations.Use(AuthMiddleware(config.AuthService))
		{
			consultations.POST("", consultationHandler.CreateConsultation)
			consultations.GET("", consultationHandler.ListConsultations)
			consultations.POST("/search", consultationHandler.SearchConsultations)
			consultations.GET("/history", consultationHandler.GetConsultationHistory)
			consultations.GET("/analytics", RequireRole(models.UserRoleAdmin), consultationHandler.GetConsultationAnalytics)

			// Session-specific endpoints
			consultations.GET("/:id", consultationHandler.GetConsultation)
			consultations.POST("/:id/continue", consultationHandler.ContinueConsultation)
			consultations.DELETE("/:id", consultationHandler.DeleteConsultation)
		}

		// Recommendation endpoints (separate path to avoid route conflicts)
		recommendations := v1.Group("/recommendations")
		recommendations.Use(AuthMiddleware(config.AuthService))
		{
			recommendations.GET("/:session_id/:recommendation_id", consultationHandler.GetRecommendation)
			recommendations.GET("/:session_id/:recommendation_id/explain", consultationHandler.ExplainRecommendation)
		}

		// Knowledge management endpoints
		knowledge := v1.Group("/knowledge")
		knowledge.Use(AuthMiddleware(config.AuthService))
		{
			knowledge.POST("", knowledgeHandler.CreateKnowledge)
			knowledge.GET("", knowledgeHandler.ListKnowledge)
			knowledge.POST("/search", knowledgeHandler.SearchKnowledge)
			knowledge.GET("/categories", knowledgeHandler.GetKnowledgeCategories)
			knowledge.GET("/types", knowledgeHandler.GetKnowledgeTypes)
			knowledge.GET("/stats", knowledgeHandler.GetKnowledgeStats)
			knowledge.GET("/graph", knowledgeHandler.GetKnowledgeGraph)

			// Knowledge item-specific endpoints
			knowledge.GET("/:id", knowledgeHandler.GetKnowledge)
			knowledge.PUT("/:id", knowledgeHandler.UpdateKnowledge)
			knowledge.DELETE("/:id", knowledgeHandler.DeleteKnowledge)
			knowledge.GET("/:id/related", knowledgeHandler.GetRelatedKnowledge)
		}

		// Speech services endpoints (only if speech service is available)
		if speechHandler != nil {
			speech := v1.Group("/speech")
			speech.Use(AuthMiddleware(config.AuthService))
			{
				// Session management
				speech.POST("/sessions", speechHandler.CreateSpeechSession)
				speech.GET("/sessions/history", speechHandler.GetSessionHistory)
				speech.DELETE("/sessions/:sessionId", speechHandler.EndSession)

				// Voice processing within sessions
				speech.POST("/sessions/:sessionId/query", speechHandler.ProcessVoiceQuery)
				speech.POST("/sessions/:sessionId/response", speechHandler.GenerateVoiceResponse)

				// Direct speech processing (without sessions)
				speech.POST("/transcribe", speechHandler.TranscribeAudio)
				speech.POST("/synthesize", speechHandler.SynthesizeSpeech)

				// Audio file upload and validation
				speech.POST("/upload", speechHandler.UploadAudioFile)

				// Voice authentication
				voiceAuth := speech.Group("/auth")
				{
					voiceAuth.POST("/enroll", speechHandler.EnrollVoice)
					voiceAuth.POST("/authenticate", speechHandler.AuthenticateVoice)
				}

				// Service capabilities
				speech.GET("/voices", speechHandler.GetAvailableVoices)
				speech.GET("/languages", speechHandler.GetSupportedLanguages)
			}
		}

		// Audit and reporting endpoints
		audit := v1.Group("/audit")
		audit.Use(AuthMiddleware(config.AuthService))
		{
			// Audit log endpoints (admin or audit role required)
			auditLogs := audit.Group("/logs")
			auditLogs.Use(RequirePermission("audit", "read"))
			{
				auditLogs.GET("", auditHandler.GetAuditLogs)
				auditLogs.GET("/:id", auditHandler.GetAuditLog)
				auditLogs.POST("/export", RequirePermission("audit", "export"), auditHandler.ExportAuditLogs)
			}

			// Report endpoints (admin or audit manager required)
			reports := audit.Group("/reports")
			reports.Use(RequirePermission("audit", "report"))
			{
				reports.POST("/generate", auditHandler.GenerateAuditReport)
				reports.GET("/compliance", auditHandler.GetComplianceReport)
			}

			// Data lineage endpoints
			audit.GET("/lineage", RequirePermission("audit", "read"), auditHandler.TrackDataLineage)

			// Activity endpoints
			audit.GET("/activity/system", RequireRole(models.UserRoleAdmin), auditHandler.GetSystemActivity)
			audit.GET("/activity/users/:user_id", auditHandler.GetUserActivity)
		}

		// System information endpoints (admin only)
		system := v1.Group("/system")
		system.Use(AuthMiddleware(config.AuthService))
		system.Use(RequireRole(models.UserRoleAdmin))
		{
			system.GET("/info", getSystemInfo)
			system.GET("/metrics", getSystemMetrics)
			system.GET("/config", getSystemConfig)
		}
	}

	// API documentation endpoint (optional auth)
	v1.GET("/docs", OptionalAuthMiddleware(config.AuthService), getAPIDocumentation)
}

// Health check handlers
func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": gin.H{},
		"version":   "1.0.0",
		"service":   "ai-government-consultant",
	})
}

func readinessCheck(c *gin.Context) {
	// In a real implementation, you would check:
	// - Database connectivity
	// - External service availability
	// - System resources
	c.JSON(200, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database":    "ok",
			"redis":       "ok",
			"ai_service":  "ok",
			"file_system": "ok",
		},
	})
}

// System information handlers (admin only)
func getSystemInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"system": gin.H{
			"name":        "AI Government Consultant",
			"version":     "1.0.0",
			"environment": "development",
			"uptime":      "0h 0m 0s",
			"go_version":  "1.23.0",
		},
	})
}

func getSystemMetrics(c *gin.Context) {
	c.JSON(200, gin.H{
		"metrics": gin.H{
			"requests_total":      0,
			"requests_per_second": 0.0,
			"response_time_avg":   "0ms",
			"memory_usage":        "0MB",
			"cpu_usage":           "0%",
			"active_sessions":     0,
		},
	})
}

func getSystemConfig(c *gin.Context) {
	// Return non-sensitive configuration information
	c.JSON(200, gin.H{
		"config": gin.H{
			"max_file_size":         "50MB",
			"supported_formats":     []string{"pdf", "doc", "docx", "txt"},
			"max_consultation_time": "60s",
			"rate_limits": gin.H{
				"requests_per_minute": 60,
				"burst_size":          10,
			},
		},
	})
}

func getAPIDocumentation(c *gin.Context) {
	// Return API documentation or redirect to Swagger UI
	c.JSON(200, gin.H{
		"documentation": gin.H{
			"title":       "AI Government Consultant API",
			"version":     "1.0.0",
			"description": "REST API for AI-powered government consulting platform",
			"swagger_url": "/api/v1/swagger.json",
			"endpoints": gin.H{
				"authentication": "/api/v1/auth/*",
				"documents":      "/api/v1/documents/*",
				"consultations":  "/api/v1/consultations/*",
				"knowledge":      "/api/v1/knowledge/*",
				"speech":         "/api/v1/speech/*",
				"audit":          "/api/v1/audit/*",
				"system":         "/api/v1/system/*",
			},
		},
	})
}
