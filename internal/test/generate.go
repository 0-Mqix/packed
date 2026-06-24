//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"

	. "github.com/0-Mqix/packed"
	"github.com/0-Mqix/packed/internal/test/types"
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

	Struct("N", false,
		Field("A", Bits[uint8](4)),
		Field("B", Bits[int8](3)),
	)

	// O and P: a contiguous run longer than 64 bits where a signed field (C)
	// straddles the 64-bit word boundary, little- and big-endian.
	Struct("O", true,
		Field("A", Bits[uint32](20)),
		Field("B", Bits[uint64](35)),
		Field("C", Bits[int16](13)),
		Field("D", Bits[uint8](7)),
		Field("E", Bit),
	)

	Struct("P", false,
		Field("A", Bits[uint32](20)),
		Field("B", Bits[uint64](35)),
		Field("C", Bits[int16](13)),
		Field("D", Bits[uint8](7)),
		Field("E", Bit),
	)

	// Q and R: a run longer than 128 bits spanning three words, with fields that
	// straddle both the 64-bit and 128-bit boundaries.
	Struct("Q", true,
		Field("A", Bits[uint64](50)),
		Field("B", Bits[uint64](50)),
		Field("C", Bits[uint64](40)),
		Field("D", Bits[int8](5)),
	)

	Struct("R", false,
		Field("A", Bits[uint64](50)),
		Field("B", Bits[uint64](50)),
		Field("C", Bits[uint64](40)),
		Field("D", Bits[int8](5)),
	)

	// S and T: the reproducer - 21 contiguous bit-fields totalling 77 bits, which
	// is 10 bytes under C __packed__ (the old splitter produced 11 and misaligned).
	Struct("S", true,
		Field("F0", Bits[uint32](5)), Field("F1", Bits[uint32](5)), Field("F2", Bits[uint32](5)),
		Field("F3", Bits[uint32](5)), Field("F4", Bits[uint32](5)), Field("F5", Bits[uint32](5)),
		Field("F6", Bits[uint32](5)), Field("F7", Bits[uint32](5)), Field("F8", Bits[uint32](5)),
		Field("F9", Bits[uint32](5)), Field("F10", Bits[uint32](5)), Field("F11", Bits[uint32](5)),
		Field("F12", Bits[uint32](7)),
		Field("F13", Bits[uint32](1)), Field("F14", Bits[uint32](1)), Field("F15", Bits[uint32](1)),
		Field("F16", Bits[uint32](1)), Field("F17", Bits[uint32](2)), Field("F18", Bits[uint32](1)),
		Field("F19", Bits[uint32](1)), Field("F20", Bits[uint32](2)),
	)

	Struct("T", false,
		Field("F0", Bits[uint32](5)), Field("F1", Bits[uint32](5)), Field("F2", Bits[uint32](5)),
		Field("F3", Bits[uint32](5)), Field("F4", Bits[uint32](5)), Field("F5", Bits[uint32](5)),
		Field("F6", Bits[uint32](5)), Field("F7", Bits[uint32](5)), Field("F8", Bits[uint32](5)),
		Field("F9", Bits[uint32](5)), Field("F10", Bits[uint32](5)), Field("F11", Bits[uint32](5)),
		Field("F12", Bits[uint32](7)),
		Field("F13", Bits[uint32](1)), Field("F14", Bits[uint32](1)), Field("F15", Bits[uint32](1)),
		Field("F16", Bits[uint32](1)), Field("F17", Bits[uint32](2)), Field("F18", Bits[uint32](1)),
		Field("F19", Bits[uint32](1)), Field("F20", Bits[uint32](2)),
	)

	workingDirectory, _ := os.Getwd()

	generated := path.Join(workingDirectory, "/output.go")

	Generate(generated, "packed", func(buffer *bytes.Buffer, name string, properties []Property) {
		for _, p := range properties {
			switch p.Type.Kind() {
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fmt.Fprintf(buffer, "func (r *%s) Get%s() int { return int(r.%s) }\n", name, p.Name, p.Name)
			case reflect.Bool:
				fmt.Fprintf(buffer, "func (r *%s) Get%s() bool { return r.%s }\n", name, p.Name, p.Name)
			}
		}
	})
}
