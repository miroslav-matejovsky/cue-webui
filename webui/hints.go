package webui

import (
	"fmt"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
)

// UIHints holds parsed UI_ directives from CUE doc comments.
//
// Supported hints (place in CUE comments as "// UI_Key: value"):
//
//	UI_Label:       Custom display label (default: field name, title-cased)
//	UI_Help:        Help text shown below the input
//	UI_Widget:      Widget override: input, select, textarea, radio, checkbox
//	UI_Hidden:      Hide field from UI (true/false)
//	UI_Readonly:    Make field read-only (true/false)
//	UI_Order:       Display order within section (integer, lower first)
//	UI_Pattern:     Regex validation pattern for text inputs
//	UI_Columns:     Grid columns for a section (default: 2)
//	UI_Colspan:     Number of grid columns a field spans
type UIHints struct {
	Label    string
	Help     string
	Widget   string
	Hidden   bool
	Readonly bool
	Order    int
	Pattern  string
	Columns  int
	Colspan  int
}

// ParseUIHints extracts UI_ directives from a CUE value's doc comments.
func ParseUIHints(val cue.Value) UIHints {
	hints := UIHints{Order: 999}
	for _, cg := range val.Doc() {
		for _, line := range strings.Split(cg.Text(), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "UI_") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			switch key {
			case "UI_Label":
				hints.Label = value
			case "UI_Help":
				hints.Help = value
			case "UI_Widget":
				hints.Widget = value
			case "UI_Hidden":
				hints.Hidden = value == "true"
			case "UI_Readonly":
				hints.Readonly = value == "true"
			case "UI_Order":
				if n, err := strconv.Atoi(value); err == nil {
					hints.Order = n
				}
			case "UI_Pattern":
				hints.Pattern = value
			case "UI_Columns":
				if n, err := strconv.Atoi(value); err == nil {
					hints.Columns = n
				}
			case "UI_Colspan":
				if n, err := strconv.Atoi(value); err == nil {
					hints.Colspan = n
				}
			}
		}
	}
	return hints
}

// ExtractOptions extracts string options from a CUE disjunction (e.g. "a" | "b" | "c").
func ExtractOptions(val cue.Value) []string {
	op, args := val.Expr()
	if op != cue.OrOp {
		return nil
	}
	var options []string
	for _, a := range args {
		if s, err := a.String(); err == nil {
			options = append(options, s)
		}
	}
	return options
}

// ExtractBounds extracts min and max bounds from CUE constraints (e.g. >=1 & <=65535).
func ExtractBounds(val cue.Value) (min, max string) {
	extractBoundsRecursive(val, &min, &max)
	return
}

func extractBoundsRecursive(val cue.Value, min, max *string) {
	op, args := val.Expr()
	switch op {
	case cue.AndOp:
		for _, a := range args {
			extractBoundsRecursive(a, min, max)
		}
	case cue.GreaterThanEqualOp, cue.GreaterThanOp:
		if len(args) > 0 {
			*min = fmt.Sprint(args[0])
		}
	case cue.LessThanEqualOp, cue.LessThanOp:
		if len(args) > 0 {
			*max = fmt.Sprint(args[0])
		}
	}
}
