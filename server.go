// server.go assembles the HTTP routes that hook2email serves.
//
// Construction is separated from cmdServe so that tests can build
// a server with the same routing as production without going
// through the CLI entry point.
package main

import (
	"log/slog"
	"net/http"
)

// newServeMux returns the ServeMux that production and tests both
// use. One handler is registered per configured endpoint, plus
// fixed health routes.
func newServeMux(cfg *Config, schema *Schema, tmpl *Template,
	sender Sender, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	for _, ep := range cfg.Endpoints {
		h := NewHandler(ep, schema, tmpl, sender, logger)
		mux.Handle(ep.Path, h)
	}

	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
