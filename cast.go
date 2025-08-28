package packed

import (
	"bytes"
	"fmt"
	"reflect"
)

type ConverterCast struct {
	converter converterHash
	reciever  reflect.Type
	target    reflect.Type
	size      int
}

func (c ConverterCast) Size() int {
	return c.size
}

func Cast[T any](converter any) ConverterCast {

	var value T

	target := reflect.TypeOf(value)

	converter = structToPointer(converter)

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

	return ConverterCast{
		converter: hash,
		reciever:  reciever,
		target:    target,
		size:      size,
	}
}

func (c ConverterCast) Write(buffer *bytes.Buffer, structure *PackedStruct, recieverVariable string, functionName string, littleEndian bool, offsetVariable string) {

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
		fmt.Fprintf(buffer, "%s = %s(r%d)\n", recieverVariable, c.target, recieverIndex)

	default:
		panic("invalid function name")
	}

}
