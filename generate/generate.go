// Package generate writes the Go file for the structures registered with
// package packed. It is the only part of packed that depends on
// golang.org/x/tools, so import it from generator programs
// (//go:build ignore) only, never from code that ships.
package generate

import (
	"os"

	"github.com/0-Mqix/packed"
	"golang.org/x/tools/imports"
)

// Generate renders every registered structure into outputFile, gofmt'ed and
// with imports resolved by goimports. On a formatting error the raw source
// is written instead so the failure can be inspected, then it panics.
func Generate(outputFile string, packageName string, hooks ...packed.GenerateHook) {
	source := packed.Source(packageName, hooks...)

	result, err := imports.Process("", source, &imports.Options{
		AllErrors:  true,
		FormatOnly: false,
		Comments:   true,
	})

	if err != nil {
		os.WriteFile(outputFile, source, 0644)
		panic("failed to generate code: " + err.Error())
	}

	os.WriteFile(outputFile, result, 0644)
}
