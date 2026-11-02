//go:build ignore

// Generates the Go, C and C++, Kotlin, Swift and TypeScript output for the
// compliance definitions: go run generate.go [directory], directory
// defaulting to this one. TestGeneratedFilesAreCurrent runs it into a
// temporary directory.
package main

import (
	"os"
	"path/filepath"

	. "github.com/0-Mqix/packed"
	"github.com/0-Mqix/packed/generate"
)

func main() {
	directory := "."
	if len(os.Args) > 1 {
		directory = os.Args[1]
	}

	Severity := Enum("Severity", Uint8, "Low", "Medium", "High")
	Signed := Enum("Signed", Int16, "Negative", "Zero", "Positive")

	Label := Struct("Label", true,
		Field("Name", VariableString(Uint8)),
		Field("Weight", Float32),
	)

	Struct("Primitives", false,
		Field("Flag", Boolean),
		Field("Small", Int8),
		Field("Byte", Uint8),
		Field("Short", Int16),
		Field("UnsignedShort", Uint16),
		Field("Integer", Int32),
		Field("UnsignedInteger", Uint32),
		Field("Long", Int64),
		Field("UnsignedLong", Uint64),
		Field("Single", Float32),
		Field("Double", Float64),
		Field("LittleInteger", Int32, LittleEndian(true)),
		Field("Code", String(6)),
	)

	Inner := Struct("Inner", false,
		Field("A", Bits[uint8](3)),
		Field("B", Bits[int16](9)),
		Field("C", Uint16),
	)

	Struct("Bits", true,
		Field("Active", Bit),
		Field("Mode", Bits[uint8](4)),
		Field("Delta", Bits[int8](5)),
		Field("Wide", Bits[uint64](60)),
		Field("Straddle", Bits[int32](17)),
		Field("Tail", Bits[uint16](11)),
		Field("Inner", Inner),
		Field("BigInner", Inner, LittleEndian(false)),
		Field("Inners", Array(2, Inner)),
	)

	Record := Struct("Record", true,
		Field("ID", Uint32),
		Field("Name", VariableString(Uint16)),
		Field("Severity", Severity),
		Field("Signed", Signed),
		Field("Flags", Struct("RecordFlags", true,
			Field("Active", Bit),
			Field("Level", Bits[uint8](3)),
		)),
		Field("Labels", Slice(Uint8, Label).Maximum(4)),
		Field("Values", Slice(Uint16, Int16).Maximum(4)),
		Field("Trailer", Uint16, LittleEndian(false)),
		Field("Pair", Array(2, VariableString(Uint8))),
		Field("Nested", Slice(Uint8, Slice(Uint32, Uint8)).Maximum(4)),
		Field("Severities", Slice(Uint8, Severity).Maximum(4)),
		Field("Matrix", Array(2, Array(2, Int16))),
	)

	Ping := Struct("Ping", false,
		Field("Sequence", Uint32),
	)

	Struct("Blob", true,
		Field("Payload", VariableBytes(Uint32)),
		Field("Small", VariableBytes(Uint8)),
		Field("Chunks", Slice(Uint8, VariableBytes(Uint16)).Maximum(4)),
	)

	MessageSet("Envelope",
		Variant(7, Record),
		Variant(200, Ping),
	)

	// Command's 3-bit tag joins Toggle's and Dial's first bit runs, takes
	// a byte of its own in front of Rename, and is all of Halt.
	Toggle := Struct("Toggle", true,
		Field("On", Bit),
		Field("Channel", Bits[uint8](4)),
		Field("Duration", Uint16),
	)

	Rename := Struct("Rename", true,
		Field("Name", VariableString(Uint8)),
	)

	Dial := Struct("Dial", true,
		Field("Level", Bits[int8](6)),
		Field("Fine", Bit),
		Field("Severity", Severity.Bits(2)),
	)

	Halt := Struct("Halt", true)

	MessageSet("Command", TagBits(3),
		Variant(0, Toggle),
		Variant(5, Rename),
		Variant(6, Dial),
		Variant(7, Halt),
	)

	// Big endian: the tag is the high bits of the first byte.
	Beacon := Struct("Beacon", false,
		Field("Urgent", Bit),
		Field("Level", Bits[int8](5)),
		Field("Signed", Signed.Bits(2)),
		Field("Count", Uint16),
	)

	Quiet := Struct("Quiet", false)

	MessageSet("Signal", TagBits(2),
		Variant(1, Beacon),
		Variant(2, Quiet),
	)

	// One variant needs no tag.
	Heartbeat := Struct("Heartbeat", true,
		Field("Count", Uint32),
	)

	MessageSet("Pulse", TagBits(0),
		Variant(0, Heartbeat),
	)

	// A microcontroller's messages, as an ATtiny's firmware takes them: a
	// frame of pixels, a status with a log and a descriptor with a list of
	// structs, all little endian.
	SetFrame := Struct("SetFrame", true,
		Field("Pixels", Array(192, Uint8)),
	)

	Read := Struct("Read", true)

	MessageSet("Display",
		Variant(1, SetFrame),
		Variant(2, Read),
	)

	Struct("Status", true,
		Field("Identifier", Uint8),
		Field("Version", Uint8),
		Field("Keys", Bits[uint8](5)),
		Field("Frames", Uint16),
		Field("Log", Slice(Uint8, Uint8)),
	)

	Level := Enum("Level", Uint8, "Debug", "Information", "Warning", "Error")
	KeyKind := Enum("KeyKind", Uint8, "Press", "Release", "Hold")

	Struct("Descriptor", true,
		Field("Level", Level),
		Field("Message", Uint16),
		Field("Keys", Slice(Uint8, Struct("KeyEvent", true,
			Field("Key", Uint16),
			Field("Kind", KeyKind),
		))),
	)

	// Structs in structs, arrays and lists, every way C holds them: a
	// nested struct with its own text and list, an array of those, an
	// array of arrays, a list in place of structs that nest a struct and
	// enum bits, lists with a maximum of structs holding lists and of big
	// endian structs, and a list three structs deep.
	Mode := Enum("Mode", Uint8, "Off", "Slow", "Fast")
	Point := Struct("Point", true, Field("X", Int16), Field("Y", Int16))
	Tagged := Struct("Tagged", true, Field("On", Bit), Field("Mode", Mode.Bits(2)), Field("At", Point))
	Named := Struct("Named", true, Field("Name", VariableString(Uint8)), Field("Points", Slice(Uint8, Point)))
	Big := Struct("Big", false, Field("Value", Uint16))

	Struct("Outer", true,
		Field("Inner", Named),
		Field("Pair", Array(2, Named)),
		Field("Grid", Array(2, Array(2, Point))),
		Field("Tags", Slice(Uint8, Tagged)),
		Field("Names", Slice(Uint8, Named).Maximum(3)),
		Field("Bigs", Slice(Uint8, Big).Maximum(2)),
		Field("Deep", Struct("Deep", true,
			Field("Level", Struct("Deeper", true,
				Field("Tags", Slice(Uint8, Tagged)),
				Field("Mode", Mode),
			)),
		)),
	)

	generate.Files(
		generate.GoFile(filepath.Join(directory, "output.go"), "compliance"),
		generate.KotlinFile(filepath.Join(directory, "kotlin/Output.kt"), "compliance"),
		generate.SwiftFile(filepath.Join(directory, "swift/Sources/Compliance/Messages.swift")),
		generate.SwiftErrorFile(filepath.Join(directory, "swift/Sources/Compliance/PackedError.swift")),
		generate.TypeScriptFile(filepath.Join(directory, "typescript/output.ts"), "../../../typescript/packed.ts"),
		generate.HeaderFile(filepath.Join(directory, "header/messages.h")),
	)
}
