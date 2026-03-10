package mfa

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
)

// MFAMethod represents different MFA methods
type MFAMethod string

const (
	MFAMethodTOTP MFAMethod = "totp"
	MFAMethodSMS  MFAMethod = "sms"
	MFAMethodEmail MFAMethod = "email"
)

// MFAStatus represents the status of MFA verification
type MFAStatus string

const (
	MFAStatusPending MFAStatus = "pending"
	MFAStatusVerified MFAStatus = "verified"
	MFAStatusFailed  MFAStatus = "failed"
)

// MFAMethod represents a configured MFA method for a user
type MFAMethodConfig struct {
	ID           uuid.UUID    `json:"id"`
	UserID       uuid.UUID    `json:"user_id"`
	Method       MFAMethod    `json:"method"`
	Secret       string       `json:"secret,omitempty"` // For TOTP
	PhoneNumber  string       `json:"phone_number,omitempty"` // For SMS
	Email        string       `json:"email,omitempty"` // For Email
	IsActive     bool         `json:"is_active"`
	IsPrimary    bool         `json:"is_primary"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// MFAChallenge represents an active MFA challenge
type MFAChallenge struct {
	ID           uuid.UUID    `json:"id"`
	UserID       uuid.UUID    `json:"user_id"`
	Method       MFAMethod    `json:"method"`
	Code         string       `json:"code"`
	ExpiresAt    time.Time    `json:"expires_at"`
	Status       MFAStatus    `json:"status"`
	Attempts     int          `json:"attempts"`
	CreatedAt    time.Time    `json:"created_at"`
	VerifiedAt   *time.Time   `json:"verified_at,omitempty"`
}

// TOTPSecret represents a TOTP secret with metadata
type TOTPSecret struct {
	Secret      string    `json:"secret"`
	AccountName string    `json:"account_name"`
	Issuer      string    `json:"issuer"`
	Algorithm   string    `json:"algorithm"` // default: SHA1
	Digits      int       `json:"digits"`    // default: 6
	Period      int       `json:"period"`    // default: 30
	QRCodeURL   string    `json:"qr_code_url"`
}

// MFAChallengeRequest represents a request to create an MFA challenge
type MFAChallengeRequest struct {
	UserID uuid.UUID   `json:"user_id"`
	Method MFAMethod   `json:"method"`
}

// MFASetupRequest represents a request to setup an MFA method
type MFASetupRequest struct {
	UserID      uuid.UUID   `json:"user_id"`
	Method      MFAMethod   `json:"method"`
	Secret      string      `json:"secret,omitempty"` // For TOTP
	PhoneNumber string      `json:"phone_number,omitempty"` // For SMS
	Email       string      `json:"email,omitempty"` // For Email
}

// MFAVerificationRequest represents a request to verify an MFA code
type MFAVerificationRequest struct {
	ChallengeID uuid.UUID `json:"challenge_id"`
	Code        string    `json:"code"`
}

// MFAVerifyCodeRequest represents a request to verify an MFA code directly
type MFAVerifyCodeRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	Method      MFAMethod `json:"method"`
	Code        string    `json:"code"`
}

// Repository defines the interface for MFA persistence
type Repository interface {
	// MFA method management
	SaveMFAMethod(ctx context.Context, method *MFAMethodConfig) error
	UpdateMFAMethod(ctx context.Context, method *MFAMethodConfig) error
	DeleteMFAMethod(ctx context.Context, id uuid.UUID) error
	FindMFAMethodByID(ctx context.Context, id uuid.UUID) (*MFAMethodConfig, error)
	FindMFAMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*MFAMethodConfig, error)
	FindPrimaryMFAMethod(ctx context.Context, userID uuid.UUID) (*MFAMethodConfig, error)

	// MFA challenge management
	SaveMFAChallenge(ctx context.Context, challenge *MFAChallenge) error
	UpdateMFAChallenge(ctx context.Context, challenge *MFAChallenge) error
	FindMFAChallenge(ctx context.Context, id uuid.UUID) (*MFAChallenge, error)
	FindActiveChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*MFAChallenge, error)
	DeleteExpiredChallenges(ctx context.Context) error
}

// Service represents the MFA service
type Service struct {
	repo           Repository
	emailService   EmailService
	fromEmail      string
	logger         *zap.Logger
}

// EmailService defines the interface for email operations
type EmailService interface {
	CreateEmail(ctx context.Context, req domainEmail.CreateEmailRequest) (*domainEmail.Email, error)
}

// NewService creates a new MFA service
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// NewServiceWithEmail creates a new MFA service with email support
func NewServiceWithEmail(repo Repository, emailService EmailService, fromEmail string, logger *zap.Logger) *Service {
	return &Service{
		repo:         repo,
		emailService: emailService,
		fromEmail:    fromEmail,
		logger:       logger,
	}
}

// SetupTOTPSecret generates a new TOTP secret for a user
func (s *Service) SetupTOTPSecret(ctx context.Context, userID uuid.UUID, accountName, issuer string) (*TOTPSecret, error) {
	// Generate random secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		s.logger.Error("Failed to generate TOTP secret", zap.Error(err))
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Generate QR code URL
	qrCodeURL := s.generateQRCodeURL(secret, accountName, issuer)

	totpSecret := &TOTPSecret{
		Secret:      secret,
		AccountName: accountName,
		Issuer:      issuer,
		Algorithm:   "SHA1",
		Digits:      6,
		Period:      30,
		QRCodeURL:   qrCodeURL,
	}

	s.logger.Info("TOTP secret generated successfully",
		zap.String("user_id", userID.String()),
		zap.String("account_name", accountName))

	return totpSecret, nil
}

// ConfirmTOTPSetup confirms TOTP setup and saves the secret
func (s *Service) ConfirmTOTPSetup(ctx context.Context, userID uuid.UUID, secret string, code string) error {
	// Verify the provided code against the secret
	if !s.verifyTOTP(secret, code) {
		s.logger.Warn("Invalid TOTP code during setup",
			zap.String("user_id", userID.String()))
		return fmt.Errorf("invalid TOTP code")
	}

	// Save TOTP method
	method := &MFAMethodConfig{
		ID:        uuid.New(),
		UserID:    userID,
		Method:    MFAMethodTOTP,
		Secret:    secret,
		IsActive:  true,
		IsPrimary: true, // Make TOTP primary by default
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveMFAMethod(ctx, method); err != nil {
		s.logger.Error("Failed to save TOTP method",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to save TOTP method: %w", err)
	}

	s.logger.Info("TOTP setup completed successfully",
		zap.String("user_id", userID.String()))

	return nil
}

// StartMFAChallenge starts an MFA challenge for a user
func (s *Service) StartMFAChallenge(ctx context.Context, req MFAChallengeRequest) (*MFAChallenge, error) {
	// Find user's MFA methods
	methods, err := s.repo.FindMFAMethodsByUserID(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to find user MFA methods",
			zap.String("user_id", req.UserID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to find MFA methods: %w", err)
	}

	// Find the requested method
	var targetMethod *MFAMethodConfig
	for _, method := range methods {
		if method.Method == req.Method && method.IsActive {
			targetMethod = method
			break
		}
	}

	if targetMethod == nil {
		s.logger.Warn("MFA method not found or inactive",
			zap.String("user_id", req.UserID.String()),
			zap.String("method", string(req.Method)))
		return nil, fmt.Errorf("MFA method not found or inactive")
	}

	// Generate challenge
	challenge := &MFAChallenge{
		ID:        uuid.New(),
		UserID:    req.UserID,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Status:    MFAStatusPending,
		Attempts:  0,
		CreatedAt: time.Now().UTC(),
	}

	// Generate and send code based on method
	switch req.Method {
	case MFAMethodTOTP:
		// TOTP is verified directly during challenge verification
		challenge.Code = "" // TOTP doesn't need a stored code
	case MFAMethodSMS:
		code := s.generateNumericCode(6)
		challenge.Code = code
		if err := s.sendSMSCode(ctx, req.UserID, targetMethod.PhoneNumber, code); err != nil {
			return nil, fmt.Errorf("failed to send SMS code: %w", err)
		}
	case MFAMethodEmail:
		code := s.generateNumericCode(6)
		challenge.Code = code
		if err := s.sendEmailCode(ctx, req.UserID, targetMethod.Email, code); err != nil {
			return nil, fmt.Errorf("failed to send email code: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported MFA method: %s", req.Method)
	}

	// Save challenge
	if err := s.repo.SaveMFAChallenge(ctx, challenge); err != nil {
		s.logger.Error("Failed to save MFA challenge",
			zap.String("challenge_id", challenge.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to save MFA challenge: %w", err)
	}

	s.logger.Info("MFA challenge started",
		zap.String("user_id", req.UserID.String()),
		zap.String("method", string(req.Method)))

	return challenge, nil
}

// VerifyMFAChallenge verifies an MFA challenge
func (s *Service) VerifyMFAChallenge(ctx context.Context, req MFAVerificationRequest) error {
	challenge, err := s.repo.FindMFAChallenge(ctx, req.ChallengeID)
	if err != nil {
		s.logger.Error("Failed to find MFA challenge",
			zap.String("challenge_id", req.ChallengeID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to find challenge: %w", err)
	}

	// Check if challenge is expired
	if time.Now().After(challenge.ExpiresAt) {
		challenge.Status = MFAStatusFailed
		s.repo.UpdateMFAChallenge(ctx, challenge)
		return fmt.Errorf("challenge has expired")
	}

	// Check if challenge is already verified or failed
	if challenge.Status != MFAStatusPending {
		return fmt.Errorf("challenge is not in pending status")
	}

	challenge.Attempts++

	// Verify code based on method
	var isValid bool
	switch challenge.Method {
	case MFAMethodTOTP:
		// For TOTP, we need to find the user's secret
		methods, err := s.repo.FindMFAMethodsByUserID(ctx, challenge.UserID)
		if err == nil {
			for _, method := range methods {
				if method.Method == MFAMethodTOTP && method.IsActive {
					isValid = s.verifyTOTP(method.Secret, req.Code)
					break
				}
			}
		}
	case MFAMethodSMS, MFAMethodEmail:
		isValid = challenge.Code == req.Code
	}

	if isValid {
		challenge.Status = MFAStatusVerified
		now := time.Now().UTC()
		challenge.VerifiedAt = &now
		s.logger.Info("MFA challenge verified successfully",
			zap.String("challenge_id", challenge.ID.String()))
	} else {
		challenge.Status = MFAStatusFailed
		s.logger.Warn("MFA challenge verification failed",
			zap.String("challenge_id", challenge.ID.String()),
			zap.Int("attempts", challenge.Attempts))
	}

	return s.repo.UpdateMFAChallenge(ctx, challenge)
}

// verifyTOTP verifies a TOTP code against a secret
func (s *Service) verifyTOTP(secret, code string) bool {
	// Decode secret
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		s.logger.Error("Failed to decode TOTP secret", zap.Error(err))
		return false
	}

	// Get current time step
	timeStep := time.Now().Unix() / 30

	// Check current time step and previous one (for clock skew)
	for i := 0; i <= 1; i++ {
		if s.checkTOTPCode(secretBytes, timeStep-int64(i), code) {
			return true
		}
	}

	return false
}

// checkTOTPCode checks a TOTP code for a specific time step
func (s *Service) checkTOTPCode(secretBytes []byte, timeStep int64, code string) bool {
	// Create HMAC-SHA1
	timeBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		timeBytes[i] = byte(timeStep & 0xFF)
		timeStep >>= 8
	}
	
	hmac := hmac.New(sha1.New, secretBytes)
	hmac.Write(timeBytes)

	// Get hash
	hashBytes := hmac.Sum(nil)

	// Dynamic truncation
	offset := hashBytes[len(hashBytes)-1] & 0x0F
	binCode := int64(hashBytes[offset]&0x7F)<<24 |
		int64(hashBytes[offset+1]&0xFF)<<16 |
		int64(hashBytes[offset+2]&0xFF)<<8 |
		int64(hashBytes[offset+3]&0xFF)

	// Get digits
	otp := binCode % 1000000 // 6 digits

	// Pad with zeros if needed
	expectedCode := fmt.Sprintf("%06d", otp)

	return expectedCode == code
}

// generateNumericCode generates a random numeric code
func (s *Service) generateNumericCode(length int) string {
	const digits = "0123456789"
	code := make([]byte, length)
	for i := range code {
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			// Fallback to pseudo-random
			randomByte[0] = byte(time.Now().UnixNano() % 256)
		}
		code[i] = digits[int(randomByte[0])%len(digits)]
	}
	return string(code)
}

// generateQRCodeURL generates a QR code URL for TOTP setup
func (s *Service) generateQRCodeURL(secret, accountName, issuer string) string {
	// URL encode the parameters
	secret = url.QueryEscape(secret)
	accountName = url.QueryEscape(accountName)
	issuer = url.QueryEscape(issuer)

	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, accountName, secret, issuer)
}

// sendSMSCode sends an SMS code (placeholder - SMS integration not implemented for current phase)
func (s *Service) sendSMSCode(ctx context.Context, userID uuid.UUID, phoneNumber, code string) error {
	s.logger.Info("SMS code would be sent",
		zap.String("user_id", userID.String()),
		zap.String("phone_number", phoneNumber),
		zap.String("code", code))
	// TODO: Integrate with SMS service (Twilio, AWS SNS, etc.) - Skipped for current phase
	return nil
}

// sendEmailCode sends an email code (placeholder - would integrate with email service)
func (s *Service) sendEmailCode(ctx context.Context, userID uuid.UUID, email, code string) error {
	if s.emailService == nil {
		return errors.New("email service not configured")
	}

	// Create email request
	req := domainEmail.CreateEmailRequest{
		TenantID:  uuid.Nil, // TODO: Get tenant ID from user context
		ToEmail:   email,
		FromEmail: s.fromEmail,
		Subject:   "Your MFA Code",
		BodyText:  fmt.Sprintf("Your MFA code is: %s\n\nThis code will expire in 10 minutes.", code),
		BodyHTML:  fmt.Sprintf("<p>Your MFA code is: <strong>%s</strong></p><p>This code will expire in 10 minutes.</p>", code),
		EmailType: domainEmail.EmailTypeNotification, // TODO: Define proper email type
		Priority:  1,
	}

	_, err := s.emailService.CreateEmail(ctx, req)
	if err != nil {
		s.logger.Error("failed to send MFA email", zap.Error(err), zap.String("userID", userID.String()), zap.String("email", email))
		return fmt.Errorf("failed to send MFA email: %w", err)
	}

	s.logger.Info("MFA email sent successfully", zap.String("userID", userID.String()), zap.String("email", email))
	return nil
}