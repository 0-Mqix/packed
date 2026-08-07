# packed

**Define your binary layout once. Get fast, zero-reflection Go code that matches it byte-for-byte.**

`packed` is a code generator for binary data structures. You describe your wire format with a small builder DSL — down to individual bits — and `packed` generates plain Go structs with hand-tight `ToBytes` / `FromBytes` methods. No reflection, no allocations, no interface dispatch at runtime: every offset is precomputed at generation time, and bit fields compile down to raw shifts and masks. The output is the code you would have written by hand, if you had the patience to get every offset and mask right — and to keep them right every time the format changes.

Where `packed` really shines is talking to microcontrollers. Its bit-field layout reproduces C `struct __attribute__((__packed__))` byte-for-byte, in both little- and big-endian. If your firmware sends this:

```c
struct __attribute__((__packed__)) status {
    uint8_t online   : 1;
    uint8_t charging : 1;
    uint8_t error    : 3;
    uint8_t signal   : 3;
};
```

then this definition on the Go side decodes it — same bits, same bytes, no manual masking:

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

Say your IoT devices report telemetry: a device ID, two sensor readings, and that packed status byte. Write a `generate.go` (kept out of normal builds with `//go:build ignore`) that describes the packet:

```go
//go:build ignore

package main

import . "github.com/0-Mqix/packed"

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

    Generate("telemetry.go", "mypackage")
}
```

Run `go run generate.go` (or hook it up with `//go:generate`) and you get real, readable Go. This is the actual output:

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

func (reciever *Status) ToBytes(bytes []byte, index int) {
    var b0 uint64
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Online))) & 1) << 0
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Charging))) & 1) << 1
    b0 |= (uint64(reciever.ErrorCode) & 0x7) << 2
    b0 |= (uint64(reciever.SignalStrength) & 0x7) << 5
    bytes[index+0+0] = byte(b0)
}

func (reciever *Status) FromBytes(bytes []byte, index int) {
    var b0 uint64
    b0 |= uint64(bytes[index+0+0]) << 0
    reciever.Online = ((b0 >> 0) & 0x1) != 0
    reciever.Charging = ((b0 >> 1) & 0x1) != 0
    reciever.ErrorCode = uint8(uint64((b0 >> 2) & 0x7))
    reciever.SignalStrength = uint8(uint64((b0 >> 5) & 0x7))
}

type Telemetry struct {
    DeviceID    uint16  `json:"device_id"`
    Temperature float32 `json:"temperature"`
    Humidity    float32 `json:"humidity"`
    Status      Status
}

func (reciever *Telemetry) Size() int {
    return 11
}

func (reciever *Telemetry) ToBytes(bytes []byte, index int) {
    c1.ToBytesLittleEndian(&reciever.DeviceID, bytes, index+0)
    c0.ToBytesLittleEndian(&reciever.Temperature, bytes, index+2)
    c0.ToBytesLittleEndian(&reciever.Humidity, bytes, index+6)
    var b0 uint64
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Status.Online))) & 1) << 0
    b0 |= (uint64(*(*uint8)(unsafe.Pointer(&reciever.Status.Charging))) & 1) << 1
    b0 |= (uint64(reciever.Status.ErrorCode) & 0x7) << 2
    b0 |= (uint64(reciever.Status.SignalStrength) & 0x7) << 5
    bytes[index+10+0] = byte(b0)
}

func (reciever *Telemetry) FromBytes(bytes []byte, index int) {
    c1.FromBytesLittleEndian(&reciever.DeviceID, bytes, index+0)
    c0.FromBytesLittleEndian(&reciever.Temperature, bytes, index+2)
    c0.FromBytesLittleEndian(&reciever.Humidity, bytes, index+6)
    var b0 uint64
    b0 |= uint64(bytes[index+10+0]) << 0
    reciever.Status.Online = ((b0 >> 0) & 0x1) != 0
    reciever.Status.Charging = ((b0 >> 1) & 0x1) != 0
    reciever.Status.ErrorCode = uint8(uint64((b0 >> 2) & 0x7))
    reciever.Status.SignalStrength = uint8(uint64((b0 >> 5) & 0x7))
}
```

Notice what happened: four status fields collapsed into one byte of shift-and-mask code, every offset is a compile-time constant, and the nested `Status` was inlined straight into `Telemetry`'s methods — the whole 11-byte packet encodes and decodes without a single reflection call, allocation, or loop. This is code the Go compiler loves to optimize.

Decoding a packet off the wire is now one line:

```go
var t Telemetry
t.FromBytes(payload, 0)

fmt.Printf("device %d: %.1f°C, signal %d/7\n", t.DeviceID, t.Temperature, t.Status.SignalStrength)
```

And encoding is just as direct — the `index` parameter lets you pack a batch of readings back-to-back into one buffer:

```go
readings := []Telemetry{ /* ... */ }
buf := make([]byte, len(readings)*readings[0].Size())

for i := range readings {
    readings[i].ToBytes(buf, i*readings[i].Size())
}
```

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

**Enums** come out properly typed instead of as bare integers. `Cast` stores the field as the converter's wire type but exposes it as your own type:

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

## Custom Types

When the built-ins run out, you can teach `packed` your own types.

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

Hooks run after each struct's core methods are generated, letting you emit extra code — getters, stringers, validation, whatever you need:

```go
Generate("output.go", "mypackage",
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
```

## Constraints

`packed` trades flexibility for layout guarantees, so a few rules are enforced — loudly, at generation time, never silently at runtime:

- Struct names and field names must be unique (duplicates panic)
- All sizes are fixed at definition time — there are no variable-length fields
- Array elements cannot be bare bit fields
- A `Bits` size cannot exceed its underlying integer type's width (as in C, where an over-wide bit field is a compile error)
- Bit fields inherit the struct's endianness and cannot be overridden per field
- Values that overflow a bit field's width are silently truncated (masked), matching C behavior

## Documentation

The complete API reference — every builder function, interface, and generation hook — lives in [llms.md](llms.md).
