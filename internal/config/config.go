package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
)

// Load reads the JSON config file at path and returns its contents as a flat
// dot-separated key-value map (e.g. {"server.host": "localhost"}).
// Returns nil, nil when the file does not exist so callers can distinguish
// "no config yet" from a parse error.
func Load(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	flat, err := jsonToFlat(data)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return flat, nil
}

// ToJSON converts a flat dot-separated key-value map to indented nested JSON bytes.
// fieldTypes maps field paths (e.g. "server.port") to their CUE type names
// ("int", "float", "bool", "string"). Fields not present in fieldTypes are written
// as JSON strings.
func ToJSON(flat map[string]string, fieldTypes map[string]string) ([]byte, error) {
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
				current[part] = coerceValue(val, fieldTypes[key])
			} else {
				next, ok := current[part]
				if !ok {
					m := map[string]any{}
					current[part] = m
					current = m
				} else if m, ok := next.(map[string]any); ok {
					current = m
				} else {
					return nil, fmt.Errorf("key conflict at %q: parent is a scalar, not an object", strings.Join(parts[:i+1], "."))
				}
			}
		}
	}

	return json.MarshalIndent(root, "", "  ")
}

// Validate checks that jsonBytes conform to the provided CUE schema value.
// Returns nil if the JSON is valid according to the schema, or a descriptive
// error if validation fails.
func Validate(jsonBytes []byte, schema cue.Value) error {
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

// jsonToFlat parses raw JSON bytes and flattens the nested object into a
// dot-separated key-value map.
func jsonToFlat(data []byte) (map[string]string, error) {
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
