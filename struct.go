package packed

import (
	"fmt"
	"reflect"
)

type packedStruct struct {
	name                   string
	properties             []packedProperty
	size                   int
	littleEndian           bool
	converterCastRecievers map[reflect.Type]int
}

func (p packedStruct) Size() int { return p.size }

func (p *packedStruct) setEndianProperties(littleEndian bool, forceOverride bool) {

	for i, child := range p.properties {

		if !child.endianOverride || forceOverride {

			child.littleEndian = littleEndian

			if forceOverride {
				child.endianOverride = true
			}
		}

		if packed, ok := child.packed.(packedStruct); ok {
			packed.setEndianProperties(littleEndian, forceOverride)
			child.packed = packed
		}

		p.properties[i] = child
	}
}

func (p *packedStruct) setBitFieldGroupIndexes(index *int) {

	for i, child := range p.properties {

		switch child.kind {

		case kindBitFieldGroup:
			group := child.packed.(packedBitFieldGroup)
			group.groupIndex = *index
			child.packed = group
			*index++

		case kindStruct:
			packed := child.packed.(packedStruct)
			packed.setBitFieldGroupIndexes(index)
			child.packed = packed

		default:
			continue
		}

		p.properties[i] = child
	}
}

func (p *packedStruct) getConverterCastRecievers(converterCastRecievers map[reflect.Type]int) {
	for i, child := range p.properties {

		switch child.kind {

		case kindConverterCast:
			cast := child.packed.(converterCast)

			if _, ok := converterCastRecievers[cast.reciever]; !ok {
				converterCastRecievers[cast.reciever] = len(converterCastRecievers)
			}

			child.packed = cast

		case kindStruct:
			packed := child.packed.(packedStruct)
			packed.getConverterCastRecievers(converterCastRecievers)
			child.packed = packed

		case kindArray:
			packed := child.packed.(packedArray)

			switch packed.ElementKind {

			case kindStruct:
				structure := packed.Element.(packedStruct)
				structure.getConverterCastRecievers(converterCastRecievers)
				packed.Element = structure

			case kindConverterCast:
				cast := packed.Element.(converterCast)

				if _, ok := converterCastRecievers[cast.reciever]; !ok {
					converterCastRecievers[cast.reciever] = len(converterCastRecievers)
				}

				packed.Element = cast
			}

			child.packed = packed

		default:
			continue
		}

		p.properties[i] = child
	}
}

func createBitFieldGroup(fields []packedBitField, littleEndian bool) packedProperty {

	group := initPackedBitFieldGroup(0, fields)

	packed := packedProperty{
		size:         group.size,
		packed:       group,
		kind:         kindBitFieldGroup,
		littleEndian: littleEndian,
	}

	return packed
}

func Struct(name string, littleEndian bool, properties ...packedProperty) packedStruct {

	if _, ok := structs[name]; ok {
		panic(fmt.Sprintf("struct %s already exists", name))
	}

	processedProperties := []packedProperty{}
	currentBitFields := []packedBitField{}
	propertyNames := map[string]bool{}
	size := 0

	addBitFieldGroup := func(fields []packedBitField, littleEndian bool) {
		property := createBitFieldGroup(fields, littleEndian)
		processedProperties = append(processedProperties, property)
		size += property.size
	}

	for _, property := range properties {

		if _, ok := propertyNames[property.name]; ok {
			panic(fmt.Sprintf("property %s already exists", property.name))
		} else {
			propertyNames[property.name] = true
		}

		if property.kind != kindBitField {
			if len(currentBitFields) > 0 {
				addBitFieldGroup(currentBitFields, littleEndian)
				currentBitFields = nil
			}

			processedProperties = append(processedProperties, property)
			size += property.size
			continue
		}

		bitField := property.packed.(packedBitField)
		bitField.packedProperty = property

		totalBits := 0

		for _, existingField := range currentBitFields {
			totalBits += existingField.bitSize
		}

		if (totalBits+bitField.bitSize+7)/8 > 8 {
			addBitFieldGroup(currentBitFields, littleEndian)
			currentBitFields = []packedBitField{bitField}
		} else {
			currentBitFields = append(currentBitFields, bitField)
		}

	}

	if len(currentBitFields) > 0 {
		addBitFieldGroup(currentBitFields, littleEndian)
	}

	packed := packedStruct{
		name:                   name,
		size:                   size,
		littleEndian:           littleEndian,
		properties:             processedProperties,
		converterCastRecievers: map[reflect.Type]int{},
	}

	bitFieldGroupIndex := 0

	packed.setEndianProperties(littleEndian, false)
	packed.setBitFieldGroupIndexes(&bitFieldGroupIndex)
	packed.getConverterCastRecievers(packed.converterCastRecievers)

	structs[name] = packed

	return packed
}
