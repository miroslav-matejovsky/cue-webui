package webui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

func applyStoredValues(formData webform.FormData, values map[string]string) webform.FormData {
	cloned := formData
	cloned.Sections = cloneSectionsWithValues(formData.Sections, values)
	return cloned
}

func cloneSectionsWithValues(sections []webform.Section, values map[string]string) []webform.Section {
	if len(sections) == 0 {
		return nil
	}

	cloned := make([]webform.Section, 0, len(sections))
	for _, section := range sections {
		sectionCopy := section
		if len(section.Fields) > 0 {
			sectionCopy.Fields = make([]webform.Field, len(section.Fields))
			for index, field := range section.Fields {
				fieldCopy := field
				if storedValue, ok := values[field.Path]; ok {
					fieldCopy.Default = storedValue
				}
				if len(field.Options) > 0 {
					fieldCopy.Options = append([]string(nil), field.Options...)
				}
				sectionCopy.Fields[index] = fieldCopy
			}
		}
		sectionCopy.Sections = cloneSectionsWithValues(section.Sections, values)
		cloned = append(cloned, sectionCopy)
	}

	return cloned
}

func mergeSubmittedValues(formData webform.FormData, existing map[string]string, submitted url.Values) map[string]string {
	merged := cloneValueMap(existing)
	for key, values := range submitted {
		merged[key] = strings.Join(values, ", ")
	}

	visitFields(formData.Sections, func(field webform.Field) {
		values, ok := submitted[field.Path]
		if ok {
			merged[field.Path] = strings.Join(values, ", ")
			return
		}

		if field.Hidden || fieldIsDisabled(field) {
			return
		}

		if field.Widget == "checkbox" {
			merged[field.Path] = "false"
			return
		}

		delete(merged, field.Path)
	})
	return merged
}

func visitFields(sections []webform.Section, visit func(webform.Field)) {
	for _, section := range sections {
		for _, field := range section.Fields {
			visit(field)
		}
		visitFields(section.Sections, visit)
	}
}

func fieldIsDisabled(field webform.Field) bool {
	if !field.Readonly {
		return false
	}

	switch field.Widget {
	case "checkbox", "radio", "select":
		return true
	default:
		return false
	}
}

func cloneValueMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// collectFieldTypes builds a map from field path to CUE kind string (e.g. "int", "bool")
// by walking all sections recursively.
func collectFieldTypes(formData webform.FormData) map[string]string {
	types := map[string]string{}
	visitFields(formData.Sections, func(f webform.Field) {
		types[f.Path] = f.Type
	})
	return types
}

// flatMapToNestedJSON converts a flat dot-separated key-value map into nested JSON bytes.
// Field types from formData are used to coerce string values to the correct JSON types
// (int, float, bool). Unknown fields or fields with empty values are written as strings.
func flatMapToNestedJSON(flat map[string]string, formData webform.FormData) ([]byte, error) {
	types := collectFieldTypes(formData)
	root := map[string]any{}

	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := flat[key]
		parts := strings.Split(key, ".")
		current := root
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = coerceValue(val, types[key])
			} else {
				next, ok := current[part]
				if !ok {
					m := map[string]any{}
					current[part] = m
					current = m
				} else if m, ok := next.(map[string]any); ok {
					current = m
				} else {
					return nil, fmt.Errorf("conflict at key %q: expected object, got value", strings.Join(parts[:i+1], "."))
				}
			}
		}
	}

	return json.MarshalIndent(root, "", "  ")
}

func coerceValue(val string, cueType string) any {
	switch cueType {
	case "int":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	case "float", "number":
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	case "bool":
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return val
}

// nestedJSONToFlatMap parses nested JSON bytes into a flat dot-separated key-value map.
// Nested objects are flattened with dots: {"a":{"b":"c"}} → {"a.b":"c"}.
func nestedJSONToFlatMap(data []byte) (map[string]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	flat := map[string]string{}
	flattenMap("", raw, flat)
	return flat, nil
}

func flattenMap(prefix string, m map[string]any, out map[string]string) {
	for key, val := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := val.(type) {
		case map[string]any:
			flattenMap(fullKey, v, out)
		case bool:
			out[fullKey] = strconv.FormatBool(v)
		case float64:
			if v == float64(int64(v)) {
				out[fullKey] = strconv.FormatInt(int64(v), 10)
			} else {
				out[fullKey] = strconv.FormatFloat(v, 'f', -1, 64)
			}
		case string:
			out[fullKey] = v
		default:
			out[fullKey] = fmt.Sprintf("%v", v)
		}
	}
}

// validateJSONWithCUE validates JSON bytes against a compiled CUE schema.
// It compiles the JSON into a CUE value, unifies it with the schema, and
// returns an error if validation fails.
func validateJSONWithCUE(jsonBytes []byte, schema cue.Value) error {
	ctx := schema.Context()
	jsonVal := ctx.CompileBytes(jsonBytes)
	if jsonVal.Err() != nil {
		return fmt.Errorf("compiling JSON as CUE: %w", jsonVal.Err())
	}
	unified := schema.Unify(jsonVal)
	if err := unified.Validate(); err != nil {
		return fmt.Errorf("CUE validation failed: %w", err)
	}
	return nil
}
