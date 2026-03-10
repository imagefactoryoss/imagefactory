package tenant

import (
	"crypto/rand"
	"fmt"
)

const (
	tenantCodeLength = 8
)

// GenerateTenantCode generates a random 8-digit numeric tenant code (e.g., "12345678")
func GenerateTenantCode() (string, error) {
	// Generate 8 random bytes
	randomBytes := make([]byte, tenantCodeLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Convert to 8-digit code (0-99999999)
	code := 0
	for i := 0; i < tenantCodeLength; i++ {
		code = code*10 + int(randomBytes[i]%10)
	}

	// Format as 8-digit string with leading zeros
	return fmt.Sprintf("%08d", code), nil
}
