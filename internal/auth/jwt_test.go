package auth

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestJWTService_GenerateTokenPair(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	tokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	if tokenPair.ExpiresAt.Before(time.Now()) {
		t.Error("Token expiration is in the past")
	}
}

func TestJWTService_ValidateAccessToken(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	tokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Validate access token
	claims, err := jwtService.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.UserID != userID.Hex() {
		t.Errorf("Expected user ID %s, got %s", userID.Hex(), claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected email %s, got %s", email, claims.Email)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}

	if claims.SecurityClearance != securityClearance {
		t.Errorf("Expected security clearance %s, got %s", securityClearance, claims.SecurityClearance)
	}

	if claims.TokenType != "access" {
		t.Errorf("Expected token type 'access', got %s", claims.TokenType)
	}
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	tokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Validate refresh token
	claims, err := jwtService.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("Expected token type 'refresh', got %s", claims.TokenType)
	}
}

func TestJWTService_RefreshToken(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	// Generate initial token pair
	initialTokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate initial token pair: %v", err)
	}

	// Refresh tokens
	newTokenPair, err := jwtService.RefreshToken(initialTokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newTokenPair.AccessToken == initialTokenPair.AccessToken {
		t.Error("New access token should be different from the original")
	}

	if newTokenPair.RefreshToken == initialTokenPair.RefreshToken {
		t.Error("New refresh token should be different from the original")
	}

	// Validate new access token
	claims, err := jwtService.ValidateAccessToken(newTokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate new access token: %v", err)
	}

	if claims.UserID != userID.Hex() {
		t.Errorf("Expected user ID %s, got %s", userID.Hex(), claims.UserID)
	}
}

func TestJWTService_InvalidToken(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	// Test with invalid token
	_, err := jwtService.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// Test with empty token
	_, err = jwtService.ValidateAccessToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	// Create service with very short TTL
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		1*time.Millisecond, // Very short TTL
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	tokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = jwtService.ValidateAccessToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestJWTService_WrongTokenType(t *testing.T) {
	jwtService := NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		15*time.Minute,
		24*time.Hour,
		"test-issuer",
	)

	userID := primitive.NewObjectID()
	email := "test@example.com"
	role := "admin"
	securityClearance := "secret"
	permissions := []string{"documents:read", "documents:write"}

	tokenPair, err := jwtService.GenerateTokenPair(userID, email, role, securityClearance, permissions)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Try to validate refresh token as access token
	_, err = jwtService.ValidateAccessToken(tokenPair.RefreshToken)
	if err == nil {
		t.Error("Expected error when validating refresh token as access token")
	}

	// Try to validate access token as refresh token
	_, err = jwtService.ValidateRefreshToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected error when validating access token as refresh token")
	}
}
