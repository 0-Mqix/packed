package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/exp/constraints"
)

type PackedBitFieldGroup struct {
	groupIndex int
	converter  converterHash
	fields     []PackedBitField
	size       int
}

func InitPackedBitFieldGroup(groupIndex int, fields []PackedBitField) PackedBitFieldGroup {

	var converter any
	var size int

	for _, field := range fields {
		size += field.bitSize
	}

	size = (size + 7) / 8

	switch size {

	case 1:
		converter = Uint8Converter{}
	case 2:
		converter = Uint16Converter{}
	case 4:
		converter = Uint32Converter{}
	case 8:
		converter = Uint64Converter{}
	}

	hash := createConverterHash(converter)

	if _, exists := converters[hash.hash]; !exists {
		converters[hash.hash] = hash
	}

	return PackedBitFieldGroup{groupIndex: groupIndex, fields: fields, converter: hash, size: size}
}

type PackedBitField struct {
	bitSize        int
	reflection     reflect.Type
	packedProperty PackedProperty
}

func (p PackedBitField) Signed() bool {

	switch p.reflection.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return false

	default:
		panic("invalid type")
	}
}

func Bits[T constraints.Integer](bits int) PackedBitField {
	var value T

	size := unsafe.Sizeof(value) * 8

	if bits > int(size) {
		panic("bits cannot be larger than the underlying type size")
	}

	return PackedBitField{bitSize: bits, reflection: reflect.TypeOf(value)}
}
func (g PackedBitFieldGroup) WriteToBytes(buffer *bytes.Buffer, recieverVariable string, endian string, offset string) {
	fmt.Fprintf(buffer, "var b%d uint%d\n", g.groupIndex, g.size*8)

	var bitOffset int

	for _, field := range g.fields {

		reciever := recieverVariable + field.packedProperty.name

		mask := (uint64(1) << field.bitSize) - 1

		if bitOffset == 0 {
			fmt.Fprintf(buffer,
				"b%d |= (uint%d(%s) & 0x%X)\n",
				g.groupIndex,
				g.size*8,
				reciever,
				mask,
			)

		} else {
			fmt.Fprintf(buffer,
				"b%d |= (uint%d(%s) & 0x%X) << %d\n",
				g.groupIndex,
				g.size*8,
				reciever,
				mask,
				bitOffset,
			)
		}

		bitOffset += field.bitSize
	}

	fmt.Fprintf(buffer,
		"%s.ToBytes%s(&b%d, bytes, %s)\n",
		getConverterName(g.converter.hash),
		endian,
		g.groupIndex,
		offset,
	)
}

func (g PackedBitFieldGroup) WriteFromBytes(buffer *bytes.Buffer, recieverVariable string, endian string, offset string) {
	fmt.Fprintf(buffer, "var b%d uint%d\n", g.groupIndex, g.size*8)
	fmt.Fprintf(buffer, "%s.FromBytes%s(&b%d, bytes, %s)\n",
		getConverterName(g.converter.hash), endian, g.groupIndex, offset)

	var bitOffset int

	for _, field := range g.fields {
		reciever := recieverVariable + field.packedProperty.name

		if field.Signed() {
			leftShift := g.size*8 - (field.bitSize + bitOffset)
			rightShift := g.size*8 - field.bitSize

			fmt.Fprintf(buffer,
				"%s = %s((int%d(b%d << %d)) >> %d)\n",
				reciever,
				field.reflection.String(),
				g.size*8,
				g.groupIndex,
				leftShift,
				rightShift,
			)

		} else {
			mask := (1 << field.bitSize) - 1
			fmt.Fprintf(buffer,
				"%s = %s(uint%d((b%d >> %d) & 0x%X))\n",
				reciever,
				field.reflection.String(),
				g.size*8,
				g.groupIndex,
				bitOffset,
				mask,
			)
		}

		bitOffset += field.bitSize
	}
}
