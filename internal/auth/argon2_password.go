package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2 parameters
	Argon2Time      = 1      // Number of passes
	Argon2Memory    = 64*1024 // Memory in KiB (64 MB)
	Argon2Threads   = 4      // Number of threads
	Argon2KeyLength = 32     // Length of the derived key

	// Salt length for Argon2
	Argon2SaltLength = 16
)

// Argon2PasswordService handles password hashing and validation using Argon2
type Argon2PasswordService struct {
	time      uint32
	memory    uint32
	threads   uint8
	keyLength uint32
	saltLength int
}

// NewArgon2PasswordService creates a new Argon2 password service with default parameters
func NewArgon2PasswordService() *Argon2PasswordService {
	return &Argon2PasswordService{
		time:       Argon2Time,
		memory:     Argon2Memory,
		threads:    Argon2Threads,
		keyLength:  Argon2KeyLength,
		saltLength: Argon2SaltLength,
	}
}

// NewArgon2PasswordServiceWithParams creates a new Argon2 password service with custom parameters
func NewArgon2PasswordServiceWithParams(time, memory uint32, threads uint8, keyLength uint32) *Argon2PasswordService {
	return &Argon2PasswordService{
		time:       time,
		memory:     memory,
		threads:    threads,
		keyLength:  keyLength,
		saltLength: Argon2SaltLength,
	}
}

// HashPassword hashes a password using Argon2id
func (a *Argon2PasswordService) HashPassword(password string) (string, error) {
	// Note: We don't validate password here - validation should be done at API level
	// This allows system-generated passwords (like default admin) to bypass validation
	
	// Generate a cryptographically secure random salt
	salt, err := a.generateSalt()
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the hash using Argon2id
	hash := argon2.IDKey([]byte(password), salt, a.time, a.memory, a.threads, a.keyLength)

	// Encode salt and hash as base64 for storage
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=memory,t=time,p=threads$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", 
		a.memory, a.time, a.threads, saltB64, hashB64)

	return encodedHash, nil
}

// VerifyPassword verifies a password against its Argon2 hash
func (a *Argon2PasswordService) VerifyPassword(password, encodedHash string) error {
	// Parse the encoded hash
	params, salt, hash, err := a.parseEncodedHash(encodedHash)
	if err != nil {
		return fmt.Errorf("invalid hash format: %w", err)
	}

	// Generate hash with the same parameters
	newHash := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(hash)))

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, newHash) == 1 {
		return nil
	}

	return fmt.Errorf("password verification failed")
}

// parseEncodedHash parses an encoded Argon2 hash string
func (a *Argon2PasswordService) parseEncodedHash(encodedHash string) (*argon2Params, []byte, []byte, error) {
	// Expected format: $argon2id$v=19$m=memory,t=time,p=threads$salt$hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported variant: %s", parts[1])
	}

	if parts[2] != "v=19" {
		return nil, nil, nil, fmt.Errorf("unsupported version: %s", parts[2])
	}

	// Parse parameters
	params := &argon2Params{}
	paramParts := strings.Split(parts[3], ",")
	for _, param := range paramParts {
		keyVal := strings.Split(param, "=")
		if len(keyVal) != 2 {
			continue
		}

		var value uint32
		if _, err := fmt.Sscanf(keyVal[1], "%d", &value); err != nil {
			return nil, nil, nil, fmt.Errorf("invalid parameter value: %s", keyVal[1])
		}

		switch keyVal[0] {
		case "m":
			params.memory = value
		case "t":
			params.time = value
		case "p":
			params.threads = uint8(value)
		}
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	return params, salt, hash, nil
}

// generateSalt generates a cryptographically secure random salt
func (a *Argon2PasswordService) generateSalt() ([]byte, error) {
	salt := make([]byte, a.saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// ValidatePassword validates password strength requirements
func (a *Argon2PasswordService) ValidatePassword(password string) error {
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
func (a *Argon2PasswordService) GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	if length > MaxPasswordLength {
		length = MaxPasswordLength
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"

	password := make([]byte, length)
	for i := range password {
		randomIndex, err := a.secureRandomInt(len(charset))
		if err != nil {
			return "", fmt.Errorf("failed to generate secure password: %w", err)
		}
		password[i] = charset[randomIndex]
	}

	// Ensure the generated password meets requirements
	if err := a.ValidatePassword(string(password)); err != nil {
		// If validation fails, try again (recursive call with small probability of infinite loop)
		return a.GenerateSecurePassword(length)
	}

	return string(password), nil
}

// secureRandomInt generates a cryptographically secure random integer in range [0, max)
func (a *Argon2PasswordService) secureRandomInt(max int) (int, error) {
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
func (a *Argon2PasswordService) GenerateSalt(length int) (string, error) {
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
func (a *Argon2PasswordService) SecureCompare(a1, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a1), []byte(b)) == 1
}

// argon2Params holds the parameters for Argon2
type argon2Params struct {
	time    uint32
	memory  uint32
	threads uint8
}
