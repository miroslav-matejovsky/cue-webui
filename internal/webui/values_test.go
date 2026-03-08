package webui

import (
	"encoding/json"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
	"github.com/stretchr/testify/require"
)

func TestFlatMapToNestedJSON(t *testing.T) {
	fd := webform.FormData{
		Sections: []webform.Section{{
			Fields: []webform.Field{
				{Path: "server.host", Type: "string"},
				{Path: "server.port", Type: "int"},
				{Path: "server.enabled", Type: "bool"},
				{Path: "server.ratio", Type: "float"},
			},
		}},
	}
	flat := map[string]string{
		"server.host":    "localhost",
		"server.port":    "8080",
		"server.enabled": "true",
		"server.ratio":   "3.14",
	}

	data, err := flatMapToNestedJSON(flat, fd)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	server := parsed["server"].(map[string]any)
	require.Equal(t, "localhost", server["host"])
	require.Equal(t, float64(8080), server["port"])
	require.Equal(t, true, server["enabled"])
	require.Equal(t, 3.14, server["ratio"])
}

func TestFlatMapToNestedJSON_EmptyMap(t *testing.T) {
	fd := webform.FormData{}
	data, err := flatMapToNestedJSON(map[string]string{}, fd)
	require.NoError(t, err)
	require.Equal(t, "{}", string(data))
}

func TestNestedJSONToFlatMap(t *testing.T) {
	input := `{"server":{"host":"localhost","port":8080,"enabled":true,"ratio":3.14}}`
	flat, err := nestedJSONToFlatMap([]byte(input))
	require.NoError(t, err)

	require.Equal(t, "localhost", flat["server.host"])
	require.Equal(t, "8080", flat["server.port"])
	require.Equal(t, "true", flat["server.enabled"])
	require.Equal(t, "3.14", flat["server.ratio"])
}

func TestNestedJSONToFlatMap_InvalidJSON(t *testing.T) {
	_, err := nestedJSONToFlatMap([]byte(`{invalid`))
	require.Error(t, err)
}

func TestRoundTrip_FlatMap_JSON_FlatMap(t *testing.T) {
	fd := webform.FormData{
		Sections: []webform.Section{{
			Fields: []webform.Field{
				{Path: "db.host", Type: "string"},
				{Path: "db.port", Type: "int"},
				{Path: "db.ssl", Type: "bool"},
			},
		}},
	}
	original := map[string]string{
		"db.host": "postgres.local",
		"db.port": "5432",
		"db.ssl":  "true",
	}

	jsonBytes, err := flatMapToNestedJSON(original, fd)
	require.NoError(t, err)

	roundTripped, err := nestedJSONToFlatMap(jsonBytes)
	require.NoError(t, err)

	require.Equal(t, original, roundTripped)
}

func TestValidateJSONWithCUE_Valid(t *testing.T) {
	schema := `#Config: { host: string, port: int & >=1 & <=65535 }`
	ctx := cuecontext.New()
	val := ctx.CompileString(schema).LookupPath(cue.ParsePath("#Config"))
	require.NoError(t, val.Err())

	jsonBytes := []byte(`{"host": "localhost", "port": 8080}`)
	err := validateJSONWithCUE(jsonBytes, val)
	require.NoError(t, err)
}

func TestValidateJSONWithCUE_Invalid(t *testing.T) {
	schema := `#Config: { host: string, port: int & >=1 & <=65535 }`
	ctx := cuecontext.New()
	val := ctx.CompileString(schema).LookupPath(cue.ParsePath("#Config"))
	require.NoError(t, val.Err())

	jsonBytes := []byte(`{"host": "localhost", "port": 99999}`)
	err := validateJSONWithCUE(jsonBytes, val)
	require.Error(t, err)
}

func TestValidateJSONWithCUE_InvalidJSON(t *testing.T) {
	schema := `#Config: { host: string }`
	ctx := cuecontext.New()
	val := ctx.CompileString(schema).LookupPath(cue.ParsePath("#Config"))
	require.NoError(t, val.Err())

	err := validateJSONWithCUE([]byte(`{invalid`), val)
	require.Error(t, err)
}
