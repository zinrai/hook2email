package main

import (
	"strings"
	"testing"
)

// TestBuildMessage_EnvelopeHeaders verifies that buildMessage
// prepends the envelope headers hook2email owns, in the right
// order, before the template output. If this regresses,
// recipients see broken or missing headers.
func TestBuildMessage_EnvelopeHeaders(t *testing.T) {
	rendered := []byte("Subject: hello\r\n\r\nbody\r\n")
	msg := string(buildMessage("alerts@example.com", "oncall@example.com", rendered))

	for _, want := range []string{
		"From: alerts@example.com\r\n",
		"To: oncall@example.com\r\n",
		"Date: ",
		"Message-ID: <",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"Subject: hello\r\n",
		"\r\nbody\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}

	// Envelope headers must precede the template output.
	envelopeEnd := strings.Index(msg, "Subject: hello")
	if envelopeEnd <= 0 {
		t.Fatalf("Subject not found in:\n%s", msg)
	}
	envelope := msg[:envelopeEnd]
	if !strings.Contains(envelope, "From: ") || !strings.Contains(envelope, "Content-Type: ") {
		t.Errorf("envelope headers not before template output:\n%s", envelope)
	}
}

// TestBuildMessage_LFInputBecomesCRLF verifies that a template
// authored with bare LF line endings (the common case for
// hand-edited template files) is normalised to CRLF before
// submission. If this regresses, strict MTAs reject the message.
func TestBuildMessage_LFInputBecomesCRLF(t *testing.T) {
	rendered := []byte("Subject: hello\n\nfirst line\nsecond line\n")
	msg := string(buildMessage("a@example.com", "b@example.com", rendered))

	if strings.Contains(msg, "\nSubject") && !strings.Contains(msg, "\r\nSubject") {
		t.Errorf("bare LF before Subject not normalised:\n%s", msg)
	}
	if !strings.Contains(msg, "first line\r\nsecond line\r\n") {
		t.Errorf("body LF not normalised to CRLF:\n%s", msg)
	}
	// No bare LF (LF not preceded by CR) should remain.
	for i := 0; i < len(msg); i++ {
		if msg[i] == '\n' && (i == 0 || msg[i-1] != '\r') {
			t.Errorf("bare LF at offset %d in:\n%s", i, msg)
			break
		}
	}
}

// TestBuildMessage_PreNormalisedCRLFUnchanged verifies the
// idempotence of CRLF normalisation: input already in CRLF must
// not become CRCRLF. If this regresses, MTAs see malformed line
// endings.
func TestBuildMessage_PreNormalisedCRLFUnchanged(t *testing.T) {
	rendered := []byte("Subject: hello\r\n\r\nbody\r\n")
	msg := string(buildMessage("a@example.com", "b@example.com", rendered))

	if strings.Contains(msg, "\r\r\n") {
		t.Errorf("CRLF doubled in:\n%s", msg)
	}
}
