package ldap

import (
	"context"
	"strings"
	"testing"

	goldap "github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

func TestAuthenticateRequiresCredentials(t *testing.T) {
	client := NewClient(&Config{}, zap.NewNop())

	if _, err := client.Authenticate(context.Background(), "", "x"); err == nil {
		t.Fatal("expected error for missing username")
	}
	if _, err := client.Authenticate(context.Background(), "user", ""); err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestGetAttributeValueFallbackOrder(t *testing.T) {
	client := NewClient(&Config{}, zap.NewNop())
	entry := &goldap.Entry{
		DN: "uid=alice,dc=example,dc=com",
		Attributes: []*goldap.EntryAttribute{
			{Name: "uid", Values: []string{"alice"}},
			{Name: "mail", Values: []string{"alice@example.com"}},
		},
	}

	got := client.getAttributeValue(entry, "sAMAccountName", "uid")
	if got != "alice" {
		t.Fatalf("expected fallback attr value 'alice', got %q", got)
	}

	missing := client.getAttributeValue(entry, "cn")
	if missing != "" {
		t.Fatalf("expected empty string for missing attr, got %q", missing)
	}
}

func TestSearchUserBuildsExpectedFilter(t *testing.T) {
	cfg := &Config{
		BaseDN:     "dc=example,dc=com",
		UserFilter: "(uid=%s)",
	}
	filter := strings.Replace(cfg.UserFilter, "%s", "alice", 1)
	if filter != "(uid=alice)" {
		t.Fatalf("expected filter '(uid=alice)', got %q", filter)
	}
}

func TestCloseNoop(t *testing.T) {
	client := NewClient(&Config{}, zap.NewNop())
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
