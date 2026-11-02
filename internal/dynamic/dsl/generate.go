//go:build ignore

// Generates output.go: dynamic.Small and dynamic.Inner defined in the DSL,
// which TestSmallMatchesTheDSL encodes beside the Go types. go run
// generate.go [directory], directory defaulting to this one.
package main

import (
	"os"
	"path/filepath"

	. "github.com/0-Mqix/packed"
	"github.com/0-Mqix/packed/generate"
)

func main() {
	directory := "."
	if len(os.Args) > 1 {
		directory = os.Args[1]
	}

	Inner := Struct("Inner", true,
		Field("On", Bit),
		Field("Value", Uint32),
	)

	Struct("Small", true,
		Field("Active", Bit),
		Field("Visible", Bit),
		Field("Count", Uint16),
		Field("Total", Int64),
		Field("Name", VariableString(Uint32)),
		Field("Ratio", Float32),
		Field("Level", Int8),
		Field("Values", Slice(Uint32, Int32)),
		Field("Inner", Inner),
		Field("Codes", Array(2, Uint16)),
	)

	generate.Generate(filepath.Join(directory, "output.go"), "dsl")
}
