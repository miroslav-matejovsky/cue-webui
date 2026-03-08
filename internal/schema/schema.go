// Package schema provides helpers for converting a compiled CUE schema value
// into a form suitable for JSON Schema generation.
package schema

import (
	"cuelang.org/go/cue"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

// RootValue returns the cue.Value that should be passed to jsonschema.Generate
// to produce a meaningful JSON Schema for the given CUE schema.
//
// When the schema contains CUE definitions (#Name), the function finds the
// single "root" definition using the same heuristic as BuildFormData (the
// definition that contains struct sub-fields). If multiple roots exist the
// whole schema value is returned so all definitions appear in $defs. If no
// definitions are found the schema value itself is returned unchanged.
func RootValue(cueSchema cue.Value) cue.Value {
	iter, err := cueSchema.Fields(cue.Definitions(true))
	if err != nil {
		return cueSchema
	}

	type defEntry struct {
		sel cue.Selector
		val cue.Value
	}
	var allDefs []defEntry
	for iter.Next() {
		allDefs = append(allDefs, defEntry{iter.Selector(), iter.Value()})
	}
	if len(allDefs) == 0 {
		return cueSchema
	}

	var roots []defEntry
	for _, d := range allDefs {
		if webform.HasStructFields(d.val) {
			roots = append(roots, d)
		}
	}
	if len(roots) == 1 {
		return cueSchema.LookupPath(cue.MakePath(roots[0].sel))
	}
	return cueSchema
}
