# packed

**Define your binary layout once. Get fast, zero-reflection code that matches it byte-for-byte, in Go, C and C++, Kotlin, Swift and TypeScript.**

`packed` is a code generator for binary data structures. You describe your wire format with a small builder DSL — down to individual bits — and `packed` generates plain Go structs with hand-tight `ToBytes` / `FromBytes` methods. No reflection, no interface dispatch at runtime: every offset of a fixed layout is precomputed at generation time, and bit fields compile down to raw shifts and masks. The same definitions generate a header for firmware that C and C++ both include, Kotlin for Android, Swift for iOS and TypeScript for the web, all writing the same bytes.

Where `packed` really shines is talking to microcontrollers. Its bit-field layout reproduces C `struct __attribute__((__packed__))` byte-for-byte, in both little- and big-endian. If your firmware sends this:

```c
struct __attribute__((__packed__)) status {
    uint8_t online   : 1;
    uint8_t charging : 1;
    uint8_t error    : 3;
    uint8_t signal   : 3;
};
```

then this definition on the Go side decodes it — same bits, same bytes, no manual masking — and generates that very struct for the firmware:

```go
Struct("Status", true,
    Field("Online", Bit),
    Field("Charging", Bit),
    Field("ErrorCode", Bits[uint8](3)),
    Field("SignalStrength", Bits[uint8](3)),
)
```

## Install

```
go get github.com/0-Mqix/packed
```

## From Definition to Working Code

Say your IoT devices report telemetry: a device ID, two sensor readings, and that packed status byte. Write a `generate.go` (kept out of normal builds with `//go:build ignore`) that describes the packet and lists the files you want:

```go
//go:build ignore

package main

import (
    . "github.com/0-Mqix/packed"
    "github.com/0-Mqix/packed/generate"
)

func main() {
    Struct("Telemetry", true, // true = little-endian
        Field("DeviceID", Uint16, Tag("json", "device_id")),
        Field("Temperature", Float32, Tag("json", "temperature")),
        Field("Humidity", Float32, Tag("json", "humidity")),
        Field("Status", Struct("Status", true,
            Field("Online", Bit),
            Field("Charging", Bit),
            Field("ErrorCode", Bits[uint8](3)),
            Field("SignalStrength", Bits[uint8](3)),
        )),
    )

    generate.Files(
        generate.GoFile("telemetry.go", "telemetry"),
        generate.HeaderFile("../firmware/telemetry.h"),
    )
}
```

Run `go run generate.go` (or hook it up with `//go:generate`) and you get real, readable Go. This is the actual output for `Status`:

```go
type Status struct {
    Online         bool
    Charging       bool
    ErrorCode      uint8
    SignalStrength uint8
}

func (reciever *Status) Size() int {
    return 1
}

func (reciever *Status) ToBytes(bytes []byte, index int) int {
    var b0 uint64
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Online))) & 1) << 0
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Charging))) & 1) << 1
    b0 |= (uint64(reciever.ErrorCode) & 0x7) << 2
    b0 |= (uint64(reciever.SignalStrength) & 0x7) << 5
    bytes[index+0+0] = byte(b0)
    return index + 1
}

func (reciever *Status) FromBytes(bytes []byte, index int) int {
    var b0 uint64
    b0 |= uint64(bytes[index+0+0]) << 0
    reciever.Online = ((b0 >> 0) & 0x1) != 0
    reciever.Charging = ((b0 >> 1) & 0x1) != 0
    reciever.ErrorCode = uint8(uint64((b0 >> 2) & 0x7))
    reciever.SignalStrength = uint8(uint64((b0 >> 5) & 0x7))
    return index + 1
}
```

`Telemetry` gets the same, with `Status` inlined into its methods: the whole 11-byte packet encodes and decodes without a reflection call, an allocation or a loop. `ToBytes` and `FromBytes` return the index just past the struct, so packets pack back to back:

```go
var t Telemetry
t.FromBytes(payload, 0)

fmt.Printf("device %d: %.1f°C, signal %d/7\n", t.DeviceID, t.Temperature, t.Status.SignalStrength)

buffer := make([]byte, len(readings)*readings[0].Size())
index := 0
for i := range readings {
    index = readings[i].ToBytes(buffer, index)
}
```

## Choosing the Outputs

`generate.Files` writes the outputs it is given, creating their folders, so a project lists only the languages it uses:

| Output | What it writes |
| --- | --- |
| `generate.GoFile(file, package, hooks...)` | Go, gofmt'ed, imports resolved |
| `generate.HeaderFile(file)` | one header for C11 and C++20, for firmware and hosts |
| `generate.KotlinFile(file, package)` | Kotlin |
| `generate.SwiftFile(file, imports...)` | Swift; `generate.SwiftErrorFile(file)` writes `PackedError` once per module |
| `generate.TypeScriptFile(file, runtime)` | a TypeScript module importing [`typescript/packed.ts`](typescript/packed.ts) from `runtime` |

`generate.Generate`, `generate.Header`, `generate.Kotlin`, `generate.Swift`, `generate.SwiftError` and `generate.TypeScript` write one each.

## Beyond the Basics

Real wire formats are rarely just a few integers, so the DSL covers the rest of what firmware tends to throw at you.

**Fixed-size arrays** — including nested ones — map to Go arrays. A 64-byte payload buffer or a 3×3 calibration matrix is one field each:

```go
Field("Payload", Array(64, Uint8))
Field("Calibration", Array(3, Array(3, Float64)))
```

**Fixed-length strings** are null-padded on write and null-trimmed on read, so a 16-byte device-name field just becomes a Go `string`:

```go
Field("Name", String(16))
```

**Enums** are named types with a constant per value, numbered from zero and stored with an integer converter. `Severity.Bits(2)` stores one in a bit field instead:

```go
Severity := Enum("Severity", Uint8, "Low", "Medium", "High")

Field("Severity", Severity)         // a byte; Go gets type Severity uint8, SeverityLow…, String()
Field("Severity", Severity.Bits(2)) // two bits of a bit run
```

Go does not check a decoded value; C, Kotlin, Swift and TypeScript report one that has no value. `Cast` is the other way to give a field your own type: it stores the converter's wire type and exposes your type:

```go
Field("Action", Cast[types.ActionEnum](Int32)) // 4 bytes on the wire, ActionEnum in Go
```

**Endianness** is set per struct and can be overridden per field — useful for those protocols where one vendor's field is inexplicably big-endian in an otherwise little-endian packet:

```go
Struct("Mixed", true,
    Field("Counter", Uint32),
    Field("Checksum", Uint32, LittleEndian(false)), // this one field is big-endian
)
```

**Bit runs of any length.** Adjacent `Bit`/`Bits` fields are packed into a single contiguous run occupying `ceil(totalBits/8)` bytes — there is no 64-bit limit, and a field may straddle a 64-bit word boundary mid-run without you ever noticing. Signed bit fields are sign-extended on read, exactly as C does it. The layout rules match C `__attribute__((__packed__))` precisely: fields pack consecutively with bit alignment 1, little-endian fills from the least-significant bit up, big-endian from the most-significant bit down, and the layout depends only on each field's width and signedness — not its declared integer type.

**The wire types you'd expect**: `Uint8`–`Uint64`, `Int8`–`Int64`, `Float32`, `Float64`, and `Boolean`, each mapping to its natural Go type at its natural size.

## Variable-Length Fields

`VariableString(Uint16)` is a byte length, then UTF-8; `Slice(Uint8, Label)` an element count, then the elements (anything an `Array` takes, slices, strings and enums too); `VariableBytes(Uint32)` the same layout as `Slice(Uint32, Uint8)` but `[]byte`, `ByteArray` and `[UInt8]`. The prefix follows the field's endianness. A struct holding one anywhere is dynamic: `Size()` computes from the value. `ToBytes` panics when a value is longer than its prefix can say.

```go
Struct("Record", true,
    Field("Name", VariableString(Uint16)),
    Field("Labels", Slice(Uint8, Label).Maximum(8)),
    Field("Payload", VariableBytes(Uint32)),
)
```

`Slice(...).Maximum(n)` caps a list at `n` items. Every language checks it when it encodes and decodes (Go panics, as for a too-long prefix), and the C header holds such a list in an array of `n`. The bytes on the wire are the same: a list of one item is still its count and that item.

## Message Sets

`MessageSet` is a tagged union: a type byte, then the struct bound to it.

```go
MessageSet("Display",
    Variant(1, SetFrame),
    Variant(2, Read),
)
```

Go gets an interface with `EncodeDisplay`, `AppendDisplay(bytes, message)` and `DecodeDisplay(bytes)`, which reports an unknown type, truncated input and trailing bytes. Kotlin gets a `sealed interface` with `encode(prefix)` and `decode(bytes, index)`, Swift an `enum` with `encode(prefix:)` and `decode(_:from:)`, TypeScript a union of `{ variant, message }`, C a struct of its type and a union of the variants.

**Tag bits.** `TagBits(n)`, 0 to 8, makes the tag `n` bits wide:

```go
MessageSet("Command", TagBits(3),
    Variant(0, Toggle),
    Variant(5, Rename),
)
```

The tag goes in front of the variant's first field and joins its first run of bit fields and Bools, so it costs no byte while the run has room; a variant that starts with anything else gets a run of its own. `TagBits(0)` is a set of one variant with no tag. A variant of a set with tag bits cannot be a field of another struct, and all variants share one endianness.

## Other Languages

Custom `TypeInterface`, converter and bit-field types have no equivalent outside Go and make the other generators panic with the field's name.

**C and C++** (`generate.HeaderFile`): one header that C11 and C++20 both include, for microcontrollers and hosts alike — an ATtiny88 builds it with avr-gcc and avr-g++ — so no heap and nothing beyond `<stdint.h>`, `<stddef.h>`, `<stdbool.h>` and `<string.h>`. The layout and the code are C; C++ gets methods, enum classes and `[]` on top of the same structs.

- A fixed struct is a `struct __attribute__((packed))` with C bit fields in the same layout, and a static assertion checks its `sizeof` against Go's `Size()`. Field names are lower camel case (`DeviceID` is `deviceId`), with an underscore after a C or C++ keyword (`signed_`). A variant of a set with `TagBits` holds its tag in `tag`, which encoding writes whatever it holds.
- Every struct `Status` gets `StatusSize(&status)`, `StatusToBytes(&status, bytes, capacity, index, &end)` and `StatusFromBytes(&status, bytes, length, index, &end)`; a message set `Display` gets `DisplaySize`, `DisplayEncode(&display, bytes, capacity, &end)` and `DisplayDecode(&display, bytes, length)`. They return a `PackedError`: `PackedErrorTruncated`, `Full`, `TooLong`, `UnknownValue`, and from a set's `Empty`, `UnknownType` and `TrailingBytes`.
- An enum is its wire type with a constant per value, as in Go (`LevelWarning`); in C++ an `enum class` (`Level::Warning`), which has the constants too. A message set is a struct of its type (`DisplayTypeRead`, `Display::Type::Read`) and a union of the variants.
- Variable-length fields are plain C. Text, bytes, and a list whose elements are in the input as they are in memory (bytes, and fixed structs that are little endian throughout, like a `KeyEvent`) are a pointer and a count: `const char* name; size_t nameLength;`, `const KeyEvent* keys; size_t keysCount;`. Encoding reads them from wherever you point them; decoding points them into the input, without a copy, so the input must outlive them. Any other list, enums included (C++ may not read input bytes as an enum class), needs `Slice(...).Maximum(n)` and is an array of `n` and a count, `Label labels[8]; uint8_t labelsCount;`, which decoding copies into; without a maximum, `HeaderFile` fails with the field's path. Text and bytes inside an array or a list are a `PackedText` (`text`, `length`) or `PackedBytes` (`bytes`, `length`). Pointing a list at structs needs a little-endian machine, which every common one is; the header asserts it.
- A fixed struct that is little endian throughout has `SetFrameView(bytes, length, index)`: on a little-endian machine, the struct in the input itself, without a copy (a 192-byte frame on a chip with 512 bytes of RAM).
- In C++ the same structs have `size()`, `toBytes(bytes, capacity, index, end)`, `fromBytes(...)` and `view(...)`, and sets `encode(...)` and `Display::decode(bytes, length, display)`. Fixed arrays work with range-for, and a pointer and count with `std::span(descriptor.keys, descriptor.keysCount)`; in C++20 a `PackedText` converts to `std::string_view` and a `PackedBytes` to `std::span<const uint8_t>`.
- A `Float64` needs a 64-bit `double`: with avr-gcc, build with `-mdouble=64`. GCC on x86 notes that packed bit fields moved in GCC 4.4; `-Wno-packed-bitfield-compat` silences it.

```c
#include "messages.h"

Display display;
if (DisplayDecode(&display, received, length) == PackedErrorNone && display.type == DisplayTypeRead) {
    static const uint8_t log[] = {'o', 'k'};
    Status status = {0};
    status.keys = 0x15;
    status.log = log;
    status.logCount = sizeof log;
    size_t end = 0;
    if (StatusToBytes(&status, sent, sizeof sent, 0, &end) == PackedErrorNone) {
        transmit(sent, end);
    }
}
```

The same in C++:

```cpp
Display display;
if (Display::decode(received, length, display) == PackedErrorNone && display.type == Display::Type::Read) {
    static const uint8_t log[] = {'o', 'k'};
    Status status{};
    status.keys = 0x15;
    status.log = log;
    status.logCount = sizeof log;
    size_t end = 0;
    if (status.toBytes(sent, sizeof sent, 0, end) == PackedErrorNone) {
        transmit(sent, end);
    }
}

Descriptor descriptor{};
descriptor.fromBytes(bytes, length, 0, end);
for (const KeyEvent& key : std::span(descriptor.keys, descriptor.keysCount)) { /* ... */ }
```

**Kotlin** (`generate.KotlinFile`): data classes with `size()`, `toBytes` and `fromBytes`, `enum class`es with `fromValue`, sealed interfaces for sets, unsigned types as `UByte` to `ULong`, lists for arrays and slices. Decoding errors throw `IllegalArgumentException`.

**Swift** (`generate.SwiftFile`): standard library only, Swift 6 mode. `Equatable, Sendable` structs with `toBytes(_:_:)` and `fromBytes(_:_:)` returning the value and its end, `CaseIterable` enums with `init(validating:)`, enums for sets. Every read is bounds-checked and throws `PackedError`; encoding errors trap. `generate.SwiftErrorFile` writes `PackedError` once, in one module; other files import it.

**TypeScript** (`generate.TypeScriptFile`): an ES module in erasable syntax only, so Node and Vite run it as it is, importing its reader and writer from `runtime`, [`typescript/packed.ts`](typescript/packed.ts) as the file is to import it (copy it into your project). An enum is a frozen array of its value names and the type of those names, and travels as the name; a struct is an interface of its fields by camelName and an object of the same name with `write(writer, value)` and `read(reader)`; a set is the union of its variants, `{ variant, message }`, and an object with `encode({ variant, message }, prefix)` and `decode(bytes, index)`. Integers of 64 bits are BigInts, the others numbers; bytes are a `Uint8Array`. Reads throw `RangeError` past the end; `decode` throws `Error` with Go's messages.

## The Dynamic Layer

`packed/dynamic` generates encoders and decoders from ordinary Go types, with no DSL file: a struct that embeds `packed.Dynamic` travels, with everything reachable from it.

```go
//go:generate go run github.com/0-Mqix/packed/dynamic

type Device struct {
    packed.Dynamic

    ID     int
    Name   string
    Online bool
    Owner  *User
    Tags   []string
    Scores map[string]int64
    Seen   time.Time
    cache  string             // unexported: not encoded
    Draft  string `packed:"-"` // skipped
}
```

`go generate` writes `packed_dynamic.go` beside it:

- `const DeviceSchema uint64`, the hash every message starts with;
- `(value Device) AppendPacked(bytes []byte) ([]byte, error)`: a value the layer can't hold (a `time.Time` Unix nanoseconds can't, a string or slice longer than its `Uint32` length) is a `*packed.EncodeError`, never a panic;
- `DecodeDevice(bytes []byte) (Device, error)`: a message written for another version of the types is `packed.ErrOutOfDate` (`errors.Is`); truncated, oversized, too deep or otherwise malformed input is an error, never a panic, and gives the zero value.

Reachable types get unexported helpers in the same file, from any package (exported types only, since the file has to name them). Types that can't travel (interfaces, channels, functions, map keys holding a float, a pointer, an interface or a `time.Time`, structs that hold themselves through a pointer) fail the generator with each field's path and a suggestion. `packed.DefaultLimits` bounds every decode: the nesting depth, and the size of the message and of the values decoded from it.

From Go, `generate.Dynamic(directory)` writes the file and `generate.DynamicSource(directory)` returns it. The layout is the DSL's: a struct the DSL can express (Bools as bit runs, integers, floats, `VariableString(Uint32)`, `Slice(Uint32, ...)`, nested structs, arrays) gives the same bytes from its Go twin, after the schema.

## Custom Types

When the built-ins run out, you can teach `packed` your own types (Go only).

### A type that serializes itself (`TypeInterface`)

Implement `Size()` plus the four `ToBytes*`/`FromBytes*` methods, and the type can be used directly as a field. The generated struct field keeps your type — so a device's 4-byte unix timestamp can land in your struct as a real `time.Time`:

```go
type Timestamp struct {
    time.Time
}

func (t *Timestamp) Size() int { return 4 }

func (t *Timestamp) ToBytesLittleEndian(bytes []byte, index int) {
    binary.LittleEndian.PutUint32(bytes[index:], uint32(t.Unix()))
}

func (t *Timestamp) FromBytesLittleEndian(bytes []byte, index int) {
    t.Time = time.Unix(int64(binary.LittleEndian.Uint32(bytes[index:])), 0)
}

func (t *Timestamp) ToBytesBigEndian(bytes []byte, index int) {
    binary.BigEndian.PutUint32(bytes[index:], uint32(t.Unix()))
}

func (t *Timestamp) FromBytesBigEndian(bytes []byte, index int) {
    t.Time = time.Unix(int64(binary.BigEndian.Uint32(bytes[index:])), 0)
}
```

```go
Field("LastSeen", Timestamp{}) // 4 bytes on the wire, time.Time in Go
```

### An external converter (`ConverterInterface[Receiver]`)

When you'd rather not put serialization methods on the type itself, write a separate converter. The generated field gets the *receiver's* type (`Vec2` here), not the converter's:

```go
type Vec2 struct{ X, Y float32 }

type Vec2Converter struct{}

func (Vec2Converter) Size() int { return 8 }

func (Vec2Converter) ToBytesLittleEndian(v *Vec2, bytes []byte, index int) {
    binary.LittleEndian.PutUint32(bytes[index:], math.Float32bits(v.X))
    binary.LittleEndian.PutUint32(bytes[index+4:], math.Float32bits(v.Y))
}

func (Vec2Converter) FromBytesLittleEndian(v *Vec2, bytes []byte, index int) {
    v.X = math.Float32frombits(binary.LittleEndian.Uint32(bytes[index:]))
    v.Y = math.Float32frombits(binary.LittleEndian.Uint32(bytes[index+4:]))
}

func (Vec2Converter) ToBytesBigEndian(v *Vec2, bytes []byte, index int) { /* big-endian variant */ }
func (Vec2Converter) FromBytesBigEndian(v *Vec2, bytes []byte, index int) { /* big-endian variant */ }
```

```go
Field("Position", Vec2Converter{}) // generated field: Position Vec2
```

### Custom bit field types (`BitsTypeInterface`)

Bit fields aren't limited to integers. Any type that can convert itself to and from an integer can live inside a bit run — here a set of four permission flags packed into 4 bits:

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
        if p[i] {
            v |= 1 << i
        }
    }
    return v
}
```

```go
Field("Perms", Bits[uint8](4, Permissions{})) // generated field: Perms Permissions, stored in 4 bits
```

### Custom bit field converters (`BitsConverterInterface`)

The external-converter flavor of the same idea — pack any receiver type into a bit field without adding methods to it. Here a plain `[8]bool` becomes a single byte:

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
        if receiver[i] {
            v |= 1 << i
        }
    }
    return v
}
```

```go
Field("Flags", Bits[uint8](8, BitArrayConverter{})) // generated field: Flags [8]bool, stored in 8 bits
```

## Generate Hooks

Hooks run after each struct's core methods are generated, letting you emit extra Go — getters, stringers, validation, whatever you need:

```go
generate.Files(generate.GoFile("output.go", "mypackage",
    func(buffer *bytes.Buffer, name string, properties []Property) {
        for _, p := range properties {
            switch p.Type.Kind() {
            case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
                reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                fmt.Fprintf(buffer, "func (r *%s) Get%s() int { return int(r.%s) }\n", name, p.Name, p.Name)
            }
        }
    },
))
```

## Constraints

`packed` trades flexibility for layout guarantees, so a few rules are enforced — loudly, at generation time, never silently at runtime:

- Struct, enum and message set names and field names must be unique (duplicates panic)
- A field cannot take the name of a generated method: `Size`, `ToBytes`, `FromBytes`, and `<Set>Type` in a variant of a message set
- Array and slice elements cannot be bare bit fields
- A `Bits` size cannot exceed its underlying integer type's width (as in C, where an over-wide bit field is a compile error)
- Bit fields inherit the struct's endianness and cannot be overridden per field
- Values that overflow a bit field's width are silently truncated (masked), matching C behavior
- Go's `FromBytes` has no bounds checks of its own: `Decode<Set>` turns the panic into an error, a direct caller still gets the panic

## Test

```sh
go test ./...
go test ./internal/compliance -update   # rewrite the checked-in outputs after a change
go test ./internal/dynamic -update      # the same for the dynamic layer's fixtures
go test ./internal/dynamic -run '^$' -fuzz FuzzDevice -fuzztime 1m   # also FuzzTree, FuzzSmall
```

`internal/compliance` holds one set of definitions generated for Go, C and C++ (`header/messages.h`), Kotlin (`kotlin/Output.kt`), Swift (`swift/`) and TypeScript (`typescript/output.ts`). `go test` writes and checks `testdata/vectors.txt` and fails when a checked-in output is stale. It runs the Swift package against the vectors when `swift` is on PATH and the TypeScript test when `node` is. With each of `clang`, `gcc`, `clang++` and `g++` on PATH (warnings as errors, pedantic, the undefined behaviour sanitizer) it builds `header/compliance.c`, which checks every sample through the C functions, as C11 and as C++20, and `header/compliance.cpp`, which checks the C++ layer, and runs them against the vectors. With an `avr-gcc` of GCC 10 or newer on PATH it builds `header/firmware.c` for an ATtiny88 as C and as C++ and checks that it fits its flash. `kotlin/ComplianceTest.kt` is a JUnit test of the Kotlin output against the same vectors, for a Gradle module that compiles it with `Output.kt` and passes the vectors' path as the `packed.vectors` system property.

`internal/test` covers layouts, round trips and errors. `internal/dynamic` holds Go types for the dynamic layer (`types.go`, with types of another package in `other/`), their generated `packed_dynamic.go`, and `dsl/`, the DSL's twin of one of them; `go test` pins their bytes, round trips, hostile input and the generator's errors.

## Documentation

The builder API — every builder function, interface, and generation hook — is in [llms.md](llms.md).
