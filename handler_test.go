package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSender is an in-memory Sender that records what the handler
// would have submitted, and can be configured to return an error.
// Tests that verify SMTP wire format belong in smtp_test.go; this
// fake exists to isolate handler flow.
type fakeSender struct {
	calls []sentCall
	err   error
}

type sentCall struct {
	server, from, to string
	rendered         []byte
}

func (f *fakeSender) Send(_ context.Context, server, from, to string, rendered []byte) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, sentCall{server, from, to, rendered})
	return nil
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func minimalFixtures(t *testing.T) (*Schema, *Template) {
	t.Helper()
	dir := t.TempDir()

	schemaPath := filepath.Join(dir, "schema.json")
	schemaBody := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["status"],
  "properties": {
    "status": { "type": "string", "enum": ["firing", "resolved"] }
  }
}`
	if err := os.WriteFile(schemaPath, []byte(schemaBody), 0o600); err != nil {
		t.Fatal(err)
	}
	schema, err := LoadSchema(schemaPath)
	if err != nil {
		t.Fatal(err)
	}

	tmplPath := filepath.Join(dir, "template.tmpl")
	tmplBody := "Subject: [{{.status}}]\n\nbody for {{.status}}\n"
	if err := os.WriteFile(tmplPath, []byte(tmplBody), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadTemplate(tmplPath)
	if err != nil {
		t.Fatal(err)
	}
	return schema, tmpl
}

// TestHandler_HappyPath verifies the validate -> render -> send
// flow: a valid payload reaches the sender with the rendered
// template as its body and the endpoint's server/from/to as its
// destination. SMTP wire format is verified separately in
// smtp_test.go.
func TestHandler_HappyPath(t *testing.T) {
	sender := &fakeSender{}
	schema, tmpl := minimalFixtures(t)

	h := NewHandler(
		Endpoint{Path: "/x", Server: "smtp.example:25", From: "alerts@example.com", To: "oncall@example.com"},
		schema, tmpl, sender, silentLogger(),
	)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json",
		strings.NewReader(`{"status":"firing"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("HTTP %d: %s", resp.StatusCode, body)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("sender call count: got %d, want 1", len(sender.calls))
	}
	got := sender.calls[0]
	if got.server != "smtp.example:25" || got.from != "alerts@example.com" || got.to != "oncall@example.com" {
		t.Errorf("sender args wrong: %+v", got)
	}
	rendered := string(got.rendered)
	if !strings.Contains(rendered, "Subject: [firing]") {
		t.Errorf("rendered missing Subject:\n%s", rendered)
	}
	if !strings.Contains(rendered, "body for firing") {
		t.Errorf("rendered missing body:\n%s", rendered)
	}
}

// TestHandler_SchemaRejectedDoesNotSend guards the property that
// a payload failing schema validation does not reach the sender.
// If this regresses, malformed payloads from misconfigured senders
// flow through to mailboxes.
func TestHandler_SchemaRejectedDoesNotSend(t *testing.T) {
	sender := &fakeSender{}
	schema, tmpl := minimalFixtures(t)

	h := NewHandler(
		Endpoint{Path: "/x", Server: "smtp.example:25", From: "alerts@example.com", To: "oncall@example.com"},
		schema, tmpl, sender, silentLogger(),
	)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json",
		strings.NewReader(`{"status":"bogus"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("HTTP: got %d, want 400", resp.StatusCode)
	}
	if len(sender.calls) != 0 {
		t.Errorf("sender should not have been called, got %d calls", len(sender.calls))
	}
}

// TestHandler_SenderFailureReturns500 protects the retry-delegation
// contract documented in DESIGN.md: when the sender returns an
// error (the MTA rejected the submission), hook2email returns 5xx
// so the upstream sender (Alertmanager, alertchain, ...) knows to
// retry. If this regresses to e.g. returning 204 on sender failure,
// alerts are silently lost.
func TestHandler_SenderFailureReturns500(t *testing.T) {
	sender := &fakeSender{err: fmt.Errorf("mta rejected")}
	schema, tmpl := minimalFixtures(t)

	h := NewHandler(
		Endpoint{Path: "/x", Server: "smtp.example:25", From: "alerts@example.com", To: "oncall@example.com"},
		schema, tmpl, sender, silentLogger(),
	)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json",
		strings.NewReader(`{"status":"firing"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("HTTP: got %d, want 500", resp.StatusCode)
	}
}
