package structured

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	profileSchemaURL  = "https://hackathon.example/schemas/profile.schema.json"
	scenarioSchemaURL = "https://hackathon.example/schemas/scenario.schema.json"
)

//go:embed schemas/profile.schema.json schemas/scenario.schema.json
var schemas embed.FS

// Validator validates fixture-agent output against the frozen contract schemas.
// Compiled schemas are immutable and safe to reuse across requests.
type Validator struct {
	profile  *jsonschema.Schema
	scenario *jsonschema.Schema
}

var _ ports.StructuredValidator = (*Validator)(nil)

func NewValidator() (*Validator, error) {
	profile, err := compileSchema(profileSchemaURL, "schemas/profile.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile profile schema: %w", err)
	}
	scenario, err := compileSchema(scenarioSchemaURL, "schemas/scenario.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile scenario schema: %w", err)
	}
	return &Validator{profile: profile, scenario: scenario}, nil
}

func (validator *Validator) ValidateProfile(raw []byte) error {
	return validate(validator.profile, raw)
}

func (validator *Validator) ValidateScenario(raw []byte) error {
	return validate(validator.scenario, raw)
}

func compileSchema(url, name string) (*jsonschema.Schema, error) {
	raw, err := schemas.ReadFile(name)
	if err != nil {
		return nil, err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(url, document); err != nil {
		return nil, err
	}
	return compiler.Compile(url)
}

func validate(schema *jsonschema.Schema, raw []byte) error {
	if schema == nil {
		return errors.New("validator is not initialized")
	}
	document, err := decodeSingleDocument(raw)
	if err != nil {
		return err
	}
	if err := schema.Validate(document); err != nil {
		return safeValidationError(err)
	}
	return nil
}

func decodeSingleDocument(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			return nil, fmt.Errorf("invalid structured JSON at byte %d", syntaxError.Offset)
		}
		return nil, errors.New("invalid structured JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid structured JSON: trailing content")
	}
	return document, nil
}

func safeValidationError(err error) error {
	var validationError *jsonschema.ValidationError
	if !errors.As(err, &validationError) {
		return errors.New("structured output validation failed")
	}
	return fmt.Errorf("structured output validation failed at %s", validationPath(validationError))
}

func validationPath(err *jsonschema.ValidationError) string {
	if len(err.InstanceLocation) > 0 {
		return "/" + strings.Join(err.InstanceLocation, "/")
	}
	for _, cause := range err.Causes {
		if path := validationPath(cause); path != "/" {
			return path
		}
	}
	return "/"
}
