package webui

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestParseUIHints_AllDirectives(t *testing.T) {
	src := `
// UI_Label: Server Address
// UI_Help: Hostname or IP
// UI_Widget: textarea
// UI_Hidden: true
// UI_Readonly: true
// UI_Order: 3
// UI_Columns: 4
// UI_Colspan: 2
// UI_Navigation: tabs
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)

	require.Equal(t, "Server Address", h.Label)
	require.Equal(t, "Hostname or IP", h.Help)
	require.Equal(t, "textarea", h.Widget)
	require.True(t, h.Hidden)
	require.True(t, h.Readonly)
	require.Equal(t, 3, h.Order)
	require.Equal(t, 4, h.Columns)
	require.Equal(t, 2, h.Colspan)
	require.Equal(t, "tabs", h.Navigation)
}

func TestParseUIHints_Defaults(t *testing.T) {
	src := `
// Just a plain comment, no UI hints.
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)

	require.Empty(t, h.Label)
	require.Equal(t, 999, h.Order)
	require.False(t, h.Hidden)
	require.False(t, h.Readonly)
	require.Empty(t, h.Navigation)
}

func TestParseUIHints_PartialDirectives(t *testing.T) {
	src := `
// UI_Label: My Field
// UI_Help: Some help text
field: int
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)

	require.Equal(t, "My Field", h.Label)
	require.Equal(t, "Some help text", h.Help)
	require.Empty(t, h.Widget)
	require.Equal(t, 999, h.Order)
}

func TestExtractOptions_Disjunction(t *testing.T) {
	src := `field: "debug" | "info" | "warn" | "error"`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	options := ExtractOptions(val)
	require.Equal(t, []string{"debug", "info", "warn", "error"}, options)
}

func TestExtractOptions_NoDisjunction(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	options := ExtractOptions(val)
	require.Empty(t, options)
}

func TestExtractBounds(t *testing.T) {
	src := `field: int & >=1 & <=65535`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	min, max := ExtractBounds(val)
	require.Equal(t, "1", min)
	require.Equal(t, "65535", max)
}

func TestExtractBounds_NoBounds(t *testing.T) {
	src := `field: int`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	min, max := ExtractBounds(val)
	require.Empty(t, min)
	require.Empty(t, max)
}

func TestExtractBounds_OnlyMin(t *testing.T) {
	src := `field: int & >=0`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	min, max := ExtractBounds(val)
	require.Equal(t, "0", min)
	require.Empty(t, max)
}

func TestExtractPattern(t *testing.T) {
	src := `field: string & =~"^[a-z]+$"`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	pattern := ExtractPattern(val)
	require.Equal(t, "^[a-z]+$", pattern)
}

func TestExtractPattern_NoPattern(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	pattern := ExtractPattern(val)
	require.Empty(t, pattern)
}

func TestParseUIHints_InvalidOrder(t *testing.T) {
	src := `
// UI_Order: notanumber
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)
	require.Equal(t, 999, h.Order)
}

func TestParseUIHints_MalformedLine(t *testing.T) {
	src := `
// UI_LabelNoColon
// UI_Help: Valid help
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)
	require.Empty(t, h.Label, "malformed line should be skipped")
	require.Equal(t, "Valid help", h.Help)
}

func TestParseUIHints_HiddenFalse(t *testing.T) {
	src := `
// UI_Hidden: false
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)
	require.False(t, h.Hidden)
}

func TestParseUIHints_NoDocComments(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	require.NoError(t, val.Err())

	h := ParseUIHints(val)
	require.Equal(t, 999, h.Order)
	require.Empty(t, h.Label)
}
