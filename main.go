package main

import (
	_ "embed"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:embed schema.cue
var schemaFile string

func main() {
	ctx := cuecontext.New()
	rootValue := ctx.CompileString(schemaFile)

	configPath := cue.ParsePath("#Configuration")
	config := rootValue.LookupPath(configPath)

	fmt.Println("=== #Configuration ===")
	iter, err := config.Fields(cue.Optional(true))
	if err != nil {
		fmt.Println("Error iterating fields:", err)
		return
	}
	for iter.Next() {
		field := iter.Value()
		label := iter.Selector()
		typ := field.IncompleteKind().String()
		doc := ""
		for _, d := range field.Doc() {
			doc += d.Text()
		}
		fmt.Printf("Field:  %s\n", label)
		fmt.Printf("Type:   %s\n", typ)
		if doc != "" {
			fmt.Printf("Doc:    %s", doc)
		}
		fmt.Println("---")
	}
}
