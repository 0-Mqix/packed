package packed

import (
	"fmt"
	"reflect"
	"slices"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindType
	KindStruct
	KindConverter
	KindArray
	KindBitField
	KindBitFieldGroup
)

type PackedProperty struct {
	name           string
	size           int
	packed         any
	recieverType   reflect.Type
	propertyType   reflect.Type
	tags           map[string]string
	kind           Kind
	littleEndian   bool
	endianOverride bool
	converter      *converterHash
}

type FieldOption func(*PackedProperty)

func Tag(key, value string) FieldOption {
	return func(definition *PackedProperty) {
		definition.tags[key] = value
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

		if reflect.TypeOf(definition.packed).String() != "*interface {}" {
			panic("type option can only be used if the field generic argument is any")
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

	if _, ok := propertyType.(PackedArray); ok {
		return KindArray, nil
	}

	if _, ok := propertyType.(PackedBitField); ok {
		return KindBitField, nil
	}

	if reciever, ok := implementsConverterInterface(propertyType); ok {

		if overwrite, ok := reciever.(OverwriteConverterReciverReflectionInterface); ok {
			reciever = reflect.TypeOf(overwrite.OverwriteConverterReciverReflection(reflect.TypeOf(propertyType)))
		}

		return KindConverter, reciever
	}

	return KindInvalid, nil
}

func Field(name string, propertyType any, options ...FieldOption) PackedProperty {

	property := PackedProperty{name: name, tags: make(map[string]string)}

	if packed, ok := propertyType.(PackedStruct); ok {
		packed.replacePropertiesWithClone()
		property.packed = packed
	} else {
		property.packed = propertyType
	}

	for _, option := range options {
		option(&property)
	}

	property.kind, property.recieverType = validatePropertyType(property.packed)

	if property.kind == KindInvalid {
		panic(fmt.Sprintf("invalid property type: %T", property.packed))
	}

	property.propertyType = reflect.TypeOf(property.packed)

	if property.kind == KindConverter {

		hash := createConverterHash(property.packed)

		if _, exists := converters[hash.hash]; !exists {
			converters[hash.hash] = hash
		}

		property.converter = &hash
	}

	if property.kind != KindBitField {
		property.size = property.packed.(interface{ Size() int }).Size()
	}

	imported[property.propertyType.PkgPath()] = true

	if property.recieverType != nil {
		imported[property.recieverType.PkgPath()] = true
	}

	return property
}
