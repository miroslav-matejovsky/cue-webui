package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/webui"
)

//go:embed schema.cue
var schemaFile string

//go:embed templates/form.html
var formTemplateStr string

//go:embed static/style.css
var cssFile string

func main() {
	ctx := cuecontext.New()
	rootValue := ctx.CompileString(schemaFile)
	if rootValue.Err() != nil {
		log.Fatalf("Failed to compile CUE schema: %v", rootValue.Err())
	}

	formData := webui.BuildFormData(rootValue)

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
		var values []webui.KeyValue
		for key, vals := range r.PostForm {
			values = append(values, webui.KeyValue{Key: key, Value: strings.Join(vals, ", ")})
		}
		sort.Slice(values, func(i, j int) bool {
			return values[i].Key < values[j].Key
		})
		result := webui.ResultData{Title: formData.Title, Values: values}
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
