package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a tiny helper that creates a file with content under
// t.TempDir() and returns its absolute path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestLoadConfig_Valid covers the happy path. If config loading
// itself is broken, the tool does not start.
func TestLoadConfig_Valid(t *testing.T) {
	yamlPath := writeFile(t, "config.yaml", `
listen: ":8080"
schema_file: /tmp/schema.json
template_file: /tmp/template.tmpl
endpoints:
  - path: /alerts/ops
    smtp:
      server: localhost:25
      from: alerts@example.com
      to: oncall-ops@example.com
  - path: /alerts/dev
    smtp:
      server: localhost:25
      from: alerts@example.com
      to: oncall-dev@example.com
`)
	cfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Endpoints) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0].To != "oncall-ops@example.com" {
		t.Errorf("endpoint[0] To: got %q", cfg.Endpoints[0].To)
	}
	if cfg.Endpoints[1].To != "oncall-dev@example.com" {
		t.Errorf("endpoint[1] To: got %q", cfg.Endpoints[1].To)
	}
}

// TestLoadConfig_DuplicatePath protects against a runtime panic.
// http.ServeMux panics on duplicate path registration, so duplicate
// endpoint paths must be rejected at config load time or the server
// cannot start.
func TestLoadConfig_DuplicatePath(t *testing.T) {
	path := writeFile(t, "config.yaml", `
listen: ":8080"
schema_file: /tmp/schema.json
template_file: /tmp/template.tmpl
endpoints:
  - path: /x
    smtp:
      server: localhost:25
      from: a@example.com
      to: b@example.com
  - path: /x
    smtp:
      server: localhost:25
      from: a@example.com
      to: c@example.com
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for duplicate path, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate path") {
		t.Errorf("error %q does not mention duplicate path", err.Error())
	}
}

// TestLoadConfig_ReservedPath protects health endpoints from being
// shadowed. If a user configures an endpoint at /-/healthy, the
// liveness probe stops working — the tool is technically running
// but operationally broken.
func TestLoadConfig_ReservedPath(t *testing.T) {
	path := writeFile(t, "config.yaml", `
listen: ":8080"
schema_file: /tmp/schema.json
template_file: /tmp/template.tmpl
endpoints:
  - path: /-/healthy
    smtp:
      server: localhost:25
      from: a@example.com
      to: b@example.com
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for reserved path, got nil")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error %q does not mention reserved path", err.Error())
	}
}
