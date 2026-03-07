package main

import (
	_ "embed"
	"fmt"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"
)

//go:embed schema.cue
var schemaFile string

// commentText joins all comment lines in a group into a single trimmed string.
func commentText(g *ast.CommentGroup) string {
	var parts []string
	for _, c := range g.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

// docGroups returns all comment groups attached before the node (Position==0, not inline).
func docGroups(n ast.Node) []*ast.CommentGroup {
	var result []*ast.CommentGroup
	for _, g := range ast.Comments(n) {
		if g.Position == 0 && !g.Line {
			result = append(result, g)
		}
	}
	return result
}

func main() {
	f, err := parser.ParseFile("schema.cue", schemaFile, parser.ParseComments)
	if err != nil {
		fmt.Println("Parse error:", err)
		return
	}

	// structDocs holds per-field docs collected across all #Configuration blocks.
	// Key = field label, value = slice of doc texts (one per occurrence).
	type fieldInfo struct {
		typ  string
		docs []string
	}
	definitionDocs := []string{}
	fields := map[string]*fieldInfo{}
	fieldOrder := []string{}

	for _, decl := range f.Decls {
		topField, ok := decl.(*ast.Field)
		if !ok {
			continue
		}
		label, _, _ := ast.LabelName(topField.Label)
		if label != "#Configuration" {
			continue
		}

		// Collect docs on the definition itself.
		for _, g := range docGroups(topField) {
			definitionDocs = append(definitionDocs, commentText(g))
		}

		structLit, ok := topField.Value.(*ast.StructLit)
		if !ok {
			continue
		}
		for _, elt := range structLit.Elts {
			f2, ok := elt.(*ast.Field)
			if !ok {
				continue
			}
			fieldLabel, _, _ := ast.LabelName(f2.Label)
			typStr := ""
			if ident, ok := f2.Value.(*ast.Ident); ok {
				typStr = ident.Name
			}
			if _, exists := fields[fieldLabel]; !exists {
				fields[fieldLabel] = &fieldInfo{typ: typStr}
				fieldOrder = append(fieldOrder, fieldLabel)
			}
			for _, g := range docGroups(f2) {
				fields[fieldLabel].docs = append(fields[fieldLabel].docs, commentText(g))
			}
		}
	}

	fmt.Println("=== #Configuration ===")
	for i, d := range definitionDocs {
		fmt.Printf("Doc[%d]: %s\n", i+1, d)
	}
	fmt.Println("---")

	for _, label := range fieldOrder {
		info := fields[label]
		fmt.Printf("Field:  %s\n", label)
		if info.typ != "" {
			fmt.Printf("Type:   %s\n", info.typ)
		}
		for i, d := range info.docs {
			fmt.Printf("Doc[%d]: %s\n", i+1, d)
		}
		fmt.Println("---")
	}
}
