package packed

import (
	"bytes"
	"fmt"
)

type packedArray struct {
	Length       int
	Element      any
	ElementKind  kind
	ElementSize  int
	recieverType string
}

func (a packedArray) Size() int {
	return a.Length * a.ElementSize
}

func getArrayRecieverType(propertyType any) string {

	kind, recieverType, propertyType := validatePropertyType(propertyType)

	if kind == kindArray {
		array := propertyType.(packedArray)
		return fmt.Sprintf("[%d]%s", array.Length, getArrayRecieverType(array.Element))
	}

	if kind == kindStruct {
		return propertyType.(packedStruct).name
	}

	return recieverType.String()
}

func Array(length int, elementType any) packedArray {

	kind, _, elementType := validatePropertyType(elementType)

	if kind == kindBitField {
		panic("bit fields as direct array elements are not supported")
	}

	if kind == kindConverter {
		hash := createConverterHash(elementType)

		if _, exists := converters[hash.hash]; !exists {
			converters[hash.hash] = hash
		}
	}

	recieverType := fmt.Sprintf("[%d]%s", length, getArrayRecieverType(elementType))
	elementSize := elementType.(interface{ Size() int }).Size()

	return packedArray{
		Length:       length,
		Element:      elementType,
		ElementKind:  kind,
		recieverType: recieverType,
		ElementSize:  elementSize,
	}
}

func (p *packedProperty) writeArrayElement(buffer *bytes.Buffer, structure *packedStruct, functionName, recieverPrefix string, offsetVariable string, depth int) {
	endian := "LittleEndian"

	if !p.littleEndian {
		endian = "BigEndian"
	}

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case kindStruct:
		for _, child := range p.packed.(packedStruct).properties {
			child.writeArrayElement(buffer, structure, functionName, reciever, offsetVariable, depth)
		}

	case kindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(p.converter.hash), functionName, endian, reciever, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, p.size)

	case kindConverterCast:
		cast := p.packed.(converterCast)
		cast.Write(buffer, structure, reciever, functionName, p.littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, cast.size)

	case kindArray:
		array := p.packed.(packedArray)
		array.write(buffer, structure, reciever, functionName, p.littleEndian, offsetVariable, depth+1)

	case kindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", reciever, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, p.size)

	case kindBitFieldGroup:
		group := p.packed.(packedBitFieldGroup)

		switch functionName {
		case "ToBytes":
			group.writeToBytes(buffer, reciever, p.littleEndian, offsetVariable)
		case "FromBytes":
			group.writeFromBytes(buffer, reciever, p.littleEndian, offsetVariable)
		}

		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, group.size)

	default:
		panic("invalid property kind")
	}
}

func (a packedArray) write(buffer *bytes.Buffer, structure *packedStruct, recieverVariable string, functionName string, littleEndian bool, offsetVariable string, depth int) {

	indexVariable := fmt.Sprintf("i%d", depth)

	recieverVariable = fmt.Sprintf("%s[%s]", recieverVariable, indexVariable)

	fmt.Fprintf(buffer, "for %s := 0; %s < %d; %s++ {\n", indexVariable, indexVariable, a.Length, indexVariable)

	endian := "LittleEndian"

	if !littleEndian {
		endian = "BigEndian"
	}

	switch a.ElementKind {

	case kindStruct:
		childStruct := a.Element.(packedStruct)
		for _, property := range childStruct.properties {
			property.writeArrayElement(buffer, structure, functionName, recieverVariable, offsetVariable, depth)
		}

	case kindConverter:
		hash := createConverterHash(a.Element)
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(hash.hash), functionName, endian, recieverVariable, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, a.ElementSize)

	case kindConverterCast:
		cast := a.Element.(converterCast)
		cast.Write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, cast.size)

	case kindArray:
		a.Element.(packedArray).write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable, depth+1)

	case kindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", recieverVariable, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, a.ElementSize)

	default:
		panic("invalid property kind")
	}

	fmt.Fprintf(buffer, "}\n")
}
