package webui

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
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

// NewHandler returns an http.Handler that serves three endpoints:
//   - GET  /                  — renders the HTML form for the given FormData.
//   - GET  /static/style.css  — serves the embedded CSS stylesheet.
//   - POST /submit            — processes form submission and renders a results page.
//
// Non-POST requests to /submit are redirected to /. Any other path returns 404.
func NewHandler(formData FormData) (http.Handler, error) {
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "form", formData); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, cssFile)
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
		}
	})

	return mux, nil
}
