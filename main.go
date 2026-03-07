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

	formData := webui.BuildFormData(rootValue)
	handler := webui.NewHandler(formData)

	addr := "localhost:8080"
	log.Printf("Server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
