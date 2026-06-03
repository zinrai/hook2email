// hook2email is a webhook-to-email relay. See README.md.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		usage()
		return fmt.Errorf("no subcommand given")
	}
	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "serve":
		return cmdServe(args)
	case "check":
		return cmdCheck(args)
	case "render":
		return cmdRender(args)
	case "version":
		printVersion()
		return nil
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", sub)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: hook2email <subcommand> [flags]

Subcommands:
  serve    run the HTTP server
  check    validate configuration, schema, and template without starting the server
  render   render a payload through a schema+template and print the RFC 5322 message it would submit
  version  print build version

Run "hook2email <subcommand> -h" for subcommand-specific flags.
`)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func cmdServe(args []string) error {
	fs := newFlagSet("serve")
	var configPath string
	fs.StringVar(&configPath, "config", "hook2email.yaml", "path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	schema, err := LoadSchema(cfg.SchemaFile)
	if err != nil {
		return fmt.Errorf("load schema: %w", err)
	}

	tmpl, err := LoadTemplate(cfg.TemplateFile)
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}

	smtpClient := NewSMTPClient()

	mux := newServeMux(cfg, schema, tmpl, smtpClient, logger)

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("hook2email serving", "listen", cfg.Listen)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// cmdCheck loads config, schema, and template without starting the
// server. Intended for CI use before deployment.
func cmdCheck(args []string) error {
	fs := newFlagSet("check")
	var configPath string
	fs.StringVar(&configPath, "config", "hook2email.yaml", "path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if _, err := LoadSchema(cfg.SchemaFile); err != nil {
		return fmt.Errorf("load schema: %w", err)
	}
	if _, err := LoadTemplate(cfg.TemplateFile); err != nil {
		return fmt.Errorf("load template: %w", err)
	}
	fmt.Fprintln(os.Stdout, "ok")
	return nil
}

// cmdRender runs the schema-validation + template-rendering pipeline
// against a sample payload and prints the RFC 5322 message body that
// would be submitted to the MTA. Intended for iterating on a new
// schema+template pair before deploying it. No SMTP request is made.
// Envelope headers (From, To, Date, Message-ID) are not added because
// they require endpoint config and a real send.
func cmdRender(args []string) error {
	fs := newFlagSet("render")
	var schemaPath, templatePath, payloadPath string
	fs.StringVar(&schemaPath, "schema", "", "path to the JSON Schema file")
	fs.StringVar(&templatePath, "template", "", "path to the Go text/template file")
	fs.StringVar(&payloadPath, "payload", "", "path to a sample JSON payload file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if schemaPath == "" || templatePath == "" || payloadPath == "" {
		fs.Usage()
		return fmt.Errorf("--schema, --template, and --payload are all required")
	}

	schema, err := LoadSchema(schemaPath)
	if err != nil {
		return fmt.Errorf("load schema: %w", err)
	}
	tmpl, err := LoadTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}

	payloadBytes, err := os.ReadFile(payloadPath)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", payloadPath, err)
	}
	var payload any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("parse payload %s: %w", payloadPath, err)
	}

	if err := schema.Validate(payload); err != nil {
		return err
	}

	rendered, err := tmpl.Render(payload)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, string(rendered))
	if len(rendered) == 0 || rendered[len(rendered)-1] != '\n' {
		fmt.Fprintln(os.Stdout)
	}
	return nil
}
