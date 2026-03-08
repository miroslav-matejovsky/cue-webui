package webui

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/jsonschema"
	"github.com/miroslav-matejovsky/cue-webui/internal/config"
	"github.com/miroslav-matejovsky/cue-webui/internal/schema"
	"github.com/miroslav-matejovsky/cue-webui/internal/watcher"
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
	ErrorMessage   string
	SuccessMessage string
	LiveReload     bool
}

// options holds optional configuration for NewHandler.
type options struct {
	watcher *watcher.SchemaWatcher
}

// Option configures NewHandler behaviour.
type Option func(*options)

// WithWatcher enables live reload: when the schema file changes, the handler
// uses the updated form data/schema and signals connected browsers via SSE.
func WithWatcher(w *watcher.SchemaWatcher) Option {
	return func(o *options) { o.watcher = w }
}

// NewHandler returns an http.Handler that serves the web UI endpoints.
// If a non-nil SchemaWatcher is provided, the handler dynamically reads the
// latest form data and schema from it, and exposes a /events SSE endpoint
// that signals browsers to reload when the schema file changes.
func NewHandler(formData webform.FormData, cueSchema cue.Value, configPath string, opts ...Option) (http.Handler, error) {
	cfg := options{}
	for _, o := range opts {
		o(&cfg)
	}
	mux := http.NewServeMux()
	tmpl, err := ParseFormTemplate()
	if err != nil {
		return nil, fmt.Errorf("parsing form template: %w", err)
	}

	getFormData := func() webform.FormData {
		if cfg.watcher != nil {
			return cfg.watcher.FormData()
		}
		return formData
	}
	getSchema := func() cue.Value {
		if cfg.watcher != nil {
			return cfg.watcher.Schema()
		}
		return cueSchema
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		currentFormData := getFormData()
		populated := currentFormData
		if flat, err := config.Load(configPath); err != nil {
			log.Printf("Warning: failed to parse config file %s: %v", configPath, err)
		} else if flat != nil {
			populated = applyStoredValues(currentFormData, flat)
		}

		var successMsg string
		if savedAt := r.URL.Query().Get("saved"); savedAt != "" {
			if t, err := time.Parse(time.RFC3339, savedAt); err == nil {
				successMsg = fmt.Sprintf("Configuration saved to %s at %s", configPath, t.Format("15:04:05"))
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "form", formPageData{FormData: populated, SuccessMessage: successMsg, LiveReload: cfg.watcher != nil}); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, cssFile)
	})

	mux.HandleFunc("/schema.json", func(w http.ResponseWriter, r *http.Request) {
		rootVal, err := schema.RootValue(getSchema())
		if err != nil {
			http.Error(w, fmt.Sprintf("resolving schema root: %v", err), http.StatusInternalServerError)
			return
		}
		expr, err := jsonschema.Generate(rootVal, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("generating JSON Schema: %v", err), http.StatusInternalServerError)
			return
		}
		jsonBytes, err := getSchema().Context().BuildExpr(expr).MarshalJSON()
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

		currentFormData := getFormData()
		updatedValues := mergeSubmittedValues(currentFormData, existing, r.PostForm)

		// Convert to nested JSON
		jsonBytes, err := config.ToJSON(updatedValues, CollectFieldTypes(currentFormData))
		if err != nil {
			log.Printf("Failed to build JSON: %v", err)
			renderFormWithError(w, tmpl, currentFormData, updatedValues, fmt.Sprintf("Failed to build JSON: %v", err))
			return
		}

		// Validate against CUE schema — use the root definition value so that
		// definition-based schemas (#Configuration etc.) are validated correctly.
		rootVal, err := schema.RootValue(getSchema())
		if err != nil {
			log.Printf("Schema root error: %v", err)
			renderFormWithError(w, tmpl, currentFormData, updatedValues, fmt.Sprintf("Schema root error: %v", err))
			return
		}
		if err := config.Validate(jsonBytes, rootVal); err != nil {
			log.Printf("Validation error: %v", err)
			renderFormWithError(w, tmpl, currentFormData, updatedValues, fmt.Sprintf("Validation error: %v", err))
			return
		}

		// Write validated JSON to config file
		if err := os.WriteFile(configPath, jsonBytes, 0644); err != nil {
			log.Printf("Failed to save config: %v", err)
			renderFormWithError(w, tmpl, currentFormData, updatedValues, fmt.Sprintf("Failed to save config: %v", err))
			return
		}

		savedAt := url.QueryEscape(time.Now().Format(time.RFC3339))
		http.Redirect(w, r, "/?saved="+savedAt, http.StatusSeeOther)
	})

	if cfg.watcher != nil {
		mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher.Flush()

			ch := cfg.watcher.Subscribe()
			defer cfg.watcher.Unsubscribe(ch)

			for {
				select {
				case <-r.Context().Done():
					return
				case <-ch:
					fmt.Fprintf(w, "event: reload\ndata: schema changed\n\n")
					flusher.Flush()
				}
			}
		})
	}

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
