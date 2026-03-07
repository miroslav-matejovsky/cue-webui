package main

import (
	_ "embed"
	"log"
	"net/http"

	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/webui"
)

//go:embed schema.cue
var schemaFile string

func main() {
	ctx := cuecontext.New()
	rootValue := ctx.CompileString(schemaFile)
	if rootValue.Err() != nil {
		log.Fatalf("Failed to compile CUE schema: %v", rootValue.Err())
	}

	formData, err := webui.BuildFormData(rootValue)
	if err != nil {
		log.Fatalf("Failed to build form data: %v", err)
	}

	handler, err := webui.NewHandler(formData)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	addr := "localhost:8080"
	log.Printf("Platform stack example running on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
