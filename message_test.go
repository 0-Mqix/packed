package packed

import (
	"fmt"
	"strings"
	"testing"
)

func panics(t *testing.T, want string, define func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recovered := recover(); recovered == nil || !strings.Contains(fmt.Sprint(recovered), want) {
			t.Errorf("got %v, want a panic with %q", recovered, want)
		}
	}()
	define()
}

func TestMethodNamesAreNoFieldNames(t *testing.T) {
	panics(t, "field MethodSize.Size has the name of the Size method every struct gets: rename the field", func() {
		Struct("MethodSize", true, Field("Key", Uint16), Field("Size", Uint8))
	})
	panics(t, "field MethodToBytes.ToBytes has the name of the ToBytes method every struct gets", func() {
		Struct("MethodToBytes", true, Field("ToBytes", Bit))
	})
	panics(t, "field MethodFromBytes.FromBytes has the name of the FromBytes method every struct gets", func() {
		Struct("MethodFromBytes", true, Field("FromBytes", VariableString(Uint8)))
	})
	panics(t, "field MethodVariant.MethodSetType has the name of the MethodSetType method message set MethodSet gives its variants", func() {
		MessageSet("MethodSet", TagBits(2), Variant(1, Struct("MethodVariant", true, Field("MethodSetType", Bits[uint8](3)))))
	})
}

func TestTagBitsDefinitionErrors(t *testing.T) {
	panics(t, "TagBits(9) is not between 0 and 8", func() { TagBits(9) })

	panics(t, "tags TagTooWideVariant with 4, which does not fit in 2 bits", func() {
		MessageSet("TagTooWide", TagBits(2), Variant(4, Struct("TagTooWideVariant", true)))
	})

	panics(t, "message set TagNone has 2 variants and no tag bits", func() {
		MessageSet("TagNone", TagBits(0), Variant(0, Struct("TagNoneFirst", true)), Variant(1, Struct("TagNoneSecond", true)))
	})

	panics(t, "message set TagLate has TagBits after a variant", func() {
		MessageSet("TagLate", Variant(0, Struct("TagLateVariant", true)), TagBits(1))
	})

	panics(t, "so TagMixedBig needs the endianness of TagMixedLittle", func() {
		MessageSet("TagMixed", TagBits(4), Variant(0, Struct("TagMixedLittle", true)), Variant(1, Struct("TagMixedBig", false)))
	})

	colors := Enum("TagColor", Uint8, "Red", "Green", "Blue")
	panics(t, "enum TagColor has more values than 1 bits can hold", func() { colors.Bits(1) })
	panics(t, "enum TagColor cannot be 9 bits", func() { colors.Bits(9) })

	nested := Struct("TagNested", true, Field("Count", Uint8))
	Struct("TagOuterBefore", true, Field("Inner", nested))
	panics(t, "struct TagOuterBefore has TagNested as a field, which carries the tag of message set TagNestedSet", func() {
		MessageSet("TagNestedSet", TagBits(1), Variant(0, nested))
	})

	later := Struct("TagLater", true)
	MessageSet("TagLaterSet", TagBits(1), Variant(1, later))
	panics(t, "struct TagOuterAfter has TagLater as a field, which carries the tag of message set TagLaterSet", func() {
		Struct("TagOuterAfter", true, Field("Items", Slice(Uint8, later)))
	})
}

// TestTagBitsJoinTheFirstBitRun checks the size each variant gets: the tag
// joins a leading bit run while it has room and takes bytes of its own
// otherwise. Eight bits lay out as the one-byte form does.
func TestTagBitsJoinTheFirstBitRun(t *testing.T) {
	colorBits := Enum("JoinColor", Uint8, "Red", "Green", "Blue").Bits(2)

	cases := []struct {
		structure packedStruct
		bits      int
		size      int
	}{
		{Struct("JoinRoom", true, Field("On", Bit), Field("Level", Bits[uint8](4)), Field("Count", Uint16)), 3, 3},
		{Struct("JoinFull", true, Field("Level", Bits[uint8](6)), Field("Count", Uint16)), 3, 4},
		{Struct("JoinNone", true, Field("Count", Uint16)), 3, 3},
		{Struct("JoinEmpty", true), 3, 1},
		{Struct("JoinByte", false, Field("On", Bit)), 8, 2},
		{Struct("JoinEnum", true, Field("Color", colorBits), Field("On", Bit)), 4, 1},
		{Struct("JoinUntagged", true, Field("Count", Uint16)), 0, 2},
	}

	for _, test := range cases {
		MessageSet(test.structure.name+"Set", TagBits(test.bits), Variant(0, test.structure))
		if size := structs[test.structure.name].size; size != test.size {
			t.Errorf("%s with %d tag bits: %d bytes, want %d", test.structure.name, test.bits, size, test.size)
		}
	}
}
