package main

import "fmt"

// Build-time variables injected via goreleaser ldflags.
// See .goreleaser.yaml.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	fmt.Printf("hook2email %s (commit %s, built %s)\n", version, commit, date)
}
