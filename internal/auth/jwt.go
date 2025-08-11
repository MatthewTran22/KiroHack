package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID            string   `json:"user_id"`
	Email             string   `json:"email"`
	Role              string   `json:"role"`
	SecurityClearance string   `json:"security_clearance"`
	Permissions       []string `json:"permissions"`
	TokenType         string   `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

// NewJWTService creates a new JWT service
func NewJWTService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer string) *JWTService {
	return &JWTService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTService) GenerateTokenPair(userID primitive.ObjectID, email, role, securityClearance string, permissions []string) (*TokenPair, error) {
	userIDStr := userID.Hex()

	// Generate access token
	accessToken, err := j.generateToken(userIDStr, email, role, securityClearance, permissions, "access", j.accessSecret, j.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(userIDStr, email, role, securityClearance, permissions, "refresh", j.refreshSecret, j.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(j.accessTTL),
	}, nil
}

// generateToken creates a JWT token with the specified parameters
func (j *JWTService) generateToken(userID, email, role, securityClearance string, permissions []string, tokenType string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := JWTClaims{
		UserID:            userID,
		Email:             email,
		Role:              role,
		SecurityClearance: securityClearance,
		Permissions:       permissions,
		TokenType:         tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        j.generateJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateAccessToken validates an access token and returns the claims
func (j *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	return j.validateToken(tokenString, j.accessSecret, "access")
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (j *JWTService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	return j.validateToken(tokenString, j.refreshSecret, "refresh")
}

// validateToken validates a token with the specified secret and type
func (j *JWTService) validateToken(tokenString string, secret []byte, expectedType string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.TokenType)
	}

	return claims, nil
}

// RefreshToken generates a new access token using a valid refresh token
func (j *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return j.GenerateTokenPair(userID, claims.Email, claims.Role, claims.SecurityClearance, claims.Permissions)
}

// generateJTI generates a unique JWT ID
func (j *JWTService) generateJTI() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
