package packed

import (
	"fmt"
	"reflect"
	"slices"
)

type kind int

const (
	kindInvalid kind = iota
	kindType
	kindStruct
	kindConverter
	kindConverterCast
	kindArray
	kindBitField
	kindBitFieldGroup
)

type structTag struct {
	key   string
	value string
}

type packedProperty struct {
	name           string
	size           int
	packed         any
	recieverType   reflect.Type
	propertyType   reflect.Type
	tags           []structTag
	kind           kind
	littleEndian   bool
	endianOverride bool
	converter      *converterHash
}

type fieldOption func(*packedProperty)

func Tag(key, value string) fieldOption {
	return func(definition *packedProperty) {
		definition.tags = append(definition.tags, structTag{key: key, value: value})
	}
}

func (p *packedStruct) replacePropertiesWithClone() {
	p.properties = slices.Clone(p.properties)

	for i, child := range p.properties {
		if packed, ok := child.packed.(packedStruct); ok {
			packed.replacePropertiesWithClone()
			child.packed = packed
			p.properties[i] = child
		}
	}
}

func LittleEndian(value bool) fieldOption {
	return func(definition *packedProperty) {

		if definition.kind == kindBitField {
			panic("endianness cannot be set for bit fields")
		}

		definition.littleEndian = value
		definition.endianOverride = true

		if packed, ok := definition.packed.(packedStruct); ok {
			packed.setEndianProperties(value, true)
			definition.packed = packed
		}
	}
}

func toPointer(structure any) any {
	value := reflect.ValueOf(structure)

	if value.Kind() == reflect.Ptr {
		return structure
	}

	pointer := reflect.New(value.Type())
	pointer.Elem().Set(value)

	return pointer.Interface()
}

func validatePropertyType(propertyType any) (kind, reflect.Type, any) {

	if _, ok := propertyType.(packedStruct); ok {
		return kindStruct, nil, propertyType
	}

	if _, ok := propertyType.(packedArray); ok {
		return kindArray, nil, propertyType
	}

	if _, ok := propertyType.(packedBitField); ok {
		return kindBitField, nil, propertyType
	}

	if cast, ok := propertyType.(converterCast); ok {
		return kindConverterCast, cast.target, propertyType
	}

	propertyType = toPointer(propertyType)

	if _, ok := propertyType.(TypeInterface); ok {
		return kindType, reflect.TypeOf(propertyType).Elem(), propertyType
	}

	if reciever, ok := implementsConverterInterface(propertyType); ok {

		if overwrite, ok := reciever.(OverwriteConverterReciverReflectionInterface); ok {
			reciever = reflect.TypeOf(overwrite.OverwriteConverterReciverReflection(reflect.TypeOf(propertyType)))
		}

		return kindConverter, reciever, propertyType
	}

	return kindInvalid, nil, propertyType
}

func Field(name string, propertyType any, options ...fieldOption) packedProperty {

	property := packedProperty{name: name, tags: []structTag{}}

	if packed, ok := any(propertyType).(packedStruct); ok {
		packed.replacePropertiesWithClone()
		property.packed = packed
	} else {
		property.packed = propertyType
	}

	property.kind, property.recieverType, property.packed = validatePropertyType(property.packed)

	if property.kind == kindInvalid {
		panic(fmt.Sprintf("invalid property type: %T", property.packed))
	}

	for _, option := range options {
		option(&property)
	}

	property.propertyType = reflect.TypeOf(property.packed)

	if property.kind == kindConverter {

		hash := createConverterHash(property.packed)

		if _, exists := converters[hash.hash]; !exists {
			converters[hash.hash] = hash
		}

		property.converter = &hash
	}

	if property.kind != kindBitField {
		property.size = property.packed.(interface{ Size() int }).Size()
	}

	imported[property.propertyType.PkgPath()] = true

	if property.recieverType != nil {
		imported[property.recieverType.PkgPath()] = true
	}

	return property
}
