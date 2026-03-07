package webui

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
)

// Field represents a single form field.
type Field struct {
	Name      string
	Path      string
	Type      string
	InputType string
	Label     string
	Help      string
	Widget    string
	Options   []string
	Hidden    bool
	Readonly  bool
	Order     int
	Min       string
	Max       string
	Pattern   string
	Default   string
	Colspan   int
}

// Section represents a group of fields (a CUE struct).
type Section struct {
	Name     string
	Label    string
	Help     string
	Columns  int
	Fields   []Field
	Sections []Section
}

// FormData is passed to the form template.
type FormData struct {
	Title    string
	Sections []Section
}

// KeyValue represents a submitted form value.
type KeyValue struct {
	Key   string
	Value string
}

// ResultData is passed to the result template.
type ResultData struct {
	Title  string
	Values []KeyValue
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
func ParseSection(name string, val cue.Value, pathPrefix string, sectionHints UIHints) Section {
	section := Section{
		Name:    name,
		Label:   sectionHints.Label,
		Help:    sectionHints.Help,
		Columns: sectionHints.Columns,
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
func BuildFormData(rootValue cue.Value) (FormData, error) {
	if err := rootValue.Err(); err != nil {
		return FormData{}, err
	}

	type defEntry struct {
		name string
		val  cue.Value
	}
	var allDefs []defEntry
	defIter, err := rootValue.Fields(cue.Definitions(true))
	if err != nil {
		return FormData{}, err
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
		formData.Sections = top.Sections
		// If the root also has direct scalar fields, show them in a "General" section.
		if len(top.Fields) > 0 {
			general := Section{
				Name:    top.Name,
				Label:   "General",
				Columns: top.Columns,
				Fields:  top.Fields,
			}
			formData.Sections = append([]Section{general}, formData.Sections...)
		}
	} else {
		formData.Title = "Configuration"
		for _, r := range roots {
			hints := ParseUIHints(r.val)
			s := ParseSection(r.name, r.val, "", hints)
			formData.Sections = append(formData.Sections, s)
		}
	}

	return formData, nil
}
