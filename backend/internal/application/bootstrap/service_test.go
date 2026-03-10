package bootstrap

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateInitialPassword(t *testing.T) {
	pwd, err := generateInitialPassword(8) // should be clamped to 16
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pwd) != 16 {
		t.Fatalf("expected length 16 for clamped short input, got %d", len(pwd))
	}

	pwd2, err := generateInitialPassword(40)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pwd2) != 40 {
		t.Fatalf("expected length 40, got %d", len(pwd2))
	}
	if pwd == pwd2 {
		t.Fatal("expected different random passwords")
	}
}

func TestIsMissingBootstrapTableError(t *testing.T) {
	if isMissingBootstrapTableError(nil) {
		t.Fatal("expected false for nil error")
	}

	err := errors.New(`pq: relation "system_bootstrap_state" does not exist`)
	if !isMissingBootstrapTableError(err) {
		t.Fatal("expected true for missing bootstrap table error")
	}

	err2 := errors.New("some other database error")
	if isMissingBootstrapTableError(err2) {
		t.Fatal("expected false for unrelated error")
	}

	err3 := errors.New(strings.ToUpper(`relation "system_bootstrap_state" does not exist`))
	if !isMissingBootstrapTableError(err3) {
		t.Fatal("expected true for case-insensitive matching")
	}
}
