package watcher

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/fsnotify/fsnotify"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

// SchemaWatcher watches a CUE schema file for changes and notifies subscribers.
type SchemaWatcher struct {
	schemaPath string
	fsWatcher  *fsnotify.Watcher

	mu       sync.RWMutex
	formData webform.FormData
	schema   cue.Value

	// subscribers receive a signal when the schema changes.
	subsMu sync.Mutex
	subs   map[chan struct{}]struct{}
}

// New creates a SchemaWatcher for the given schema file path with initial form data and schema.
func New(schemaPath string, formData webform.FormData, schema cue.Value) (*SchemaWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating file watcher: %w", err)
	}
	if err := fsw.Add(schemaPath); err != nil {
		fsw.Close()
		return nil, fmt.Errorf("watching schema file %s: %w", schemaPath, err)
	}

	w := &SchemaWatcher{
		schemaPath: schemaPath,
		fsWatcher:  fsw,
		formData:   formData,
		schema:     schema,
		subs:       make(map[chan struct{}]struct{}),
	}

	go w.run()
	return w, nil
}

// FormData returns the current form data (thread-safe).
func (w *SchemaWatcher) FormData() webform.FormData {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.formData
}

// Schema returns the current CUE schema value (thread-safe).
func (w *SchemaWatcher) Schema() cue.Value {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.schema
}

// Subscribe returns a channel that receives a signal whenever the schema changes.
// Call Unsubscribe when done.
func (w *SchemaWatcher) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	w.subsMu.Lock()
	w.subs[ch] = struct{}{}
	w.subsMu.Unlock()
	return ch
}

// Unsubscribe removes a previously subscribed channel.
func (w *SchemaWatcher) Unsubscribe(ch chan struct{}) {
	w.subsMu.Lock()
	delete(w.subs, ch)
	w.subsMu.Unlock()
}

// Close stops watching the file.
func (w *SchemaWatcher) Close() error {
	return w.fsWatcher.Close()
}

func (w *SchemaWatcher) run() {
	// Debounce rapid writes — many editors write multiple events.
	var debounce *time.Timer

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(200*time.Millisecond, func() {
				w.reload()
			})

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *SchemaWatcher) reload() {
	schemaBytes, err := os.ReadFile(w.schemaPath)
	if err != nil {
		log.Printf("live reload: failed to read schema file: %v", err)
		return
	}

	ctx := cuecontext.New()
	cueSchema := ctx.CompileString(string(schemaBytes))
	if cueSchema.Err() != nil {
		log.Printf("live reload: failed to compile CUE schema: %v", cueSchema.Err())
		return
	}

	formData, err := webform.BuildFormData(cueSchema)
	if err != nil {
		log.Printf("live reload: failed to build form data: %v", err)
		return
	}

	w.mu.Lock()
	w.formData = formData
	w.schema = cueSchema
	w.mu.Unlock()

	log.Printf("live reload: schema reloaded from %s", w.schemaPath)
	w.notify()
}

func (w *SchemaWatcher) notify() {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	for ch := range w.subs {
		select {
		case ch <- struct{}{}:
		default:
			// channel already has a pending signal, skip
		}
	}
}
