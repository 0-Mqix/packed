package packed

import (
	"bytes"
	"fmt"
)

type PackedArray struct {
	Length       int
	Element      any
	ElementKind  Kind
	ElementSize  int
	recieverType string
}

func (a PackedArray) Size() int {
	return a.Length * a.ElementSize
}

func getArrayRecieverType(propertyType any) string {

	kind, recieverType, propertyType := validatePropertyType(propertyType)

	if kind == KindArray {
		array := propertyType.(PackedArray)
		return fmt.Sprintf("[%d]%s", array.Length, getArrayRecieverType(array.Element))
	}

	if kind == KindStruct {
		return propertyType.(PackedStruct).name
	}

	return recieverType.String()
}

func Array(length int, elementType any) PackedArray {

	kind, _, elementType := validatePropertyType(elementType)

	if kind == KindBitField {
		panic("bit fields as direct array elements are not supported")
	}

	if kind == KindConverter {
		hash := createConverterHash(elementType)

		if _, exists := converters[hash.hash]; !exists {
			converters[hash.hash] = hash
		}
	}

	recieverType := fmt.Sprintf("[%d]%s", length, getArrayRecieverType(elementType))
	elementSize := elementType.(interface{ Size() int }).Size()

	return PackedArray{
		Length:       length,
		Element:      elementType,
		ElementKind:  kind,
		recieverType: recieverType,
		ElementSize:  elementSize,
	}
}

func (p *PackedProperty) WriteArrayElement(buffer *bytes.Buffer, structure *PackedStruct, functionName, recieverPrefix string, offsetVariable string, depth int) {
	endian := "LittleEndian"

	if !p.littleEndian {
		endian = "BigEndian"
	}

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case KindStruct:
		for _, child := range p.packed.(PackedStruct).properties {
			child.WriteArrayElement(buffer, structure, functionName, reciever, offsetVariable, depth)
		}

	case KindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(p.converter.hash), functionName, endian, reciever, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, p.size)

	case KindConverterCast:
		cast := p.packed.(ConverterCast)
		cast.Write(buffer, structure, reciever, functionName, p.littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, cast.size)

	case KindArray:
		array := p.packed.(PackedArray)
		array.Write(buffer, structure, reciever, functionName, p.littleEndian, offsetVariable, depth+1)

	case KindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", reciever, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, p.size)

	case KindBitFieldGroup:
		group := p.packed.(PackedBitFieldGroup)

		switch functionName {
		case "ToBytes":
			group.WriteToBytes(buffer, reciever, p.littleEndian, offsetVariable)
		case "FromBytes":
			group.WriteFromBytes(buffer, reciever, p.littleEndian, offsetVariable)
		}

		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, group.size)

	default:
		panic("invalid property kind")
	}
}

func (a PackedArray) Write(buffer *bytes.Buffer, structure *PackedStruct, recieverVariable string, functionName string, littleEndian bool, offsetVariable string, depth int) {

	indexVariable := fmt.Sprintf("i%d", depth)

	recieverVariable = fmt.Sprintf("%s[%s]", recieverVariable, indexVariable)

	fmt.Fprintf(buffer, "for %s := 0; %s < %d; %s++ {\n", indexVariable, indexVariable, a.Length, indexVariable)

	endian := "LittleEndian"

	if !littleEndian {
		endian = "BigEndian"
	}

	switch a.ElementKind {

	case KindStruct:
		childStruct := a.Element.(PackedStruct)
		for _, property := range childStruct.properties {
			property.WriteArrayElement(buffer, structure, functionName, recieverVariable, offsetVariable, depth)
		}

	case KindConverter:
		hash := createConverterHash(a.Element)
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(hash.hash), functionName, endian, recieverVariable, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, a.ElementSize)

	case KindConverterCast:
		cast := a.Element.(ConverterCast)
		cast.Write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, cast.size)

	case KindArray:
		a.Element.(PackedArray).Write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable, depth+1)

	case KindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", recieverVariable, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, a.ElementSize)

	default:
		panic("invalid property kind")
	}

	fmt.Fprintf(buffer, "}\n")
}
