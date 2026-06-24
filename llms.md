# packed

`github.com/0-Mqix/packed` is a Go library for defining binary data structures with precise memory layout control and generating efficient serialization code. You define structures using a builder DSL, then run code generation to produce Go structs with `ToBytes`/`FromBytes` methods.

Bit-field runs reproduce C `struct __attribute__((__packed__))` layout byte-for-byte, in both little- and big-endian, which makes it a drop-in for decoding messages from microcontrollers (e.g. ARM).

## Install

```
go get github.com/0-Mqix/packed
```

## How It Works

1. Write a `generate.go` file (with `//go:build ignore`) that defines your structures using the DSL
2. Run it with `go run generate.go` to produce a generated `.go` file
3. The generated file contains Go structs with `Size()`, `ToBytes()`, and `FromBytes()` methods

## Core API

### `Struct(name string, littleEndian bool, properties ...packedProperty) packedStruct`

Defines a binary structure. `littleEndian` sets the default byte order for all fields.

### `Field(name string, propertyType any, options ...fieldOption) packedProperty`

Defines a field in a structure. `propertyType` can be:
- A built-in converter (`Uint8`, `Uint16`, `Uint32`, `Uint64`, `Int8`, `Int16`, `Int32`, `Int64`, `Float32`, `Float64`, `Boolean`)
- `String(length)` for fixed-length strings
- A `packedStruct` (nested struct)
- An `Array(length, elementType)`
- A `Bits[T](bitSize)` or `Bit` for bit-packed fields
- A `Cast[T](converter)` for type casting
- A custom type implementing `TypeInterface`
- A custom converter implementing `ConverterInterface[T]`

### Field Options

- `Tag(key, value string)` - Adds a Go struct tag (e.g. `Tag("json", "myField")`)
- `LittleEndian(value bool)` - Overrides the struct-level endianness for this field

### `Array(length int, elementType any) packedArray`

Fixed-size array. Supports nesting: `Array(2, Array(3, Uint8))`. Elements can be any supported field type except bare bit fields.

### `Bits[Integer](bitSize int, bitsTarget ...any) packedBitField`

Bit-packed integer field. `Integer` must be a Go integer type (`uint8`, `int16`, etc.). `bitSize` is the number of bits to use. Adjacent bit fields are packed together bit-by-bit and the whole run is rounded up to `ceil(totalBits/8)` bytes, matching C `__attribute__((__packed__))` layout. A run can be any length — there is no 64-bit limit.

Optional `bitsTarget` argument for custom bit packing types (see BitsTypeInterface / BitsConverterInterface below).

### `Bit`

Pre-defined single-bit boolean field. Shorthand for a 1-bit bool.

### `Cast[T any](converter any) converterCast`

Casts between a converter's receiver type and target type `T`. The converter's receiver must be convertible to `T`. Useful for enums.

### `String(length int) StringConverter`

Fixed-length string converter. Pads with null bytes on write, trims null bytes on read.

### `Load(structures ...packedStruct)`

Registers structures for code generation. Call this before `Generate` if you define structs separately from the `Generate` call. Note: `Struct()` already registers internally, so `Load` is only needed if you want explicit control.

### `Generate(outputFile string, packageName string, hooks ...GenerateHook)`

Generates the output Go file with all registered structures. Runs `goimports` and `gofmt` on the result.

## Built-in Converters

| Converter | Go Type  | Size (bytes) |
|-----------|----------|--------------|
| `Boolean` | `bool`   | 1            |
| `Int8`    | `int8`   | 1            |
| `Int16`   | `int16`  | 2            |
| `Int32`   | `int32`  | 4            |
| `Int64`   | `int64`  | 8            |
| `Uint8`   | `uint8`  | 1            |
| `Uint16`  | `uint16` | 2            |
| `Uint32`  | `uint32` | 4            |
| `Uint64`  | `uint64` | 8            |
| `Float32` | `float32`| 4            |
| `Float64` | `float64`| 8            |

## Generated Code

For each struct, the generator produces:

```go
type MyStruct struct {
    FieldA uint32 `json:"field_a"`
    FieldB int16
}

func (reciever *MyStruct) Size() int { return 6 }

func (reciever *MyStruct) ToBytes(bytes []byte, index int) {
    // serialization code
}

func (reciever *MyStruct) FromBytes(bytes []byte, index int) {
    // deserialization code
}
```

Usage of generated structs:

```go
instance := MyStruct{FieldA: 42, FieldB: -1}
buf := make([]byte, instance.Size())
instance.ToBytes(buf, 0)

var result MyStruct
result.FromBytes(buf, 0)
```

The `index` parameter allows packing multiple structs into the same byte slice at different offsets.

## Interfaces

### TypeInterface

For types that handle their own serialization. The generated struct field type will be the type itself.

```go
type TypeInterface interface {
    Size() int
    ToBytesLittleEndian(bytes []byte, index int)
    FromBytesLittleEndian(bytes []byte, index int)
    ToBytesBigEndian(bytes []byte, index int)
    FromBytesBigEndian(bytes []byte, index int)
}
```

### ConverterInterface[Receiver any]

For external converters that serialize a separate receiver type. The generated struct field type will be `Receiver`.

```go
type ConverterInterface[Receiver any] interface {
    Size() int
    ToBytesLittleEndian(receiver *Receiver, bytes []byte, index int)
    FromBytesLittleEndian(receiver *Receiver, bytes []byte, index int)
    ToBytesBigEndian(receiver *Receiver, bytes []byte, index int)
    FromBytesBigEndian(receiver *Receiver, bytes []byte, index int)
}
```

### BitsTypeInterface[Integer constraints.Integer]

For custom types that pack/unpack themselves from a bit field integer. The generated struct field type will be the implementing type.

```go
type BitsTypeInterface[Integer constraints.Integer] interface {
    Set(Integer)
    Integer() Integer
}
```

### BitsConverterInterface[Integer constraints.Integer, Receiver any]

For external converters that pack/unpack a separate receiver type from a bit field integer.

```go
type BitsConverterInterface[Integer constraints.Integer, Receiver any] interface {
    Set(*Receiver, Integer)
    Integer(*Receiver) Integer
}
```

### InitializeConverterFieldInterface

Allows converters with configurable fields (like `StringConverter` with `Length`) to specify initialization values in generated code.

```go
type InitializeConverterFieldInterface interface {
    InitializeConverterFields() map[string]string
}
```

Mark fields that affect converter identity with the `packed_hash_field` struct tag:

```go
type StringConverter struct {
    Length int `packed_hash_field:"length"`
}
```

### OverwriteConverterReflectionInterface / OverwriteConverterReciverReflectionInterface

Allow converters to override the reflected type used in generated code.

## Generate Hooks

Hooks run after each struct's core methods are generated, allowing you to emit additional methods.

```go
type GenerateHook func(buffer *bytes.Buffer, structName string, properties []Property)

type Property struct {
    Name string
    Type reflect.Type
}
```

## Complete Example

### generate.go

```go
//go:build ignore

package main

import (
    "bytes"
    "fmt"
    "reflect"

    . "github.com/0-Mqix/packed"
    "myproject/types"
)

func main() {
    // Basic struct with primitive fields and struct tags
    Struct("Header", true,
        Field("Version", Uint8, Tag("json", "version")),
        Field("Length", Uint32, Tag("json", "length")),
        Field("Name", String(16), Tag("json", "name")),
    )

    // Bit-packed flags (all fit in 1 byte = 8 bits)
    Struct("Flags", false,
        Field("Active", Bit),
        Field("Priority", Bits[uint8](3)),
        Field("Mode", Bits[uint8](4)),
    )

    // Nested struct
    Struct("Packet", true,
        Field("Header", Struct("PacketHeader", true,
            Field("ID", Uint16),
            Field("Type", Uint8),
        )),
        Field("Payload", Array(64, Uint8)),
    )

    // Enum via Cast
    Struct("Command", true,
        Field("Action", Cast[types.ActionEnum](Int32)),
    )

    // Nested arrays
    Struct("Matrix", true,
        Field("Data", Array(3, Array(3, Float64))),
    )

    Generate("output.go", "mypackage",
        // Optional hook: generate getters
        func(buffer *bytes.Buffer, name string, properties []Property) {
            for _, p := range properties {
                switch p.Type.Kind() {
                case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
                    reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                    fmt.Fprintf(buffer, "func (r *%s) Get%s() int { return int(r.%s) }\n", name, p.Name, p.Name)
                }
            }
        },
    )
}
```

### Custom TypeInterface

```go
type Color struct {
    R, G, B uint8
}

func (c *Color) Size() int { return 3 }

func (c *Color) ToBytesLittleEndian(bytes []byte, index int) {
    bytes[index] = c.R
    bytes[index+1] = c.G
    bytes[index+2] = c.B
}

func (c *Color) FromBytesLittleEndian(bytes []byte, index int) {
    c.R = bytes[index]
    c.G = bytes[index+1]
    c.B = bytes[index+2]
}

func (c *Color) ToBytesBigEndian(bytes []byte, index int) {
    c.ToBytesLittleEndian(bytes, index) // same for single-byte fields
}

func (c *Color) FromBytesBigEndian(bytes []byte, index int) {
    c.FromBytesLittleEndian(bytes, index)
}
```

Use it: `Field("Color", Color{})`

### Custom ConverterInterface

```go
type Vec2 struct{ X, Y float32 }

type Vec2Converter struct{}

func (Vec2Converter) Size() int { return 8 }

func (Vec2Converter) ToBytesLittleEndian(v *Vec2, bytes []byte, index int) {
    // serialize v.X and v.Y as little-endian float32s
}

func (Vec2Converter) FromBytesLittleEndian(v *Vec2, bytes []byte, index int) {
    // deserialize
}

func (Vec2Converter) ToBytesBigEndian(v *Vec2, bytes []byte, index int) {
    // serialize big-endian
}

func (Vec2Converter) FromBytesBigEndian(v *Vec2, bytes []byte, index int) {
    // deserialize big-endian
}
```

Use it: `Field("Position", Vec2Converter{})`

The generated struct field will be of type `Vec2`.

### Custom BitsTypeInterface

```go
type Permissions [4]bool // read, write, execute, admin

func (p *Permissions) Set(integer uint8) {
    for i := range 4 {
        p[i] = (integer & (1 << i)) != 0
    }
}

func (p *Permissions) Integer() uint8 {
    var v uint8
    for i := range 4 {
        if p[i] { v |= 1 << i }
    }
    return v
}
```

Use it: `Field("Perms", Bits[uint8](4, Permissions{}))`

The generated field type will be `Permissions`, packed into 4 bits.

### Custom BitsConverterInterface

```go
type BitArrayConverter struct{}

func (BitArrayConverter) Set(receiver *[8]bool, integer uint8) {
    for i := range 8 {
        receiver[i] = (integer & (1 << i)) != 0
    }
}

func (BitArrayConverter) Integer(receiver *[8]bool) uint8 {
    var v uint8
    for i := range 8 {
        if receiver[i] { v |= 1 << i }
    }
    return v
}
```

Use it: `Field("Flags", Bits[uint8](8, BitArrayConverter{}))`

The generated field type will be `[8]bool`, packed into 8 bits.

### Running Generation

```bash
go run generate.go
```

Or with `go:generate`:

```go
//go:generate go run generate.go
```

## Bit Field Details

- Adjacent `Bits` and `Bit` fields are automatically grouped into a single contiguous run
- A run can be any length; the whole run is rounded up to `ceil(totalBits/8)` bytes, exactly like C `__packed__` (there is no 64-bit limit)
- A field may straddle a 64-bit boundary inside a long run; this is handled transparently
- A non-bit field between bit fields ends the current run and starts a new one
- Signed bit fields use sign extension on deserialization
- Values that overflow the bit width are silently truncated (masked)
- Bit fields do not support per-field endianness override (panics if attempted); they inherit the struct's endianness
- Little-endian: bits are packed starting from bit 0 (LSB) of the first byte upward; trailing padding lands in the high bits of the last byte
- Big-endian: bits are packed starting from the MSB of the first byte downward; trailing padding lands in the low bits of the last byte

## C `__attribute__((__packed__))` Compatibility

Bit-field runs reproduce the layout a C compiler produces for a `struct` marked `__attribute__((__packed__))`, byte-for-byte, in both endiannesses:

- Fields are packed consecutively with a bit alignment of 1 (no padding between fields)
- The run occupies `ceil(totalBits/8)` bytes
- The layout depends only on each field's bit width and signedness, not on its declared integer type
- Little-endian fills from the least-significant bit; big-endian from the most-significant bit

This makes generated `FromBytes`/`ToBytes` a drop-in for decoding and encoding packed C structs as emitted by microcontrollers.

## Endianness

- Set at struct level via the `littleEndian` parameter of `Struct()`
- Override per-field with `LittleEndian(true)` or `LittleEndian(false)` option
- Nested structs inherit the parent field's endianness override
- Bit field groups use the struct-level endianness (cannot be overridden per-field)

## Key Constraints

- All struct names must be unique (panics on duplicate)
- All field names within a struct must be unique (panics on duplicate)
- Array elements cannot be bare bit fields (panics)
- `Bits` bit size cannot exceed the underlying integer type's size (panics) — matching C, where an over-wide bit field is a compile error
- All sizes are fixed at definition time; no variable-length fields
