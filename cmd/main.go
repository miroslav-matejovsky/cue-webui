package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/storage"
	"github.com/miroslav-matejovsky/cue-webui/webui"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Usage: cue-webui [flags] <schema.cue>\n  Flags:\n    -addr string  address to listen on (default \"localhost:8080\")")
	}

	schemaPath := flag.Arg(0)
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	ctx := cuecontext.New()
	rootValue := ctx.CompileString(string(schemaBytes))
	if rootValue.Err() != nil {
		log.Fatalf("Failed to compile CUE schema: %v", rootValue.Err())
	}

	formData, err := webui.BuildFormData(rootValue)
	if err != nil {
		log.Fatalf("Failed to build form data: %v", err)
	}

	handler, err := webui.NewHandlerWithStorage(formData, storage.NewMock(nil))
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	log.Printf("Serving on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
