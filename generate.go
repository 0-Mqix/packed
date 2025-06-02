package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

func prefixWithPackage(reflection reflect.Type, target string) string {

	if reflection.Kind() == reflect.Ptr {
		reflection = reflection.Elem()
	}

	split := strings.Split(reflection.String(), ".")

	if len(split) > 1 {
		return split[0] + "." + target
	}

	return target
}

func (p *PackedStruct) StructDefinition() []byte {

	buffer := bytes.Buffer{}

	buffer.WriteString(fmt.Sprintf("type %s struct {\n", p.name))

	for _, property := range p.properties {
		tags := []string{}

		for key, value := range property.tags {
			tags = append(tags, fmt.Sprintf("`%s:\"%s\"`", key, value))
		}

		var propertyType string
		var reflection reflect.Type

		switch property.kind {
		case KindStruct:
			propertyType = property.packed.(*PackedStruct).name
		case KindConverter:
			reflection = property.finalType
		case KindType:
			reflection = property.propertyType.Elem()
		}

		if property.kind != KindStruct {
			propertyType = reflection.String()
		}

		buffer.WriteString(fmt.Sprintf("%s %s %s\n", property.name, propertyType, strings.Join(tags, " ")))
	}

	buffer.WriteString("}\n")

	return buffer.Bytes()
}

func (p *PackedStruct) ConversionDefinition(functionName string) []byte {

	buffer := bytes.Buffer{}

	buffer.WriteString(fmt.Sprintf("func (reciever *%s) %s(bytes []byte, index int) {\n", p.name, functionName))
	offset := 0

	for _, property := range p.properties {

		if property.kind == KindConverter {
			variable := converters[property.propertyType]
			var converter string

			if variable.external {
				converter = prefixWithPackage(property.propertyType, variable.variable)
			} else {
				converter = variable.variable
			}

			buffer.WriteString(fmt.Sprintf("%s.%s(&reciever.%s, bytes, index + %d)\n", converter, functionName, property.name, offset))
		} else {
			buffer.WriteString(fmt.Sprintf("reciever.%s.%s(bytes, index + %d)\n", property.name, functionName, offset))
		}

		offset += property.size
	}

	buffer.WriteString("}\n")

	return buffer.Bytes()
}
