// Package auth implements user authentication with OTP, Passkey, and 2FA support
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// OTPCodeLength is the length of generated OTP codes
	OTPCodeLength = 6
	// OTPTTL is the time-to-live for OTP codes
	OTPTTL = 5 * time.Minute
	// SessionTTL is the default session lifetime
	SessionTTL = 30 * 24 * time.Hour
	// TOTPDigits for TOTP codes
	TOTPDigits = 6
	// TOTPPeriod in seconds
	TOTPPeriod = 30
)

// User represents an authenticated user
type User struct {
	ID            string    `json:"id"`
	Phone         string    `json:"phone,omitempty"`
	Email         string    `json:"email,omitempty"`
	PasswordHash  string    `json:"-"`
	PasskeyID     string    `json:"passkey_id,omitempty"`
	TOTPSecret    string    `json:"-"`
	Is2FAEnabled  bool      `json:"is_2fa_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	LastSeen      time.Time `json:"last_seen"`
	IsActive      bool      `json:"is_active"`
	SeedPhrase    string    `json:"-"` // For account recovery
}

// OTPRequest represents an OTP generation request
type OTPRequest struct {
	Identifier string `json:"identifier"` // phone or email
	Method     string `json:"method"`     // "sms", "email"
}

// OTPVerification represents an OTP verification request
type OTPVerification struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
}

// Session represents an active user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret       string        `json:"jwt_secret"`
	JWTExpiration   time.Duration `json:"jwt_expiration"`
	RefreshTTL      time.Duration `json:"refresh_ttl"`
	OTPExpiration   time.Duration `json:"otp_expiration"`
	MaxLoginAttempts int          `json:"max_login_attempts"`
	LockoutDuration time.Duration `json:"lockout_duration"`
}

// Service handles authentication operations
type Service struct {
	config      AuthConfig
	logger      *zap.Logger
	otpStore    map[string]OTPRecord // In production: Redis
	sessions    map[string]*Session  // In production: Redis
	loginAttempts map[string]int
}

// OTPRecord stores OTP code information
type OTPRecord struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Verified  bool      `json:"verified"`
}

// NewService creates a new authentication service
func NewService(config AuthConfig, logger *zap.Logger) *Service {
	return &Service{
		config:        config,
		logger:        logger,
		otpStore:      make(map[string]OTPRecord),
		sessions:      make(map[string]*Session),
		loginAttempts: make(map[string]int),
	}
}

// GenerateOTP generates a random OTP code
func (s *Service) GenerateOTP(identifier string) (string, error) {
	// Generate random 6-digit code
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	// Store OTP
	s.otpStore[identifier] = OTPRecord{
		Code:      code,
		ExpiresAt: time.Now().Add(s.config.OTPExpiration),
		Verified:  false,
	}

	s.logger.Info("OTP generated", zap.String("identifier", identifier))
	return code, nil
}

// VerifyOTP verifies an OTP code
func (s *Service) VerifyOTP(identifier, code string) (bool, error) {
	record, exists := s.otpStore[identifier]
	if !exists {
		return false, errors.New("OTP not found")
	}

	if record.Verified {
		return false, errors.New("OTP already used")
	}

	if time.Now().After(record.ExpiresAt) {
		delete(s.otpStore, identifier)
		return false, errors.New("OTP expired")
	}

	if record.Code != code {
		return false, errors.New("invalid OTP code")
	}

	// Mark as verified
	record.Verified = true
	s.otpStore[identifier] = record

	s.logger.Info("OTP verified", zap.String("identifier", identifier))
	return true, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTOTPSecret generates a new TOTP secret
func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// VerifyTOTP verifies a TOTP code
func VerifyTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateSeedPhrase generates a 12-word recovery phrase
func GenerateSeedPhrase() (string, error) {
	// Simplified version - in production use proper BIP39 wordlist
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent",
		"absorb", "abstract", "absurd", "abuse", "access", "accident",
		"account", "accuse", "achieve", "acid", "acoustic", "acquire",
		"across", "act", "action", "actor", "actress", "actual",
		"adapt", "add", "addict", "address", "adjust", "admit",
		"adult", "advance", "advice", "aerobic", "affair", "afford",
		"afraid", "again", "age", "agent", "agree", "ahead",
		"aim", "air", "airport", "aisle", "alarm", "album",
		"alcohol", "alert", "alien", "all", "alley", "allow",
		"almost", "alone", "alpha", "already", "also", "alter",
		"always", "amateur", "amazing", "among", "amount", "amused",
		"analyst", "anchor", "ancient", "anger", "angle", "angry",
		"animal", "ankle", "announce", "annual", "another", "answer",
		"antenna", "anticipate", "anxiety", "any", "apart", "apology",
		"appear", "apple", "approve", "april", "arch", "arctic",
		"area", "arena", "argue", "arm", "armed", "armor",
		"army", "around", "arrange", "arrest", "arrive", "arrow",
		"art", "artefact", "artist", "artwork", "ask", "aspect",
		"assault", "asset", "assist", "assume", "asthma", "athlete",
		"atom", "attack", "attend", "attitude", "attract", "auction",
		"audit", "august", "aunt", "author", "auto", "autumn",
		"average", "avocado", "avoid", "awake", "aware", "away",
		"awesome", "awful", "awkward", "axis", "baby", "bachelor",
	}

	phrase := make([]string, 12)
	for i := 0; i < 12; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
		if err != nil {
			return "", err
		}
		phrase[i] = words[idx.Int64()]
	}

	return fmt.Sprintf("%s", phrase), nil
}

// CreateSession creates a new session for a user
func (s *Service) CreateSession(ctx context.Context, userID, deviceID, ip, userAgent string) (*Session, error) {
	accessToken := make([]byte, 32)
	refreshToken := make([]byte, 32)

	if _, err := rand.Read(accessToken); err != nil {
		return nil, err
	}
	if _, err := rand.Read(refreshToken); err != nil {
		return nil, err
	}

	session := &Session{
		ID:           hex.EncodeToString(accessToken[:16]),
		UserID:       userID,
		DeviceID:     deviceID,
		AccessToken:  hex.EncodeToString(accessToken),
		RefreshToken: hex.EncodeToString(refreshToken),
		ExpiresAt:    time.Now().Add(s.config.JWTExpiration),
		CreatedAt:    time.Now(),
		IPAddress:    ip,
		UserAgent:    userAgent,
	}

	s.sessions[session.ID] = session
	s.logger.Info("Session created", zap.String("user_id", userID), zap.String("session_id", session.ID))

	return session, nil
}

// ValidateSession validates a session token
func (s *Service) ValidateSession(token string) (*Session, error) {
	for _, session := range s.sessions {
		if session.AccessToken == token {
			if time.Now().After(session.ExpiresAt) {
				return nil, errors.New("session expired")
			}
			return session, nil
		}
	}
	return nil, errors.New("invalid session")
}

// RefreshSession refreshes an existing session
func (s *Service) RefreshSession(refreshToken string) (*Session, error) {
	for _, session := range s.sessions {
		if session.RefreshToken == refreshToken {
			if time.Now().After(session.ExpiresAt.Add(s.config.RefreshTTL)) {
				return nil, errors.New("refresh token expired")
			}

			// Generate new tokens
			newAccessToken := make([]byte, 32)
			if _, err := rand.Read(newAccessToken); err != nil {
				return nil, err
			}

			session.AccessToken = hex.EncodeToString(newAccessToken)
			session.ExpiresAt = time.Now().Add(s.config.JWTExpiration)

			s.logger.Info("Session refreshed", zap.String("session_id", session.ID))
			return session, nil
		}
	}
	return nil, errors.New("invalid refresh token")
}

// RevokeSession revokes a session
func (s *Service) RevokeSession(sessionID string) {
	delete(s.sessions, sessionID)
	s.logger.Info("Session revoked", zap.String("session_id", sessionID))
}

// TrackLoginAttempt tracks a login attempt for rate limiting
func (s *Service) TrackLoginAttempt(identifier string, success bool) {
	if success {
		delete(s.loginAttempts, identifier)
		return
	}

	s.loginAttempts[identifier]++
	s.logger.Warn("Login attempt failed", zap.String("identifier", identifier), zap.Int("attempts", s.loginAttempts[identifier]))
}

// IsLockedOut checks if an identifier is locked out due to too many failed attempts
func (s *Service) IsLockedOut(identifier string) bool {
	attempts, exists := s.loginAttempts[identifier]
	return exists && attempts >= s.config.MaxLoginAttempts
}
