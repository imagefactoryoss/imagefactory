package sender

import (
	"context"
	"strings"
	"testing"
)

func TestGoTemplateRenderer_RenderHTML(t *testing.T) {
	r := NewGoTemplateRenderer()
	out, err := r.RenderHTML(context.Background(), "<p>{{.name}}</p>", map[string]interface{}{
		"name": "<b>Alice</b>",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(out, "&lt;b&gt;Alice&lt;/b&gt;") {
		t.Fatalf("expected escaped HTML output, got %q", out)
	}
}

func TestGoTemplateRenderer_RenderText(t *testing.T) {
	r := NewGoTemplateRenderer()
	out, err := r.RenderText(context.Background(), "Hello {{.name}}", map[string]interface{}{
		"name": "<b>Alice</b>",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out != "Hello <b>Alice</b>" {
		t.Fatalf("unexpected text output: %q", out)
	}
}

func TestGoTemplateRenderer_ContextCanceled(t *testing.T) {
	r := NewGoTemplateRenderer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := r.RenderHTML(ctx, "x", nil); err == nil {
		t.Fatal("expected context error for RenderHTML")
	}
	if _, err := r.RenderText(ctx, "x", nil); err == nil {
		t.Fatal("expected context error for RenderText")
	}
}

func TestGoTemplateRenderer_TemplateParseErrors(t *testing.T) {
	r := NewGoTemplateRenderer()

	if _, err := r.RenderHTML(context.Background(), "{{", nil); err == nil {
		t.Fatal("expected parse error for invalid HTML template")
	}
	if _, err := r.RenderText(context.Background(), "{{", nil); err == nil {
		t.Fatal("expected parse error for invalid text template")
	}
}
