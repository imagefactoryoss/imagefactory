package notification

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockTemplateRepo struct {
	template *Template
	err      error
}

func (m *mockTemplateRepo) GetNotificationTemplate(ctx context.Context, companyID, tenantID uuid.UUID, notificationType, channelType, locale string) (*Template, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.template, nil
}

func TestRenderTemplateSuccessWithHTML(t *testing.T) {
	html := "<p>Hello {{.name}}</p>"
	repo := &mockTemplateRepo{
		template: &Template{
			SubjectTemplate: " Welcome {{.name}} ",
			BodyTemplate:    " Body for {{.name}} ",
			HTMLTemplate:    &html,
		},
	}
	svc := NewService(repo, zap.NewNop())

	rendered, err := svc.RenderTemplate(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"build_complete",
		"email",
		"en",
		map[string]interface{}{"name": "Alice"},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rendered.Subject != "Welcome Alice" {
		t.Fatalf("unexpected subject: %q", rendered.Subject)
	}
	if rendered.Body != "Body for Alice" {
		t.Fatalf("unexpected body: %q", rendered.Body)
	}
	if rendered.HTML == nil || !strings.Contains(*rendered.HTML, "Hello Alice") {
		t.Fatalf("unexpected html rendering: %v", rendered.HTML)
	}
}

func TestRenderTemplateRepoError(t *testing.T) {
	svc := NewService(&mockTemplateRepo{err: errors.New("db error")}, zap.NewNop())
	if _, err := svc.RenderTemplate(context.Background(), uuid.New(), uuid.New(), "n", "c", "en", nil); err == nil {
		t.Fatal("expected error when repository fails")
	}
}

func TestRenderTemplateMissingTemplate(t *testing.T) {
	svc := NewService(&mockTemplateRepo{template: nil}, zap.NewNop())
	if _, err := svc.RenderTemplate(context.Background(), uuid.New(), uuid.New(), "n", "c", "en", nil); err == nil {
		t.Fatal("expected error when template is not found")
	}
}

func TestRenderTemplateInvalidTemplateSyntax(t *testing.T) {
	repo := &mockTemplateRepo{
		template: &Template{
			SubjectTemplate: "{{",
			BodyTemplate:    "ok",
		},
	}
	svc := NewService(repo, zap.NewNop())
	if _, err := svc.RenderTemplate(context.Background(), uuid.New(), uuid.New(), "n", "c", "en", nil); err == nil {
		t.Fatal("expected parse error for invalid template")
	}
}

func TestPrepareTemplateDataDefaults(t *testing.T) {
	svc := NewService(&mockTemplateRepo{}, zap.NewNop())
	data := svc.prepareTemplateData(nil)
	if data["timestamp"] == nil || data["date"] == nil || data["time"] == nil {
		t.Fatalf("expected default timestamp/date/time fields, got %+v", data)
	}
}
