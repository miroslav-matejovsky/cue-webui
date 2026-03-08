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
	require.Equal(t, v, schema.RootValue(v))
}

func TestRootValue_NoDefs_ReturnsSelf(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{...}`)
	require.Equal(t, v, schema.RootValue(v))
}

func TestRootValue_SingleRootDef_ReturnsRootValue(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString("#Connection: { host: string }\n#Config: { conn: #Connection }\n")
	got := schema.RootValue(v)
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

func TestRootValue_MultipleRootDefs_ReturnsTopLevel(t *testing.T) {
	ctx := cuecontext.New()
	// Two definitions both with struct sub-fields — no single root, return top level.
	v := ctx.CompileString("#A: { sub: { x: int } }\n#B: { sub: { y: string } }\n")
	got := schema.RootValue(v)
	// Should return the top-level value (both definitions remain as definitions).
	iter, err := got.Fields(cue.Definitions(true))
	require.NoError(t, err)
	var defs []string
	for iter.Next() {
		defs = append(defs, iter.Label())
	}
	require.Contains(t, defs, "#A")
	require.Contains(t, defs, "#B")
}
