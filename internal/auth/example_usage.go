package auth

import (
	"net/http"
	"time"

	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// ExampleUsage demonstrates how to use the authentication system
func ExampleUsage() {
	// This is an example of how to set up and use the authentication system
	// In a real application, you would get these from your configuration

	// Setup dependencies (these would come from your app initialization)
	var userCollection *mongo.Collection // From your MongoDB setup
	var redisClient *redis.Client        // From your Redis setup

	// JWT configuration
	jwtConfig := JWTConfig{
		AccessSecret:  "your-access-secret-key",
		RefreshSecret: "your-refresh-secret-key",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
		SessionTTL:    30 * time.Minute,
		BlacklistTTL:  24 * time.Hour,
		Issuer:        "ai-government-consultant",
	}

	// Create services
	authService := NewAuthService(userCollection, redisClient, jwtConfig)
	authzService := NewAuthorizationService()
	middleware := NewAuthMiddleware(authService, authzService)

	// Setup Gin router
	router := gin.New()

	// Add middleware
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.AuditLog())

	// Public routes (no authentication required)
	public := router.Group("/api/v1/public")
	{
		public.POST("/register", registerHandler(authService))
		public.POST("/login", loginHandler(authService))
		public.POST("/refresh", refreshTokenHandler(authService))
	}

	// Protected routes (authentication required)
	protected := router.Group("/api/v1")
	protected.Use(middleware.RequireAuth())
	{
		protected.POST("/logout", logoutHandler(authService))
		protected.GET("/profile", profileHandler())
		protected.PUT("/profile", updateProfileHandler())

		// MFA routes
		protected.POST("/mfa/setup", setupMFAHandler(authService))
		protected.POST("/mfa/enable", enableMFAHandler(authService))
		protected.POST("/mfa/disable", disableMFAHandler(authService))
	}

	// Admin routes (admin role required)
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.RequireAuth())
	admin.Use(middleware.RequireRole(models.UserRoleAdmin))
	{
		admin.GET("/users", listUsersHandler())
		admin.POST("/users", createUserHandler(authService))
		admin.DELETE("/users/:id", deleteUserHandler())
	}

	// Document routes with permission checks
	documents := router.Group("/api/v1/documents")
	documents.Use(middleware.RequireAuth())
	{
		documents.GET("/", middleware.RequirePermission(ResourceDocuments, ActionRead), listDocumentsHandler())
		documents.POST("/", middleware.RequirePermission(ResourceDocuments, ActionWrite), createDocumentHandler())
		documents.DELETE("/:id", middleware.RequirePermission(ResourceDocuments, ActionDelete), deleteDocumentHandler())

		// Classified documents require security clearance
		documents.GET("/classified/:id",
			middleware.RequirePermissionWithClearance(ResourceDocuments, ActionRead, "SECRET"),
			getClassifiedDocumentHandler())
	}

	// Start server
	router.Run(":8080")
}

// Example handlers

func registerHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email             string `json:"email" binding:"required,email"`
			Name              string `json:"name" binding:"required"`
			Department        string `json:"department" binding:"required"`
			Password          string `json:"password" binding:"required"`
			Role              string `json:"role" binding:"required"`
			SecurityClearance string `json:"security_clearance" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user := &models.User{
			Email:             req.Email,
			Name:              req.Name,
			Department:        req.Department,
			Role:              models.UserRole(req.Role),
			SecurityClearance: models.SecurityClearance(req.SecurityClearance),
		}

		err := authService.RegisterUser(c.Request.Context(), user, req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user_id": user.ID.Hex(),
		})
	}
}

func loginHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var credentials AuthCredentials
		if err := c.ShouldBindJSON(&credentials); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := authService.Authenticate(
			c.Request.Context(),
			credentials,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if result.MFARequired {
			c.JSON(http.StatusOK, gin.H{
				"mfa_required": true,
				"message":      "MFA code required",
			})
			return
		}

		c.Header("X-Session-ID", result.SessionID)
		c.JSON(http.StatusOK, gin.H{
			"access_token":  result.TokenPair.AccessToken,
			"refresh_token": result.TokenPair.RefreshToken,
			"expires_at":    result.TokenPair.ExpiresAt,
			"user":          result.User,
		})
	}
}

func refreshTokenHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tokenPair, err := authService.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt,
		})
	}
}

func logoutHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Get("token")
		sessionID := c.GetHeader("X-Session-ID")

		if tokenStr, ok := token.(string); ok {
			authService.Logout(c.Request.Context(), tokenStr, sessionID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

func profileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func updateProfileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := RequireCurrentUser(c)
		if user == nil {
			return // RequireCurrentUser already sent error response
		}

		var req struct {
			Name       string `json:"name"`
			Department string `json:"department"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update user profile logic here
		c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
	}
}

func setupMFAHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := RequireCurrentUser(c)
		if user == nil {
			return
		}

		mfaSetup, err := authService.SetupMFA(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, mfaSetup)
	}
}

func enableMFAHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := RequireCurrentUser(c)
		if user == nil {
			return
		}

		var req struct {
			Code string `json:"code" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := authService.EnableMFA(c.Request.Context(), user.ID, req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
	}
}

func disableMFAHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := RequireCurrentUser(c)
		if user == nil {
			return
		}

		var req struct {
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := authService.DisableMFA(c.Request.Context(), user.ID, req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "MFA disabled successfully"})
	}
}

// Placeholder handlers for demonstration
func listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	}
}

func createUserHandler(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "User created"})
	}
}

func deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
	}
}

func listDocumentsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"documents": []string{}})
	}
}

func createDocumentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "Document created"})
	}
}

func deleteDocumentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
	}
}

func getClassifiedDocumentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"document": "classified content"})
	}
}
