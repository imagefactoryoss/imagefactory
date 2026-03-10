package kubernetes

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// GoTemplateEngine implements TemplateEngine using Go templates
type GoTemplateEngine struct{}

// NewGoTemplateEngine creates a new template engine
func NewGoTemplateEngine() *GoTemplateEngine {
	return &GoTemplateEngine{}
}

// Render renders a template with the given data
func (e *GoTemplateEngine) Render(templateStr string, data interface{}) (string, error) {
	normalized := strings.ReplaceAll(templateStr, `\"`, `"`)
	tmpl, err := template.New("pipeline").Funcs(sprig.TxtFuncMap()).Parse(normalized)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
