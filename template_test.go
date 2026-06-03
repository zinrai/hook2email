package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplate_JsonFunc protects the only custom function we add
// to text/template. The json function exists so operators can embed
// JSON-derived strings without breaking quoting in any structured
// header value they choose to render.
func TestTemplate_JsonFunc(t *testing.T) {
	tmplPath := filepath.Join(t.TempDir(), "template.tmpl")
	body := `X-Raw: {{ .msg | json }}`
	if err := os.WriteFile(tmplPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadTemplate(tmplPath)
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	rendered, err := tmpl.Render(map[string]any{
		"msg": `line one with "quote" and = sign`,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// The rendered fragment after "X-Raw: " must be a valid JSON string.
	got := strings.TrimPrefix(string(rendered), "X-Raw: ")
	var probe string
	if err := json.Unmarshal([]byte(got), &probe); err != nil {
		t.Fatalf("rendered fragment is not valid JSON: %v\noutput: %s", err, got)
	}
	if probe != `line one with "quote" and = sign` {
		t.Errorf("string roundtrip failed: got %q", probe)
	}
}

// TestTemplate_Example renders the bundled example template against
// the bundled example payload. The example is documentation — if it
// stops rendering, users following the README are misled.
func TestTemplate_Example(t *testing.T) {
	tmpl, err := LoadTemplate(filepath.Join("examples", "template.tmpl"))
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}

	payloadBytes, err := os.ReadFile(filepath.Join("examples", "payload.json"))
	if err != nil {
		t.Fatalf("read payload.json: %v", err)
	}
	var payload any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("parse payload.json: %v", err)
	}

	rendered, err := tmpl.Render(payload)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// payload.json has level=warning and title="Something happened",
	// so the rendered Subject line should contain both. If this
	// assertion fails, either the example payload, the example
	// template, or both have drifted.
	out := string(rendered)
	if !strings.Contains(out, "Subject: [warning] Something happened") {
		t.Errorf("Subject line missing or wrong:\n%s", out)
	}
}
