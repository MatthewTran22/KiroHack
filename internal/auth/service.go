package auth

import (
	"context"
	"fmt"
	"time"

	"ai-government-consultant/internal/models"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// PasswordHasher defines the interface for password hashing services
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) error
	ValidatePassword(password string) error
}

// AuthService provides authentication and authorization functionality
type AuthService struct {
	userCollection  *mongo.Collection
	jwtService      *JWTService
	passwordService PasswordHasher
	mfaService      *MFAService
	sessionService  *SessionService
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userCollection *mongo.Collection,
	redisClient *redis.Client,
	jwtConfig JWTConfig,
) *AuthService {
	jwtService := NewJWTService(
		jwtConfig.AccessSecret,
		jwtConfig.RefreshSecret,
		jwtConfig.AccessTTL,
		jwtConfig.RefreshTTL,
		jwtConfig.Issuer,
	)

	passwordService := NewArgon2PasswordService()
	mfaService := NewMFAService(jwtConfig.Issuer)
	sessionService := NewSessionService(redisClient, jwtConfig.SessionTTL, jwtConfig.BlacklistTTL)

	return &AuthService{
		userCollection:  userCollection,
		jwtService:      jwtService,
		passwordService: passwordService,
		mfaService:      mfaService,
		sessionService:  sessionService,
	}
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	SessionTTL    time.Duration
	BlacklistTTL  time.Duration
	Issuer        string
}

// AuthCredentials represents login credentials
type AuthCredentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// AuthResult represents the result of authentication
type AuthResult struct {
	User        *models.User `json:"user"`
	TokenPair   *TokenPair   `json:"tokens"`
	SessionID   string       `json:"session_id"`
	MFARequired bool         `json:"mfa_required"`
	MFASetupURL string       `json:"mfa_setup_url,omitempty"`
}

// RegisterUser creates a new user account
func (a *AuthService) RegisterUser(ctx context.Context, user *models.User, password string) error {
	// Validate user data
	if err := user.Validate(); err != nil {
		return err
	}

	// Check if user already exists
	existingUser, err := a.getUserByEmail(ctx, user.Email)
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return models.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := a.passwordService.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Set user fields
	user.ID = primitive.NewObjectID()
	user.PasswordHash = hashedPassword
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	// Generate MFA secret
	mfaSecret, err := a.mfaService.GenerateSecret()
	if err != nil {
		return fmt.Errorf("failed to generate MFA secret: %w", err)
	}
	user.MFASecret = mfaSecret

	// Insert user into database
	_, err = a.userCollection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Authenticate authenticates a user with email and password
func (a *AuthService) Authenticate(ctx context.Context, credentials AuthCredentials, ipAddress, userAgent string) (*AuthResult, error) {
	// Get user by email
	user, err := a.getUserByEmail(ctx, credentials.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, models.ErrUserInactive
	}

	// Verify password
	err = a.passwordService.VerifyPassword(credentials.Password, user.PasswordHash)
	if err != nil {
		return nil, models.ErrInvalidCredentials
	}

	// Check MFA if enabled
	if user.MFAEnabled {
		if credentials.MFACode == "" {
			return &AuthResult{
				MFARequired: true,
			}, nil
		}

		valid, err := a.mfaService.ValidateCode(user.MFASecret, credentials.MFACode)
		if err != nil || !valid {
			return nil, models.ErrInvalidMFACode
		}
	}

	// Generate session ID
	sessionID := primitive.NewObjectID().Hex()

	// Create session data
	sessionData := &SessionData{
		UserID:            user.ID,
		Email:             user.Email,
		Role:              string(user.Role),
		SecurityClearance: string(user.SecurityClearance),
		Permissions:       a.getUserPermissions(user),
		LoginTime:         time.Now(),
		LastActivity:      time.Now(),
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		MFAVerified:       user.MFAEnabled,
	}

	// Create session
	err = a.sessionService.CreateSession(ctx, sessionID, sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT tokens
	tokenPair, err := a.jwtService.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		string(user.SecurityClearance),
		sessionData.Permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login
	user.UpdateLastLogin()
	a.updateUser(ctx, user)

	return &AuthResult{
		User:      user,
		TokenPair: tokenPair,
		SessionID: sessionID,
	}, nil
}

// ValidateToken validates a JWT access token
func (a *AuthService) ValidateToken(ctx context.Context, tokenString string) (*TokenValidation, error) {
	// Parse and validate token
	claims, err := a.jwtService.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, models.ErrInvalidToken
	}

	// Check if token is blacklisted
	blacklisted, err := a.sessionService.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if blacklisted {
		return nil, models.ErrTokenBlacklisted
	}

	// Get user to ensure they still exist and are active
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, models.ErrInvalidToken
	}

	user, err := a.getUserByID(ctx, userID)
	if err != nil {
		return nil, models.ErrInvalidToken
	}

	if !user.IsActive {
		return nil, models.ErrUserInactive
	}

	return &TokenValidation{
		Valid:  true,
		Claims: claims,
		User:   user,
	}, nil
}

// RefreshToken generates new tokens using a refresh token
func (a *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := a.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, models.ErrInvalidRefreshToken
	}

	// Check if token is blacklisted
	blacklisted, err := a.sessionService.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if blacklisted {
		return nil, models.ErrTokenBlacklisted
	}

	// Get user to ensure they still exist and are active
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, models.ErrInvalidRefreshToken
	}

	user, err := a.getUserByID(ctx, userID)
	if err != nil {
		return nil, models.ErrInvalidRefreshToken
	}

	if !user.IsActive {
		return nil, models.ErrUserInactive
	}

	// Generate new token pair
	tokenPair, err := a.jwtService.GenerateTokenPair(
		user.ID,
		user.Email,
		string(user.Role),
		string(user.SecurityClearance),
		a.getUserPermissions(user),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	// Blacklist the old refresh token
	err = a.sessionService.BlacklistToken(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to blacklist old token: %w", err)
	}

	return tokenPair, nil
}

// Logout logs out a user by blacklisting their tokens and removing session
func (a *AuthService) Logout(ctx context.Context, tokenString, sessionID string) error {
	// Parse token to get token ID
	claims, err := a.jwtService.ValidateAccessToken(tokenString)
	if err != nil {
		// Even if token is invalid, try to clean up session
		if sessionID != "" {
			a.sessionService.DeleteSession(ctx, sessionID)
		}
		return nil
	}

	// Blacklist the token
	err = a.sessionService.BlacklistToken(ctx, claims.ID)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// Remove session
	if sessionID != "" {
		err = a.sessionService.DeleteSession(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}
	}

	return nil
}

// SetupMFA sets up multi-factor authentication for a user
func (a *AuthService) SetupMFA(ctx context.Context, userID primitive.ObjectID) (*MFASetup, error) {
	user, err := a.getUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Generate new MFA secret if not already set
	if user.MFASecret == "" {
		secret, err := a.mfaService.GenerateSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate MFA secret: %w", err)
		}
		user.MFASecret = secret
		user.UpdatedAt = time.Now()

		// Update user in database
		err = a.updateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Generate QR code URL
	qrURL := a.mfaService.GenerateQRCodeURL(user.MFASecret, user.Email)

	// Generate backup codes
	backupCodes, err := a.mfaService.GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	return &MFASetup{
		Secret:      user.MFASecret,
		QRCodeURL:   qrURL,
		BackupCodes: backupCodes,
	}, nil
}

// EnableMFA enables multi-factor authentication for a user after verification
func (a *AuthService) EnableMFA(ctx context.Context, userID primitive.ObjectID, verificationCode string) error {
	user, err := a.getUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.MFASecret == "" {
		return fmt.Errorf("MFA not set up for user")
	}

	// Verify the code
	valid, err := a.mfaService.ValidateCode(user.MFASecret, verificationCode)
	if err != nil || !valid {
		return models.ErrInvalidMFACode
	}

	// Enable MFA
	user.MFAEnabled = true
	user.UpdatedAt = time.Now()

	return a.updateUser(ctx, user)
}

// DisableMFA disables multi-factor authentication for a user
func (a *AuthService) DisableMFA(ctx context.Context, userID primitive.ObjectID, password string) error {
	user, err := a.getUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify password
	err = a.passwordService.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return models.ErrInvalidCredentials
	}

	// Disable MFA
	user.MFAEnabled = false
	user.MFASecret = ""
	user.UpdatedAt = time.Now()

	return a.updateUser(ctx, user)
}

// Helper methods

func (a *AuthService) getUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := a.userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	return &user, err
}

func (a *AuthService) getUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := a.userCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	return &user, err
}

func (a *AuthService) updateUser(ctx context.Context, user *models.User) error {
	_, err := a.userCollection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

func (a *AuthService) getUserPermissions(user *models.User) []string {
	var permissions []string
	for _, perm := range user.Permissions {
		for _, action := range perm.Actions {
			permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, action))
		}
	}
	return permissions
}

// TokenValidation represents the result of token validation
type TokenValidation struct {
	Valid  bool         `json:"valid"`
	Claims *JWTClaims   `json:"claims"`
	User   *models.User `json:"user"`
}

// MFASetup represents MFA setup information
type MFASetup struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}
