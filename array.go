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

	switch kind {
	case kindStruct:
		return propertyType.(packedStruct).name
	case kindSlice:
		return propertyType.(packedSlice).recieverType
	case kindVariableString:
		if propertyType.(packedVariableString).Bytes {
			return "[]byte"
		}
		return "string"
	case kindEnum:
		return propertyType.(packedEnum).name
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

	case kindSlice:
		p.packed.(packedSlice).write(buffer, structure, reciever, functionName, p.littleEndian, offsetVariable, depth+1)

	case kindVariableString:
		p.packed.(packedVariableString).write(buffer, reciever, functionName, p.littleEndian, offsetVariable)

	case kindEnum:
		enum := p.packed.(packedEnum)
		enum.write(buffer, reciever, functionName, p.littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, enum.size)

	case kindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", reciever, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, p.size)

	case kindBitFieldGroup:
		group := p.packed.(packedBitFieldGroup)

		switch functionName {
		case "ToBytes":
			group.writeToBytes(buffer, reciever, p.littleEndian, offsetVariable, 0, true)
		case "FromBytes":
			group.writeFromBytes(buffer, reciever, p.littleEndian, offsetVariable, 0, true)
		}
		// the offset variable is advanced per word inside write{To,From}Bytes

	default:
		panic("invalid property kind")
	}
}

func (a packedArray) write(buffer *bytes.Buffer, structure *packedStruct, recieverVariable string, functionName string, littleEndian bool, offsetVariable string, depth int) {

	indexVariable := fmt.Sprintf("i%d", depth)

	recieverVariable = fmt.Sprintf("%s[%s]", recieverVariable, indexVariable)

	fmt.Fprintf(buffer, "for %s := 0; %s < %d; %s++ {\n", indexVariable, indexVariable, a.Length, indexVariable)

	writeElement(buffer, structure, functionName, a.ElementKind, a.Element, a.ElementSize, recieverVariable, littleEndian, offsetVariable, depth)

	fmt.Fprintf(buffer, "}\n")
}

func writeElement(buffer *bytes.Buffer, structure *packedStruct, functionName string, elementKind kind, element any, elementSize int, recieverVariable string, littleEndian bool, offsetVariable string, depth int) {

	endian := endianName(littleEndian)

	switch elementKind {

	case kindStruct:
		childStruct := element.(packedStruct)
		for _, property := range childStruct.properties {
			property.writeArrayElement(buffer, structure, functionName, recieverVariable, offsetVariable, depth)
		}

	case kindConverter:
		hash := createConverterHash(element)
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(hash.hash), functionName, endian, recieverVariable, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, elementSize)

	case kindConverterCast:
		cast := element.(converterCast)
		cast.Write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, cast.size)

	case kindArray:
		element.(packedArray).write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable, depth+1)

	case kindSlice:
		element.(packedSlice).write(buffer, structure, recieverVariable, functionName, littleEndian, offsetVariable, depth+1)

	case kindVariableString:
		element.(packedVariableString).write(buffer, recieverVariable, functionName, littleEndian, offsetVariable)

	case kindEnum:
		enum := element.(packedEnum)
		enum.write(buffer, recieverVariable, functionName, littleEndian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, enum.size)

	case kindType:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", recieverVariable, functionName, endian, offsetVariable)
		fmt.Fprintf(buffer, "%s += %d\n", offsetVariable, elementSize)

	default:
		panic("invalid property kind")
	}
}
