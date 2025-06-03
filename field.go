package packed

import (
	"reflect"
	"slices"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindType
	KindStruct
	KindConverter
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
	recieverType   reflect.Type
	propertyType   reflect.Type
	tags           map[string]string
	bitField       *BitField
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

func (p *PackedStruct) replacePropertiesWithClone() {
	p.properties = slices.Clone(p.properties)

	for i, child := range p.properties {
		if packed, ok := child.packed.(PackedStruct); ok {
			packed.replacePropertiesWithClone()
			child.packed = packed
			p.properties[i] = child
		}
	}
}

func Type(propertyType any) FieldOption {
	return func(definition *PackedProperty) {
		if packed, ok := propertyType.(PackedStruct); ok {
			packed.replacePropertiesWithClone()
			definition.packed = packed
		} else {
			definition.packed = propertyType
		}
	}
}

func (p PackedStruct) SetEndianProperties(littleEndian bool, forceOverride bool) {

	for i, child := range p.properties {

		if !child.endianOverride || forceOverride {

			child.littleEndian = littleEndian

			if forceOverride {
				child.endianOverride = true
			}
		}

		if packed, ok := child.packed.(PackedStruct); ok {
			packed.SetEndianProperties(littleEndian, forceOverride)
			child.packed = packed
		}

		p.properties[i] = child
	}
}

func LittleEndian(value bool) FieldOption {
	return func(definition *PackedProperty) {

		definition.littleEndian = value
		definition.endianOverride = true

		if packed, ok := definition.packed.(PackedStruct); ok {
			packed.SetEndianProperties(value, true)
			definition.packed = packed
		}
	}
}

func validatePropertyType(propertyType any) (Kind, reflect.Type) {

	if _, ok := propertyType.(TypeInterface); ok {
		return KindType, reflect.TypeOf(propertyType)
	}

	if _, ok := propertyType.(PackedStruct); ok {
		return KindStruct, nil
	}

	if reciever, ok := implementsConverterInterface(propertyType); ok {
		return KindConverter, reciever
	}

	return KindInvalid, nil
}

func Field[T any](name string, options ...FieldOption) PackedProperty {

	property := PackedProperty{name: name, tags: make(map[string]string)}

	property.packed = new(T)

	for _, option := range options {
		option(&property)
	}

	property.kind, property.recieverType = validatePropertyType(property.packed)

	property.propertyType = reflect.TypeOf(property.packed)

	if _, exists := converters[property.propertyType]; !exists && property.kind == KindConverter {
		converters[property.propertyType] = converter{variable: property.propertyType.Elem().Name() + "Converter", external: false}
	}

	property.size = property.packed.(interface{ Size() int }).Size()

	imported[property.propertyType.PkgPath()] = true

	if property.recieverType != nil {
		imported[property.recieverType.PkgPath()] = true
	}

	return property
}
