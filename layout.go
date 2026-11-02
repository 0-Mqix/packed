package packed

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// LayoutKind is what a Layout lays out.
type LayoutKind int

const (
	LayoutBool LayoutKind = iota
	LayoutInt8
	LayoutInt16
	LayoutInt32
	LayoutInt64
	LayoutInt
	LayoutUint8
	LayoutUint16
	LayoutUint32
	LayoutUint64
	LayoutUint
	LayoutFloat32
	LayoutFloat64
	LayoutString
	LayoutTime
	LayoutSlice
	LayoutArray
	LayoutMap
	LayoutPointer
	LayoutStruct
)

var layoutKindNames = [...]string{
	LayoutBool:    "bool",
	LayoutInt8:    "int8",
	LayoutInt16:   "int16",
	LayoutInt32:   "int32",
	LayoutInt64:   "int64",
	LayoutInt:     "int",
	LayoutUint8:   "uint8",
	LayoutUint16:  "uint16",
	LayoutUint32:  "uint32",
	LayoutUint64:  "uint64",
	LayoutUint:    "uint",
	LayoutFloat32: "float32",
	LayoutFloat64: "float64",
	LayoutString:  "string",
	LayoutTime:    "time",
	LayoutSlice:   "slice",
	LayoutArray:   "array",
	LayoutMap:     "map",
	LayoutPointer: "pointer",
	LayoutStruct:  "struct",
}

func (kind LayoutKind) String() string { return layoutKindNames[kind] }

// A Layout is a Go type as packed's dynamic layer lays it out. The dynamic
// generator (package generate) builds it from go/types; DynamicSource turns
// it into packed's own model, the one the DSL builds, and writes its code
// with packed's emitter.
type Layout struct {
	Kind LayoutKind
	// Type is the type as the generated file writes it: "Device",
	// "other.Item", "[]string", "time.Time".
	Type string
	// Name is a named type's name, "" for an unnamed type and time.Time.
	Name string
	// Fields are a struct's encoded fields, in declaration order.
	Fields []LayoutField
	// Element is a slice's, an array's, a map's or a pointer's element.
	Element *Layout
	// Key is a map's key.
	Key *Layout
	// Length is an array's length.
	Length int
	// Function names a struct's helpers, appendPacked<Function> and
	// decodePacked<Function>.
	Function string
}

// LayoutField is a struct's field.
type LayoutField struct {
	Name   string
	Layout *Layout
}

// Import is a package the generated file imports, by the name it uses, ""
// when that is the package's own.
type Import struct {
	Path string
	Name string
}

// fixed reports whether the layout is the same number of bytes for every
// value, which packed's converters and arrays write.
func (l *Layout) fixed() bool {
	switch l.Kind {
	case LayoutString, LayoutSlice, LayoutMap, LayoutPointer, LayoutStruct:
		return false
	case LayoutArray:
		return l.Element.fixed()
	}
	return true
}

// converter is the converter of a fixed layout that is no array, and the
// type it converts.
func (l *Layout) converter() (any, reflect.Type) {
	switch l.Kind {
	case LayoutBool:
		return &BooleanConverter{}, reflect.TypeFor[bool]()
	case LayoutInt8:
		return &Int8Converter{}, reflect.TypeFor[int8]()
	case LayoutInt16:
		return &Int16Converter{}, reflect.TypeFor[int16]()
	case LayoutInt32:
		return &Int32Converter{}, reflect.TypeFor[int32]()
	case LayoutInt64, LayoutInt:
		return &Int64Converter{}, reflect.TypeFor[int64]()
	case LayoutUint8:
		return &Uint8Converter{}, reflect.TypeFor[uint8]()
	case LayoutUint16:
		return &Uint16Converter{}, reflect.TypeFor[uint16]()
	case LayoutUint32:
		return &Uint32Converter{}, reflect.TypeFor[uint32]()
	case LayoutUint64, LayoutUint:
		return &Uint64Converter{}, reflect.TypeFor[uint64]()
	case LayoutFloat32:
		return &Float32Converter{}, reflect.TypeFor[float32]()
	case LayoutFloat64:
		return &Float64Converter{}, reflect.TypeFor[float64]()
	case LayoutTime:
		return &TimeConverter{}, reflect.TypeFor[time.Time]()
	}
	panic(fmt.Sprintf("packed: a %s layout has no converter", l.Kind))
}

// MinimumSize is the fewest bytes a value of the layout takes: a decoder
// refuses a count of elements that can't fit in the bytes left.
func (l *Layout) MinimumSize() int {
	return l.minimumSize(map[*Layout]int{})
}

func (l *Layout) minimumSize(structs map[*Layout]int) int {
	switch l.Kind {
	case LayoutBool, LayoutInt8, LayoutUint8:
		return 1
	case LayoutInt16, LayoutUint16:
		return 2
	case LayoutInt32, LayoutUint32, LayoutFloat32:
		return 4
	case LayoutString, LayoutSlice, LayoutMap:
		return 4
	case LayoutPointer:
		return 1
	case LayoutArray:
		return l.Length * l.Element.minimumSize(structs)
	case LayoutStruct:
		if size, ok := structs[l]; ok {
			return size
		}
		// as parts lays it out: bools and pointers share bit runs, and a
		// pointer's value may be absent
		size, bits := 0, 0
		for _, field := range l.Fields {
			if field.Layout.Kind == LayoutBool || field.Layout.Kind == LayoutPointer {
				bits++
				continue
			}
			size += (bits + 7) / 8
			bits = 0
			size += field.Layout.minimumSize(structs)
		}
		size += (bits + 7) / 8
		structs[l] = size
		return size
	}
	return 8
}

// schemaVersion starts every description, so a change to the layer's own
// rules changes every schema.
const schemaVersion = "packed dynamic 1"

// Description is the canonical description of the layout the schema hash
// is taken of: every struct's name and its fields' names and types in
// order, a struct described where it is first reached and referred to by
// its number after that.
func (l *Layout) Description() string {
	builder := &strings.Builder{}
	builder.WriteString(schemaVersion + "\n")
	l.describe(builder, map[*Layout]int{})
	return builder.String()
}

func (l *Layout) describe(builder *strings.Builder, structs map[*Layout]int) {
	if l.Name != "" && l.Kind != LayoutStruct {
		builder.WriteString(l.Name + "(")
		defer builder.WriteString(")")
	}

	switch l.Kind {
	case LayoutSlice:
		builder.WriteString("[]")
		l.Element.describe(builder, structs)
	case LayoutArray:
		fmt.Fprintf(builder, "[%d]", l.Length)
		l.Element.describe(builder, structs)
	case LayoutMap:
		builder.WriteString("map[")
		l.Key.describe(builder, structs)
		builder.WriteString("]")
		l.Element.describe(builder, structs)
	case LayoutPointer:
		builder.WriteString("*")
		l.Element.describe(builder, structs)
	case LayoutStruct:
		if number, ok := structs[l]; ok {
			fmt.Fprintf(builder, "#%d", number)
			return
		}
		number := len(structs)
		structs[l] = number
		fmt.Fprintf(builder, "#%d %s{", number, l.Name)
		for index, field := range l.Fields {
			if index > 0 {
				builder.WriteString("; ")
			}
			builder.WriteString(field.Name + " ")
			field.Layout.describe(builder, structs)
		}
		builder.WriteString("}")
	default:
		builder.WriteString(l.Kind.String())
	}
}

// Schema is the 64-bit hash of the layout's Description that starts each
// of its messages: the first 8 bytes of its SHA-256, little endian.
func (l *Layout) Schema() uint64 {
	sum := sha256.Sum256([]byte(l.Description()))
	return binary.LittleEndian.Uint64(sum[:8])
}
