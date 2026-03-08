// Package schema provides helpers for converting a compiled CUE schema value
// into a form suitable for JSON Schema generation.
package schema

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

// RootValue returns the cue.Value that should be passed to jsonschema.Generate
// to produce a meaningful JSON Schema for the given CUE schema.
//
// If a definition carries a UI_Root: true hint, that definition is returned.
//
// Otherwise, the function finds the single "root" definition using the same
// heuristic as BuildFormData (the definition that contains struct sub-fields).
// If multiple roots exist an error is returned. If no definitions are found
// the schema value itself is returned unchanged.
func RootValue(cueSchema cue.Value) (cue.Value, error) {
	iter, err := cueSchema.Fields(cue.Definitions(true))
	if err != nil {
		return cueSchema, nil
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
		return cueSchema, nil
	}

	// Check for explicit UI_Root: true hint.
	rootDef, err := webform.FindRootDef(cueSchema)
	if err != nil {
		return cue.Value{}, err
	}
	if rootDef != "" {
		sel := cue.Def("#" + rootDef)
		for _, d := range allDefs {
			if d.sel == sel {
				return cueSchema.LookupPath(cue.MakePath(d.sel)), nil
			}
		}
	}

	var roots []defEntry
	for _, d := range allDefs {
		if webform.HasStructFields(d.val) {
			roots = append(roots, d)
		}
	}
	if len(roots) == 1 {
		return cueSchema.LookupPath(cue.MakePath(roots[0].sel)), nil
	}
	if len(roots) > 1 {
		return cue.Value{}, fmt.Errorf("multiple root definitions found; add UI_Root: true to exactly one definition")
	}
	return cueSchema, nil
}
