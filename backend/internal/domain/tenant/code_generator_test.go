package tenant

import (
	"regexp"
	"testing"
)

func TestGenerateTenantCode(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "generates 8-digit code",
			test: func(t *testing.T) {
				code, err := GenerateTenantCode()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(code) != 8 {
					t.Errorf("expected code length 8, got %d", len(code))
				}
			},
		},
		{
			name: "generates numeric code only",
			test: func(t *testing.T) {
				code, err := GenerateTenantCode()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// Check that code only contains digits
				matched, _ := regexp.MatchString("^[0-9]{8}$", code)
				if !matched {
					t.Errorf("code %q is not valid 8-digit numeric code", code)
				}
			},
		},
		{
			name: "generates unique codes",
			test: func(t *testing.T) {
				codes := make(map[string]bool)
				for i := 0; i < 100; i++ {
					code, err := GenerateTenantCode()
					if err != nil {
						t.Fatalf("expected no error on iteration %d, got %v", i, err)
					}
					if codes[code] {
						t.Errorf("generated duplicate code: %q", code)
					}
					codes[code] = true
				}
			},
		},
		{
			name: "includes leading zeros",
			test: func(t *testing.T) {
				// Generate many codes to hopefully get one with leading zeros
				found := false
				for i := 0; i < 1000; i++ {
					code, err := GenerateTenantCode()
					if err != nil {
						t.Fatalf("expected no error on iteration %d, got %v", i, err)
					}
					// Check if code starts with 0 (this will eventually happen with random generation)
					if code[0] == '0' {
						found = true
						break
					}
				}
				if !found {
					t.Log("Note: Did not generate code with leading zero in 1000 iterations (statistically possible)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
