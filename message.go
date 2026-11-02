package packed

import (
	"bytes"
	"fmt"
	"reflect"
)

var (
	messageSets        = map[string]packedMessageSet{}
	messageSetOfStruct = map[string]string{}
)

type packedVariant struct {
	typeByte  uint8
	structure packedStruct
}

type packedTagBits struct {
	bits int
}

type packedMessageSet struct {
	name     string
	variants []packedVariant
	// tagInBits is set by TagBits: the tag is the first tagBits bits of
	// the variant instead of a byte in front of it.
	tagInBits    bool
	tagBits      int
	littleEndian bool
}

// messageSetPart is what MessageSet takes after the name: variants, and
// optionally the tag's width.
type messageSetPart interface {
	messageSetPart()
}

func (packedVariant) messageSetPart() {}

func (packedTagBits) messageSetPart() {}

// Variant binds a struct to the type byte that precedes it on the wire, or
// to its tag when the set has TagBits.
func Variant(typeByte uint8, structure packedStruct) packedVariant {
	return packedVariant{typeByte: typeByte, structure: structure}
}

// TagBits makes a message set's tag bits wide instead of a byte: 0 to 8,
// enough for the set's highest tag, and 0 only for a set of one variant,
// which then has no tag at all. The tag goes in front of the variant's
// first field and joins the variant's first run of bit fields and Bools,
// so it costs no byte of its own while the run has room. A variant of such
// a set carries its tag: its Size, ToBytes and FromBytes include the tag's
// bits, so it cannot be a field of another struct. Below 8 bits the tag is
// at the low bits of the first byte (little endian) or at its high bits
// (big endian), so the variants of such a set share their endianness.
func TagBits(bits int) packedTagBits {
	if bits < 0 || bits > 8 {
		panic(fmt.Sprintf("TagBits(%d) is not between 0 and 8", bits))
	}
	return packedTagBits{bits: bits}
}

// MessageSet defines a tagged union: a message is one type byte followed by
// the packed bytes of the struct bound to it, or with TagBits the struct
// with its tag in its first bits. Generators emit an encoder and a decoder
// that checks the tag and the length.
func MessageSet(name string, parts ...messageSetPart) packedMessageSet {

	if _, ok := messageSets[name]; ok {
		panic(fmt.Sprintf("message set %s already exists", name))
	}

	set := packedMessageSet{name: name}

	for index, part := range parts {
		switch part := part.(type) {
		case packedTagBits:
			if index != 0 {
				panic(fmt.Sprintf("message set %s has TagBits after a variant", name))
			}
			set.tagInBits = true
			set.tagBits = part.bits
		case packedVariant:
			set.variants = append(set.variants, part)
		}
	}

	if len(set.variants) == 0 {
		panic(fmt.Sprintf("message set %s has no variants", name))
	}

	if set.tagInBits && set.tagBits == 0 && len(set.variants) > 1 {
		panic(fmt.Sprintf("message set %s has %d variants and no tag bits", name, len(set.variants)))
	}

	types := map[uint8]string{}
	structNames := map[string]bool{}

	for _, variant := range set.variants {
		structName := variant.structure.name

		if set.tagInBits && int(variant.typeByte) >= 1<<set.tagBits {
			panic(fmt.Sprintf("message set %s tags %s with %d, which does not fit in %d bits", name, structName, variant.typeByte, set.tagBits))
		}

		if other, ok := types[variant.typeByte]; ok {
			panic(fmt.Sprintf("message set %s uses type %d for both %s and %s", name, variant.typeByte, other, structName))
		}
		types[variant.typeByte] = structName

		if other, ok := messageSetOfStruct[structName]; ok {
			panic(fmt.Sprintf("struct %s is already a variant of message set %s", structName, other))
		}
		if structNames[structName] {
			panic(fmt.Sprintf("struct %s is already a variant of message set %s", structName, name))
		}
		structNames[structName] = true

		for _, field := range variant.structure.fieldNames() {
			if field == name+"Type" {
				panic(fmt.Sprintf("field %s.%s has the name of the %s method message set %s gives its variants: rename the field", structName, field, field, name))
			}
		}
	}

	if set.tagInBits && set.tagBits > 0 {
		set.littleEndian = set.variants[0].structure.littleEndian

		for index, variant := range set.variants {
			structure := structs[variant.structure.name]

			if set.tagBits < 8 && structure.littleEndian != set.littleEndian {
				panic(fmt.Sprintf("message set %s packs its tag in %d bits, so %s needs the endianness of %s", name, set.tagBits, structure.name, set.variants[0].structure.name))
			}

			for _, other := range structs {
				if other.name != structure.name && nestsStruct(other.properties, structure.name) {
					panic(fmt.Sprintf("struct %s has %s as a field, which carries the tag of message set %s", other.name, structure.name, name))
				}
			}

			structure = structure.withTag(set.tagBits, variant.typeByte)
			structs[structure.name] = structure
			set.variants[index].structure = structure
		}
	}

	for _, variant := range set.variants {
		messageSetOfStruct[variant.structure.name] = name
	}

	messageSets[name] = set

	return set
}

// withTag returns the struct with a tag of bits bits in front of its first
// field: in its first bit run when it starts with one, else in a run of its
// own.
func (p packedStruct) withTag(bits int, value uint8) packedStruct {
	tag := packedBitField{
		bitSize:      bits,
		reflection:   reflect.TypeFor[uint8](),
		bitFieldKind: bitFieldKindTag,
		tagValue:     value,
	}

	p.replacePropertiesWithClone()

	if len(p.properties) > 0 && p.properties[0].kind == kindBitFieldGroup {
		first := p.properties[0]
		fields := append([]packedBitField{tag}, first.packed.(packedBitFieldGroup).fields...)
		merged := createBitFieldGroup(fields, first.littleEndian)
		p.size += merged.size - first.size
		p.properties[0] = merged
	} else {
		group := createBitFieldGroup([]packedBitField{tag}, p.littleEndian)
		p.size += group.size
		p.properties = append([]packedProperty{group}, p.properties...)
	}

	bitFieldGroupIndex := 0
	p.setBitFieldGroupIndexes(&bitFieldGroupIndex)

	return p
}

// taggedVariant reports whether name is a variant of a message set that
// packs its tag into the variant.
func taggedVariant(name string) (string, bool) {
	setName, ok := messageSetOfStruct[name]
	if !ok {
		return "", false
	}
	return setName, messageSets[setName].tagInBits && messageSets[setName].tagBits > 0
}

// nestsStruct reports whether properties hold the struct name, directly, in
// a nested struct, or as the element of an array or slice.
func nestsStruct(properties []packedProperty, name string) bool {
	for _, property := range properties {
		if nestsStructValue(property.kind, property.packed, name) {
			return true
		}
	}
	return false
}

func nestsStructValue(kind kind, packed any, name string) bool {
	switch kind {
	case kindStruct:
		structure := packed.(packedStruct)
		return structure.name == name || nestsStruct(structure.properties, name)
	case kindArray:
		array := packed.(packedArray)
		return nestsStructValue(array.ElementKind, array.Element, name)
	case kindSlice:
		slice := packed.(packedSlice)
		return nestsStructValue(slice.ElementKind, slice.Element, name)
	}
	return false
}

// tagExpression is the Go expression for the tag of the message at the
// start of bytes; the caller has checked that bytes is not empty.
func (m packedMessageSet) tagExpression() string {
	switch {
	case !m.tagInBits || m.tagBits == 8:
		return "bytes[0]"
	case m.littleEndian:
		return fmt.Sprintf("bytes[0] & 0x%X", bitMask(m.tagBits))
	default:
		return fmt.Sprintf("bytes[0] >> %d", 8-m.tagBits)
	}
}

// start is the index the variant's struct starts at: after the type byte,
// or at 0 when the tag is in the struct.
func (m packedMessageSet) start() int {
	if m.tagInBits {
		return 0
	}
	return 1
}

func (m packedMessageSet) definition() []byte {
	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "type %s interface {\n", m.name)
	fmt.Fprintf(buffer, "%sType() uint8\n", m.name)
	fmt.Fprintf(buffer, "Size() int\n")
	fmt.Fprintf(buffer, "ToBytes(bytes []byte, index int) int\n")
	fmt.Fprintf(buffer, "FromBytes(bytes []byte, index int) int\n")
	fmt.Fprintf(buffer, "}\n\n")

	for _, variant := range m.variants {
		fmt.Fprintf(buffer, "func (reciever *%s) %sType() uint8 {\n", variant.structure.name, m.name)
		fmt.Fprintf(buffer, "return %d\n", variant.typeByte)
		fmt.Fprintf(buffer, "}\n\n")
	}

	start := m.start()

	fmt.Fprintf(buffer, "func Encode%s(message %s) []byte {\n", m.name, m.name)
	fmt.Fprintf(buffer, "bytes := make([]byte, %d+message.Size())\n", start)
	if !m.tagInBits {
		fmt.Fprintf(buffer, "bytes[0] = message.%sType()\n", m.name)
	}
	fmt.Fprintf(buffer, "message.ToBytes(bytes, %d)\n", start)
	fmt.Fprintf(buffer, "return bytes\n")
	fmt.Fprintf(buffer, "}\n\n")

	fmt.Fprintf(buffer, "// Append%s appends the encoded message to bytes.\n", m.name)
	fmt.Fprintf(buffer, "func Append%s(bytes []byte, message %s) []byte {\n", m.name, m.name)
	fmt.Fprintf(buffer, "index := len(bytes)\n")
	fmt.Fprintf(buffer, "bytes = append(bytes, make([]byte, %d+message.Size())...)\n", start)
	if !m.tagInBits {
		fmt.Fprintf(buffer, "bytes[index] = message.%sType()\n", m.name)
	}
	fmt.Fprintf(buffer, "message.ToBytes(bytes, index+%d)\n", start)
	fmt.Fprintf(buffer, "return bytes\n")
	fmt.Fprintf(buffer, "}\n\n")

	tag := "tag"
	if m.tagInBits && m.tagBits == 0 {
		tag = "0"
	}

	fmt.Fprintf(buffer, "func Decode%s(bytes []byte) (message %s, err error) {\n", m.name, m.name)
	if m.tagInBits && m.tagBits == 0 {
		fmt.Fprintf(buffer, "message = &%s{}\n", m.variants[0].structure.name)
	} else {
		fmt.Fprintf(buffer, "if len(bytes) == 0 {\n")
		fmt.Fprintf(buffer, "return nil, errors.New(\"packed: empty %s message\")\n", m.name)
		fmt.Fprintf(buffer, "}\n")
		fmt.Fprintf(buffer, "tag := %s\n", m.tagExpression())
		fmt.Fprintf(buffer, "switch tag {\n")
		for _, variant := range m.variants {
			fmt.Fprintf(buffer, "case %d:\n", variant.typeByte)
			fmt.Fprintf(buffer, "message = &%s{}\n", variant.structure.name)
		}
		fmt.Fprintf(buffer, "default:\n")
		fmt.Fprintf(buffer, "return nil, fmt.Errorf(\"packed: unknown %s type %%d\", tag)\n", m.name)
		fmt.Fprintf(buffer, "}\n")
	}
	fmt.Fprintf(buffer, "defer func() {\n")
	fmt.Fprintf(buffer, "if recover() != nil {\n")
	fmt.Fprintf(buffer, "message, err = nil, fmt.Errorf(\"packed: truncated %s type %%d\", %s)\n", m.name, tag)
	fmt.Fprintf(buffer, "}\n")
	fmt.Fprintf(buffer, "}()\n")
	fmt.Fprintf(buffer, "if end := message.FromBytes(bytes, %d); end != len(bytes) {\n", start)
	fmt.Fprintf(buffer, "return nil, fmt.Errorf(\"packed: %s type %%d has %%d trailing bytes\", %s, len(bytes)-end)\n", m.name, tag)
	fmt.Fprintf(buffer, "}\n")
	fmt.Fprintf(buffer, "return message, nil\n")
	fmt.Fprintf(buffer, "}\n")

	return buffer.Bytes()
}
