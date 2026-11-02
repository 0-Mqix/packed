// Package generate writes the files for the structures registered with
// package packed: Go, and whichever of C and C++, Kotlin, Swift and
// TypeScript a project asks for. It is the only part of packed that depends on
// golang.org/x/tools, so import it from generator programs
// (//go:build ignore) only, never from code that ships.
package generate

import (
	"os"
	"path/filepath"

	"github.com/0-Mqix/packed"
	"golang.org/x/tools/imports"
)

// An Output is one file Files writes: every registered enum, struct and
// message set in one language, with the same byte layout as the others.
type Output struct {
	file string
	// source returns the file's content; on an error, the content to
	// write anyway so the failure can be inspected.
	source func() ([]byte, error)
}

// Files writes each output, creating its folder, so a generator lists only
// the languages its project uses:
//
//	generate.Files(
//		generate.GoFile("messages.go", "messages"),
//		generate.HeaderFile("../firmware/messages.h"),
//	)
//
// It panics when an output can't be rendered or written.
func Files(outputs ...Output) {
	for _, output := range outputs {
		content, failure := output.source()

		if err := os.MkdirAll(filepath.Dir(output.file), 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(output.file, content, 0644); err != nil {
			panic(err)
		}
		if failure != nil {
			panic("failed to generate " + output.file + ": " + failure.Error())
		}
	}
}

// GoFile is the Go file, in package packageName, gofmt'ed and with imports
// resolved by goimports. On a formatting error the raw source is written.
func GoFile(file string, packageName string, hooks ...packed.GenerateHook) Output {
	return Output{file, func() ([]byte, error) {
		source := packed.Source(packageName, hooks...)

		result, err := imports.Process("", source, &imports.Options{
			AllErrors:  true,
			FormatOnly: false,
			Comments:   true,
		})
		if err != nil {
			return source, err
		}
		return result, nil
	}}
}

// KotlinFile is the Kotlin file, in package packageName.
func KotlinFile(file string, packageName string) Output {
	return Output{file, func() ([]byte, error) {
		return append(packed.KotlinSource(packageName), '\n'), nil
	}}
}

// SwiftFile is the Swift file. It imports imports: the module SwiftErrorFile
// is in, when it is not in the file's own.
func SwiftFile(file string, imports ...string) Output {
	return Output{file, func() ([]byte, error) {
		return append(packed.SwiftSource(imports...), '\n'), nil
	}}
}

// SwiftErrorFile is PackedError: once, in one module, for all the Swift
// files SwiftFile writes in it and in modules that import it.
func SwiftErrorFile(file string) Output {
	return Output{file, func() ([]byte, error) {
		return append(packed.SwiftErrorSource(), '\n'), nil
	}}
}

// TypeScriptFile is the TypeScript module. It imports its reader and writer
// from runtime: packed's typescript/packed.ts, as the file is to import it.
func TypeScriptFile(file string, runtime string) Output {
	return Output{file, func() ([]byte, error) {
		return append(packed.TypeScriptSource(runtime), '\n'), nil
	}}
}

// HeaderFile is the header C11 and C++20 both include: C structs and
// functions, with methods, enum classes and a list's [] for C++. No heap
// and nothing beyond <stdint.h>, <stddef.h>, <stdbool.h> and <string.h>, so
// it builds for microcontrollers (avr-gcc) as well as hosts.
func HeaderFile(file string) Output {
	return Output{file, func() ([]byte, error) {
		return append(packed.HeaderSource(), '\n'), nil
	}}
}

// Generate writes the Go file alone: Files(GoFile(...)).
func Generate(outputFile string, packageName string, hooks ...packed.GenerateHook) {
	Files(GoFile(outputFile, packageName, hooks...))
}

// Kotlin writes the Kotlin file alone: Files(KotlinFile(...)).
func Kotlin(outputFile string, packageName string) {
	Files(KotlinFile(outputFile, packageName))
}

// Swift writes the Swift file alone: Files(SwiftFile(...)).
func Swift(outputFile string, imports ...string) {
	Files(SwiftFile(outputFile, imports...))
}

// SwiftError writes PackedError alone: Files(SwiftErrorFile(...)).
func SwiftError(outputFile string) {
	Files(SwiftErrorFile(outputFile))
}

// TypeScript writes the TypeScript module alone: Files(TypeScriptFile(...)).
func TypeScript(outputFile string, runtime string) {
	Files(TypeScriptFile(outputFile, runtime))
}

// Header writes the C and C++ header alone: Files(HeaderFile(...)).
func Header(outputFile string) {
	Files(HeaderFile(outputFile))
}
