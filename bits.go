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
	if size < 1 {
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

func bitMask(bits int) uint64 { return (uint64(1) << uint(bits)) - 1 }

func wordCount(size int) int { return (size + 7) / 8 }

func wordBytes(size, word int) int {
	if remaining := size - 8*word; remaining < 8 {
		return remaining
	}
	return 8
}

type bitDeposit struct {
	word             int
	sourceShift      int
	bitCount         int
	destinationShift int
}

type bitPlacement struct {
	field    packedBitField
	index    int
	straddle bool
	deposits []bitDeposit
}

// placements assigns each field to the uint64 word(s) backing it. The run is
// ceil(size/8) words; word w holds output bytes [8w, 8w+8). Little-endian packs
// from the least-significant bit, big-endian from the most-significant (C
// __packed__ bit order); a field crossing a word boundary becomes two deposits.
func (g packedBitFieldGroup) placements(littleEndian bool) []bitPlacement {
	result := make([]bitPlacement, 0, len(g.fields))

	bitOffset := 0
	for index, field := range g.fields {
		bitSize := field.bitSize
		firstWord := bitOffset / 64
		bitInWord := bitOffset - 64*firstWord

		placement := bitPlacement{field: field, index: index}

		if (bitOffset+bitSize-1)/64 == firstWord {
			var destinationShift int
			if littleEndian {
				destinationShift = bitInWord
			} else {
				destinationShift = wordBytes(g.size, firstWord)*8 - bitInWord - bitSize
			}
			placement.deposits = []bitDeposit{{word: firstWord, bitCount: bitSize, destinationShift: destinationShift}}
		} else {
			placement.straddle = true
			firstBits := 64*(firstWord+1) - bitOffset
			secondBits := bitSize - firstBits

			if littleEndian {
				placement.deposits = []bitDeposit{
					{word: firstWord, bitCount: firstBits, destinationShift: bitInWord},
					{word: firstWord + 1, sourceShift: firstBits, bitCount: secondBits},
				}
			} else {
				upperWordBytes := wordBytes(g.size, firstWord+1)
				placement.deposits = []bitDeposit{
					{word: firstWord, sourceShift: secondBits, bitCount: firstBits},
					{word: firstWord + 1, bitCount: secondBits, destinationShift: upperWordBytes*8 - secondBits},
				}
			}
		}

		result = append(result, placement)
		bitOffset += bitSize
	}

	return result
}

func wordByteBase(offsetBase string, offsetConst, word int, advance bool) string {
	if advance {
		return offsetBase
	}
	return fmt.Sprintf("%s + %d", offsetBase, offsetConst+8*word)
}

// writeWordStore spills the low byteCount bytes of word into bytes at base in the
// given byte order. The Go compiler folds a run of consecutive byte loads into a
// single wide load but does not do the same for byte stores, so the store side is
// emitted as binary.*Endian.PutUintNN over the widest power-of-two chunks (one
// MOVQ/MOVL/MOVW each) while the load side in writeFromBytes stays the byte loop
// the compiler already combines.
func writeWordStore(buffer *bytes.Buffer, base, word string, byteCount int, littleEndian bool) {
	order := "binary.LittleEndian"
	if !littleEndian {
		order = "binary.BigEndian"
	}

	for sourceByte := 0; sourceByte < byteCount; {
		chunk := 1
		for chunk*2 <= byteCount-sourceByte {
			chunk *= 2
		}

		value := word
		if sourceByte != 0 {
			value = fmt.Sprintf("%s >> %d", word, 8*sourceByte)
		}

		position := sourceByte
		if !littleEndian {
			position = byteCount - sourceByte - chunk
		}

		if chunk == 1 {
			fmt.Fprintf(buffer, "bytes[%s+%d] = byte(%s)\n", base, position, value)
		} else {
			bits := chunk * 8
			fmt.Fprintf(buffer, "%s.PutUint%d(bytes[%s+%d:], uint%d(%s))\n", order, bits, base, position, bits, value)
		}

		sourceByte += chunk
	}
}

func (field packedBitField) toBytesReceiver(receiverVariable string) string {
	receiver := receiverVariable + field.packedProperty.name

	switch field.bitFieldKind {
	case bitFieldKindBitsType:
		return receiver + ".Integer()"
	case bitFieldKindBitsConverter:
		return fmt.Sprintf("%s.Integer(&%s)", getConverterName(field.converter.hash), receiver)
	}

	return receiver
}

func (g packedBitFieldGroup) writeToBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offsetBase string,
	offsetConst int,
	advance bool,
) {
	placements := g.placements(littleEndian)

	for word := 0; word < wordCount(g.size); word++ {
		wordVariable := g.groupIndex + word
		fmt.Fprintf(buffer, "var b%d uint64\n", wordVariable)

		for _, placement := range placements {
			field := placement.field

			for depositIndex, deposit := range placement.deposits {
				if deposit.word != word {
					continue
				}

				if !placement.straddle {
					if field.bitFieldKind == bitFieldKindBoolean {
						fmt.Fprintf(
							buffer,
							"b%d |= (uint64(*(*uint8)(unsafe.Pointer(&%s))) & 1) << %d\n",
							wordVariable,
							receiverVariable+field.packedProperty.name,
							deposit.destinationShift,
						)
						continue
					}

					receiver := field.toBytesReceiver(receiverVariable)
					mask := bitMask(deposit.bitCount)
					if deposit.destinationShift == 0 {
						fmt.Fprintf(buffer, "b%d |= (uint64(%s) & 0x%X)\n", wordVariable, receiver, mask)
					} else {
						fmt.Fprintf(buffer, "b%d |= (uint64(%s) & 0x%X) << %d\n", wordVariable, receiver, mask, deposit.destinationShift)
					}
					continue
				}

				temporary := fmt.Sprintf("s%d_%d", g.groupIndex, placement.index)
				if depositIndex == 0 {
					fmt.Fprintf(buffer, "%s := uint64(%s) & 0x%X\n", temporary, field.toBytesReceiver(receiverVariable), bitMask(field.bitSize))
				}

				source := temporary
				if deposit.sourceShift != 0 {
					source = fmt.Sprintf("(%s >> %d)", temporary, deposit.sourceShift)
				}
				expression := fmt.Sprintf("(%s & 0x%X)", source, bitMask(deposit.bitCount))
				if deposit.destinationShift != 0 {
					expression = fmt.Sprintf("%s << %d", expression, deposit.destinationShift)
				}
				fmt.Fprintf(buffer, "b%d |= %s\n", g.groupIndex+deposit.word, expression)
			}
		}

		byteCount := wordBytes(g.size, word)
		base := wordByteBase(offsetBase, offsetConst, word, advance)
		writeWordStore(buffer, base, fmt.Sprintf("b%d", wordVariable), byteCount, littleEndian)

		if advance {
			fmt.Fprintf(buffer, "%s += %d\n", offsetBase, byteCount)
		}
	}
}

func (g packedBitFieldGroup) writeFromBytes(
	buffer *bytes.Buffer,
	receiverVariable string,
	littleEndian bool,
	offsetBase string,
	offsetConst int,
	advance bool,
) {
	placements := g.placements(littleEndian)

	for word := 0; word < wordCount(g.size); word++ {
		wordVariable := g.groupIndex + word
		fmt.Fprintf(buffer, "var b%d uint64\n", wordVariable)

		byteCount := wordBytes(g.size, word)
		base := wordByteBase(offsetBase, offsetConst, word, advance)
		for byteIndex := 0; byteIndex < byteCount; byteIndex++ {
			localByte := byteIndex
			if !littleEndian {
				localByte = byteCount - 1 - byteIndex
			}
			fmt.Fprintf(buffer, "b%d |= uint64(bytes[%s+%d]) << %d\n", wordVariable, base, localByte, 8*byteIndex)
		}

		// extract every field whose last (highest-addressed) word is this one,
		// so that a straddling field is reconstructed only after both words exist.
		for _, placement := range placements {
			if placement.deposits[len(placement.deposits)-1].word != word {
				continue
			}

			field := placement.field
			receiver := receiverVariable + field.packedProperty.name
			rawValue := g.rawValue(placement)

			if placement.straddle {
				temporary := fmt.Sprintf("s%d_%d", g.groupIndex, placement.index)
				fmt.Fprintf(buffer, "%s := %s\n", temporary, rawValue)
				rawValue = temporary
			}

			emitFieldExtract(buffer, field, receiver, rawValue)
		}

		if advance {
			fmt.Fprintf(buffer, "%s += %d\n", offsetBase, byteCount)
		}
	}
}

// rawValue rebuilds the unsigned field value from its word deposits. For a
// single-word field this is the historical "(bX >> shift) & mask" expression.
func (g packedBitFieldGroup) rawValue(placement bitPlacement) string {
	term := func(deposit bitDeposit) string {
		text := fmt.Sprintf("(b%d >> %d) & 0x%X", g.groupIndex+deposit.word, deposit.destinationShift, bitMask(deposit.bitCount))
		if deposit.sourceShift != 0 {
			text = fmt.Sprintf("(%s) << %d", text, deposit.sourceShift)
		}
		return text
	}

	if len(placement.deposits) == 1 {
		return term(placement.deposits[0])
	}

	return fmt.Sprintf("(%s) | (%s)", term(placement.deposits[0]), term(placement.deposits[1]))
}

func convertRawValue(field packedBitField, rawValue string) string {
	if field.bitFieldKind == bitFieldKindBoolean {
		return fmt.Sprintf("(%s) != 0", rawValue)
	}

	if field.signed() {
		return fmt.Sprintf("%s((( %s ) ^ (1 << %d)) - (1 << %d))", field.reflection, rawValue, field.bitSize-1, field.bitSize-1)
	}

	return fmt.Sprintf("%s(uint64(%s))", field.reflection, rawValue)
}

func emitFieldExtract(buffer *bytes.Buffer, field packedBitField, receiver, rawValue string) {
	value := convertRawValue(field, rawValue)

	switch field.bitFieldKind {
	case bitFieldKindBitsType:
		fmt.Fprintf(buffer, "%s.Set(%s)\n", receiver, value)
	case bitFieldKindBitsConverter:
		fmt.Fprintf(buffer, "%s.Set(&%s, %s)\n", getConverterName(field.converter.hash), receiver, value)
	default:
		fmt.Fprintf(buffer, "%s = %s\n", receiver, value)
	}
}
