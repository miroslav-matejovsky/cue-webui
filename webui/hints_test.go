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
// UI_Placeholder: e.g. 0.0.0.0
// UI_Widget: textarea
// UI_Options: a, b, c
// UI_Hidden: true
// UI_Readonly: true
// UI_Order: 3
// UI_Min: 1
// UI_Max: 100
// UI_Pattern: ^[a-z]+$
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
	if h.Placeholder != "e.g. 0.0.0.0" {
		t.Errorf("Placeholder = %q, want %q", h.Placeholder, "e.g. 0.0.0.0")
	}
	if h.Widget != "textarea" {
		t.Errorf("Widget = %q, want %q", h.Widget, "textarea")
	}
	if len(h.Options) != 3 || h.Options[0] != "a" || h.Options[1] != "b" || h.Options[2] != "c" {
		t.Errorf("Options = %v, want [a b c]", h.Options)
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
	if h.Min != "1" {
		t.Errorf("Min = %q, want %q", h.Min, "1")
	}
	if h.Max != "100" {
		t.Errorf("Max = %q, want %q", h.Max, "100")
	}
	if h.Pattern != "^[a-z]+$" {
		t.Errorf("Pattern = %q, want %q", h.Pattern, "^[a-z]+$")
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
	if len(h.Options) != 0 {
		t.Errorf("Options = %v, want empty", h.Options)
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

func TestParseUIHints_OptionsWithSpaces(t *testing.T) {
	src := `
// UI_Options: debug , info , warn , error
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	want := []string{"debug", "info", "warn", "error"}
	if len(h.Options) != len(want) {
		t.Fatalf("Options length = %d, want %d", len(h.Options), len(want))
	}
	for i, got := range h.Options {
		if got != want[i] {
			t.Errorf("Options[%d] = %q, want %q", i, got, want[i])
		}
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

func TestParseUIHints_EmptyOptions(t *testing.T) {
	src := `
// UI_Options: ,, ,
field: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src).LookupPath(cue.ParsePath("field"))
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	h := ParseUIHints(val)
	if len(h.Options) != 0 {
		t.Errorf("Options = %v, want empty (blank entries filtered)", h.Options)
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
