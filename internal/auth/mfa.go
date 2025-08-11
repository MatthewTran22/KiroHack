package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// MFASecretLength is the length of the MFA secret in bytes
	MFASecretLength = 20
	// MFACodeLength is the length of the MFA code
	MFACodeLength = 6
	// MFATimeStep is the time step for TOTP (30 seconds)
	MFATimeStep = 30
	// MFASkew is the number of time steps to allow for clock skew
	MFASkew = 1
)

// MFAService handles multi-factor authentication operations
type MFAService struct {
	issuer string
}

// NewMFAService creates a new MFA service
func NewMFAService(issuer string) *MFAService {
	return &MFAService{
		issuer: issuer,
	}
}

// GenerateSecret generates a new MFA secret for a user
func (m *MFAService) GenerateSecret() (string, error) {
	secret := make([]byte, MFASecretLength)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate MFA secret: %w", err)
	}

	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateQRCodeURL generates a QR code URL for setting up MFA
func (m *MFAService) GenerateQRCodeURL(secret, userEmail string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s",
		m.issuer,
		userEmail,
		secret,
		m.issuer,
	)
}

// GenerateCode generates a TOTP code for the given secret at the current time
func (m *MFAService) GenerateCode(secret string) (string, error) {
	return m.GenerateCodeAtTime(secret, time.Now())
}

// GenerateCodeAtTime generates a TOTP code for the given secret at a specific time
func (m *MFAService) GenerateCodeAtTime(secret string, t time.Time) (string, error) {
	secretBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("invalid secret format: %w", err)
	}

	timeCounter := uint64(t.Unix()) / MFATimeStep
	return m.generateHOTP(secretBytes, timeCounter)
}

// ValidateCode validates a TOTP code against the given secret
func (m *MFAService) ValidateCode(secret, code string) (bool, error) {
	return m.ValidateCodeAtTime(secret, code, time.Now())
}

// ValidateCodeAtTime validates a TOTP code against the given secret at a specific time
func (m *MFAService) ValidateCodeAtTime(secret, code string, t time.Time) (bool, error) {
	if len(code) != MFACodeLength {
		return false, fmt.Errorf("invalid code length: expected %d, got %d", MFACodeLength, len(code))
	}

	secretBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false, fmt.Errorf("invalid secret format: %w", err)
	}

	timeCounter := uint64(t.Unix()) / MFATimeStep

	// Check current time and allow for clock skew
	for i := -MFASkew; i <= MFASkew; i++ {
		testCounter := timeCounter + uint64(i)
		expectedCode, err := m.generateHOTP(secretBytes, testCounter)
		if err != nil {
			return false, err
		}

		if code == expectedCode {
			return true, nil
		}
	}

	return false, nil
}

// generateHOTP generates an HMAC-based One-Time Password
func (m *MFAService) generateHOTP(secret []byte, counter uint64) (string, error) {
	// Convert counter to byte array
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// Generate HMAC-SHA1
	h := hmac.New(sha1.New, secret)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0F
	truncatedHash := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF

	// Generate the final code
	code := truncatedHash % uint32(math.Pow10(MFACodeLength))

	return fmt.Sprintf("%0*d", MFACodeLength, code), nil
}

// GenerateBackupCodes generates backup codes for MFA recovery
func (m *MFAService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10 // Default number of backup codes
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := m.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code %d: %w", i+1, err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateBackupCode generates a single backup code
func (m *MFAService) generateBackupCode() (string, error) {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Convert to a readable format (e.g., "1234-5678")
	code := fmt.Sprintf("%04x-%04x",
		binary.BigEndian.Uint16(bytes[0:2]),
		binary.BigEndian.Uint16(bytes[2:4]))

	return strings.ToUpper(code), nil
}

// ValidateBackupCode validates a backup code (this would typically check against stored hashed codes)
func (m *MFAService) ValidateBackupCode(providedCode, storedHashedCode string) (bool, error) {
	// In a real implementation, you would hash the provided code and compare with stored hash
	// For now, this is a placeholder that shows the interface

	// Normalize the provided code
	normalizedCode := strings.ToUpper(strings.ReplaceAll(providedCode, "-", ""))

	// This is a simplified validation - in production, use proper hashing
	return normalizedCode == storedHashedCode, nil
}
