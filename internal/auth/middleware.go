package auth

import (
	"net/http"
	"strings"
	"time"

	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides authentication middleware for Gin
type AuthMiddleware struct {
	authService  *AuthService
	authzService *AuthorizationService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *AuthService, authzService *AuthorizationService) *AuthMiddleware {
	return &AuthMiddleware{
		authService:  authService,
		authzService: authzService,
	}
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization token",
			})
			c.Abort()
			return
		}

		// Validate token
		validation, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}

		// Store user and claims in context
		c.Set("user", validation.User)
		c.Set("claims", validation.Claims)
		c.Set("token", token)

		// Update session activity if session ID is provided
		if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
			m.authService.sessionService.UpdateLastActivity(c.Request.Context(), sessionID)
		}

		c.Next()
	}
}

// RequireRole middleware that requires a specific role
func (m *AuthMiddleware) RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.getCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "insufficient role permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware that requires a specific permission
func (m *AuthMiddleware) RequirePermission(resource Resource, action Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.getCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Check authorization
		authorized, err := m.authzService.Authorize(c.Request.Context(), user, resource, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "authorization check failed",
			})
			c.Abort()
			return
		}

		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSecurityClearance middleware that requires a specific security clearance
func (m *AuthMiddleware) RequireSecurityClearance(requiredClearance string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.getCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		if !user.CanAccessClassification(requiredClearance) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "insufficient security clearance",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissionWithClearance middleware that requires both permission and security clearance
func (m *AuthMiddleware) RequirePermissionWithClearance(resource Resource, action Action, requiredClearance string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := m.getCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Check authorization with security clearance
		authorized, err := m.authzService.AuthorizeWithSecurityClearance(
			c.Request.Context(),
			user,
			resource,
			action,
			requiredClearance,
		)
		if err != nil {
			if err == models.ErrInvalidSecurityClearance {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "insufficient security clearance",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "authorization check failed",
				})
			}
			c.Abort()
			return
		}

		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware that optionally authenticates (doesn't fail if no token)
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		// Validate token
		validation, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Don't fail, just continue without user context
			c.Next()
			return
		}

		// Store user and claims in context
		c.Set("user", validation.User)
		c.Set("claims", validation.Claims)
		c.Set("token", token)

		c.Next()
	}
}

// RateLimitByUser middleware that applies rate limiting per user
func (m *AuthMiddleware) RateLimitByUser(requestsPerMinute int) gin.HandlerFunc {
	// This is a simplified rate limiter - in production, use a proper rate limiting library
	userRequests := make(map[string][]time.Time)

	return func(c *gin.Context) {
		user := m.getCurrentUser(c)
		if user == nil {
			c.Next()
			return
		}

		userID := user.ID.Hex()
		now := time.Now()

		// Clean old requests
		if requests, exists := userRequests[userID]; exists {
			var validRequests []time.Time
			cutoff := now.Add(-time.Minute)
			for _, reqTime := range requests {
				if reqTime.After(cutoff) {
					validRequests = append(validRequests, reqTime)
				}
			}
			userRequests[userID] = validRequests
		}

		// Check rate limit
		if len(userRequests[userID]) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Add current request
		userRequests[userID] = append(userRequests[userID], now)

		c.Next()
	}
}

// AuditLog middleware that logs all requests for audit purposes
func (m *AuthMiddleware) AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after request is processed
		user := m.getCurrentUser(c)
		userID := "anonymous"
		if user != nil {
			userID = user.ID.Hex()
		}

		// In a real implementation, you would send this to an audit service
		// For now, we'll just log it (you could integrate with your logger)
		duration := time.Since(start)

		// Store audit information in context for potential use by handlers
		c.Set("audit_info", map[string]interface{}{
			"user_id":     userID,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
			"duration":    duration,
			"ip_address":  c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"timestamp":   start,
		})
	}
}

// CORS middleware for handling cross-origin requests
func (m *AuthMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Define allowed origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"https://localhost:3000",
		}

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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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

// SecurityHeaders middleware that adds security headers
func (m *AuthMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// Helper methods

// extractToken extracts the JWT token from the Authorization header
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer token format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// getCurrentUser gets the current user from the Gin context
func (m *AuthMiddleware) getCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// getCurrentClaims gets the current JWT claims from the Gin context
func (m *AuthMiddleware) getCurrentClaims(c *gin.Context) *JWTClaims {
	if claims, exists := c.Get("claims"); exists {
		if cl, ok := claims.(*JWTClaims); ok {
			return cl
		}
	}
	return nil
}

// Helper functions for use in handlers

// GetCurrentUser returns the current authenticated user from context
func GetCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

// GetCurrentClaims returns the current JWT claims from context
func GetCurrentClaims(c *gin.Context) *JWTClaims {
	if claims, exists := c.Get("claims"); exists {
		if cl, ok := claims.(*JWTClaims); ok {
			return cl
		}
	}
	return nil
}

// RequireCurrentUser returns the current user or aborts with 401
func RequireCurrentUser(c *gin.Context) *models.User {
	user := GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		c.Abort()
		return nil
	}
	return user
}
