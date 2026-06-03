// schema.go loads a JSON Schema and validates payloads against it.
package main

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Schema wraps a compiled JSON Schema.
type Schema struct {
	compiled *jsonschema.Schema
}

// LoadSchema compiles a JSON Schema from a file path.
func LoadSchema(path string) (*Schema, error) {
	compiled, err := jsonschema.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", path, err)
	}
	return &Schema{compiled: compiled}, nil
}

// Validate checks the decoded JSON value against the schema.
// Returns nil on success. On failure, the returned error includes
// the path within the document and the validation rule that was
// violated, as produced by the underlying jsonschema library.
func (s *Schema) Validate(value any) error {
	if err := s.compiled.Validate(value); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
