package webui

import (
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
//	UI_Placeholder: Input placeholder text
//	UI_Widget:      Widget override: input, select, textarea, radio, checkbox
//	UI_Options:     Comma-separated values for select/radio widgets
//	UI_Hidden:      Hide field from UI (true/false)
//	UI_Readonly:    Make field read-only (true/false)
//	UI_Order:       Display order within section (integer, lower first)
//	UI_Min:         Minimum value for number inputs
//	UI_Max:         Maximum value for number inputs
//	UI_Pattern:     Regex validation pattern for text inputs
//	UI_Columns:     Grid columns for a section (default: 2)
//	UI_Colspan:     Number of grid columns a field spans
type UIHints struct {
	Label       string
	Help        string
	Placeholder string
	Widget      string
	Options     []string
	Hidden      bool
	Readonly    bool
	Order       int
	Min         string
	Max         string
	Pattern     string
	Columns     int
	Colspan     int
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
			case "UI_Placeholder":
				hints.Placeholder = value
			case "UI_Widget":
				hints.Widget = value
			case "UI_Options":
				for _, opt := range strings.Split(value, ",") {
					opt = strings.TrimSpace(opt)
					if opt != "" {
						hints.Options = append(hints.Options, opt)
					}
				}
			case "UI_Hidden":
				hints.Hidden = value == "true"
			case "UI_Readonly":
				hints.Readonly = value == "true"
			case "UI_Order":
				if n, err := strconv.Atoi(value); err == nil {
					hints.Order = n
				}
			case "UI_Min":
				hints.Min = value
			case "UI_Max":
				hints.Max = value
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
