package packed

type PackedStruct struct {
	name         string
	properties   []PackedProperty
	size         int
	littleEndian bool
}

func (s PackedStruct) Size() int { return s.size }

func Struct(name string, littleEndian bool, properties ...PackedProperty) PackedStruct {
	var lastBitField *BitField

	var size int

	packed := PackedStruct{
		name:         name,
		size:         size,
		littleEndian: littleEndian,
		properties:   make([]PackedProperty, len(properties)),
	}

	for i, property := range properties {

		if property.bitField != nil {
			property.bitField.previous = lastBitField
		}

		if lastBitField != nil {
			lastBitField.next = property.bitField
		}

		lastBitField = property.bitField

		size += property.size

		packed.properties[i] = property
	}

	packed.SetEndianProperties(littleEndian, false)

	structs[name] = packed

	return packed
}
