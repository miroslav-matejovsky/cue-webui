package app

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/watcher"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

func Run() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	liveReload := flag.Bool("live", false, "enable live reload on schema file changes")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Usage: cue-webui [flags] <schema.cue> <config.json>")
		fmt.Fprintln(os.Stderr, "  Flags:")
		fmt.Fprintln(os.Stderr, "    -addr string  address to listen on (default \"localhost:8080\")")
		fmt.Fprintln(os.Stderr, "    -live          enable live reload on schema file changes")
		os.Exit(1)
	}

	schemaPath := flag.Arg(0)
	configPath := flag.Arg(1)

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	ctx := cuecontext.New()
	cueSchema := ctx.CompileString(string(schemaBytes))
	if cueSchema.Err() != nil {
		log.Fatalf("Failed to compile CUE schema: %v", cueSchema.Err())
	}

	formData, err := webform.BuildFormData(cueSchema)
	if err != nil {
		log.Fatalf("Failed to build form data: %v", err)
	}

	var handlerOpts []webui.Option
	if *liveReload {
		w, err := watcher.New(schemaPath, formData, cueSchema)
		if err != nil {
			log.Fatalf("Failed to start schema watcher: %v", err)
		}
		defer w.Close()
		handlerOpts = append(handlerOpts, webui.WithWatcher(w))
		log.Printf("Live reload enabled for %s", schemaPath)
	}

	handler, err := webui.NewHandler(formData, cueSchema, configPath, handlerOpts...)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	log.Printf("Serving on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
