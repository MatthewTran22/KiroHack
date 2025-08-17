package audit

import (
	"ai-government-consultant/internal/config"
	"ai-government-consultant/internal/database"

	"github.com/gin-gonic/gin"
)

// IntegrateAuditSystem shows how to integrate the audit system into the main server
func IntegrateAuditSystem(router *gin.Engine, cfg *config.Config) (Service, error) {
	// Initialize MongoDB connection
	db, err := database.NewMongoDB(cfg.Database)
	if err != nil {
		return nil, err
	}

	// Create audit repository and service
	auditRepo := NewMongoRepository(db)
	auditService := NewService(auditRepo)

	// Add audit middleware to all routes
	router.Use(AuditMiddleware(auditService))
	router.Use(SecurityEventMiddleware(auditService))

	// Add audit API routes
	auditHandler := NewAuditHandler(auditService)
	auditGroup := router.Group("/api/v1/audit")
	{
		// Audit logs endpoints
		auditGroup.GET("/logs", auditHandler.SearchAuditLogs)
		auditGroup.GET("/logs/:id", auditHandler.GetAuditEntry)
		auditGroup.GET("/users/:user_id/activity", auditHandler.GetUserActivity)

		// Data lineage endpoints
		auditGroup.GET("/lineage/:data_id", auditHandler.GetDataLineage)
		auditGroup.GET("/provenance/:data_id", auditHandler.GetDataProvenance)

		// Audit reports endpoints
		auditGroup.POST("/reports", auditHandler.GenerateAuditReport)
		auditGroup.GET("/reports", auditHandler.ListAuditReports)
		auditGroup.GET("/reports/:id", auditHandler.GetAuditReport)
		auditGroup.GET("/reports/:id/export", auditHandler.ExportAuditReport)

		// Security endpoints
		securityGroup := auditGroup.Group("/security")
		{
			securityGroup.GET("/alerts", auditHandler.GetSecurityAlerts)
			securityGroup.PUT("/alerts/:id", auditHandler.UpdateSecurityAlert)
			securityGroup.POST("/anomalies", auditHandler.DetectAnomalies)
		}

		// Compliance endpoints
		complianceGroup := auditGroup.Group("/compliance")
		{
			complianceGroup.POST("/reports", auditHandler.GenerateComplianceReport)
			complianceGroup.GET("/reports/:id", auditHandler.GetComplianceReport)
			complianceGroup.GET("/validate/:standard", auditHandler.ValidateCompliance)
		}
	}

	return auditService, nil
}

// Example usage in server.go:
//
// func (s *Server) setupRoutes() {
//     // Initialize audit system
//     auditService, err := audit.IntegrateAuditSystem(s.router, s.config)
//     if err != nil {
//         s.logger.Error("Failed to initialize audit system", err, nil)
//         return
//     }
//
//     // Log system startup
//     auditService.LogSystemEvent(context.Background(), audit.EventSystemStartup, map[string]interface{}{
//         "version": "1.0.0",
//         "environment": s.config.Environment,
//     })
//
//     // Health check endpoint
//     s.router.GET("/health", s.healthCheck)
//
//     // API version 1 routes
//     v1 := s.router.Group("/api/v1")
//     {
//         // Your existing routes...
//     }
// }
