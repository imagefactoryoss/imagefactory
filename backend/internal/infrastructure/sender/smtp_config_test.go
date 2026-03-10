package sender

import (
	"testing"
	"time"
)

func TestDefaultSMTPConfig(t *testing.T) {
	cfg := DefaultSMTPConfig()
	if cfg.Port != 587 {
		t.Fatalf("expected default port 587, got %d", cfg.Port)
	}
	if !cfg.UseTLS {
		t.Fatal("expected UseTLS=true by default")
	}
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("expected timeout 30s, got %v", cfg.Timeout)
	}
}

func TestSMTPConfigValidate(t *testing.T) {
	valid := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
		UseTLS:   true,
		Timeout:  10 * time.Second,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	cases := []SMTPConfig{
		{Port: 587, Username: "u", Password: "p", From: "noreply@example.com"},
		{Host: "smtp.example.com", Port: 0, Username: "u", Password: "p", From: "noreply@example.com"},
		{Host: "smtp.example.com", Port: 70000, Username: "u", Password: "p", From: "noreply@example.com"},
		{Host: "smtp.example.com", Port: 587, Password: "p", From: "noreply@example.com"},
		{Host: "smtp.example.com", Port: 587, Username: "u", From: "noreply@example.com"},
		{Host: "smtp.example.com", Port: 587, Username: "u", Password: "p", From: "invalid"},
		{Host: "smtp.example.com", Port: 587, Username: "u", Password: "p", From: "noreply@example.com", Timeout: -1 * time.Second},
	}
	for i, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error, got nil", i)
		}
	}
}
