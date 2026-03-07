package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed schema.cue
var schemaFile string

//go:embed templates/form.html
var formTemplateStr string

//go:embed static/style.css
var cssFile string

// --- UI hint parsing ---

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

// Field represents a single form field.
type Field struct {
	Name        string
	Path        string
	Type        string
	InputType   string
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
	Default     string
	Colspan     int
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

// parseUIHints extracts UI_ directives from a CUE value's doc comments.
func parseUIHints(val cue.Value) UIHints {
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

// --- CUE to form model ---

func cueTypeToInputType(kind cue.Kind) string {
	switch kind {
	case cue.IntKind, cue.FloatKind, cue.NumberKind:
		return "number"
	case cue.BoolKind:
		return "checkbox"
	default:
		return "text"
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// hasStructFields returns true if val contains at least one struct-typed field.
func hasStructFields(val cue.Value) bool {
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

// parseSection recursively converts a CUE struct value into a Section.
func parseSection(name string, val cue.Value, pathPrefix string, sectionHints UIHints) Section {
	section := Section{
		Name:    name,
		Label:   sectionHints.Label,
		Help:    sectionHints.Help,
		Columns: sectionHints.Columns,
	}
	if section.Label == "" {
		section.Label = titleCase(name)
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

		fieldHints := parseUIHints(fieldVal)

		// Struct field → nested section
		if fieldVal.IncompleteKind() == cue.StructKind {
			sub := parseSection(fieldName, fieldVal, fieldPath, fieldHints)
			section.Sections = append(section.Sections, sub)
			continue
		}

		// Scalar field → form input
		inputType := cueTypeToInputType(fieldVal.IncompleteKind())
		widget := fieldHints.Widget
		if widget == "" {
			if len(fieldHints.Options) > 0 {
				widget = "select"
			} else if fieldVal.IncompleteKind() == cue.BoolKind {
				widget = "checkbox"
			} else {
				widget = "input"
			}
		}

		label := fieldHints.Label
		if label == "" {
			label = titleCase(fieldName)
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
			Name:        fieldName,
			Path:        fieldPath,
			Type:        fieldVal.IncompleteKind().String(),
			InputType:   inputType,
			Label:       label,
			Help:        fieldHints.Help,
			Placeholder: fieldHints.Placeholder,
			Widget:      widget,
			Options:     fieldHints.Options,
			Hidden:      fieldHints.Hidden,
			Readonly:    fieldHints.Readonly,
			Order:       fieldHints.Order,
			Min:         fieldHints.Min,
			Max:         fieldHints.Max,
			Pattern:     fieldHints.Pattern,
			Default:     defVal,
			Colspan:     fieldHints.Colspan,
		})
	}

	sort.Slice(section.Fields, func(i, j int) bool {
		return section.Fields[i].Order < section.Fields[j].Order
	})

	return section
}

// --- HTTP server ---

func main() {
	ctx := cuecontext.New()
	rootValue := ctx.CompileString(schemaFile)
	if rootValue.Err() != nil {
		log.Fatalf("Failed to compile CUE schema: %v", rootValue.Err())
	}

	// Discover top-level definitions
	type defEntry struct {
		name string
		val  cue.Value
	}
	var allDefs []defEntry
	defIter, _ := rootValue.Fields(cue.Definitions(true))
	for defIter.Next() {
		name := strings.TrimPrefix(defIter.Selector().String(), "#")
		allDefs = append(allDefs, defEntry{name, defIter.Value()})
	}

	// Root definitions are those with struct sub-fields (they aggregate others).
	// If none qualify, fall back to rendering all definitions.
	var roots []defEntry
	for _, d := range allDefs {
		if hasStructFields(d.val) {
			roots = append(roots, d)
		}
	}
	if len(roots) == 0 {
		roots = allDefs
	}

	var formData FormData
	if len(roots) == 1 {
		// Single root: unwrap it — its sub-sections become the top-level sections.
		hints := parseUIHints(roots[0].val)
		top := parseSection(roots[0].name, roots[0].val, "", hints)
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
			hints := parseUIHints(r.val)
			s := parseSection(r.name, r.val, "", hints)
			formData.Sections = append(formData.Sections, s)
		}
	}

	tmpl := template.Must(template.New("base").Parse(formTemplateStr))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "form", formData); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			log.Printf("Template error: %v", err)
		}
	})

	http.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, cssFile)
	})

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		var values []KeyValue
		for key, vals := range r.PostForm {
			values = append(values, KeyValue{Key: key, Value: strings.Join(vals, ", ")})
		}
		sort.Slice(values, func(i, j int) bool {
			return values[i].Key < values[j].Key
		})
		result := ResultData{Title: formData.Title, Values: values}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "result", result); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			log.Printf("Template error: %v", err)
		}
	})

	addr := "localhost:8080"
	log.Printf("Server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
