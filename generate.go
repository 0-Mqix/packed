package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

func (p *packedStruct) sizeDefinition() []byte {
	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "func (reciever *%s) Size() int {\n", p.name)
	fmt.Fprintf(buffer, "return %d\n", p.size)
	fmt.Fprintf(buffer, "}\n")
	return buffer.Bytes()
}

func (p *packedStruct) structDefinition() []byte {

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

		case kindStruct:
			propertyType = property.packed.(packedStruct).name

		case kindConverter:

			if overwrite, ok := property.packed.(OverwriteConverterReciverReflectionInterface); ok {
				propertyType = overwrite.OverwriteConverterReciverReflection(property.recieverType).String()

			} else {
				propertyType = property.recieverType.String()
			}

		case kindConverterCast:
			propertyType = property.packed.(converterCast).target.String()

		case kindType:
			propertyType = property.propertyType.Elem().String()

		case kindArray:
			propertyType = property.packed.(packedArray).recieverType

		case kindBitFieldGroup:
			group := property.packed.(packedBitFieldGroup)

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

				var reflection reflect.Type

				switch field.bitFieldKind {

				case bitFieldKindBitsType, bitFieldKindBitsConverter:
					reflection = field.bitsTargetReflection.Elem()

				default:
					reflection = field.reflection
				}

				fmt.Fprintf(buffer, "%s %s %s\n", property.name, reflection, tagString)
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

func (p *packedProperty) writeProperty(buffer *bytes.Buffer, structure *packedStruct, functionName, recieverPrefix string, offset *int) {
	endian := "LittleEndian"

	if !p.littleEndian {
		endian = "BigEndian"
	}

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case kindStruct:
		for _, child := range p.packed.(packedStruct).properties {
			child.writeProperty(buffer, structure, functionName, reciever, offset)
		}
		return

	case kindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, index + %d)\n", getConverterName(p.converter.hash), functionName, endian, reciever, *offset)

	case kindConverterCast:
		cast := p.packed.(converterCast)
		cast.Write(buffer, structure, reciever, functionName, p.littleEndian, fmt.Sprintf("index + %d", *offset))
		*offset += cast.size
		return

	case kindArray:
		array := p.packed.(packedArray)
		fmt.Fprintf(buffer, "o%d := index + %d\n", *offset, *offset)
		array.write(buffer, structure, reciever, functionName, p.littleEndian, fmt.Sprintf("o%d", *offset), 0)
		*offset += p.size
		return

	case kindBitFieldGroup:
		group := p.packed.(packedBitFieldGroup)

		offsetString := fmt.Sprintf("index + %d", *offset)

		switch functionName {
		case "ToBytes":
			group.writeToBytes(buffer, reciever, p.littleEndian, offsetString)
		case "FromBytes":
			group.writeFromBytes(buffer, reciever, p.littleEndian, offsetString)
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

func (p *packedStruct) conversionDefinition(functionName string) []byte {

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "func (reciever *%s) %s(bytes []byte, index int) {\n", p.name, functionName)
	offset := 0

	for reciever, index := range p.converterCastRecievers {
		fmt.Fprintf(buffer, "var r%d %s\n", index, reciever)
	}

	for _, property := range p.properties {
		property.writeProperty(buffer, p, functionName, "reciever", &offset)
	}

	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}
