package webform

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
)

// Field represents a single form field derived from a scalar CUE value.
//
// Constraint-derived attributes (Options, Min, Max, Pattern) are extracted
// from native CUE expressions (disjunctions, bounds, =~ regex), while
// display attributes (Label, Help, Widget, etc.) come from UI_ doc-comment hints.
type Field struct {
	Name      string   // CUE field name.
	Path      string   // Dot-separated path used as the HTML input name (e.g. "connection.port").
	Type      string   // CUE kind as a string (e.g. "string", "int").
	InputType string   // HTML input type ("text", "number", or "checkbox").
	Label     string   // Human-readable label shown in the form.
	Help      string   // Help text displayed below the input.
	Widget    string   // Rendered widget: "input", "select", "textarea", "radio", or "checkbox".
	Options   []string // Allowed values extracted from a CUE disjunction.
	Hidden    bool     // Whether the field is hidden from the UI.
	Readonly  bool     // Whether the field is rendered as read-only.
	Order     int      // Sort weight within its section (lower values first).
	Min       string   // Minimum value from a CUE >= or > bound.
	Max       string   // Maximum value from a CUE <= or < bound.
	Pattern   string   // Regex pattern from a CUE =~ constraint.
	Default   string   // Default value from a CUE default marker (*value).
	Colspan   int      // Number of grid columns the field spans.
}

// Section represents a group of fields derived from a CUE struct.
// Sections can be nested: a struct field within a struct becomes a child Section.
type Section struct {
	ID         string    // Stable HTML-safe identifier for this section.
	Name       string    // CUE field name of the struct.
	Label      string    // Display label (from UI_Label or title-cased Name).
	Help       string    // Help text shown below the section legend.
	Columns    int       // Number of CSS grid columns (default 2).
	Navigation string    // Child section layout mode (e.g. "tabs").
	Fields     []Field   // Scalar fields, sorted by Order.
	Sections   []Section // Nested sub-sections.
}

// FormData is the top-level view model passed to the "form" HTML template.
type FormData struct {
	Title      string    // Page title, derived from the root definition's UI_Label.
	ID         string    // Stable HTML-safe identifier for top-level tab groups.
	Navigation string    // Top-level section layout mode (e.g. "tabs").
	Sections   []Section // Top-level sections rendered as fieldsets.
}

// CueTypeToInputType maps a CUE kind to an HTML input type.
func CueTypeToInputType(kind cue.Kind) string {
	switch kind {
	case cue.IntKind, cue.FloatKind, cue.NumberKind:
		return "number"
	case cue.BoolKind:
		return "checkbox"
	default:
		return "text"
	}
}

// TitleCase capitalises the first letter of a string.
func TitleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func buildSectionID(name string, pathPrefix string) string {
	raw := name
	if pathPrefix != "" {
		raw = pathPrefix
	}

	var builder strings.Builder
	builder.Grow(len(raw))
	lastDash := false
	for _, r := range raw {
		isLetter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	id := strings.Trim(builder.String(), "-")
	if id == "" {
		return "section"
	}
	return id
}

// HasStructFields returns true if val contains at least one struct-typed field.
func HasStructFields(val cue.Value) bool {
	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return false
	}
	for iter.Next() {
		if iter.Value().IncompleteKind() == cue.StructKind {
			return true
		}
	}
	return false
}

// ParseSection recursively converts a CUE struct value into a Section.
// Struct-typed fields become nested sub-sections; scalar fields become form Fields.
// The pathPrefix is prepended to field names to build dot-separated input paths
// (e.g. passing "connection" produces paths like "connection.port").
// Widget type, options, bounds, and pattern are inferred from native CUE
// constraints; display hints are taken from sectionHints and per-field UI_ comments.
func ParseSection(name string, val cue.Value, pathPrefix string, sectionHints UIHints) Section {
	section := Section{
		ID:         buildSectionID(name, pathPrefix),
		Name:       name,
		Label:      sectionHints.Label,
		Help:       sectionHints.Help,
		Columns:    sectionHints.Columns,
		Navigation: sectionHints.Navigation,
	}
	if section.Label == "" {
		section.Label = TitleCase(name)
	}
	if section.Columns <= 0 {
		section.Columns = 2
	}

	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return section
	}
	for iter.Next() {
		fieldVal := iter.Value()
		fieldName := iter.Selector().String()
		fieldPath := fieldName
		if pathPrefix != "" {
			fieldPath = pathPrefix + "." + fieldName
		}

		fieldHints := ParseUIHints(fieldVal)

		// Struct field → nested section
		if fieldVal.IncompleteKind() == cue.StructKind {
			sub := ParseSection(fieldName, fieldVal, fieldPath, fieldHints)
			section.Sections = append(section.Sections, sub)
			continue
		}

		// Scalar field → form input
		inputType := CueTypeToInputType(fieldVal.IncompleteKind())

		options := ExtractOptions(fieldVal)
		min, max := ExtractBounds(fieldVal)
		pattern := ExtractPattern(fieldVal)

		widget := fieldHints.Widget
		if widget == "" {
			if len(options) > 0 {
				widget = "select"
			} else if fieldVal.IncompleteKind() == cue.BoolKind {
				widget = "checkbox"
			} else {
				widget = "input"
			}
		}

		label := fieldHints.Label
		if label == "" {
			label = TitleCase(fieldName)
		}

		defVal := ""
		if d, exists := fieldVal.Default(); exists {
			switch fieldVal.IncompleteKind() {
			case cue.StringKind:
				if s, err := d.String(); err == nil {
					defVal = s
				}
			case cue.IntKind:
				if n, err := d.Int64(); err == nil {
					defVal = strconv.FormatInt(n, 10)
				}
			case cue.FloatKind:
				if f, err := d.Float64(); err == nil {
					defVal = strconv.FormatFloat(f, 'f', -1, 64)
				}
			case cue.BoolKind:
				if b, err := d.Bool(); err == nil {
					defVal = strconv.FormatBool(b)
				}
			}
		}

		section.Fields = append(section.Fields, Field{
			Name:      fieldName,
			Path:      fieldPath,
			Type:      fieldVal.IncompleteKind().String(),
			InputType: inputType,
			Label:     label,
			Help:      fieldHints.Help,
			Widget:    widget,
			Options:   options,
			Hidden:    fieldHints.Hidden,
			Readonly:  fieldHints.Readonly,
			Order:     fieldHints.Order,
			Min:       min,
			Max:       max,
			Pattern:   pattern,
			Default:   defVal,
			Colspan:   fieldHints.Colspan,
		})
	}

	sort.Slice(section.Fields, func(i, j int) bool {
		return section.Fields[i].Order < section.Fields[j].Order
	})

	return section
}

// BuildFormData constructs a FormData from a compiled CUE schema value.
// It discovers all top-level CUE definitions (#Name) and identifies "root"
// definitions — those containing struct sub-fields that aggregate other definitions.
//
// When a single root is found it is unwrapped: its label becomes the page title
// and its struct fields become top-level sections. Any direct scalar fields on
// the root are collected into a prepended "General" section.
//
// When multiple roots exist, each is rendered as its own section under a
// generic "Configuration" title. If no definition contains struct sub-fields,
// all definitions are treated as roots.
func BuildFormData(cueSchema cue.Value) (FormData, error) {
	if err := cueSchema.Err(); err != nil {
		return FormData{}, fmt.Errorf("invalid CUE schema: %w", err)
	}

	type defEntry struct {
		name string
		val  cue.Value
	}
	var allDefs []defEntry
	defIter, err := cueSchema.Fields(cue.Definitions(true))
	if err != nil {
		return FormData{}, fmt.Errorf("failed to iterate CUE definitions: %w", err)
	}
	for defIter.Next() {
		name := strings.TrimPrefix(defIter.Selector().String(), "#")
		allDefs = append(allDefs, defEntry{name, defIter.Value()})
	}
	if len(allDefs) == 0 {
		return FormData{}, errors.New("no CUE definitions found in schema")
	}

	// Root definitions are those with struct sub-fields (they aggregate others).
	// If none qualify, fall back to rendering all definitions.
	var roots []defEntry
	for _, d := range allDefs {
		if HasStructFields(d.val) {
			roots = append(roots, d)
		}
	}
	if len(roots) == 0 {
		roots = allDefs
	}

	var formData FormData
	if len(roots) == 1 {
		// Single root: unwrap it — its sub-sections become the top-level sections.
		hints := ParseUIHints(roots[0].val)
		top := ParseSection(roots[0].name, roots[0].val, "", hints)
		formData.Title = top.Label
		formData.ID = top.ID
		formData.Navigation = top.Navigation
		formData.Sections = top.Sections
		// If the root also has direct scalar fields, show them in a "General" section.
		if len(top.Fields) > 0 {
			general := Section{
				ID:      top.ID + "-general",
				Name:    top.Name,
				Label:   "General",
				Columns: top.Columns,
				Fields:  top.Fields,
			}
			formData.Sections = append([]Section{general}, formData.Sections...)
		}
	} else {
		formData.Title = "Configuration"
		formData.ID = "configuration"
		for _, r := range roots {
			hints := ParseUIHints(r.val)
			s := ParseSection(r.name, r.val, "", hints)
			formData.Sections = append(formData.Sections, s)
		}
	}

	return formData, nil
}
