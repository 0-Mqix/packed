package packed

import (
	"reflect"
	"testing"

	types "github.com/0-mqix/packed/internal/test/types"
)

func TestStructTags(t *testing.T) {

	aType := reflect.TypeOf(A{})

	if aType.NumField() != 7 {
		t.Errorf("Expected A to have 7 fields, got %d", aType.NumField())
	}

	expectedTags := map[string]map[string]string{
		"A": {"json": "a", "xml": "a"},
		"B": {"json": "b", "xml": "b"},
		"C": {"json": "c", "xml": "c"},
		"D": {"json": "d", "xml": "d"},
		"E": {"json": "e", "xml": "e"},
		"F": {"json": "f", "xml": "f"},
	}

	for i := 0; i < aType.NumField() && i < 6; i++ {
		field := aType.Field(i)
		fieldName := field.Name

		if expected, ok := expectedTags[fieldName]; ok {
			jsonTag := field.Tag.Get("json")
			xmlTag := field.Tag.Get("xml")

			if jsonTag != expected["json"] {
				t.Errorf("field %s json tag: expected %q, got %q", fieldName, expected["json"], jsonTag)
			}
			if xmlTag != expected["xml"] {
				t.Errorf("field %s xml tag: expected %q, got %q", fieldName, expected["xml"], xmlTag)
			}
		}
	}

	bType := reflect.TypeOf(B{})
	if bType.NumField() != 6 {
		t.Errorf("Expected B to have 6 fields, got %d", bType.NumField())
	}

	for i := 0; i < bType.NumField(); i++ {
		field := bType.Field(i)
		fieldName := field.Name

		if expected, ok := expectedTags[fieldName]; ok {
			jsonTag := field.Tag.Get("json")
			xmlTag := field.Tag.Get("xml")

			if jsonTag != expected["json"] {
				t.Errorf("field %s json tag: expected %q, got %q", fieldName, expected["json"], jsonTag)
			}
			if xmlTag != expected["xml"] {
				t.Errorf("field %s xml tag: expected %q, got %q", fieldName, expected["xml"], xmlTag)
			}
		}
	}
}

func TestBitPackedStructsBothEndian(t *testing.T) {

	bytesFromCompilerStructB := []byte{255, 255, 255, 255, 255, 254, 121, 46, 120}
	bytesFromCompilerStructC := []byte{255, 255, 255, 255, 187, 228, 249, 255, 135}

	b := B{
		A: 15,
		B: 1023,
		C: 1048575,
		D: -100050,
		E: 7,
		F: -8,
	}

	c := C{
		A: 15,
		B: 1023,
		C: 1048575,
		D: -100050,
		E: 7,
		F: -8,
	}

	bytesB := make([]byte, b.Size())
	bytesC := make([]byte, c.Size())

	b.ToBytes(bytesB, 0)
	c.ToBytes(bytesC, 0)

	if !reflect.DeepEqual(bytesB, bytesFromCompilerStructB) {
		t.Errorf("c bytes b: expected %v, got %v", bytesFromCompilerStructB, bytesB)
	}

	if !reflect.DeepEqual(bytesC, bytesFromCompilerStructC) {
		t.Errorf("c bytes c: expected %v, got %v", bytesFromCompilerStructC, bytesC)
	}

	var resultB B
	resultB.FromBytes(bytesB, 0)

	if !reflect.DeepEqual(b, resultB) {
		t.Errorf("b: expected %v, got %v", b, resultB)
	}

	var resultC C
	resultC.FromBytes(bytesC, 0)

	if !reflect.DeepEqual(c, resultC) {
		t.Errorf("c: expected %v, got %v", c, resultC)
	}
}

func TestArrayOfStructOfBitPackedStructsBothEndian(t *testing.T) {

	definition := E{
		A: [2]D{
			{A: B{A: 15, B: 1023, C: 1048575, D: -100050, E: 7, F: -8}, B: C{A: 15, B: 1023, C: 1048575, D: -100050, E: 7, F: -8}},
			{A: B{A: 15, B: 1023, C: 1048575, D: -100050, E: 7, F: -8}, B: C{A: 15, B: 1023, C: 1048575, D: -100050, E: 7, F: -8}},
		},
	}

	bytes := make([]byte, definition.Size())
	definition.ToBytes(bytes, 0)

	var result E
	result.FromBytes(bytes, 0)

	if !reflect.DeepEqual(definition, result) {
		t.Errorf("e: expected %v, got %v", definition, result)
	}
}

func TestMatrixOfExampleTypeInterface(t *testing.T) {

	definition := F{
		A: [2][2][2]types.ExampleTypeInterface{
			{
				{{A: 1}, {A: 2}},
				{{A: 3}, {A: 4}},
			},
			{
				{{A: 5}, {A: 6}},
				{{A: 7}, {A: 8}},
			},
		},
	}

	bytes := make([]byte, definition.Size())
	definition.ToBytes(bytes, 0)

	var result F
	result.FromBytes(bytes, 0)

	if !reflect.DeepEqual(definition, result) {
		t.Errorf("f: expected %v, got %v", definition, result)
	}
}

func TestMatrixOfExampleConverterBitArray(t *testing.T) {

	g := G{
		A: [2][2][2]types.ExampleRecieverType{
			{
				{{A: 1}, {A: 2}},
				{{A: 3}, {A: 4}},
			},
			{
				{{A: 5}, {A: 6}},
				{{A: 7}, {A: 8}},
			},
		},
	}

	bytesG := make([]byte, g.Size())
	g.ToBytes(bytesG, 0)

	var resultG G
	resultG.FromBytes(bytesG, 0)

	if !reflect.DeepEqual(g, resultG) {
		t.Errorf("g: expected %v, got %v", g, resultG)
	}
}

func TestConverterCast(t *testing.T) {

	definition := I{
		A: types.ExampleEnumValueA,
		B: [2]types.ExampleEnum{types.ExampleEnumValueB, types.ExampleEnumValueC},
		C: [2]H{
			{A: types.ExampleEnumValueA},
			{A: types.ExampleEnumValueB},
		},
		D: types.ExampleEnumStringValueA,
	}

	bytes := make([]byte, definition.Size())
	definition.ToBytes(bytes, 0)

	var result I
	result.FromBytes(bytes, 0)

	if !reflect.DeepEqual(definition, result) {
		t.Errorf("i: expected %v, got %v", definition, result)
	}
}
