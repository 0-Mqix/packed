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
		panic(fmt.Sprintf(
			"invalid bit group size: %d bytes (%d bits)",
			size,
			totalBits,
		))
	}
	return packedBitFieldGroup{
		groupIndex: groupIndex,
		fields:     fields,
		size:       size,
	}
}

type bitFieldKind int

const (
	bitFieldKindInteger bitFieldKind = iota
	bitFieldKindBoolean
	bitFieldKindBitsType
	bitFieldKindBitsConverter
)

type packedBitField struct {
	bitSize              int
	reflection           reflect.Type
	packedProperty       packedProperty
	bitFieldKind         bitFieldKind
	bitsTargetReflection reflect.Type
	converter            *converterHash
}

func (p packedBitField) signed() bool {
	switch p.reflection.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Bool, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return false
	default:
		panic("invalid type for bitfield")
	}
}

type BitsTypeInterface[Integer constraints.Integer] interface {
	Set(Integer)
	Integer() Integer
}

type BitsConverterInterface[Integer constraints.Integer, Reciever any] interface {
	Set(*Reciever, Integer)
	Integer(*Reciever) Integer
}

func Bits[Interger constraints.Integer](bits int, bitsTarget ...any) packedBitField {
	var value Interger

	size := unsafe.Sizeof(value) * 8
	if bits > int(size) {
		panic("bits cannot be larger than the underlying type size")
	}

	reflection := reflect.TypeOf(value)

	field := packedBitField{
		bitSize:      bits,
		reflection:   reflection,
		bitFieldKind: bitFieldKindInteger,
	}

	if len(bitsTarget) == 0 || bitsTarget == nil {
		return field
	}

	target := toPointer(bitsTarget[0])

	if _, ok := target.(BitsTypeInterface[Interger]); ok {
		field.bitFieldKind = bitFieldKindBitsType
		field.bitsTargetReflection = reflect.TypeOf(target)
		return field
	}

	reciever, ok := implementsBitsConverterInterface(target, reflection)

	if !ok {
		panic("invalid type for bitfield")
	}

	field.bitFieldKind = bitFieldKindBitsConverter
	field.bitsTargetReflection = reciever

	hash := createConverterHash(target)

	if _, exists := converters[hash.hash]; !exists {
		converters[hash.hash] = hash
	}

	field.converter = &hash

	return field
}

var Bit = packedBitField{
	bitSize:      1,
	reflection:   reflect.TypeOf(bool(false)),
	bitFieldKind: bitFieldKindBoolean,
}

type fieldOffset struct {
	field     packedBitField
	bitOffset int
}

func (g packedBitFieldGroup) computeOffsets(littleEndian bool) []fieldOffset {
	result := make([]fieldOffset, 0, len(g.fields))

	if littleEndian {
		runningOffset := 0
		for _, field := range g.fields {
			result = append(result, fieldOffset{field: field, bitOffset: runningOffset})
			runningOffset += field.bitSize
		}
		return result
	}

	remainingBits := g.size * 8

	for _, field := range g.fields {
		bitOffset := remainingBits - field.bitSize
		result = append(result, fieldOffset{field: field, bitOffset: bitOffset})
		remainingBits -= field.bitSize
	}

	return result
}

func (g packedBitFieldGroup) writeToBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offset string,
) {
	fmt.Fprintf(buffer, "var b%d uint64\n", g.groupIndex)

	offsets := g.computeOffsets(littleEndian)
	for _, fieldOffset := range offsets {
		field := fieldOffset.field
		bitOffset := fieldOffset.bitOffset
		receiver := receiverVariable + field.packedProperty.name

		if field.bitFieldKind == bitFieldKindBoolean {
			fmt.Fprintf(
				buffer,
				"b%d |= (uint64(*(*uint8)(unsafe.Pointer(&%s))) & 1) << %d\n",
				g.groupIndex,
				receiver,
				bitOffset,
			)
			continue
		}

		switch field.bitFieldKind {

		case bitFieldKindBitsType:
			receiver += ".Integer()"

		case bitFieldKindBitsConverter:
			receiver = fmt.Sprintf("%s.Integer(&%s)", getConverterName(field.converter.hash), receiver)
		}

		mask := (uint64(1) << field.bitSize) - 1
		if bitOffset == 0 {
			fmt.Fprintf(
				buffer,
				"b%d |= (uint64(%s) & 0x%X)\n",
				g.groupIndex,
				receiver,
				mask,
			)
		} else {
			fmt.Fprintf(
				buffer,
				"b%d |= (uint64(%s) & 0x%X) << %d\n",
				g.groupIndex,
				receiver,
				mask,
				bitOffset,
			)
		}
	}

	for i := 0; i < g.size; i++ {
		idx := i
		if !littleEndian {
			idx = g.size - 1 - i
		}
		fmt.Fprintf(
			buffer,
			"bytes[%s+%d] = byte(b%d >> %d)\n",
			offset,
			idx,
			g.groupIndex,
			8*i,
		)
	}
}

func (g packedBitFieldGroup) writeFromBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offset string,
) {
	fmt.Fprintf(buffer, "var b%d uint64\n", g.groupIndex)

	for i := 0; i < g.size; i++ {
		idx := i
		if !littleEndian {
			idx = g.size - 1 - i
		}
		fmt.Fprintf(
			buffer,
			"b%d |= uint64(bytes[%s+%d]) << %d\n",
			g.groupIndex,
			offset,
			idx,
			8*i,
		)
	}

	offsets := g.computeOffsets(littleEndian)

	for _, fieldOffset := range offsets {
		field := fieldOffset.field
		bitOffset := fieldOffset.bitOffset
		receiver := receiverVariable + field.packedProperty.name

		mask := (uint64(1) << field.bitSize) - 1

		if field.bitFieldKind == bitFieldKindBoolean {
			fmt.Fprintf(
				buffer,
				"%s = ((b%d >> %d) & 0x%X) != 0\n",
				receiver,
				g.groupIndex,
				bitOffset,
				mask,
			)
			continue
		}

		switch field.bitFieldKind {

		case bitFieldKindBitsType:
			fmt.Fprintf(buffer, "%s.Set(", receiver)

		case bitFieldKindBitsConverter:
			fmt.Fprintf(buffer, "%s.Set(&%s, ", getConverterName(field.converter.hash), receiver)

		default:
			fmt.Fprintf(buffer, "%s = ", receiver)
		}

		if field.signed() {
			fmt.Fprintf(
				buffer,
				"%s((( (b%d >> %d) & 0x%X ) ^ (1 << %d)) - (1 << %d))",
				field.reflection,
				g.groupIndex,
				bitOffset,
				mask,
				field.bitSize-1,
				field.bitSize-1,
			)
		} else {
			fmt.Fprintf(
				buffer,
				"%s(uint64((b%d >> %d) & 0x%X))",
				field.reflection,
				g.groupIndex,
				bitOffset,
				mask,
			)
		}

		switch field.bitFieldKind {

		case bitFieldKindBitsType, bitFieldKindBitsConverter:
			fmt.Fprintf(buffer, ")\n")

		default:
			fmt.Fprintf(buffer, "\n")
		}
	}
}
