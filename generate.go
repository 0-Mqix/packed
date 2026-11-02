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

	if !p.dynamic {
		fmt.Fprintf(buffer, "return %d\n", p.size)
		fmt.Fprintf(buffer, "}\n")
		return buffer.Bytes()
	}

	fmt.Fprintf(buffer, "size := %d\n", p.size)
	for _, property := range p.properties {
		if property.dynamic {
			writeDynamicSize(buffer, property.kind, property.packed, "reciever."+property.name, 0)
		}
	}
	fmt.Fprintf(buffer, "return size\n")
	fmt.Fprintf(buffer, "}\n")
	return buffer.Bytes()
}

// writeDynamicSize adds the size of a variable-length value to the local size.
func writeDynamicSize(buffer *bytes.Buffer, kind kind, packed any, reciever string, depth int) {
	switch kind {

	case kindVariableString:
		fmt.Fprintf(buffer, "size += %d + len(%s)\n", packed.(packedVariableString).Length.size, reciever)

	case kindStruct:
		fmt.Fprintf(buffer, "size += %s.Size()\n", reciever)

	case kindSlice:
		slice := packed.(packedSlice)
		fmt.Fprintf(buffer, "size += %d\n", slice.Length.size)

		if !isDynamic(slice.ElementKind, slice.Element) {
			fmt.Fprintf(buffer, "size += len(%s) * %d\n", reciever, slice.ElementSize)
			return
		}

		indexVariable := fmt.Sprintf("i%d", depth)
		fmt.Fprintf(buffer, "for %s := range %s {\n", indexVariable, reciever)
		writeDynamicSize(buffer, slice.ElementKind, slice.Element, fmt.Sprintf("%s[%s]", reciever, indexVariable), depth+1)
		fmt.Fprintf(buffer, "}\n")

	case kindArray:
		array := packed.(packedArray)
		indexVariable := fmt.Sprintf("i%d", depth)
		fmt.Fprintf(buffer, "for %s := range %s {\n", indexVariable, reciever)
		writeDynamicSize(buffer, array.ElementKind, array.Element, fmt.Sprintf("%s[%s]", reciever, indexVariable), depth+1)
		fmt.Fprintf(buffer, "}\n")

	default:
		panic("invalid dynamic property kind")
	}
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

		case kindSlice:
			propertyType = property.packed.(packedSlice).recieverType

		case kindVariableString:
			propertyType = "string"
			if property.packed.(packedVariableString).Bytes {
				propertyType = "[]byte"
			}

		case kindEnum:
			propertyType = property.packed.(packedEnum).name

		case kindBitFieldGroup:
			group := property.packed.(packedBitFieldGroup)

			for _, field := range group.fields {

				if field.bitFieldKind == bitFieldKindTag {
					continue
				}

				property := field.packedProperty

				tags := []string{}

				for _, tag := range property.tags {
					tags = append(tags, fmt.Sprintf("%s:\"%s\"", tag.key, tag.value))
				}

				var tagString string

				if len(tags) > 0 {
					tagString = "`" + strings.Join(tags, " ") + "`"
				}

				var fieldType string

				switch field.bitFieldKind {

				case bitFieldKindBitsType, bitFieldKindBitsConverter:
					fieldType = field.bitsTargetReflection.Elem().String()

				case bitFieldKindEnum:
					fieldType = field.enum.name

				default:
					fieldType = field.reflection.String()
				}

				fmt.Fprintf(buffer, "%s %s %s\n", property.name, fieldType, tagString)
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

// position is where the next property starts: a constant offset from index
// while every property so far had a fixed size, then a constant offset from
// the running offset variable once a variable-length property was written.
type position struct {
	variable string
	constant int
}

func (p *position) base() string {
	if p.variable == "" {
		return "index"
	}
	return p.variable
}

func (p *position) expression() string {
	return fmt.Sprintf("%s + %d", p.base(), p.constant)
}

// settle moves the position into the offset variable, so that
// variable-length code can advance it at runtime.
func (p *position) settle(buffer *bytes.Buffer) string {
	if p.variable == "" {
		fmt.Fprintf(buffer, "offset := index + %d\n", p.constant)
		p.variable = "offset"
	} else if p.constant != 0 {
		fmt.Fprintf(buffer, "offset += %d\n", p.constant)
	}
	p.constant = 0
	return p.variable
}

func (p *packedProperty) writeProperty(buffer *bytes.Buffer, structure *packedStruct, functionName, recieverPrefix string, offset *position) {
	endian := endianName(p.littleEndian)

	reciever := recieverPrefix + "." + p.name

	switch p.kind {

	case kindStruct:
		for _, child := range p.packed.(packedStruct).properties {
			child.writeProperty(buffer, structure, functionName, reciever, offset)
		}
		return

	case kindConverter:
		fmt.Fprintf(buffer, "%s.%s%s(&%s, bytes, %s)\n", getConverterName(p.converter.hash), functionName, endian, reciever, offset.expression())

	case kindConverterCast:
		cast := p.packed.(converterCast)
		cast.Write(buffer, structure, reciever, functionName, p.littleEndian, offset.expression())
		offset.constant += cast.size
		return

	case kindEnum:
		p.packed.(packedEnum).write(buffer, reciever, functionName, p.littleEndian, offset.expression())

	case kindArray:
		array := p.packed.(packedArray)

		if offset.variable != "" || p.dynamic {
			array.write(buffer, structure, reciever, functionName, p.littleEndian, offset.settle(buffer), 0)
			return
		}

		variable := fmt.Sprintf("o%d", offset.constant)
		fmt.Fprintf(buffer, "%s := %s\n", variable, offset.expression())
		array.write(buffer, structure, reciever, functionName, p.littleEndian, variable, 0)
		offset.constant += p.size
		return

	case kindSlice:
		p.packed.(packedSlice).write(buffer, structure, reciever, functionName, p.littleEndian, offset.settle(buffer), 0)
		return

	case kindVariableString:
		p.packed.(packedVariableString).write(buffer, reciever, functionName, p.littleEndian, offset.settle(buffer))
		return

	case kindBitFieldGroup:
		group := p.packed.(packedBitFieldGroup)

		switch functionName {
		case "ToBytes":
			group.writeToBytes(buffer, reciever, p.littleEndian, offset.base(), offset.constant, false)
		case "FromBytes":
			group.writeFromBytes(buffer, reciever, p.littleEndian, offset.base(), offset.constant, false)
		default:
			panic("invalid function name")
		}

		offset.constant += group.size
		return

	default:
		fmt.Fprintf(buffer, "%s.%s%s(bytes, %s)\n", reciever, functionName, endian, offset.expression())
	}

	offset.constant += p.size
}

func (p *packedStruct) conversionDefinition(functionName string) []byte {

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "func (reciever *%s) %s(bytes []byte, index int) int {\n", p.name, functionName)
	offset := &position{}

	recieverByIndex := make([]reflect.Type, len(p.converterCastRecievers))
	for reciever, index := range p.converterCastRecievers {
		recieverByIndex[index] = reciever
	}
	for index, reciever := range recieverByIndex {
		fmt.Fprintf(buffer, "var r%d %s\n", index, reciever)
	}

	for _, property := range p.properties {
		property.writeProperty(buffer, p, functionName, "reciever", offset)
	}

	fmt.Fprintf(buffer, "return %s\n", offset.expression())
	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}
