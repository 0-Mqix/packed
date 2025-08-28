package packed

import "fmt"

type PackedStruct struct {
	name         string
	properties   []PackedProperty
	size         int
	littleEndian bool
}

func (s PackedStruct) Size() int { return s.size }

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

func (p *PackedStruct) SetBitFieldGroupIndexes(index *int) {

	for i, child := range p.properties {

		switch child.kind {

		case KindBitFieldGroup:
			group := child.packed.(PackedBitFieldGroup)
			group.groupIndex = *index
			child.packed = group
			*index++

		case KindStruct:
			packed := child.packed.(PackedStruct)
			packed.SetBitFieldGroupIndexes(index)
			child.packed = packed

		default:
			continue
		}

		p.properties[i] = child
	}
}

func createBitFieldGroup(fields []PackedBitField, littleEndian bool) PackedProperty {

	group := InitPackedBitFieldGroup(0, fields)

	packed := PackedProperty{
		size:         group.size,
		packed:       group,
		kind:         KindBitFieldGroup,
		littleEndian: littleEndian,
	}

	return packed
}

func Struct(name string, littleEndian bool, properties ...PackedProperty) PackedStruct {

	if _, ok := structs[name]; ok {
		panic(fmt.Sprintf("struct %s already exists", name))
	}

	processedProperties := []PackedProperty{}
	currentBitFields := []PackedBitField{}
	propertyNames := map[string]bool{}
	size := 0

	addBitFieldGroup := func(fields []PackedBitField, littleEndian bool) {
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

		if property.kind != KindBitField {
			if len(currentBitFields) > 0 {
				addBitFieldGroup(currentBitFields, littleEndian)
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

		if (totalBits+bitField.bitSize+7)/8 > 8 {
			addBitFieldGroup(currentBitFields, littleEndian)
			currentBitFields = []PackedBitField{bitField}
		} else {
			currentBitFields = append(currentBitFields, bitField)
		}

	}

	if len(currentBitFields) > 0 {
		addBitFieldGroup(currentBitFields, littleEndian)
	}

	packed := PackedStruct{
		name:         name,
		size:         size,
		littleEndian: littleEndian,
		properties:   processedProperties,
	}

	packed.SetEndianProperties(littleEndian, false)

	groupIndex := 0

	packed.SetBitFieldGroupIndexes(&groupIndex)

	structs[name] = packed

	return packed
}
