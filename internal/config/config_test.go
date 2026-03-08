package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoad_FileNotExist(t *testing.T) {
	flat, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	require.NoError(t, err)
	require.Nil(t, flat, "missing file should return nil map")
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(p, []byte(`{"server":{"host":"localhost","port":8080}}`), 0644))

	flat, err := config.Load(p)
	require.NoError(t, err)
	require.Equal(t, "localhost", flat["server.host"])
	require.Equal(t, "8080", flat["server.port"])
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(p, []byte(`{invalid`), 0644))

	_, err := config.Load(p)
	require.Error(t, err)
}

func TestToJSON_TypeCoercion(t *testing.T) {
	flat := map[string]string{
		"server.host":    "localhost",
		"server.port":    "8080",
		"server.enabled": "true",
		"server.ratio":   "3.14",
	}
	types := map[string]string{
		"server.host":    "string",
		"server.port":    "int",
		"server.enabled": "bool",
		"server.ratio":   "float",
	}

	data, err := config.ToJSON(flat, types)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	server := parsed["server"].(map[string]any)
	require.Equal(t, "localhost", server["host"])
	require.Equal(t, float64(8080), server["port"])
	require.Equal(t, true, server["enabled"])
	require.Equal(t, 3.14, server["ratio"])
}

func TestToJSON_EmptyMap(t *testing.T) {
	data, err := config.ToJSON(map[string]string{}, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, "{}", string(data))
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")

	original := map[string]string{
		"db.host": "postgres.local",
		"db.port": "5432",
		"db.ssl":  "true",
	}
	types := map[string]string{
		"db.host": "string",
		"db.port": "int",
		"db.ssl":  "bool",
	}

	data, err := config.ToJSON(original, types)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, data, 0644))

	loaded, err := config.Load(p)
	require.NoError(t, err)
	require.Equal(t, original, loaded)
}

func TestValidate_Valid(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ host: string, port: int & >=1 & <=65535 }`)
	require.NoError(t, schema.Err())

	err := config.Validate([]byte(`{"host":"localhost","port":8080}`), schema)
	require.NoError(t, err)
}

func TestValidate_Invalid(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ host: string, port: int & >=1 & <=65535 }`)
	require.NoError(t, schema.Err())

	err := config.Validate([]byte(`{"host":"localhost","port":99999}`), schema)
	require.Error(t, err)
}

func TestValidate_InvalidJSON(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ host: string }`)
	_ = schema.LookupPath(cue.ParsePath(""))

	err := config.Validate([]byte(`{invalid`), schema)
	require.Error(t, err)
}
