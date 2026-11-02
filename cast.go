package packed

import (
	"bytes"
	"fmt"
	"reflect"
)

type converterCast struct {
	converter converterHash
	reciever  reflect.Type
	target    reflect.Type
	size      int
	// targetName is the target as the generated file writes it, for a type
	// the dynamic layer read from go/types and so has no reflect.Type of.
	targetName string
	// overflow is a statement the decoder runs when the value read does not
	// survive the conversion to the target (an Int64 read into an int where
	// int is 32 bits).
	overflow string
}

// targetType is the target as the generated file writes it.
func (c converterCast) targetType() string {
	if c.targetName != "" {
		return c.targetName
	}
	return c.target.String()
}

func (c converterCast) Size() int {
	return c.size
}

func Cast[T any](converter any) converterCast {

	var value T

	target := reflect.TypeOf(value)

	converter = toPointer(converter)

	reciever, ok := implementsConverterInterface(converter)

	if !ok {
		panic("not a valid converter")
	}

	if overwrite, ok := reciever.(OverwriteConverterReciverReflectionInterface); ok {
		reciever = overwrite.OverwriteConverterReciverReflection(reflect.TypeOf(converter))
	}

	if !reciever.ConvertibleTo(target) {
		panic("reciever of converter is not assignable to the target type")
	}

	hash := createConverterHash(converter)

	if _, exists := converters[hash.hash]; !exists {
		converters[hash.hash] = hash
	}

	size := converter.(interface{ Size() int }).Size()

	return converterCast{
		converter: hash,
		reciever:  reciever,
		target:    target,
		size:      size,
	}
}

func (c converterCast) Write(buffer *bytes.Buffer, structure *packedStruct, recieverVariable string, functionName string, littleEndian bool, offsetVariable string) {

	endian := "LittleEndian"

	if !littleEndian {
		endian = "BigEndian"
	}

	recieverIndex := structure.converterCastRecievers[c.reciever]

	switch functionName {
	case "ToBytes":
		fmt.Fprintf(buffer, "r%d = %s(%s)\n", recieverIndex, c.reciever, recieverVariable)
		fmt.Fprintf(buffer, "%s.ToBytes%s(&r%d, bytes, %s)\n", getConverterName(c.converter.hash), endian, recieverIndex, offsetVariable)

	case "FromBytes":
		fmt.Fprintf(buffer, "%s.FromBytes%s(&r%d, bytes, %s)\n", getConverterName(c.converter.hash), endian, recieverIndex, offsetVariable)
		fmt.Fprintf(buffer, "%s = %s(r%d)\n", recieverVariable, c.targetType(), recieverIndex)
		if c.overflow != "" {
			fmt.Fprintf(buffer, "if %s(%s) != r%d {\n%s\n}\n", c.reciever, recieverVariable, recieverIndex, c.overflow)
		}

	default:
		panic("invalid function name")
	}

}
