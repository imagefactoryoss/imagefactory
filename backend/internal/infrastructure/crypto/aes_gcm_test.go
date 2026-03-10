package crypto

import (
	"encoding/base64"
	"testing"
)

func TestNewAESGCMEncryptor(t *testing.T) {
	_, err := NewAESGCMEncryptor([]byte("short"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}

	key := make([]byte, 32)
	enc, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enc == nil {
		t.Fatal("expected encryptor, got nil")
	}
}

func TestAESGCMEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	enc, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := []byte("hello world")
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("expected non-empty ciphertext")
	}

	out, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}
	if string(out) != string(plaintext) {
		t.Fatalf("expected %q, got %q", string(plaintext), string(out))
	}
}

func TestAESGCMDecryptErrors(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	enc, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	_, err = enc.Decrypt([]byte("tiny"))
	if err != ErrInvalidCiphertext {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}

	ciphertext, err := enc.Encrypt([]byte("payload"))
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xFF
	_, err = enc.Decrypt(ciphertext)
	if err != ErrDecryptionFailed {
		t.Fatalf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestAESGCMStringHelpers(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	enc, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	encrypted, err := enc.EncryptString("secret")
	if err != nil {
		t.Fatalf("failed to EncryptString: %v", err)
	}
	if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
		t.Fatalf("expected base64 ciphertext, got error: %v", err)
	}

	decrypted, err := enc.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("failed to DecryptString: %v", err)
	}
	if decrypted != "secret" {
		t.Fatalf("expected %q, got %q", "secret", decrypted)
	}

	if _, err := enc.DecryptString("%%%"); err == nil {
		t.Fatal("expected base64 decode error, got nil")
	}
}

func TestNewAESGCMEncryptorFromEnv(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "")
	if _, err := NewAESGCMEncryptorFromEnv(); err == nil {
		t.Fatal("expected error when ENCRYPTION_KEY is missing")
	}

	t.Setenv("ENCRYPTION_KEY", "not-base64")
	if _, err := NewAESGCMEncryptorFromEnv(); err == nil {
		t.Fatal("expected error when ENCRYPTION_KEY is invalid base64")
	}

	key := make([]byte, 32)
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
	if _, err := NewAESGCMEncryptorFromEnv(); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}
