package agenix

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	schemaManifest    = "manifest.schema.json"
	schemaTrace       = "trace.schema.json"
	schemaCheckReport = "check-report.schema.json"
)

type jsonSchema struct {
	Type       string                `json:"type,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Properties map[string]jsonSchema `json:"properties,omitempty"`
	Items      *jsonSchema           `json:"items,omitempty"`
	Enum       []string              `json:"enum,omitempty"`
	MinItems   int                   `json:"minItems,omitempty"`
}

func ValidateManifestDocument(manifest Manifest) error {
	doc, err := structToDocument(manifest)
	if err != nil {
		return err
	}
	return ValidateSchemaDocument(schemaManifest, doc)
}

func ValidateTraceDocument(trace Trace) error {
	doc, err := structToDocument(trace)
	if err != nil {
		return err
	}
	return ValidateSchemaDocument(schemaTrace, doc)
}

func ValidateSchemaDocument(schemaName string, doc map[string]any) error {
	schema, err := loadSchema(schemaName)
	if err != nil {
		return err
	}
	return validateSchemaValue("$", doc, schema)
}

func ValidateTarget(path string) (string, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", WrapError(ErrNotFound, "read validation target", err)
	}
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err == nil && looksLikeCheckReport(doc) {
			if err := ValidateSchemaDocument(schemaCheckReport, doc); err != nil {
				return "", "", err
			}
			schemaPath, err := SchemaPath(schemaCheckReport)
			if err != nil {
				return "", "", err
			}
			return "check_report", schemaPath, nil
		}
		trace, err := ReadTrace(path)
		if err != nil {
			return "", "", err
		}
		if err := ValidateTraceDocument(*trace); err != nil {
			return "", "", err
		}
		schemaPath, err := SchemaPath(schemaTrace)
		if err != nil {
			return "", "", err
		}
		return "trace", schemaPath, nil
	}
	manifest, err := LoadManifest(path)
	if err != nil {
		return "", "", err
	}
	if err := ValidateManifestDocument(manifest); err != nil {
		return "", "", err
	}
	schemaPath, err := SchemaPath(schemaManifest)
	if err != nil {
		return "", "", err
	}
	return "manifest", schemaPath, nil
}

func looksLikeCheckReport(doc map[string]any) bool {
	for _, key := range []string{"artifact_path", "changed_files", "verifier_summary", "event_count"} {
		if _, ok := doc[key]; ok {
			return true
		}
	}
	return false
}

func SchemaPath(schemaName string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", NewError(ErrDriverError, "resolve schema path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "specs", schemaName)), nil
}

func loadSchema(schemaName string) (jsonSchema, error) {
	path, err := SchemaPath(schemaName)
	if err != nil {
		return jsonSchema{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return jsonSchema{}, WrapError(ErrNotFound, "read schema", err)
	}
	var schema jsonSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return jsonSchema{}, WrapError(ErrInvalidInput, "decode schema", err)
	}
	return schema, nil
}

func structToDocument(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, WrapError(ErrDriverError, "encode schema document", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, WrapError(ErrDriverError, "decode schema document", err)
	}
	return doc, nil
}

func validateSchemaValue(path string, value any, schema jsonSchema) error {
	switch schema.Type {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return schemaTypeError(path, "object")
		}
		for _, required := range schema.Required {
			if _, ok := obj[required]; !ok {
				return NewError(ErrInvalidInput, fmt.Sprintf("schema validation failed: %s missing required field", schemaPath(path, required)))
			}
		}
		for key, propertySchema := range schema.Properties {
			child, ok := obj[key]
			if !ok {
				continue
			}
			if child == nil {
				continue
			}
			if err := validateSchemaValue(schemaPath(path, key), child, propertySchema); err != nil {
				return err
			}
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return schemaTypeError(path, "array")
		}
		if schema.MinItems > 0 && len(items) < schema.MinItems {
			return NewError(ErrInvalidInput, fmt.Sprintf("schema validation failed: %s requires at least %d items", path, schema.MinItems))
		}
		if schema.Items != nil {
			for i, item := range items {
				if err := validateSchemaValue(fmt.Sprintf("%s[%d]", path, i), item, *schema.Items); err != nil {
					return err
				}
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return schemaTypeError(path, "string")
		}
		if len(schema.Enum) > 0 && !containsString(schema.Enum, text) {
			return NewError(ErrInvalidInput, fmt.Sprintf("schema validation failed: %s must be one of %v", path, schema.Enum))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return schemaTypeError(path, "boolean")
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || math.Trunc(number) != number {
			return schemaTypeError(path, "integer")
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return schemaTypeError(path, "number")
		}
	}
	return nil
}

func schemaTypeError(path, want string) error {
	return NewError(ErrInvalidInput, fmt.Sprintf("schema validation failed: %s must be %s", path, want))
}

func schemaPath(base, key string) string {
	if base == "$" {
		return "$." + key
	}
	return base + "." + key
}
