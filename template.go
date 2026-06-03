// template.go loads and renders Go text/templates. The only custom
// function exposed to templates is `json`, which encodes its argument
// as a JSON value. Useful for embedding JSON-derived strings into
// header values without breaking RFC 5322 framing when combined with
// other escaping the operator applies.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
)

var templateFuncs = template.FuncMap{
	"json": jsonEncode,
}

// jsonEncode encodes v as a JSON value. It is exposed to templates
// as the `json` pipeline function: `{{ .x | json }}` produces a
// valid JSON encoding of x (string, number, object, array, ...).
func jsonEncode(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Template wraps a parsed text/template.
type Template struct {
	parsed *template.Template
}

// LoadTemplate parses a text/template file.
func LoadTemplate(path string) (*Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}
	t, err := template.New("hook2email").Funcs(templateFuncs).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}
	return &Template{parsed: t}, nil
}

// Render executes the template against value (the validated
// incoming JSON, typically a map[string]any) and returns the
// rendered bytes — the DATA section from Subject: onward.
func (t *Template) Render(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.parsed.Execute(&buf, value); err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}
	return buf.Bytes(), nil
}
