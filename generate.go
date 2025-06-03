package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
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

func (p *PackedStruct) SizeDefinition() []byte {
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("func (reciever *%s) Size() int {\n", p.name))
	buffer.WriteString("return " + strconv.Itoa(p.size) + "\n")
	buffer.WriteString("}\n")
	return buffer.Bytes()
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
			propertyType = property.packed.(PackedStruct).name

		case KindConverter:
			reflection = property.recieverType

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
		variable := converters[p.propertyType]
		var converter string

		if variable.external {
			converter = prefixWithPackage(p.propertyType, variable.variable)
		} else {
			converter = variable.variable
		}

		buffer.WriteString(fmt.Sprintf("%s.%s%s(&%s, bytes, index + %d)\n", converter, functionName, endian, reciever, *offset))

	default:
		buffer.WriteString(fmt.Sprintf("%s.%s%s(bytes, index + %d)\n", reciever, functionName, endian, *offset))
	}

	*offset += p.size
}

func (p *PackedStruct) ConversionDefinition(functionName string) []byte {

	buffer := bytes.Buffer{}

	buffer.WriteString(fmt.Sprintf("func (reciever *%s) %s(bytes []byte, index int) {\n", p.name, functionName))
	offset := 0

	for _, property := range p.properties {
		property.WriteProperty(&buffer, functionName, "reciever", &offset)
	}

	buffer.WriteString("}\n")

	return buffer.Bytes()
}
