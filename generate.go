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

		for _, tag := range property.tags {
			tags = append(tags, fmt.Sprintf("%s:\"%s\"", tag.key, tag.value))
		}

		var tagString string

		if len(tags) > 0 {
			tagString = "`" + strings.Join(tags, " ") + "`"
		}

		var propertyType string

		switch property.kind {

		case KindStruct:
			propertyType = property.packed.(PackedStruct).name

		case KindConverter:

			if overwrite, ok := property.packed.(OverwriteConverterReciverReflectionInterface); ok {
				propertyType = overwrite.OverwriteConverterReciverReflection(property.recieverType).String()

			} else {
				propertyType = property.recieverType.String()
			}

		case KindConverterCast:
			propertyType = property.packed.(ConverterCast).target.String()

		case KindType:
			propertyType = property.propertyType.Elem().String()

		case KindArray:
			propertyType = property.packed.(PackedArray).recieverType

		case KindBitFieldGroup:
			group := property.packed.(PackedBitFieldGroup)

			for _, field := range group.fields {

				property := field.packedProperty

				tags := []string{}

				for _, tag := range property.tags {
					tags = append(tags, fmt.Sprintf("%s:\"%s\"", tag.key, tag.value))
				}

				var tagString string

				if len(tags) > 0 {
					tagString = "`" + strings.Join(tags, " ") + "`"
				}

				fmt.Fprintf(buffer, "%s %s %s\n", property.name, field.reflection.String(), tagString)
			}

			continue

		default:
			panic("invalid property kind")
		}

		fmt.Fprintf(buffer, "%s %s %s\n", property.name, propertyType, tagString)
	}

	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}

func (p *PackedProperty) WriteProperty(buffer *bytes.Buffer, structure *PackedStruct, functionName, recieverPrefix string, offset *int) {
	endian := "LittleEndian"

	if !p.littleEndian {
		endian = "BigEndian"
	}

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case KindStruct:
		for _, child := range p.packed.(PackedStruct).properties {
			child.WriteProperty(buffer, structure, functionName, reciever, offset)
		}
		return

	case KindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, index + %d)\n", getConverterName(p.converter.hash), functionName, endian, reciever, *offset)

	case KindConverterCast:
		cast := p.packed.(ConverterCast)
		cast.Write(buffer, structure, reciever, functionName, p.littleEndian, fmt.Sprintf("index + %d", *offset))
		*offset += cast.size
		return

	case KindArray:
		array := p.packed.(PackedArray)
		fmt.Fprintf(buffer, "o%d := index + %d\n", *offset, *offset)
		array.Write(buffer, structure, reciever, functionName, p.littleEndian, fmt.Sprintf("o%d", *offset), 0)
		*offset += p.size
		return

	case KindBitFieldGroup:
		group := p.packed.(PackedBitFieldGroup)

		offsetString := fmt.Sprintf("index + %d", *offset)

		switch functionName {
		case "ToBytes":
			group.WriteToBytes(buffer, reciever, p.littleEndian, offsetString)
		case "FromBytes":
			group.WriteFromBytes(buffer, reciever, p.littleEndian, offsetString)
		default:
			panic("invalid function name")
		}

		*offset += group.size
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

	for reciever, index := range p.converterCastRecievers {
		fmt.Fprintf(buffer, "var r%d %s\n", index, reciever)
	}

	for _, property := range p.properties {
		property.WriteProperty(buffer, p, functionName, "reciever", &offset)
	}

	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}
