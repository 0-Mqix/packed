package packed

import (
	"bytes"
	"fmt"
	"os"
	"reflect"

	"golang.org/x/tools/imports"
)

type converter struct {
	variable string
	external bool
}

var (
	structs    = map[string]*PackedStruct{}
	converters = map[reflect.Type]converter{}
	imported   = map[string]bool{"packed": true}
)

type BitField struct {
	bitSize  int
	next     *BitField
	previous *BitField
}

type PackedProperty struct {
	name           string
	size           int
	packed         any
	finalType      reflect.Type
	propertyType   reflect.Type
	bitField       *BitField
	tags           map[string]string
	kind           Kind
	littleEndian   bool
	endianOverride bool
}

type FieldOption func(*PackedProperty)

func Tag(key, value string) FieldOption {
	return func(definition *PackedProperty) {
		definition.tags[key] = value
	}
}

func Bits(bits int) FieldOption {
	return func(definition *PackedProperty) {
		definition.bitField = &BitField{bitSize: bits}
	}
}

func Type(propertyType any) FieldOption {
	return func(definition *PackedProperty) {
		definition.packed = propertyType
	}
}

func LittleEndian(value bool) FieldOption {
	return func(definition *PackedProperty) {
		definition.littleEndian = value
		definition.endianOverride = true
	}
}

type Kind int

type TypeInterface interface {
	Size() int
	ToBytesLittleEndian(bytes []byte, index int)
	FromBytesLittleEndian(bytes []byte, index int)
	ToBytesBigEndian(bytes []byte, index int)
	FromBytesBigEndian(bytes []byte, index int)
}

type ConverterInterface[Reciever any] interface {
	Size() int
	ToBytesLittleEndian(value *Reciever, bytes []byte, index int)
	FromBytesLittleEndian(reciever *Reciever, bytes []byte, index int)
	ToBytesBigEndian(value *Reciever, bytes []byte, index int)
	FromBytesBigEndian(reciever *Reciever, bytes []byte, index int)
}

const (
	KindInvalid Kind = iota
	KindType
	KindStruct
	KindConverter
)

func validatePropertyType(propertyType any) (Kind, reflect.Type) {

	if _, ok := propertyType.(TypeInterface); ok {
		return KindType, reflect.TypeOf(propertyType)
	}

	if _, ok := propertyType.(*PackedStruct); ok {
		return KindStruct, nil
	}

	if reciever, ok := implementsConverterInterface(propertyType); ok {
		return KindConverter, reciever
	}

	return KindInvalid, nil
}

func (property *PackedProperty) SetEndian(littleEndian bool) {
	if !property.endianOverride {
		property.littleEndian = littleEndian
	}

	if property.kind == KindStruct {
		for _, child := range property.packed.(*PackedStruct).properties {
			child.SetEndian(property.littleEndian)
		}
	}
}

func Field[T any](name string, options ...FieldOption) PackedProperty {

	property := PackedProperty{name: name, tags: make(map[string]string)}

	property.packed = new(T)

	for _, option := range options {
		option(&property)
	}

	property.kind, property.finalType = validatePropertyType(property.packed)

	if property.kind == KindInvalid {
		panic(fmt.Sprintf("propertyType %T does not implement ConverterInterface, TypeInterface, or is not a *PackedStruct", property.packed))
	}

	property.propertyType = reflect.TypeOf(property.packed)

	if _, exists := converters[property.propertyType]; !exists && property.kind == KindConverter {
		converters[property.propertyType] = converter{variable: property.propertyType.Elem().Name() + "Converter", external: false}
	}

	property.size = property.packed.(interface{ Size() int }).Size()

	imported[property.propertyType.PkgPath()] = true

	if property.finalType != nil {
		imported[property.finalType.PkgPath()] = true
	}

	return property
}

type PackedStruct struct {
	name         string
	properties   []PackedProperty
	size         int
	littleEndian bool
}

func (s *PackedStruct) Size() int { return s.size }

func Struct(name string, littleEndian bool, properties ...PackedProperty) *PackedStruct {
	var lastBitField *BitField

	var size int

	for _, property := range properties {

		property.SetEndian(littleEndian)

		if property.bitField != nil {
			property.bitField.previous = lastBitField
		}

		if lastBitField != nil {
			lastBitField.next = property.bitField
		}

		lastBitField = property.bitField

		size += property.size
	}

	packed := &PackedStruct{name: name, properties: properties, size: size, littleEndian: littleEndian}

	structs[name] = packed

	return packed
}

func RegisterConverter[T any](variable string) {
	converters[reflect.TypeOf(new(T))] = converter{variable: variable, external: true}
}

func Generate() {
	for reflection, variable := range converters {
		fmt.Println(reflection, variable)
	}

	buffer := bytes.Buffer{}

	buffer.WriteString("package packed\n\n")
	buffer.WriteString("import (\n")
	for importPath := range imported {
		buffer.WriteString(fmt.Sprintf("\"%s\"\n", importPath))
	}
	buffer.WriteString(")\n\n")

	buffer.WriteString("var (\n")
	for reflection, variable := range converters {

		if variable.external {
			continue
		}

		buffer.WriteString(fmt.Sprintf("%s = &%s{}\n", variable.variable, reflection.Elem().String()))

	}
	buffer.WriteString(")\n")

	for _, packed := range structs {
		buffer.Write(packed.StructDefinition())
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("ToBytesLittleEndian"))
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("FromBytesLittleEndian"))
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("ToBytesBigEndian"))
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("FromBytesBigEndian"))
		buffer.WriteString("\n")
	}

	result, err := imports.Process("", buffer.Bytes(), &imports.Options{
		AllErrors:  true,
		FormatOnly: false,
	})

	os.MkdirAll("./generated/packed", 0755)

	if err != nil {
		os.WriteFile("./generated/packed/packed.go", buffer.Bytes(), 0644)
		fmt.Println(err)
		return
	}

	os.WriteFile("./generated/packed/packed.go", result, 0644)

}
