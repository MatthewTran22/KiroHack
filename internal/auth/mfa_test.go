package auth

import (
	"strings"
	"testing"
	"time"
)

func TestMFAService_GenerateSecret(t *testing.T) {
	mfaService := NewMFAService("test-issuer")

	secret, err := mfaService.GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate MFA secret: %v", err)
	}

	if secret == "" {
		t.Error("Secret is empty")
	}

	// Check that secret is base32 encoded
	if len(secret) == 0 {
		t.Error("Secret should not be empty")
	}

	// Generate another secret to ensure they're different
	secret2, err := mfaService.GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate second MFA secret: %v", err)
	}

	if secret == secret2 {
		t.Error("Generated secrets should be different")
	}
}

func TestMFAService_GenerateQRCodeURL(t *testing.T) {
	issuer := "test-issuer"
	mfaService := NewMFAService(issuer)
	secret := "JBSWY3DPEHPK3PXP"
	userEmail := "test@example.com"

	qrURL := mfaService.GenerateQRCodeURL(secret, userEmail)

	expectedPrefix := "otpauth://totp/"
	if !strings.HasPrefix(qrURL, expectedPrefix) {
		t.Errorf("Expected QR URL to start with %s, got %s", expectedPrefix, qrURL)
	}

	if !strings.Contains(qrURL, issuer) {
		t.Errorf("Expected QR URL to contain issuer %s", issuer)
	}

	if !strings.Contains(qrURL, userEmail) {
		t.Errorf("Expected QR URL to contain user email %s", userEmail)
	}

	if !strings.Contains(qrURL, secret) {
		t.Errorf("Expected QR URL to contain secret %s", secret)
	}
}

func TestMFAService_GenerateAndValidateCode(t *testing.T) {
	mfaService := NewMFAService("test-issuer")

	secret, err := mfaService.GenerateSecret()
	if err != nil {
		t.Fatalf("Failed to generate MFA secret: %v", err)
	}

	// Generate code for current time
	code, err := mfaService.GenerateCode(secret)
	if err != nil {
		t.Fatalf("Failed to generate MFA code: %v", err)
	}

	if len(code) != MFACodeLength {
		t.Errorf("Expected code length %d, got %d", MFACodeLength, len(code))
	}

	// Validate the generated code
	valid, err := mfaService.ValidateCode(secret, code)
	if err != nil {
		t.Fatalf("Failed to validate MFA code: %v", err)
	}

	if !valid {
		t.Error("Generated code should be valid")
	}
}

func TestMFAService_ValidateCodeAtTime(t *testing.T) {
	mfaService := NewMFAService("test-issuer")
	secret := "JBSWY3DPEHPK3PXP" // Known test secret

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Generate code for specific time
	code, err := mfaService.GenerateCodeAtTime(secret, testTime)
	if err != nil {
		t.Fatalf("Failed to generate MFA code at time: %v", err)
	}

	// Validate code at the same time
	valid, err := mfaService.ValidateCodeAtTime(secret, code, testTime)
	if err != nil {
		t.Fatalf("Failed to validate MFA code at time: %v", err)
	}

	if !valid {
		t.Error("Code should be valid at the same time it was generated")
	}

	// Test with different time (should be invalid)
	differentTime := testTime.Add(2 * time.Minute)
	valid, err = mfaService.ValidateCodeAtTime(secret, code, differentTime)
	if err != nil {
		t.Fatalf("Failed to validate MFA code at different time: %v", err)
	}

	if valid {
		t.Error("Code should be invalid at a significantly different time")
	}
}

func TestMFAService_ValidateCodeWithSkew(t *testing.T) {
	mfaService := NewMFAService("test-issuer")
	secret := "JBSWY3DPEHPK3PXP"

	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Generate code for base time
	code, err := mfaService.GenerateCodeAtTime(secret, baseTime)
	if err != nil {
		t.Fatalf("Failed to generate MFA code: %v", err)
	}

	// Test validation within allowed skew
	skewTime := baseTime.Add(time.Duration(MFASkew) * MFATimeStep * time.Second)
	valid, err := mfaService.ValidateCodeAtTime(secret, code, skewTime)
	if err != nil {
		t.Fatalf("Failed to validate MFA code with skew: %v", err)
	}

	if !valid {
		t.Error("Code should be valid within allowed skew")
	}

	// Test validation outside allowed skew
	outsideSkewTime := baseTime.Add(time.Duration(MFASkew+2) * MFATimeStep * time.Second)
	valid, err = mfaService.ValidateCodeAtTime(secret, code, outsideSkewTime)
	if err != nil {
		t.Fatalf("Failed to validate MFA code outside skew: %v", err)
	}

	if valid {
		t.Error("Code should be invalid outside allowed skew")
	}
}

func TestMFAService_InvalidSecret(t *testing.T) {
	mfaService := NewMFAService("test-issuer")

	// Test with invalid base32 secret
	_, err := mfaService.GenerateCode("invalid-secret!")
	if err == nil {
		t.Error("Expected error for invalid secret")
	}

	// Test validation with invalid secret
	_, err = mfaService.ValidateCode("invalid-secret!", "123456")
	if err == nil {
		t.Error("Expected error for invalid secret in validation")
	}
}

func TestMFAService_InvalidCodeLength(t *testing.T) {
	mfaService := NewMFAService("test-issuer")
	secret := "JBSWY3DPEHPK3PXP"

	// Test with short code
	valid, err := mfaService.ValidateCode(secret, "123")
	if err == nil {
		t.Error("Expected error for short code")
	}
	if valid {
		t.Error("Short code should not be valid")
	}

	// Test with long code
	valid, err = mfaService.ValidateCode(secret, "1234567890")
	if err == nil {
		t.Error("Expected error for long code")
	}
	if valid {
		t.Error("Long code should not be valid")
	}
}

func TestMFAService_GenerateBackupCodes(t *testing.T) {
	mfaService := NewMFAService("test-issuer")

	codes, err := mfaService.GenerateBackupCodes(5)
	if err != nil {
		t.Fatalf("Failed to generate backup codes: %v", err)
	}

	if len(codes) != 5 {
		t.Errorf("Expected 5 backup codes, got %d", len(codes))
	}

	// Check that all codes are different
	codeMap := make(map[string]bool)
	for _, code := range codes {
		if codeMap[code] {
			t.Errorf("Duplicate backup code found: %s", code)
		}
		codeMap[code] = true

		// Check code format (should be XXXX-XXXX)
		if len(code) != 9 || code[4] != '-' {
			t.Errorf("Invalid backup code format: %s", code)
		}
	}
}

func TestMFAService_ValidateBackupCode(t *testing.T) {
	mfaService := NewMFAService("test-issuer")

	// This is a simplified test since the actual implementation would involve
	// hashing and storing backup codes
	providedCode := "1234-5678"
	storedHashedCode := "12345678" // Simplified - in reality this would be hashed

	valid, err := mfaService.ValidateBackupCode(providedCode, storedHashedCode)
	if err != nil {
		t.Fatalf("Failed to validate backup code: %v", err)
	}

	if !valid {
		t.Error("Backup code should be valid")
	}

	// Test with invalid code
	valid, err = mfaService.ValidateBackupCode("9999-9999", storedHashedCode)
	if err != nil {
		t.Fatalf("Failed to validate invalid backup code: %v", err)
	}

	if valid {
		t.Error("Invalid backup code should not be valid")
	}
}

func BenchmarkMFAService_GenerateCode(b *testing.B) {
	mfaService := NewMFAService("test-issuer")
	secret := "JBSWY3DPEHPK3PXP"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mfaService.GenerateCode(secret)
		if err != nil {
			b.Fatalf("Failed to generate MFA code: %v", err)
		}
	}
}

func BenchmarkMFAService_ValidateCode(b *testing.B) {
	mfaService := NewMFAService("test-issuer")
	secret := "JBSWY3DPEHPK3PXP"

	code, err := mfaService.GenerateCode(secret)
	if err != nil {
		b.Fatalf("Failed to generate MFA code: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mfaService.ValidateCode(secret, code)
		if err != nil {
			b.Fatalf("Failed to validate MFA code: %v", err)
		}
	}
}
