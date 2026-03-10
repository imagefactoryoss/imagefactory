package mfa

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
)

type mockMFARepo struct {
	methodsByUser map[uuid.UUID][]*MFAMethodConfig
	challenges    map[uuid.UUID]*MFAChallenge
}

func newMockMFARepo() *mockMFARepo {
	return &mockMFARepo{
		methodsByUser: map[uuid.UUID][]*MFAMethodConfig{},
		challenges:    map[uuid.UUID]*MFAChallenge{},
	}
}

func (m *mockMFARepo) SaveMFAMethod(ctx context.Context, method *MFAMethodConfig) error {
	m.methodsByUser[method.UserID] = append(m.methodsByUser[method.UserID], method)
	return nil
}
func (m *mockMFARepo) UpdateMFAMethod(ctx context.Context, method *MFAMethodConfig) error { return nil }
func (m *mockMFARepo) DeleteMFAMethod(ctx context.Context, id uuid.UUID) error            { return nil }
func (m *mockMFARepo) FindMFAMethodByID(ctx context.Context, id uuid.UUID) (*MFAMethodConfig, error) {
	return nil, nil
}
func (m *mockMFARepo) FindMFAMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*MFAMethodConfig, error) {
	return m.methodsByUser[userID], nil
}
func (m *mockMFARepo) FindPrimaryMFAMethod(ctx context.Context, userID uuid.UUID) (*MFAMethodConfig, error) {
	methods := m.methodsByUser[userID]
	if len(methods) == 0 {
		return nil, nil
	}
	return methods[0], nil
}
func (m *mockMFARepo) SaveMFAChallenge(ctx context.Context, challenge *MFAChallenge) error {
	m.challenges[challenge.ID] = challenge
	return nil
}
func (m *mockMFARepo) UpdateMFAChallenge(ctx context.Context, challenge *MFAChallenge) error {
	m.challenges[challenge.ID] = challenge
	return nil
}
func (m *mockMFARepo) FindMFAChallenge(ctx context.Context, id uuid.UUID) (*MFAChallenge, error) {
	return m.challenges[id], nil
}
func (m *mockMFARepo) FindActiveChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*MFAChallenge, error) {
	return nil, nil
}
func (m *mockMFARepo) DeleteExpiredChallenges(ctx context.Context) error { return nil }

type mockEmailSvc struct {
	count int
	last  domainEmail.CreateEmailRequest
}

func (m *mockEmailSvc) CreateEmail(ctx context.Context, req domainEmail.CreateEmailRequest) (*domainEmail.Email, error) {
	m.count++
	m.last = req
	return domainEmail.NewEmail(req)
}

func TestSetupTOTPSecret(t *testing.T) {
	svc := NewService(newMockMFARepo(), zap.NewNop())
	secret, err := svc.SetupTOTPSecret(context.Background(), uuid.New(), "alice@example.com", "ImageFactory")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if secret.Secret == "" || len(secret.Secret) < 16 {
		t.Fatalf("expected non-empty secret, got %q", secret.Secret)
	}
	if !strings.HasPrefix(secret.QRCodeURL, "otpauth://totp/") {
		t.Fatalf("expected otpauth URL, got %q", secret.QRCodeURL)
	}
}

func TestGenerateNumericCode(t *testing.T) {
	svc := NewService(newMockMFARepo(), zap.NewNop())
	code := svc.generateNumericCode(6)
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %q", code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("expected numeric code, got %q", code)
		}
	}
}

func TestGenerateQRCodeURL(t *testing.T) {
	svc := NewService(newMockMFARepo(), zap.NewNop())
	url := svc.generateQRCodeURL("ABCDEF", "alice@example.com", "Image Factory")
	if !strings.Contains(url, "secret=ABCDEF") {
		t.Fatalf("expected secret in QR URL, got %q", url)
	}
	if !strings.Contains(url, "issuer=Image+Factory") {
		t.Fatalf("expected URL-encoded issuer in QR URL, got %q", url)
	}
}

func TestStartMFAChallenge_Email(t *testing.T) {
	repo := newMockMFARepo()
	emailSvc := &mockEmailSvc{}
	svc := NewServiceWithEmail(repo, emailSvc, "noreply@example.com", zap.NewNop())
	userID := uuid.New()
	repo.methodsByUser[userID] = []*MFAMethodConfig{
		{
			ID:       uuid.New(),
			UserID:   userID,
			Method:   MFAMethodEmail,
			Email:    "alice@example.com",
			IsActive: true,
		},
	}

	challenge, err := svc.StartMFAChallenge(context.Background(), MFAChallengeRequest{
		UserID: userID,
		Method: MFAMethodEmail,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if challenge.Method != MFAMethodEmail || challenge.Status != MFAStatusPending {
		t.Fatalf("unexpected challenge state: %+v", challenge)
	}
	if len(challenge.Code) != 6 {
		t.Fatalf("expected 6-digit challenge code, got %q", challenge.Code)
	}
	if emailSvc.count != 1 {
		t.Fatalf("expected one email send, got %d", emailSvc.count)
	}
}

func TestVerifyMFAChallenge_Email(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewService(repo, zap.NewNop())
	ch := &MFAChallenge{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Method:    MFAMethodEmail,
		Code:      "123456",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Status:    MFAStatusPending,
	}
	repo.challenges[ch.ID] = ch

	if err := svc.VerifyMFAChallenge(context.Background(), MFAVerificationRequest{
		ChallengeID: ch.ID,
		Code:        "123456",
	}); err != nil {
		t.Fatalf("expected successful verification, got %v", err)
	}
	if repo.challenges[ch.ID].Status != MFAStatusVerified {
		t.Fatalf("expected verified status, got %s", repo.challenges[ch.ID].Status)
	}
}
