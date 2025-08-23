//go:build ignore

package main

import (
	"os"
	"path"

	. "github.com/0-mqix/packed"
)

func main() {

	// A := Struct("A", true,
	// 	Field("A", Int32),
	// 	Field("B", Int32),
	// )

	// B := Struct("B", true,
	// 	Field("A", Array(2, Array(2, Array(2, Int32)))),
	// 	Field("B", Array(2, A)),
	// 	Field("C", Float32),
	// 	Field("D", String(10)),
	// 	Field("X", Bits[uint8](2)),
	// 	Field("Y", Bits[uint8](4)),
	// )

	// Struct("C", true,
	// 	Field("A", Array(2, Array(2, B))),
	// )

	// Struct("D", true,
	// 	Field("A", Bits[uint8](1)),
	// 	Field("B", Bits[uint8](1)),
	// 	Field("C", Bits[uint16](12)),
	// )

	A := Struct("A", true,
		Field("A", Bits[uint8](4)),
		Field("B", Bits[uint16](12)),
		Field("C", String(5)),
		Field("D", Bits[uint16](14)),
		Field("E", Bits[uint64](50)),
		Field("F", String(5)),
		Field("G", Bits[int8](4)),
		Field("H", Bits[int16](12)),
	)

	Struct("B", true,
		Field("A", Array(3, A)),
	)

	workingDirectory, _ := os.Getwd()

	generated := path.Join(workingDirectory, "/output.go")

	Generate(generated, "packed")
}
