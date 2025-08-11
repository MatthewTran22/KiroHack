package auth

import (
	"strings"
	"testing"
)

func TestPasswordService_HashPassword(t *testing.T) {
	passwordService := NewPasswordService()
	password := "TestPassword123!"

	hash, err := passwordService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash is empty")
	}

	if hash == password {
		t.Error("Hash should not be the same as the original password")
	}
}

func TestPasswordService_VerifyPassword(t *testing.T) {
	passwordService := NewPasswordService()
	password := "TestPassword123!"

	hash, err := passwordService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	err = passwordService.VerifyPassword(password, hash)
	if err != nil {
		t.Errorf("Failed to verify correct password: %v", err)
	}

	// Test incorrect password
	err = passwordService.VerifyPassword("WrongPassword123!", hash)
	if err == nil {
		t.Error("Expected error for incorrect password")
	}
}

func TestPasswordService_ValidatePassword(t *testing.T) {
	passwordService := NewPasswordService()

	tests := []struct {
		password string
		valid    bool
		name     string
	}{
		{"TestPassword123!", true, "valid password"},
		{"short", false, "too short"},
		{"nouppercase123!", false, "no uppercase"},
		{"NOLOWERCASE123!", false, "no lowercase"},
		{"NoDigits!", false, "no digits"},
		{"NoSpecialChars123", false, "no special characters"},
		{"TestPassword123!" + strings.Repeat("a", 120), false, "too long"},
		{"ValidPass1!", true, "minimum valid password"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := passwordService.ValidatePassword(test.password)
			if test.valid && err != nil {
				t.Errorf("Expected password '%s' to be valid, but got error: %v", test.password, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected password '%s' to be invalid, but validation passed", test.password)
			}
		})
	}
}

func TestPasswordService_GenerateSecurePassword(t *testing.T) {
	passwordService := NewPasswordService()

	// Test default length
	password, err := passwordService.GenerateSecurePassword(12)
	if err != nil {
		t.Fatalf("Failed to generate secure password: %v", err)
	}

	if len(password) != 12 {
		t.Errorf("Expected password length 12, got %d", len(password))
	}

	// Validate the generated password meets requirements
	err = passwordService.ValidatePassword(password)
	if err != nil {
		t.Errorf("Generated password doesn't meet validation requirements: %v", err)
	}

	// Test minimum length enforcement
	shortPassword, err := passwordService.GenerateSecurePassword(4)
	if err != nil {
		t.Fatalf("Failed to generate secure password: %v", err)
	}

	if len(shortPassword) < MinPasswordLength {
		t.Errorf("Expected password length to be at least %d, got %d", MinPasswordLength, len(shortPassword))
	}

	// Test maximum length enforcement
	longPassword, err := passwordService.GenerateSecurePassword(200)
	if err != nil {
		t.Fatalf("Failed to generate secure password: %v", err)
	}

	if len(longPassword) > MaxPasswordLength {
		t.Errorf("Expected password length to be at most %d, got %d", MaxPasswordLength, len(longPassword))
	}
}

func TestPasswordService_GenerateSalt(t *testing.T) {
	passwordService := NewPasswordService()

	salt1, err := passwordService.GenerateSalt(32)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	salt2, err := passwordService.GenerateSalt(32)
	if err != nil {
		t.Fatalf("Failed to generate second salt: %v", err)
	}

	if salt1 == salt2 {
		t.Error("Generated salts should be different")
	}

	if salt1 == "" {
		t.Error("Salt should not be empty")
	}
}

func TestPasswordService_SecureCompare(t *testing.T) {
	passwordService := NewPasswordService()

	// Test equal strings
	if !passwordService.SecureCompare("test", "test") {
		t.Error("Expected equal strings to return true")
	}

	// Test different strings
	if passwordService.SecureCompare("test", "different") {
		t.Error("Expected different strings to return false")
	}

	// Test empty strings
	if !passwordService.SecureCompare("", "") {
		t.Error("Expected empty strings to return true")
	}

	// Test one empty string
	if passwordService.SecureCompare("test", "") {
		t.Error("Expected comparison with empty string to return false")
	}
}

func TestPasswordService_CustomCost(t *testing.T) {
	customCost := 10
	passwordService := NewPasswordServiceWithCost(customCost)
	password := "TestPassword123!"

	hash, err := passwordService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password with custom cost: %v", err)
	}

	// Verify the password still works
	err = passwordService.VerifyPassword(password, hash)
	if err != nil {
		t.Errorf("Failed to verify password with custom cost: %v", err)
	}
}

func BenchmarkPasswordService_HashPassword(b *testing.B) {
	passwordService := NewPasswordService()
	password := "TestPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := passwordService.HashPassword(password)
		if err != nil {
			b.Fatalf("Failed to hash password: %v", err)
		}
	}
}

func BenchmarkPasswordService_VerifyPassword(b *testing.B) {
	passwordService := NewPasswordService()
	password := "TestPassword123!"

	hash, err := passwordService.HashPassword(password)
	if err != nil {
		b.Fatalf("Failed to hash password: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := passwordService.VerifyPassword(password, hash)
		if err != nil {
			b.Fatalf("Failed to verify password: %v", err)
		}
	}
}
