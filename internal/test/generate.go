//go:build ignore

package main

import (
	"os"
	"path"

	. "github.com/0-mqix/packed"
)

func main() {

	A := Struct("A", true,
		Field[Int32]("A"),
		Field[Int32]("B"),
	)

	B := Struct("B", true,
		Field[any]("A", Type(Array(2, Array(2, Array(2, Int32{}))))),
		Field[any]("B", Type(Array(2, A))),
		Field[Float32]("C"),
		Field[any]("D", Type(String(10))),
	)

	Struct("C", true,
		Field[any]("A", Type(Array(2, Array(2, B)))),
	)

	workingDirectory, _ := os.Getwd()

	generated := path.Join(workingDirectory, "output.go")

	Generate(generated, "packed")
}
