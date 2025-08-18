package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default cost for bcrypt hashing
	DefaultBcryptCost = 12
	// MinPasswordLength is the minimum password length
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum password length
	MaxPasswordLength = 128
)

// PasswordService handles password hashing and validation
type PasswordService struct {
	cost int
}

// NewPasswordService creates a new password service
func NewPasswordService() *PasswordService {
	return &PasswordService{
		cost: DefaultBcryptCost,
	}
}

// NewPasswordServiceWithCost creates a new password service with custom bcrypt cost
func NewPasswordServiceWithCost(cost int) *PasswordService {
	return &PasswordService{
		cost: cost,
	}
}

// HashPassword hashes a password using bcrypt
// Note: Password validation should be done at the API level, not here
// This allows system-generated passwords (like default admin) to bypass validation
func (p *PasswordService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (p *PasswordService) VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// ValidatePassword validates password strength requirements
func (p *PasswordService) ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", MinPasswordLength)
	}

	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must be no more than %d characters long", MaxPasswordLength)
	}

	// Check for at least one uppercase letter
	hasUpper := false
	// Check for at least one lowercase letter
	hasLower := false
	// Check for at least one digit
	hasDigit := false
	// Check for at least one special character
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// GenerateSecurePassword generates a cryptographically secure random password
func (p *PasswordService) GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	if length > MaxPasswordLength {
		length = MaxPasswordLength
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"

	password := make([]byte, length)
	for i := range password {
		randomIndex, err := p.secureRandomInt(len(charset))
		if err != nil {
			return "", fmt.Errorf("failed to generate secure password: %w", err)
		}
		password[i] = charset[randomIndex]
	}

	// Ensure the generated password meets requirements
	if err := p.ValidatePassword(string(password)); err != nil {
		// If validation fails, try again (recursive call with small probability of infinite loop)
		return p.GenerateSecurePassword(length)
	}

	return string(password), nil
}

// secureRandomInt generates a cryptographically secure random integer in range [0, max)
func (p *PasswordService) secureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	// Calculate the number of bytes needed
	bytes := make([]byte, 4) // 4 bytes for int32
	_, err := rand.Read(bytes)
	if err != nil {
		return 0, err
	}

	// Convert bytes to int and mod by max
	num := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if num < 0 {
		num = -num
	}

	return num % max, nil
}

// GenerateSalt generates a cryptographically secure random salt
func (p *PasswordService) GenerateSalt(length int) (string, error) {
	if length <= 0 {
		length = 32 // Default salt length
	}

	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	return base64.URLEncoding.EncodeToString(salt), nil
}

// SecureCompare performs a constant-time comparison of two strings
func (p *PasswordService) SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
