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

	v1 := rootValue.LookupPath(cue.ParsePath("#V1"))
	v2 := rootValue.LookupPath(cue.ParsePath("#V2"))
	v3 := rootValue.LookupPath(cue.ParsePath("#V3"))

	fmt.Println("V2 is backwards compatible with V1:", v2.Subsume(v1) == nil)
	fmt.Println("V3 is backwards compatible with V2:", v3.Subsume(v2) == nil)
}