//go:build ignore

package main

import (
	"os"
	"path"

	. "github.com/0-Mqix/packed"
	types "github.com/0-Mqix/packed/internal/test/types"
)

func main() {

	Struct("A", true,
		Field("A", Uint8, Tag("json", "a"), Tag("xml", "a")),
		Field("B", Uint16, Tag("json", "b"), Tag("xml", "b")),
		Field("C", Uint32, Tag("json", "c"), Tag("xml", "c")),
		Field("D", Int64, Tag("json", "d"), Tag("xml", "d")),
		Field("E", Int8, Tag("json", "e"), Tag("xml", "e")),
		Field("F", Int8, Tag("json", "f"), Tag("xml", "f")),
		Field("G", types.ExampleTypeInterface{}),
	)

	B := Struct("B", false,
		Field("A", Bits[uint8](4), Tag("json", "a"), Tag("xml", "a")),
		Field("B", Bits[uint16](10), Tag("json", "b"), Tag("xml", "b")),
		Field("C", Bits[uint32](20), Tag("json", "c"), Tag("xml", "c")),
		Field("D", Bits[int64](30), Tag("json", "d"), Tag("xml", "d")),
		Field("E", Bits[int8](4), Tag("json", "e"), Tag("xml", "e")),
		Field("F", Bit, Tag("json", "f"), Tag("xml", "f")),
		Field("G", Bits[int8](3), Tag("json", "g"), Tag("xml", "g")),
	)

	C := Struct("C", true,
		Field("A", Bits[uint8](4)),
		Field("B", Bits[uint16](10)),
		Field("C", Bits[uint32](20)),
		Field("D", Bits[int64](30)),
		Field("E", Bits[int8](4)),
		Field("F", Bit),
		Field("G", Bits[int8](3)),
	)

	D := Struct("D", true,
		Field("A", B),
		Field("B", C),
	)

	Struct("E", false,
		Field("A", Array(2, D)),
	)

	Struct("F", true,
		Field("A", Array(2, Array(2, Array(2, types.ExampleTypeInterface{})))),
	)

	Struct("G", true,
		Field("A", Array(2, Array(2, Array(2, types.ExampleConverter{})))),
	)

	H := Struct("H", false,
		Field("A", Cast[types.ExampleEnum](Int16)),
	)

	Struct("I", true,
		Field("A", Cast[types.ExampleEnum](Int32)),
		Field("B", Array(2, Cast[types.ExampleEnum](Int8))),
		Field("C", Array(2, H)),
		Field("D", Cast[types.ExampleEnumString](String(1))),
	)

	Struct("J", true,
		Field("A", Bits[uint8](6)),
		Field("B", Bits[uint16](10, types.ExampleBitsType{})),
	)

	K := Struct("K", false,
		Field("A", Bits[uint8](6)),
		Field("B", Bits[uint16](10, types.ExampleBitsType{})),
	)

	L := Struct("L", true,
		Field("A", Bits[uint8](4)),
		Field("B", Bits[uint16](10, types.ExampleBitsTypeConverter{})),
	)

	Struct("M", true,
		Field("A", Array(2, L)),
		Field("B", Array(2, K)),
	)

	workingDirectory, _ := os.Getwd()

	generated := path.Join(workingDirectory, "/output.go")

	Generate(generated, "packed")
}
