package audit

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditMiddleware creates a Gin middleware for automatic audit logging
func AuditMiddleware(auditService Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		requestID := uuid.New().String()

		// Add request ID to context
		c.Set("request_id", requestID)

		// Create enriched context with audit information
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		ctx = context.WithValue(ctx, "ip_address", c.ClientIP())
		ctx = context.WithValue(ctx, "user_agent", c.GetHeader("User-Agent"))

		// Get user ID from context if available (set by auth middleware)
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				ctx = context.WithValue(ctx, "user_id", uid)
			}
		}

		if sessionID, exists := c.Get("session_id"); exists {
			if sid, ok := sessionID.(string); ok {
				ctx = context.WithValue(ctx, "session_id", sid)
			}
		}

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Determine result based on status code
		result := "success"
		if c.Writer.Status() >= 400 {
			result = "failure"
		} else if c.Writer.Status() >= 300 {
			result = "partial"
		}

		// Create audit entry
		entry := AuditEntry{
			ID:        uuid.New().String(),
			Timestamp: startTime,
			EventType: determineEventType(c.Request.Method, c.Request.URL.Path),
			Level:     determineLevelFromStatus(c.Writer.Status()),
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Resource:  c.Request.URL.Path,
			Action:    c.Request.Method,
			Result:    result,
			RequestID: requestID,
			Duration:  &duration,
			Details: map[string]interface{}{
				"method":        c.Request.Method,
				"path":          c.Request.URL.Path,
				"query":         c.Request.URL.RawQuery,
				"status_code":   c.Writer.Status(),
				"response_size": c.Writer.Size(),
			},
		}

		// Add user information if available
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				entry.UserID = &uid
			}
		}

		if sessionID, exists := c.Get("session_id"); exists {
			if sid, ok := sessionID.(string); ok {
				entry.SessionID = &sid
			}
		}

		// Add error information if request failed
		if c.Writer.Status() >= 400 {
			if errors := c.Errors; len(errors) > 0 {
				errorCode := "HTTP_ERROR"
				errorMessage := errors.Last().Error()
				entry.ErrorCode = &errorCode
				entry.ErrorMessage = &errorMessage
			}
		}

		// Log the audit entry (don't block the response)
		go func() {
			if err := auditService.LogEvent(context.Background(), entry); err != nil {
				// Log error but don't fail the request
				// In production, you might want to use a proper logger here
				println("Failed to log audit entry:", err.Error())
			}
		}()
	}
}

// determineEventType maps HTTP method and path to audit event type
func determineEventType(method, path string) AuditEventType {
	switch {
	case method == "POST" && contains(path, "/auth/login"):
		return EventUserLogin
	case method == "POST" && contains(path, "/auth/logout"):
		return EventUserLogout
	case method == "POST" && contains(path, "/documents"):
		return EventDocumentUploaded
	case method == "GET" && contains(path, "/documents"):
		return EventDocumentAccessed
	case method == "DELETE" && contains(path, "/documents"):
		return EventDocumentDeleted
	case method == "PUT" && contains(path, "/documents"):
		return EventDocumentUpdated
	case method == "POST" && contains(path, "/consultations"):
		return EventConsultationStarted
	case method == "GET" && contains(path, "/consultations"):
		return EventConsultationCompleted
	case method == "POST" && contains(path, "/knowledge"):
		return EventKnowledgeAdded
	case method == "GET" && contains(path, "/knowledge"):
		return EventKnowledgeAccessed
	case method == "PUT" && contains(path, "/knowledge"):
		return EventKnowledgeUpdated
	case method == "DELETE" && contains(path, "/knowledge"):
		return EventKnowledgeDeleted
	default:
		return AuditEventType("API_REQUEST")
	}
}

// determineLevelFromStatus determines audit level based on HTTP status code
func determineLevelFromStatus(statusCode int) AuditLevel {
	switch {
	case statusCode >= 500:
		return AuditLevelError
	case statusCode >= 400:
		return AuditLevelWarning
	case statusCode >= 300:
		return AuditLevelInfo
	default:
		return AuditLevelInfo
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr))
}

// SecurityEventMiddleware creates middleware for detecting security events
func SecurityEventMiddleware(auditService Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request first
		c.Next()

		// Check for security events after request processing
		go func() {
			ctx := context.Background()

			// Check for failed authentication attempts
			if c.Writer.Status() == 401 {
				userID := getUserIDFromContext(c)
				auditService.LogSecurityEvent(
					ctx,
					EventUserLoginFailed,
					userID,
					c.ClientIP(),
					c.Request.URL.Path,
					map[string]interface{}{
						"method":     c.Request.Method,
						"path":       c.Request.URL.Path,
						"user_agent": c.GetHeader("User-Agent"),
						"status":     c.Writer.Status(),
					},
				)
			}

			// Check for unauthorized access attempts
			if c.Writer.Status() == 403 {
				userID := getUserIDFromContext(c)
				auditService.LogSecurityEvent(
					ctx,
					EventUnauthorizedAccess,
					userID,
					c.ClientIP(),
					c.Request.URL.Path,
					map[string]interface{}{
						"method":     c.Request.Method,
						"path":       c.Request.URL.Path,
						"user_agent": c.GetHeader("User-Agent"),
						"status":     c.Writer.Status(),
					},
				)
			}

			// Check for suspicious activity (multiple rapid requests)
			if userID := getUserIDFromContext(c); userID != nil {
				alerts, err := auditService.DetectAnomalies(ctx, *userID, 5*time.Minute)
				if err == nil && len(alerts) > 0 {
					for _, alert := range alerts {
						auditService.CreateSecurityAlert(ctx, alert)
					}
				}
			}
		}()
	}
}

// getUserIDFromContext extracts user ID from Gin context
func getUserIDFromContext(c *gin.Context) *string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return &uid
		}
	}
	return nil
}
