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
	"github.com/miroslav-matejovsky/cue-webui/webui"
)

//go:embed schema.cue
var schemaFile string

//go:embed templates/form.html
var formTemplateStr string

//go:embed static/style.css
var cssFile string

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
func parseSection(name string, val cue.Value, pathPrefix string, sectionHints webui.UIHints) Section {
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

		fieldHints := webui.ParseUIHints(fieldVal)

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
		hints := webui.ParseUIHints(roots[0].val)
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
			hints := webui.ParseUIHints(r.val)
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
