// Command dynamic writes packed's dynamic layer for the package in the
// current directory (or the one given) into its packed_dynamic.go: for
// each type that embeds packed.Dynamic, its Schema, AppendPacked and
// Decode<Type>. From a package:
//
//	//go:generate go run github.com/0-Mqix/packed/dynamic
//
// generate.Dynamic does the same from Go.
package main

import (
	"fmt"
	"os"

	"github.com/0-Mqix/packed/generate"
)

func main() {
	directory := "."
	if len(os.Args) > 1 {
		directory = os.Args[1]
	}

	if err := generate.Dynamic(directory); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
