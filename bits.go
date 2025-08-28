package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/exp/constraints"
)

type packedBitFieldGroup struct {
	groupIndex int
	fields     []packedBitField
	size       int
}

func initPackedBitFieldGroup(groupIndex int, fields []packedBitField) packedBitFieldGroup {
	totalBits := 0
	for _, field := range fields {
		totalBits += field.bitSize
	}
	size := (totalBits + 7) / 8
	if size < 1 || size > 8 {
		panic(fmt.Sprintf("invalid bit group size: %d bytes (%d bits)", size, totalBits))
	}
	return packedBitFieldGroup{
		groupIndex: groupIndex,
		fields:     fields,
		size:       size,
	}
}

type packedBitField struct {
	bitSize            int
	reflection         reflect.Type
	packedProperty     packedProperty
	bitsType           bool
	bitsTypeReflection reflect.Type
}

func (p packedBitField) signed() bool {
	switch p.reflection.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return false
	default:
		panic("invalid type for bitfield")
	}
}

type bitsType[Integer constraints.Integer] interface {
	Set(Integer)
	Integer() Integer
}

func Bits[Interger constraints.Integer](bits int, bitsType ...bitsType[Interger]) packedBitField {
	var value Interger

	size := unsafe.Sizeof(value) * 8
	if bits > int(size) {
		panic("bits cannot be larger than the underlying type size")
	}

	field := packedBitField{bitSize: bits, reflection: reflect.TypeOf(value)}

	if len(bitsType) != 0 {

		if len(bitsType) != 1 {
			panic("bits type must be a single type")
		}

		if bitsType[0] == nil {
			panic("bits type cannot be nil")
		}

		field.bitsType = true
		field.bitsTypeReflection = reflect.TypeOf(bitsType[0])
	}

	return field
}

func (g packedBitFieldGroup) writeToBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offset string,
) {
	fmt.Fprintf(buffer, "var b%d uint64\n", g.groupIndex)
	if littleEndian {
		runningOffset := 0
		for _, field := range g.fields {

			receiver := receiverVariable + field.packedProperty.name

			if field.bitsType {
				receiver += ".Integer()"
			}

			mask := (uint64(1) << field.bitSize) - 1
			bitOffset := runningOffset
			if bitOffset == 0 {
				fmt.Fprintf(buffer,
					"b%d |= (uint64(%s) & 0x%X)\n",
					g.groupIndex,
					receiver,
					mask,
				)
			} else {
				fmt.Fprintf(buffer,
					"b%d |= (uint64(%s) & 0x%X) << %d\n",
					g.groupIndex,
					receiver,
					mask,
					bitOffset,
				)
			}
			runningOffset += field.bitSize
		}
	} else {
		remainingBits := g.size * 8
		for _, field := range g.fields {

			receiver := receiverVariable + field.packedProperty.name

			if field.bitsType {
				receiver += ".Integer()"
			}

			mask := (uint64(1) << field.bitSize) - 1
			bitOffset := remainingBits - field.bitSize
			if bitOffset == 0 {
				fmt.Fprintf(buffer,
					"b%d |= (uint64(%s) & 0x%X)\n",
					g.groupIndex,
					receiver,
					mask,
				)
			} else {
				fmt.Fprintf(buffer,
					"b%d |= (uint64(%s) & 0x%X) << %d\n",
					g.groupIndex,
					receiver,
					mask,
					bitOffset,
				)
			}
			remainingBits -= field.bitSize
		}
	}
	if littleEndian {
		for i := 0; i < g.size; i++ {
			fmt.Fprintf(buffer,
				"bytes[%s+%d] = byte(b%d >> %d)\n",
				offset,
				i,
				g.groupIndex,
				8*i,
			)
		}
	} else {
		for i := 0; i < g.size; i++ {
			fmt.Fprintf(buffer,
				"bytes[%s+%d] = byte(b%d >> %d)\n",
				offset,
				g.size-1-i,
				g.groupIndex,
				8*i,
			)
		}
	}
}

func (g packedBitFieldGroup) writeFromBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offset string,
) {
	fmt.Fprintf(buffer, "var b%d uint64\n", g.groupIndex)
	if littleEndian {
		for i := 0; i < g.size; i++ {
			fmt.Fprintf(buffer,
				"b%d |= uint64(bytes[%s+%d]) << %d\n",
				g.groupIndex,
				offset,
				i,
				8*i,
			)
		}
	} else {
		for i := 0; i < g.size; i++ {
			fmt.Fprintf(buffer,
				"b%d |= uint64(bytes[%s+%d]) << %d\n",
				g.groupIndex,
				offset,
				g.size-1-i,
				8*i,
			)
		}
	}
	if littleEndian {
		runningOffset := 0
		for _, field := range g.fields {

			receiver := receiverVariable + field.packedProperty.name

			if field.bitsType {
				fmt.Fprintf(buffer, "%s.Set(", receiver)
			} else {
				fmt.Fprintf(buffer, "%s = ", receiver)
			}

			mask := (uint64(1) << field.bitSize) - 1
			bitOffset := runningOffset
			if field.signed() {
				fmt.Fprintf(buffer,
					"%s((( (b%d >> %d) & 0x%X ) ^ (1 << %d)) - (1 << %d))",
					field.reflection.String(),
					g.groupIndex,
					bitOffset,
					mask,
					field.bitSize-1,
					field.bitSize-1,
				)
			} else {
				fmt.Fprintf(buffer,
					"%s(uint64((b%d >> %d) & 0x%X))",
					field.reflection.String(),
					g.groupIndex,
					bitOffset,
					mask,
				)
			}
			runningOffset += field.bitSize

			if field.bitsType {
				fmt.Fprintf(buffer, ")\n")
			} else {
				fmt.Fprintf(buffer, "\n")
			}
		}
	} else {
		remainingBits := g.size * 8
		for _, field := range g.fields {

			receiver := receiverVariable + field.packedProperty.name

			if field.bitsType {
				fmt.Fprintf(buffer, "%s.Set(", receiver)
			} else {
				fmt.Fprintf(buffer, "%s = ", receiver)
			}

			mask := (uint64(1) << field.bitSize) - 1
			bitOffset := remainingBits - field.bitSize
			if field.signed() {
				fmt.Fprintf(buffer,
					"%s((( (b%d >> %d) & 0x%X ) ^ (1 << %d)) - (1 << %d))",
					field.reflection.String(),
					g.groupIndex,
					bitOffset,
					mask,
					field.bitSize-1,
					field.bitSize-1,
				)
			} else {
				fmt.Fprintf(buffer,
					"%s(uint64((b%d >> %d) & 0x%X))",
					field.reflection.String(),
					g.groupIndex,
					bitOffset,
					mask,
				)
			}

			if field.bitsType {
				fmt.Fprintf(buffer, ")\n")
			} else {
				fmt.Fprintf(buffer, "\n")
			}

			remainingBits -= field.bitSize
		}
	}
}
