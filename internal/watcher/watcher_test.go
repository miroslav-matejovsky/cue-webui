package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
	"github.com/stretchr/testify/require"
)

func TestSchemaWatcher_ReloadsOnChange(t *testing.T) {
	// Write initial schema
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.cue")
	initialSchema := `#Config: { name: string }`
	require.NoError(t, os.WriteFile(schemaPath, []byte(initialSchema), 0644))

	ctx := cuecontext.New()
	cueVal := ctx.CompileString(initialSchema)
	require.NoError(t, cueVal.Err())

	formData, err := webform.BuildFormData(cueVal)
	require.NoError(t, err)

	w, err := New(schemaPath, formData, cueVal)
	require.NoError(t, err)
	defer w.Close()

	ch := w.Subscribe()
	defer w.Unsubscribe(ch)

	// Modify schema — add a new field
	updatedSchema := `#Config: { name: string, age: int }`
	require.NoError(t, os.WriteFile(schemaPath, []byte(updatedSchema), 0644))

	// Wait for reload notification
	select {
	case <-ch:
		// Verify updated form data has the new field
		fd := w.FormData()
		var fieldNames []string
		for _, s := range fd.Sections {
			for _, f := range s.Fields {
				fieldNames = append(fieldNames, f.Name)
			}
		}
		require.Contains(t, fieldNames, "age", "expected 'age' field after schema reload")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for schema reload notification")
	}
}

func TestSchemaWatcher_FormDataAndSchema(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.cue")
	schemaStr := `#Config: { host: string }`
	require.NoError(t, os.WriteFile(schemaPath, []byte(schemaStr), 0644))

	ctx := cuecontext.New()
	cueVal := ctx.CompileString(schemaStr)
	require.NoError(t, cueVal.Err())

	formData, err := webform.BuildFormData(cueVal)
	require.NoError(t, err)

	w, err := New(schemaPath, formData, cueVal)
	require.NoError(t, err)
	defer w.Close()

	require.Equal(t, formData.Title, w.FormData().Title)
	require.NoError(t, w.Schema().Err())
}

func TestSchemaWatcher_InvalidSchemaDoesNotCrash(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.cue")
	initialSchema := `#Config: { name: string }`
	require.NoError(t, os.WriteFile(schemaPath, []byte(initialSchema), 0644))

	ctx := cuecontext.New()
	cueVal := ctx.CompileString(initialSchema)
	require.NoError(t, cueVal.Err())

	formData, err := webform.BuildFormData(cueVal)
	require.NoError(t, err)

	w, err := New(schemaPath, formData, cueVal)
	require.NoError(t, err)
	defer w.Close()

	// Write invalid CUE — should log error but not crash, and keep old data
	require.NoError(t, os.WriteFile(schemaPath, []byte(`{ invalid !!!`), 0644))

	// Small delay to let the watcher process the event
	time.Sleep(500 * time.Millisecond)

	// Old form data should still be intact
	fd := w.FormData()
	require.Equal(t, formData.Title, fd.Title)
}
