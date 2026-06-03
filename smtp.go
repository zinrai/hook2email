// smtp.go submits a rendered message to a local MTA over SMTP.
//
// The MVP transport is plaintext on port 25 with no AUTH. TLS,
// authentication, queueing, retry, and bounce handling are the
// local MTA's responsibility. See DESIGN.md.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SMTPClient submits messages to an SMTP server.
type SMTPClient struct {
	dialer *net.Dialer
}

// NewSMTPClient returns an SMTPClient. The dial deadline bounds
// the time spent reaching the MTA; total session deadline comes
// from the context passed to Send.
func NewSMTPClient() *SMTPClient {
	return &SMTPClient{
		dialer: &net.Dialer{Timeout: 5 * time.Second},
	}
}

// Send submits one rendered message to server. The rendered
// argument is the DATA section from Subject: onward, as produced
// by Template.Render. Send prepends the envelope headers (From,
// To, Date, Message-ID, MIME-Version, Content-Type) and
// normalises line endings to CRLF before submission.
func (c *SMTPClient) Send(ctx context.Context, server, from, to string, rendered []byte) error {
	conn, err := c.dialer.DialContext(ctx, "tcp", server)
	if err != nil {
		return fmt.Errorf("dial %s: %w", server, err)
	}

	host, _, err := net.SplitHostPort(server)
	if err != nil {
		conn.Close()
		return fmt.Errorf("parse server %q: %w", server, err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(buildMessage(from, to, rendered)); err != nil {
		return fmt.Errorf("write DATA: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close DATA: %w", err)
	}

	return client.Quit()
}

// buildMessage assembles the RFC 5322 message: the envelope
// headers owned by hook2email, followed by the template output
// (Subject + any operator-chosen headers, blank line, body),
// all with CRLF line endings.
func buildMessage(from, to string, rendered []byte) []byte {
	var sb strings.Builder
	sb.WriteString("From: ")
	sb.WriteString(from)
	sb.WriteString("\r\n")
	sb.WriteString("To: ")
	sb.WriteString(to)
	sb.WriteString("\r\n")
	sb.WriteString("Date: ")
	sb.WriteString(time.Now().Format(time.RFC1123Z))
	sb.WriteString("\r\n")
	sb.WriteString("Message-ID: ")
	sb.WriteString(generateMessageID())
	sb.WriteString("\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString(toCRLF(string(rendered)))
	return []byte(sb.String())
}

func toCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func generateMessageID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("<%s.%s@%s>", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(b), hostname)
}
