package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type lengthPrefix struct {
	converter  converterHash
	reflection reflect.Type
	size       int
}

func newLengthPrefix(converter any) lengthPrefix {
	converter = toPointer(converter)

	reciever, ok := implementsConverterInterface(converter)

	if !ok {
		panic("length prefix must be an unsigned integer converter such as Uint8 or Uint16")
	}

	switch reciever.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		panic(fmt.Sprintf("length prefix must be unsigned, got %s", reciever))
	}

	hash := createConverterHash(converter)

	if _, exists := converters[hash.hash]; !exists {
		converters[hash.hash] = hash
	}

	return lengthPrefix{
		converter:  hash,
		reflection: reciever,
		size:       converter.(interface{ Size() int }).Size(),
	}
}

func (l lengthPrefix) maximum() uint64 {
	return bitMask(l.size * 8)
}

// Slice is a variable-length list: a length prefix written with the given
// unsigned integer converter, followed by that many elements.
func Slice(length any, elementType any) packedSlice {

	kind, _, elementType := validatePropertyType(elementType)

	if kind == kindBitField {
		panic("bit fields as direct slice elements are not supported")
	}

	if kind == kindConverter {
		hash := createConverterHash(elementType)

		if _, exists := converters[hash.hash]; !exists {
			converters[hash.hash] = hash
		}
	}

	elementSize := 0
	if !isDynamic(kind, elementType) {
		elementSize = elementType.(interface{ Size() int }).Size()
	}

	return packedSlice{
		Length:       newLengthPrefix(length),
		Element:      elementType,
		ElementKind:  kind,
		ElementSize:  elementSize,
		recieverType: "[]" + getArrayRecieverType(elementType),
	}
}

type packedSlice struct {
	Length       lengthPrefix
	Element      any
	ElementKind  kind
	ElementSize  int
	recieverType string
	// maximum is the most items the list holds, 0 for no maximum.
	maximum int
}

func (s packedSlice) Size() int { return s.Length.size }

// Maximum caps the list at n items, which every language checks when it
// encodes and decodes it, and lets the C header hold the list in an array
// of n instead of pointing into the input. The bytes on the wire are the
// same: a list of one item is still its count and that item.
func (s packedSlice) Maximum(n int) packedSlice {
	if n < 1 {
		panic(fmt.Sprintf("Maximum(%d) is less than one item", n))
	}
	if uint64(n) > s.Length.maximum() {
		panic(fmt.Sprintf("Maximum(%d) is more than a %s length prefix can say", n, s.Length.reflection))
	}
	s.maximum = n
	return s
}

// VariableString is a UTF-8 string preceded by its length in bytes, written
// with the given unsigned integer converter.
func VariableString(length any) packedVariableString {
	return packedVariableString{Length: newLengthPrefix(length)}
}

// VariableBytes is raw bytes preceded by their count, written with the given
// unsigned integer converter: []byte in Go, ByteArray in Kotlin, [UInt8] in
// Swift, so a payload is never a list of boxed bytes.
func VariableBytes(length any) packedVariableString {
	return packedVariableString{Length: newLengthPrefix(length), Bytes: true}
}

type packedVariableString struct {
	Length lengthPrefix
	Bytes  bool
}

func (s packedVariableString) Size() int { return s.Length.size }

func isDynamic(kind kind, packed any) bool {
	switch kind {
	case kindSlice, kindVariableString:
		return true
	case kindStruct:
		return packed.(packedStruct).dynamic
	case kindArray:
		array := packed.(packedArray)
		return isDynamic(array.ElementKind, array.Element)
	}
	return false
}

func endianName(littleEndian bool) string {
	if littleEndian {
		return "LittleEndian"
	}
	return "BigEndian"
}

func (s packedSlice) write(buffer *bytes.Buffer, structure *packedStruct, reciever string, functionName string, littleEndian bool, offsetVariable string, depth int) {
	converter := getConverterName(s.Length.converter.hash)
	endian := endianName(littleEndian)

	buffer.WriteString("{\n")

	switch functionName {
	case "ToBytes":
		if s.Length.size < 8 {
			// uint64, so a 32-bit prefix compiles where int is 32 bits.
			fmt.Fprintf(buffer, "if uint64(len(%s)) > %d {\n", reciever, s.Length.maximum())
			fmt.Fprintf(buffer, "panic(\"packed: %s holds more items than its %s length prefix allows\")\n", strings.TrimPrefix(reciever, "reciever."), s.Length.reflection)
			buffer.WriteString("}\n")
		}
		if s.maximum > 0 {
			fmt.Fprintf(buffer, "if len(%s) > %d {\n", reciever, s.maximum)
			fmt.Fprintf(buffer, "panic(\"packed: %s holds more than its maximum of %d items\")\n", strings.TrimPrefix(reciever, "reciever."), s.maximum)
			buffer.WriteString("}\n")
		}
		fmt.Fprintf(buffer, "length := %s(len(%s))\n", s.Length.reflection, reciever)
		fmt.Fprintf(buffer, "%s.ToBytes%s(&length, bytes, %s)\n", converter, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, s.Length.size)

	case "FromBytes":
		fmt.Fprintf(buffer, "var length %s\n", s.Length.reflection)
		fmt.Fprintf(buffer, "%s.FromBytes%s(&length, bytes, %s)\n", converter, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, s.Length.size)
		if s.maximum > 0 {
			fmt.Fprintf(buffer, "if length > %d {\n", s.maximum)
			fmt.Fprintf(buffer, "panic(\"packed: %s holds more than its maximum of %d items\")\n", strings.TrimPrefix(reciever, "reciever."), s.maximum)
			buffer.WriteString("}\n")
		}
		fmt.Fprintf(buffer, "%s = make(%s, length)\n", reciever, s.recieverType)

	default:
		panic("invalid function name")
	}

	indexVariable := fmt.Sprintf("i%d", depth)
	fmt.Fprintf(buffer, "for %s := range %s {\n", indexVariable, reciever)
	writeElement(buffer, structure, functionName, s.ElementKind, s.Element, s.ElementSize, fmt.Sprintf("%s[%s]", reciever, indexVariable), littleEndian, offsetVariable, depth)
	buffer.WriteString("}\n")

	buffer.WriteString("}\n")
}

func (s packedVariableString) write(buffer *bytes.Buffer, reciever string, functionName string, littleEndian bool, offsetVariable string) {
	converter := getConverterName(s.Length.converter.hash)
	endian := endianName(littleEndian)

	buffer.WriteString("{\n")

	switch functionName {
	case "ToBytes":
		if s.Length.size < 8 {
			fmt.Fprintf(buffer, "if uint64(len(%s)) > %d {\n", reciever, s.Length.maximum())
			fmt.Fprintf(buffer, "panic(\"packed: %s is longer than its %s length prefix allows\")\n", strings.TrimPrefix(reciever, "reciever."), s.Length.reflection)
			buffer.WriteString("}\n")
		}
		fmt.Fprintf(buffer, "length := %s(len(%s))\n", s.Length.reflection, reciever)
		fmt.Fprintf(buffer, "%s.ToBytes%s(&length, bytes, %s)\n", converter, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, s.Length.size)
		fmt.Fprintf(buffer, "%s += copy(bytes[%s:], %s)\n", offsetVariable, offsetVariable, reciever)

	case "FromBytes":
		fmt.Fprintf(buffer, "var length %s\n", s.Length.reflection)
		fmt.Fprintf(buffer, "%s.FromBytes%s(&length, bytes, %s)\n", converter, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, s.Length.size)
		if s.Bytes {
			fmt.Fprintf(buffer, "%s = make([]byte, length)\n", reciever)
			fmt.Fprintf(buffer, "copy(%s, bytes[%s:%s+int(length)])\n", reciever, offsetVariable, offsetVariable)
		} else {
			fmt.Fprintf(buffer, "%s = string(bytes[%s : %s+int(length)])\n", reciever, offsetVariable, offsetVariable)
		}
		fmt.Fprintf(buffer, "%s += int(length)\n", offsetVariable)

	default:
		panic("invalid function name")
	}

	buffer.WriteString("}\n")
}
