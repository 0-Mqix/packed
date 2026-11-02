package packed

import (
	"bytes"
	"fmt"
	"reflect"
)

var enums = map[string]packedEnum{}

type packedEnum struct {
	name       string
	values     []string
	converter  converterHash
	reflection reflect.Type
	size       int
}

func (e packedEnum) Size() int { return e.size }

// Enum defines a named integer type with the given values, numbered from zero
// and stored with the given integer converter. Go gets a named type with one
// constant per value, other generators get their language's enum.
func Enum(name string, wire any, values ...string) packedEnum {

	if _, ok := enums[name]; ok {
		panic(fmt.Sprintf("enum %s already exists", name))
	}

	if _, ok := structs[name]; ok {
		panic(fmt.Sprintf("enum %s has the name of a struct", name))
	}

	if len(values) == 0 {
		panic(fmt.Sprintf("enum %s has no values", name))
	}

	wire = toPointer(wire)

	reciever, ok := implementsConverterInterface(wire)

	if !ok {
		panic(fmt.Sprintf("enum %s must be stored with an integer converter", name))
	}

	switch reciever.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	default:
		panic(fmt.Sprintf("enum %s must be stored with an integer converter, got %s", name, reciever))
	}

	size := wire.(interface{ Size() int }).Size()

	if uint64(len(values)-1) > bitMask(size*8) {
		panic(fmt.Sprintf("enum %s has more values than %s can hold", name, reciever))
	}

	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			panic(fmt.Sprintf("enum %s has value %s twice", name, value))
		}
		seen[value] = true
	}

	hash := createConverterHash(wire)

	if _, exists := converters[hash.hash]; !exists {
		converters[hash.hash] = hash
	}

	enum := packedEnum{
		name:       name,
		values:     values,
		converter:  hash,
		reflection: reciever,
		size:       size,
	}

	enums[name] = enum

	return enum
}

// Bits stores the enum as a bit field of bits bits instead of its integer
// converter, so it packs with the bit fields and Bools next to it. The
// value is unsigned on the wire, whatever the converter's sign.
func (e packedEnum) Bits(bits int) packedBitField {
	if bits < 1 || bits > e.size*8 {
		panic(fmt.Sprintf("enum %s cannot be %d bits", e.name, bits))
	}
	if uint64(len(e.values)-1) > bitMask(bits) {
		panic(fmt.Sprintf("enum %s has more values than %d bits can hold", e.name, bits))
	}
	enum := e
	return packedBitField{
		bitSize:      bits,
		reflection:   e.reflection,
		bitFieldKind: bitFieldKindEnum,
		enum:         &enum,
	}
}

func (e packedEnum) definition() []byte {
	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "type %s %s\n\n", e.name, e.reflection)

	fmt.Fprintf(buffer, "const (\n")
	for index, value := range e.values {
		if index == 0 {
			fmt.Fprintf(buffer, "%s%s %s = iota\n", e.name, value, e.name)
		} else {
			fmt.Fprintf(buffer, "%s%s\n", e.name, value)
		}
	}
	fmt.Fprintf(buffer, ")\n\n")

	fmt.Fprintf(buffer, "func (value %s) String() string {\n", e.name)
	fmt.Fprintf(buffer, "switch value {\n")
	for _, value := range e.values {
		fmt.Fprintf(buffer, "case %s%s:\n", e.name, value)
		fmt.Fprintf(buffer, "return %q\n", value)
	}
	fmt.Fprintf(buffer, "}\n")
	fmt.Fprintf(buffer, "return fmt.Sprintf(\"%s(%%d)\", %s(value))\n", e.name, e.reflection)
	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}

func (e packedEnum) write(buffer *bytes.Buffer, reciever string, functionName string, littleEndian bool, offset string) {
	fmt.Fprintf(buffer, "%s.%s%s((*%s)(&%s), bytes, %s)\n", getConverterName(e.converter.hash), functionName, endianName(littleEndian), e.reflection, reciever, offset)
}
