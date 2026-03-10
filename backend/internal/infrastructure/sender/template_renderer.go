package sender

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	textTemplate "text/template"
)

// TemplateRenderer defines the interface for rendering email templates
type TemplateRenderer interface {
	// RenderHTML renders an HTML template with the provided data
	RenderHTML(ctx context.Context, templateStr string, data map[string]interface{}) (string, error)

	// RenderText renders a plain text template with the provided data
	RenderText(ctx context.Context, templateStr string, data map[string]interface{}) (string, error)
}

// GoTemplateRenderer implements TemplateRenderer using Go's html/template and text/template
type GoTemplateRenderer struct{}

// NewGoTemplateRenderer creates a new GoTemplateRenderer
func NewGoTemplateRenderer() *GoTemplateRenderer {
	return &GoTemplateRenderer{}
}

// RenderHTML renders an HTML template with automatic HTML escaping
func (r *GoTemplateRenderer) RenderHTML(ctx context.Context, templateStr string, data map[string]interface{}) (string, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context error: %w", err)
	}

	// Parse template
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.String(), nil
}

// RenderText renders a plain text template without HTML escaping
func (r *GoTemplateRenderer) RenderText(ctx context.Context, templateStr string, data map[string]interface{}) (string, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context error: %w", err)
	}

	// Parse template
	tmpl, err := textTemplate.New("email").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse text template: %w", err)
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return buf.String(), nil
}
