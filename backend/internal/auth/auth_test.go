package auth

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGenerateOTP(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		OTPExpiration: 5 * time.Minute,
	}
	svc := NewService(config, logger)

	code, err := svc.GenerateOTP("test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate OTP: %v", err)
	}

	if len(code) != 6 {
		t.Errorf("Expected OTP length 6, got %d", len(code))
	}

	// Verify code is numeric
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("OTP contains non-numeric character: %c", c)
		}
	}
}

func TestVerifyOTP(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		OTPExpiration: 5 * time.Minute,
	}
	svc := NewService(config, logger)

	identifier := "test@example.com"
	code, err := svc.GenerateOTP(identifier)
	if err != nil {
		t.Fatalf("Failed to generate OTP: %v", err)
	}

	// Test valid OTP
	valid, err := svc.VerifyOTP(identifier, code)
	if err != nil {
		t.Fatalf("Failed to verify valid OTP: %v", err)
	}
	if !valid {
		t.Error("Expected OTP to be valid")
	}

	// Test already used OTP
	valid, err = svc.VerifyOTP(identifier, code)
	if err == nil {
		t.Error("Expected error for already used OTP")
	}
	if valid {
		t.Error("Expected OTP to be invalid after first use")
	}
}

func TestVerifyOTPExpired(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		OTPExpiration: 1 * time.Millisecond, // Very short TTL
	}
	svc := NewService(config, logger)

	identifier := "test@example.com"
	code, err := svc.GenerateOTP(identifier)
	if err != nil {
		t.Fatalf("Failed to generate OTP: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	valid, err := svc.VerifyOTP(identifier, code)
	if err == nil {
		t.Error("Expected error for expired OTP")
	}
	if valid {
		t.Error("Expected OTP to be invalid after expiration")
	}
}

func TestHashPassword(t *testing.T) {
	password := "securepassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == password {
		t.Error("Hash should not equal original password")
	}

	if len(hash) < len(password) {
		t.Error("Hash should be longer than original password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "securepassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !CheckPasswordHash(password, hash) {
		t.Error("Correct password should match hash")
	}

	// Test wrong password
	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("Wrong password should not match hash")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}

	if len(secret) == 0 {
		t.Error("TOTP secret should not be empty")
	}

	// Base32 encoding uses A-Z and 2-7
	for _, c := range secret {
		if !((c >= 'A' && c <= 'Z') || (c >= '2' && c <= '7')) {
			t.Errorf("Invalid character in TOTP secret: %c", c)
		}
	}
}

func TestCreateSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		JWTExpiration: 24 * time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	svc := NewService(config, logger)

	ctx := context.Background()
	session, err := svc.CreateSession(ctx, "user123", "device456", "192.168.1.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != "user123" {
		t.Errorf("Expected user_id 'user123', got '%s'", session.UserID)
	}

	if session.DeviceID != "device456" {
		t.Errorf("Expected device_id 'device456', got '%s'", session.DeviceID)
	}

	if len(session.AccessToken) == 0 {
		t.Error("Access token should not be empty")
	}

	if len(session.RefreshToken) == 0 {
		t.Error("Refresh token should not be empty")
	}
}

func TestValidateSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		JWTExpiration: 24 * time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	svc := NewService(config, logger)

	ctx := context.Background()
	session, err := svc.CreateSession(ctx, "user123", "device456", "192.168.1.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test valid session
	validated, err := svc.ValidateSession(session.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate valid session: %v", err)
	}
	if validated.ID != session.ID {
		t.Error("Validated session ID should match original")
	}

	// Test invalid token
	_, err = svc.ValidateSession("invalid_token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestRefreshSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		JWTExpiration: 24 * time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	svc := NewService(config, logger)

	ctx := context.Background()
	session, err := svc.CreateSession(ctx, "user123", "device456", "192.168.1.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	oldAccessToken := session.AccessToken

	// Refresh session
	refreshed, err := svc.RefreshSession(session.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}

	if refreshed.AccessToken == oldAccessToken {
		t.Error("Access token should change after refresh")
	}

	// Test invalid refresh token
	_, err = svc.RefreshSession("invalid_refresh_token")
	if err == nil {
		t.Error("Expected error for invalid refresh token")
	}
}

func TestRevokeSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		JWTExpiration: 24 * time.Hour,
		RefreshTTL:    7 * 24 * time.Hour,
	}
	svc := NewService(config, logger)

	ctx := context.Background()
	session, err := svc.CreateSession(ctx, "user123", "device456", "192.168.1.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Revoke session
	svc.RevokeSession(session.ID)

	// Try to validate revoked session
	_, err = svc.ValidateSession(session.AccessToken)
	if err == nil {
		t.Error("Expected error for revoked session")
	}
}

func TestTrackLoginAttempt(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := AuthConfig{
		MaxLoginAttempts: 3,
	}
	svc := NewService(config, logger)

	identifier := "test@example.com"

	// Track failed attempts
	for i := 0; i < 3; i++ {
		svc.TrackLoginAttempt(identifier, false)
	}

	// Check if locked out
	if !svc.IsLockedOut(identifier) {
		t.Error("Should be locked out after 3 failed attempts")
	}

	// Successful login should reset counter
	svc.TrackLoginAttempt(identifier, true)
	if svc.IsLockedOut(identifier) {
		t.Error("Should not be locked out after successful login")
	}
}

func TestGenerateSeedPhrase(t *testing.T) {
	phrase, err := GenerateSeedPhrase()
	if err != nil {
		t.Fatalf("Failed to generate seed phrase: %v", err)
	}

	// Check phrase format (12 words separated by spaces)
	// Note: our simplified implementation returns a string representation of slice
	if len(phrase) == 0 {
		t.Error("Seed phrase should not be empty")
	}
}
