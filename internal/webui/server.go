package webui

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/jsonschema"
	"github.com/miroslav-matejovsky/cue-webui/internal/config"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

//go:embed templates/form.html
var formTemplateStr string

//go:embed static/style.css
var cssFile string

// FormTemplate returns the embedded HTML form template string.
func FormTemplate() string { return formTemplateStr }

// StyleCSS returns the embedded CSS stylesheet string.
func StyleCSS() string { return cssFile }

// ParseFormTemplate parses the embedded form template and returns a ready-to-use *template.Template.
func ParseFormTemplate() (*template.Template, error) {
	return template.New("base").Parse(formTemplateStr)
}

// formPageData is the view model passed to the "form" template.
type formPageData struct {
	webform.FormData
	ErrorMessage string
}

// NewHandler returns an http.Handler that serves four endpoints:
//   - GET  /                  — renders the HTML form populated from the JSON config file.
//   - GET  /static/style.css  — serves the embedded CSS stylesheet.
//   - GET  /schema.json       — serves the JSON Schema derived from the CUE schema.
//   - POST /submit            — validates submitted values against the CUE schema,
//     writes valid JSON to configPath, and redirects to /.
//
// If configPath does not exist, the form is rendered with schema defaults only.
// If CUE validation fails on submit, the form is re-rendered with an error banner.
func NewHandler(formData webform.FormData, cueSchema cue.Value, configPath string) (http.Handler, error) {
	mux := http.NewServeMux()
	tmpl, err := ParseFormTemplate()
	if err != nil {
		return nil, fmt.Errorf("parsing form template: %w", err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		populated := formData
		if flat, err := config.Load(configPath); err != nil {
			log.Printf("Warning: failed to parse config file %s: %v", configPath, err)
		} else if flat != nil {
			populated = applyStoredValues(formData, flat)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "form", formPageData{FormData: populated}); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, cssFile)
	})

	mux.HandleFunc("/schema.json", func(w http.ResponseWriter, r *http.Request) {
		expr, err := jsonschema.Generate(cueSchema, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("generating JSON Schema: %v", err), http.StatusInternalServerError)
			return
		}
		jsonBytes, err := cueSchema.Context().BuildExpr(expr).MarshalJSON()
		if err != nil {
			http.Error(w, fmt.Sprintf("marshalling JSON Schema: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonBytes)
	})

	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Load existing values from config file (if any)
		existing, _ := config.Load(configPath)
		if existing == nil {
			existing = map[string]string{}
		}

		updatedValues := mergeSubmittedValues(formData, existing, r.PostForm)

		// Convert to nested JSON
		jsonBytes, err := config.ToJSON(updatedValues, CollectFieldTypes(formData))
		if err != nil {
			renderFormWithError(w, tmpl, formData, updatedValues, fmt.Sprintf("Failed to build JSON: %v", err))
			return
		}

		// Validate against CUE schema
		if err := config.Validate(jsonBytes, cueSchema); err != nil {
			renderFormWithError(w, tmpl, formData, updatedValues, fmt.Sprintf("Validation error: %v", err))
			return
		}

		// Write validated JSON to config file
		if err := os.WriteFile(configPath, jsonBytes, 0644); err != nil {
			renderFormWithError(w, tmpl, formData, updatedValues, fmt.Sprintf("Failed to save config: %v", err))
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return mux, nil
}

func renderFormWithError(w http.ResponseWriter, tmpl *template.Template, formData webform.FormData, values map[string]string, errMsg string) {
	populated := applyStoredValues(formData, values)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := tmpl.ExecuteTemplate(w, "form", formPageData{FormData: populated, ErrorMessage: errMsg}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
