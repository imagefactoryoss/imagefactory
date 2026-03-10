package kubernetes

import "testing"

func TestGoTemplateEngine_RenderHandlesEscapedQuotes(t *testing.T) {
	engine := NewGoTemplateEngine()
	template := `value: "{{ default \"fallback\" .Value }}"`

	out, err := engine.Render(template, map[string]interface{}{"Value": ""})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	expected := `value: "fallback"`
	if out != expected {
		t.Fatalf("unexpected render output: got %q want %q", out, expected)
	}
}
