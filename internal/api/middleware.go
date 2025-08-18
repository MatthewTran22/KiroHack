package api

import (
	"net/http"
	"strings"

	"ai-government-consultant/internal/auth"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Missing authorization header",
				Code:  "MISSING_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		// Extract token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid authorization header format",
				Code:  "INVALID_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Validate token
		validation, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			switch err {
			case models.ErrInvalidToken:
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error: "Invalid token",
					Code:  "INVALID_TOKEN",
				})
			case models.ErrTokenBlacklisted:
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error: "Token has been revoked",
					Code:  "TOKEN_REVOKED",
				})
			case models.ErrUserInactive:
				c.JSON(http.StatusForbidden, ErrorResponse{
					Error: "Account inactive",
					Code:  "ACCOUNT_INACTIVE",
				})
			default:
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "Token validation failed",
					Message: err.Error(),
					Code:    "TOKEN_VALIDATION_FAILED",
				})
			}
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", validation.User)
		c.Set("claims", validation.Claims)

		c.Next()
	}
}

// OptionalAuthMiddleware creates optional authentication middleware
// This allows endpoints to work with or without authentication
func OptionalAuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without user context
			c.Next()
			return
		}

		// Extract token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			// Invalid format, continue without user context
			c.Next()
			return
		}

		token := tokenParts[1]

		// Validate token
		validation, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Invalid token, continue without user context
			c.Next()
			return
		}

		// Set user in context if valid
		c.Set("user", validation.User)
		c.Set("claims", validation.Claims)

		c.Next()
	}
}

// RequirePermission creates middleware that requires specific permissions
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "User not authenticated",
				Code:  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		user := userInterface.(*models.User)

		// Check permission
		if !user.HasPermission(resource, action) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "Insufficient permissions",
				Code:  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole creates middleware that requires specific user roles
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "User not authenticated",
				Code:  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		user := userInterface.(*models.User)

		// Check if user has any of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "Insufficient role permissions",
				Code:  "INSUFFICIENT_ROLE",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSecurityClearance creates middleware that requires minimum security clearance
func RequireSecurityClearance(minClearance models.SecurityClearance) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "User not authenticated",
				Code:  "NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		user := userInterface.(*models.User)

		// Define clearance hierarchy
		clearanceLevel := map[models.SecurityClearance]int{
			models.SecurityClearancePublic:       1,
			models.SecurityClearanceInternal:     2,
			models.SecurityClearanceConfidential: 3,
			models.SecurityClearanceSecret:       4,
			models.SecurityClearanceTopSecret:    5,
		}

		userLevel := clearanceLevel[user.SecurityClearance]
		requiredLevel := clearanceLevel[minClearance]

		if userLevel < requiredLevel {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: "Insufficient security clearance",
				Code:  "INSUFFICIENT_CLEARANCE",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Note: In a real implementation, you would use a proper rate limiter
	// like redis-based rate limiting or in-memory rate limiter
	return func(c *gin.Context) {
		// For now, just continue - implement actual rate limiting logic here
		c.Next()
	}
}

// AuditMiddleware creates audit logging middleware
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (if authenticated)
		var userID string
		if userInterface, exists := c.Get("user"); exists {
			if user, ok := userInterface.(*models.User); ok {
				userID = user.ID.Hex()
			}
		}

		// Log the request
		// Note: In a real implementation, you would use the audit service
		// to log this activity
		_ = userID

		c.Next()

		// Log the response
		// Note: You could also log response status, duration, etc.
	}
}

// CORSMiddleware creates CORS middleware with configurable origins
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if the origin is in the allowed list
		allowedOrigin := ""
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				allowedOrigin = origin
				break
			}
		}

		// Set CORS headers
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Session-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or get request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID (in production, use UUID)
			requestID = "req-" + strings.ReplaceAll(c.Request.URL.Path, "/", "-")
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// ContentTypeMiddleware ensures JSON content type for API endpoints
func ContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For POST, PUT, PATCH requests, ensure content type is JSON
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "multipart/form-data") {
				c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{
					Error: "Content-Type must be application/json or multipart/form-data",
					Code:  "UNSUPPORTED_MEDIA_TYPE",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}