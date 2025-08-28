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
	KindConverterCast
	KindArray
	KindBitField
	KindBitFieldGroup
)

type StructTag struct {
	key   string
	value string
}

type PackedProperty struct {
	name           string
	size           int
	packed         any
	recieverType   reflect.Type
	propertyType   reflect.Type
	tags           []StructTag
	kind           Kind
	littleEndian   bool
	endianOverride bool
	converter      *converterHash
}

type FieldOption func(*PackedProperty)

func Tag(key, value string) FieldOption {
	return func(definition *PackedProperty) {
		definition.tags = append(definition.tags, StructTag{key: key, value: value})
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

func LittleEndian(value bool) FieldOption {
	return func(definition *PackedProperty) {

		if definition.kind == KindBitField {
			panic("endianness cannot be set for bit fields")
		}

		definition.littleEndian = value
		definition.endianOverride = true

		if packed, ok := definition.packed.(PackedStruct); ok {
			packed.SetEndianProperties(value, true)
			definition.packed = packed
		}
	}
}

func structToPointer(structure any) any {
	value := reflect.ValueOf(structure)

	if value.Kind() != reflect.Struct {
		return structure
	}

	pointer := reflect.New(value.Type())
	pointer.Elem().Set(value)
	return pointer.Interface()
}

func validatePropertyType(propertyType any) (Kind, reflect.Type, any) {

	if _, ok := propertyType.(PackedStruct); ok {
		return KindStruct, nil, propertyType
	}

	if _, ok := propertyType.(PackedArray); ok {
		return KindArray, nil, propertyType
	}

	if _, ok := propertyType.(PackedBitField); ok {
		return KindBitField, nil, propertyType
	}

	if cast, ok := propertyType.(ConverterCast); ok {
		return KindConverterCast, cast.target, propertyType
	}

	propertyType = structToPointer(propertyType)

	if _, ok := propertyType.(TypeInterface); ok {
		return KindType, reflect.TypeOf(propertyType).Elem(), propertyType
	}

	if reciever, ok := implementsConverterInterface(propertyType); ok {

		if overwrite, ok := reciever.(OverwriteConverterReciverReflectionInterface); ok {
			reciever = reflect.TypeOf(overwrite.OverwriteConverterReciverReflection(reflect.TypeOf(propertyType)))
		}

		return KindConverter, reciever, propertyType
	}

	return KindInvalid, nil, propertyType
}

func Field(name string, propertyType any, options ...FieldOption) PackedProperty {

	property := PackedProperty{name: name, tags: []StructTag{}}

	if packed, ok := any(propertyType).(PackedStruct); ok {
		packed.replacePropertiesWithClone()
		property.packed = packed
	} else {
		property.packed = propertyType
	}

	property.kind, property.recieverType, property.packed = validatePropertyType(property.packed)

	if property.kind == KindInvalid {
		panic(fmt.Sprintf("invalid property type: %T", property.packed))
	}

	for _, option := range options {
		option(&property)
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
