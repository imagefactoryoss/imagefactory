package template

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestTemplateNotFoundErrorHelpers(t *testing.T) {
	err := &TemplateNotFoundError{
		CompanyID:    uuid.New(),
		TenantID:     uuid.New(),
		TemplateName: "welcome",
		Locale:       "en",
	}
	if !IsTemplateNotFoundError(err) {
		t.Fatal("expected true for TemplateNotFoundError")
	}
	if IsTemplateNotFoundError(errors.New("other")) {
		t.Fatal("expected false for non-TemplateNotFoundError")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error string")
	}
}
