package sender

import (
	"testing"
)

func TestEmailSenderFactory_CreateSender(t *testing.T) {
	f := NewEmailSenderFactory()

	if _, err := f.CreateSender(SenderConfig{Type: SenderTypeSMTP}); err == nil {
		t.Fatal("expected error for SMTP sender without SMTP config")
	}

	if _, err := f.CreateSender(SenderConfig{
		Type: SenderTypeSMTP,
		SMTP: &SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user",
			Password: "pass",
			From:     "noreply@example.com",
		},
	}); err != nil {
		t.Fatalf("expected SMTP sender creation success, got %v", err)
	}

	if _, err := f.CreateSender(SenderConfig{Type: SenderTypeMock}); err != nil {
		t.Fatalf("expected mock sender creation success, got %v", err)
	}

	if _, err := f.CreateSender(SenderConfig{Type: SenderType("unknown")}); err == nil {
		t.Fatal("expected unsupported sender type error")
	}
}

func TestEmailSenderFactory_ConvenienceMethods(t *testing.T) {
	f := NewEmailSenderFactory()
	if _, err := f.CreateSMTPSender(SMTPConfig{}); err == nil {
		t.Fatal("expected validation error for invalid SMTP config")
	}

	sender, err := CreateDefaultSMTPSender("smtp.example.com", 587, "u", "p", "noreply@example.com")
	if err != nil {
		t.Fatalf("expected CreateDefaultSMTPSender success, got %v", err)
	}
	if sender == nil {
		t.Fatal("expected non-nil sender")
	}

	if f.CreateMockSender() == nil {
		t.Fatal("expected non-nil mock sender")
	}
}
