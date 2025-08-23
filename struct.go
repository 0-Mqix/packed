package packed

import "fmt"

const maxBitsPerGroup = 64

type PackedStruct struct {
	name         string
	properties   []PackedProperty
	size         int
	littleEndian bool
}

func (s PackedStruct) Size() int { return s.size }

func createBitFieldGroup(fields []PackedBitField, groupIndex *int, littleEndian bool) PackedProperty {

	group := InitPackedBitFieldGroup(*groupIndex, fields)

	packed := PackedProperty{
		size:         group.size,
		packed:       group,
		kind:         KindBitFieldGroup,
		littleEndian: littleEndian,
	}

	*groupIndex++

	return packed
}

func Struct(name string, littleEndian bool, properties ...PackedProperty) PackedStruct {

	processedProperties := []PackedProperty{}
	currentBitFields := []PackedBitField{}
	propertyNames := map[string]bool{}
	size := 0

	addBitFieldGroup := func(fields []PackedBitField, groupIndex *int, littleEndian bool) {
		property := createBitFieldGroup(fields, groupIndex, littleEndian)
		processedProperties = append(processedProperties, property)
		size += property.size
	}

	groupIndex := 0

	for _, property := range properties {

		if _, ok := propertyNames[property.name]; ok {
			panic(fmt.Sprintf("property %s already exists", property.name))
		} else {
			propertyNames[property.name] = true
		}

		if property.kind != KindBitField {
			if len(currentBitFields) > 0 {
				addBitFieldGroup(currentBitFields, &groupIndex, littleEndian)
				currentBitFields = nil
			}

			processedProperties = append(processedProperties, property)
			size += property.size
			continue
		}

		bitField := property.packed.(PackedBitField)
		bitField.packedProperty = property

		totalBits := 0

		for _, existingField := range currentBitFields {
			totalBits += existingField.bitSize
		}

		if totalBits+bitField.bitSize > maxBitsPerGroup && len(currentBitFields) > 0 {
			addBitFieldGroup(currentBitFields, &groupIndex, littleEndian)
			currentBitFields = []PackedBitField{}
		} else {
			currentBitFields = append(currentBitFields, bitField)
		}

	}

	if len(currentBitFields) > 0 {
		addBitFieldGroup(currentBitFields, &groupIndex, littleEndian)
	}

	packed := PackedStruct{
		name:         name,
		size:         size,
		littleEndian: littleEndian,
		properties:   processedProperties,
	}

	packed.SetEndianProperties(littleEndian, false)

	structs[name] = packed

	return packed
}
