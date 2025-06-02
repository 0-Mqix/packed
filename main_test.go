package packed_test

import (
	"fmt"
	. "packed"
	"packed/test"
	"testing"
)

func TestStruct(t *testing.T) {

	// RegisterConverter(CustomType, "Custom")

	var A = Struct("A", true,
		Field[Int8]("A"),
		Field[Int8]("B", Bits(3)),
		Field[Int8]("C", Bits(2)),
		Field[Int16]("D"),
	)

	var D = Struct("D", true,
		Field[Int8]("A", Bits(1)),
		Field[Int8]("B", Bits(2)),
		Field[test.Monkey]("Converter", LittleEndian(false)),
		Field[test.Custom]("Custom"),
	)

	Struct("B", true,
		Field[any]("A", Type(A)),
		Field[Int8]("B", Tag("json", "b")),
		Field[any]("C", Type(D), Bits(3)),
	)

	Generate()
}

func TestBits(t *testing.T) {

	mask := ((1 << (3 - 1)) - 1) << 1
	chunk := (1 & mask) >> 1

	fmt.Println(mask)
	fmt.Println(chunk)
}
