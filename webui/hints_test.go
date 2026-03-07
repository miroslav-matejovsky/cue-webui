package webui

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
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
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)

	if h.Label != "Server Address" {
		t.Errorf("Label = %q, want %q", h.Label, "Server Address")
	}
	if h.Help != "Hostname or IP" {
		t.Errorf("Help = %q, want %q", h.Help, "Hostname or IP")
	}
	if h.Widget != "textarea" {
		t.Errorf("Widget = %q, want %q", h.Widget, "textarea")
	}
	if !h.Hidden {
		t.Error("Hidden = false, want true")
	}
	if !h.Readonly {
		t.Error("Readonly = false, want true")
	}
	if h.Order != 3 {
		t.Errorf("Order = %d, want 3", h.Order)
	}
	if h.Columns != 4 {
		t.Errorf("Columns = %d, want 4", h.Columns)
	}
	if h.Colspan != 2 {
		t.Errorf("Colspan = %d, want 2", h.Colspan)
	}
}

func TestParseUIHints_Defaults(t *testing.T) {
	src := `
// Just a plain comment, no UI hints.
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)

	if h.Label != "" {
		t.Errorf("Label = %q, want empty", h.Label)
	}
	if h.Order != 999 {
		t.Errorf("Order = %d, want 999", h.Order)
	}
	if h.Hidden {
		t.Error("Hidden = true, want false")
	}
	if h.Readonly {
		t.Error("Readonly = true, want false")
	}
}

func TestParseUIHints_PartialDirectives(t *testing.T) {
	src := `
// UI_Label: My Field
// UI_Help: Some help text
field: int
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)

	if h.Label != "My Field" {
		t.Errorf("Label = %q, want %q", h.Label, "My Field")
	}
	if h.Help != "Some help text" {
		t.Errorf("Help = %q, want %q", h.Help, "Some help text")
	}
	if h.Widget != "" {
		t.Errorf("Widget = %q, want empty", h.Widget)
	}
	if h.Order != 999 {
		t.Errorf("Order = %d, want 999", h.Order)
	}
}

func TestExtractOptions_Disjunction(t *testing.T) {
	src := `field: "debug" | "info" | "warn" | "error"`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	options := ExtractOptions(val)
	want := []string{"debug", "info", "warn", "error"}
	if len(options) != len(want) {
		t.Fatalf("Options length = %d, want %d", len(options), len(want))
	}
	for i, got := range options {
		if got != want[i] {
			t.Errorf("Options[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestExtractOptions_NoDisjunction(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	options := ExtractOptions(val)
	if len(options) != 0 {
		t.Errorf("Options = %v, want empty", options)
	}
}

func TestExtractBounds(t *testing.T) {
	src := `field: int & >=1 & <=65535`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	min, max := ExtractBounds(val)
	if min != "1" {
		t.Errorf("Min = %q, want %q", min, "1")
	}
	if max != "65535" {
		t.Errorf("Max = %q, want %q", max, "65535")
	}
}

func TestExtractBounds_NoBounds(t *testing.T) {
	src := `field: int`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	min, max := ExtractBounds(val)
	if min != "" {
		t.Errorf("Min = %q, want empty", min)
	}
	if max != "" {
		t.Errorf("Max = %q, want empty", max)
	}
}

func TestExtractBounds_OnlyMin(t *testing.T) {
	src := `field: int & >=0`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	min, max := ExtractBounds(val)
	if min != "0" {
		t.Errorf("Min = %q, want %q", min, "0")
	}
	if max != "" {
		t.Errorf("Max = %q, want empty", max)
	}
}

func TestExtractPattern(t *testing.T) {
	src := `field: string & =~"^[a-z]+$"`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	pattern := ExtractPattern(val)
	if pattern != "^[a-z]+$" {
		t.Errorf("Pattern = %q, want %q", pattern, "^[a-z]+$")
	}
}

func TestExtractPattern_NoPattern(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	pattern := ExtractPattern(val)
	if pattern != "" {
		t.Errorf("Pattern = %q, want empty", pattern)
	}
}

func TestParseUIHints_InvalidOrder(t *testing.T) {
	src := `
// UI_Order: notanumber
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	if h.Order != 999 {
		t.Errorf("Order = %d, want 999 (default)", h.Order)
	}
}

func TestParseUIHints_MalformedLine(t *testing.T) {
	src := `
// UI_LabelNoColon
// UI_Help: Valid help
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	if h.Label != "" {
		t.Errorf("Label = %q, want empty (malformed line skipped)", h.Label)
	}
	if h.Help != "Valid help" {
		t.Errorf("Help = %q, want %q", h.Help, "Valid help")
	}
}

func TestParseUIHints_HiddenFalse(t *testing.T) {
	src := `
// UI_Hidden: false
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	if h.Hidden {
		t.Error("Hidden = true, want false")
	}
}

func TestParseUIHints_NoDocComments(t *testing.T) {
	src := `field: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	if h.Order != 999 {
		t.Errorf("Order = %d, want 999", h.Order)
	}
	if h.Label != "" {
		t.Errorf("Label = %q, want empty", h.Label)
	}
}
