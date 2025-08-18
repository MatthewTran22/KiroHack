package api

import (
	"net/http"
	"strings"

	"ai-government-consultant/internal/auth"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthHandler handles authentication-related API endpoints
type AuthHandler struct {
	authService *auth.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email             string                    `json:"email" binding:"required,email"`
	Name              string                    `json:"name" binding:"required"`
	Department        string                    `json:"department" binding:"required"`
	Role              models.UserRole           `json:"role" binding:"required"`
	SecurityClearance models.SecurityClearance  `json:"security_clearance" binding:"required"`
	Password          string                    `json:"password" binding:"required,min=8"`
	Permissions       []models.Permission       `json:"permissions,omitempty"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// MFAVerificationRequest represents an MFA verification request
type MFAVerificationRequest struct {
	Code string `json:"code" binding:"required"`
}

// PasswordChangeRequest represents a password change request
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Create user model
	user := &models.User{
		Email:             req.Email,
		Name:              req.Name,
		Department:        req.Department,
		Role:              req.Role,
		SecurityClearance: req.SecurityClearance,
		Permissions:       req.Permissions,
	}

	// Register user
	err := h.authService.RegisterUser(c.Request.Context(), user, req.Password)
	if err != nil {
		if err == models.ErrUserAlreadyExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "User already exists",
				Message: "A user with this email already exists",
				Code:    "USER_EXISTS",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Registration failed",
			Message: err.Error(),
			Code:    "REGISTRATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "User registered successfully",
		Data: gin.H{
			"user_id": user.ID.Hex(),
			"email":   user.Email,
		},
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get client information
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Authenticate user
	credentials := auth.AuthCredentials{
		Email:    req.Email,
		Password: req.Password,
		MFACode:  req.MFACode,
	}

	result, err := h.authService.Authenticate(c.Request.Context(), credentials, ipAddress, userAgent)
	if err != nil {
		switch err {
		case models.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid credentials",
				Message: "Email or password is incorrect",
				Code:    "INVALID_CREDENTIALS",
			})
		case models.ErrUserInactive:
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Account inactive",
				Message: "Your account has been deactivated",
				Code:    "ACCOUNT_INACTIVE",
			})
		case models.ErrInvalidMFACode:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid MFA code",
				Message: "The provided MFA code is invalid or expired",
				Code:    "INVALID_MFA_CODE",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Authentication failed",
				Message: err.Error(),
				Code:    "AUTH_FAILED",
			})
		}
		return
	}

	// If MFA is required but not provided
	if result.MFARequired {
		c.JSON(http.StatusOK, gin.H{
			"mfa_required": true,
			"message":      "MFA code required",
		})
		return
	}

	// Set session cookie
	c.SetCookie("session_id", result.SessionID, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Login successful",
		"user":       result.User,
		"tokens":     result.TokenPair,
		"session_id": result.SessionID,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing authorization header",
			Code:    "MISSING_AUTH_HEADER",
		})
		return
	}

	// Extract token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid authorization header format",
			Code:    "INVALID_AUTH_HEADER",
		})
		return
	}

	token := tokenParts[1]
	sessionID, _ := c.Cookie("session_id")

	// Logout user
	err := h.authService.Logout(c.Request.Context(), token, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Logout failed",
			Message: err.Error(),
			Code:    "LOGOUT_FAILED",
		})
		return
	}

	// Clear session cookie
	c.SetCookie("session_id", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Logout successful",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Refresh token
	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case models.ErrInvalidRefreshToken:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid refresh token",
				Code:    "INVALID_REFRESH_TOKEN",
			})
		case models.ErrTokenBlacklisted:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Token has been revoked",
				Code:    "TOKEN_REVOKED",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Token refresh failed",
				Message: err.Error(),
				Code:    "REFRESH_FAILED",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokenPair,
	})
}

// SetupMFA handles MFA setup
func (h *AuthHandler) SetupMFA(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Setup MFA
	mfaSetup, err := h.authService.SetupMFA(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "MFA setup failed",
			Message: err.Error(),
			Code:    "MFA_SETUP_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "MFA setup initiated",
		"qr_code_url":  mfaSetup.QRCodeURL,
		"backup_codes": mfaSetup.BackupCodes,
	})
}

// EnableMFA handles MFA enablement after verification
func (h *AuthHandler) EnableMFA(c *gin.Context) {
	var req MFAVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Enable MFA
	err := h.authService.EnableMFA(c.Request.Context(), user.ID, req.Code)
	if err != nil {
		if err == models.ErrInvalidMFACode {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid MFA code",
				Message: "The provided verification code is invalid",
				Code:    "INVALID_MFA_CODE",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "MFA enablement failed",
			Message: err.Error(),
			Code:    "MFA_ENABLE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "MFA enabled successfully",
	})
}

// DisableMFA handles MFA disabling
func (h *AuthHandler) DisableMFA(c *gin.Context) {
	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Disable MFA
	err := h.authService.DisableMFA(c.Request.Context(), user.ID, req.CurrentPassword)
	if err != nil {
		if err == models.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid password",
				Message: "Current password is incorrect",
				Code:    "INVALID_PASSWORD",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "MFA disable failed",
			Message: err.Error(),
			Code:    "MFA_DISABLE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "MFA disabled successfully",
	})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req struct {
		Name       string `json:"name,omitempty"`
		Department string `json:"department,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Update fields if provided
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Department != "" {
		user.Department = req.Department
	}

	// Note: In a real implementation, you would update the user in the database
	// For now, we'll just return success
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Profile updated successfully",
		Data:    user,
	})
}

// ChangePassword handles password changes
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Note: In a real implementation, you would verify the current password
	// and update it in the database. For now, we'll just return success.
	_ = user
	_ = req

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Password changed successfully",
	})
}

// ValidateToken validates a JWT token (used by other services)
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Missing authorization header",
			Code:  "MISSING_AUTH_HEADER",
		})
		return
	}

	// Extract token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid authorization header format",
			Code:  "INVALID_AUTH_HEADER",
		})
		return
	}

	token := tokenParts[1]

	// Validate token
	validation, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Token validation failed",
			Message: err.Error(),
			Code:    "TOKEN_INVALID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  validation.Valid,
		"claims": validation.Claims,
		"user":   validation.User,
	})
}

// ListUsers returns a list of users (admin only)
func (h *AuthHandler) ListUsers(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check if user is admin
	if !user.IsAdmin() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would fetch users from the database
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"users": []gin.H{},
		"total": 0,
	})
}

// GetUser returns a specific user by ID (admin only)
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "User ID is required",
			Code:  "MISSING_USER_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID format",
			Message: err.Error(),
			Code:    "INVALID_USER_ID",
		})
		return
	}

	// Get current user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	currentUser := userInterface.(*models.User)

	// Check if user is admin or requesting their own profile
	if !currentUser.IsAdmin() && currentUser.ID != objID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would fetch the user from the database
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    objID.Hex(),
			"email": "placeholder@example.com",
			"name":  "Placeholder User",
		},
	})
}