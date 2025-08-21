package packed

import (
	"bytes"
	"fmt"
	"strings"
)

func (p *PackedStruct) SizeDefinition() []byte {
	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "func (reciever *%s) Size() int {\n", p.name)
	fmt.Fprintf(buffer, "return %d\n", p.size)
	fmt.Fprintf(buffer, "}\n")
	return buffer.Bytes()
}

func (p *PackedStruct) StructDefinition() []byte {

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "type %s struct {\n", p.name)

	for _, property := range p.properties {
		tags := []string{}

		for key, value := range property.tags {
			tags = append(tags, fmt.Sprintf("`%s:\"%s\"`", key, value))
		}

		var propertyType string

		switch property.kind {

		case KindStruct:
			propertyType = property.packed.(PackedStruct).name

		case KindConverter:

			if overwrite, ok := property.packed.(OverwriteConverterReciverReflectionInterface); ok {
				propertyType = overwrite.OverwriteConverterReciverReflection(property.recieverType)

			} else {
				propertyType = property.recieverType.String()
			}

		case KindType:
			propertyType = property.propertyType.Elem().String()

		case KindArray:
			propertyType = property.packed.(PackedArray).recieverType

		default:
			panic("invalid property kind")
		}

		fmt.Fprintf(buffer, "%s %s %s\n", property.name, propertyType, strings.Join(tags, " "))
	}

	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}

func (p *PackedProperty) WriteProperty(buffer *bytes.Buffer, functionName, recieverPrefix string, offset *int) {
	endian := "LittleEndian"

	if !p.littleEndian {
		endian = "BigEndian"
	}

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case KindStruct:
		for _, child := range p.packed.(PackedStruct).properties {
			child.WriteProperty(buffer, functionName, reciever, offset)
		}
		return

	case KindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, index + %d)\n", getConverterName(p.converter.hash), functionName, endian, reciever, *offset)

	case KindArray:
		array := p.packed.(PackedArray)
		fmt.Fprintf(buffer, "o%d := index + %d\n", *offset, *offset)
		array.Write(buffer, reciever, functionName, p.littleEndian, fmt.Sprintf("o%d", *offset), 0)
		*offset += p.size
		return

	default:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, index + %d)\n", reciever, functionName, endian, *offset)
	}

	*offset += p.size
}

func (p *PackedStruct) ConversionDefinition(functionName string) []byte {

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "func (reciever *%s) %s(bytes []byte, index int) {\n", p.name, functionName)
	offset := 0

	for _, property := range p.properties {
		property.WriteProperty(buffer, functionName, "reciever", &offset)
	}

	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}
