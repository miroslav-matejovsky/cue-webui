package schema_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/schema"
	"github.com/stretchr/testify/require"
)

func TestRootValue_PlainStruct_ReturnsSelf(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{ host: string, port: int }`)
	got, err := schema.RootValue(v)
	require.NoError(t, err)
	require.Equal(t, v, got)
}

func TestRootValue_NoDefs_ReturnsSelf(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{...}`)
	got, err := schema.RootValue(v)
	require.NoError(t, err)
	require.Equal(t, v, got)
}

func TestRootValue_SingleRootDef_ReturnsRootValue(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString("#Connection: { host: string }\n#Config: { conn: #Connection }\n")
	got, err := schema.RootValue(v)
	require.NoError(t, err)
	require.Nil(t, got.Err())
	// The returned value should represent #Config (has a "conn" field).
	iter, err := got.Fields(cue.Optional(true))
	require.NoError(t, err)
	var fields []string
	for iter.Next() {
		fields = append(fields, iter.Label())
	}
	require.Contains(t, fields, "conn", "returned value should be #Config with a 'conn' field")
}

func TestRootValue_MultipleRootDefs_ReturnsError(t *testing.T) {
	ctx := cuecontext.New()
	// Two definitions both with struct sub-fields — no UI_Root → should return error.
	v := ctx.CompileString("#A: { sub: { x: int } }\n#B: { sub: { y: string } }\n")
	_, err := schema.RootValue(v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "UI_Root")
}

func TestRootValue_UIRootHint_ReturnsNamedDef(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString("#Connection: { database: { host: string } }\n// UI_Root: true\n#Config: { conn: #Connection }\n")
	got, err := schema.RootValue(v)
	require.NoError(t, err)
	require.Nil(t, got.Err())
	iter, err := got.Fields(cue.Optional(true))
	require.NoError(t, err)
	var fields []string
	for iter.Next() {
		fields = append(fields, iter.Label())
	}
	require.Contains(t, fields, "conn")
	require.NotContains(t, fields, "database", "should not contain fields from #Connection directly")
}
