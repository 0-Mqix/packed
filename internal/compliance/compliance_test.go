package compliance

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/vectors.txt and the generated files")

// samples must match the values built in kotlin/ComplianceTest.kt and
// swift/Tests/ComplianceTests/ComplianceTests.swift; those encode them and
// compare with the bytes Go wrote to testdata/vectors.txt.
func samples() []struct {
	name    string
	message interface {
		Size() int
		ToBytes([]byte, int) int
		FromBytes([]byte, int) int
	}
} {
	return []struct {
		name    string
		message interface {
			Size() int
			ToBytes([]byte, int) int
			FromBytes([]byte, int) int
		}
	}{
		{"primitives", &Primitives{
			Flag: true, Small: -5, Byte: 250, Short: -12345, UnsignedShort: 54321,
			Integer: -123456789, UnsignedInteger: 3123456789, Long: math.MinInt64 + 7, UnsignedLong: math.MaxUint64 - 3,
			Single: -1.25, Double: math.Pi, LittleInteger: 0x01020304, Code: "ab·",
		}},
		{"bits", &Bits{
			Active: true, Mode: 9, Delta: -11, Wide: 0x0ABCDEF012345678, Straddle: -54321, Tail: 1500,
			Inner:    Inner{A: 5, B: -200, C: 0xBEEF},
			BigInner: Inner{A: 3, B: 255, C: 0x1234},
			Inners:   [2]Inner{{A: 1, B: -1, C: 1}, {A: 7, B: -256, C: 65535}},
		}},
		{"record", &Record{
			ID: 4000000000, Name: "Pump 3 · west 🚰", Severity: SeverityHigh, Signed: SignedPositive,
			Flags:      RecordFlags{Active: true, Level: 6},
			Labels:     []Label{{Name: "inlet", Weight: 0.5}, {Name: "", Weight: -3}},
			Values:     []int16{-32768, 0, 32767},
			Trailer:    0xCAFE,
			Pair:       [2]string{"left", ""},
			Nested:     [][]uint8{{1, 2, 3}, {}, {255}},
			Severities: []Severity{SeverityLow, SeverityMedium, SeverityHigh},
			Matrix:     [2][2]int16{{1, -2}, {-3, 4}},
		}},
		{"ping", &Ping{Sequence: 0xDEADBEEF}},
		{"blob", &Blob{Payload: []byte{0, 1, 254, 255}, Small: []byte{}, Chunks: [][]byte{{7}, {}, {8, 9}}}},
		{"status", &Status{Identifier: 0x5E, Version: 2, Keys: 0x15, Frames: 0x1234, Log: []uint8{'o', 'k', 0, 255}}},
		{"descriptor", &Descriptor{Level: LevelWarning, Message: 0xBEEF, Keys: []KeyEvent{{Key: 1, Kind: KeyKindPress}, {Key: 0x0203, Kind: KeyKindHold}}}},
		{"outer", outer()},
	}
}

// outer is the Outer the other languages build: structs nested every way.
func outer() *Outer {
	named := func(name string, points ...Point) Named {
		return Named{Name: name, Points: append([]Point{}, points...)}
	}
	tag := Tagged{On: true, Mode: ModeFast, At: Point{X: -3, Y: 7}}
	return &Outer{
		Inner: named("inner", Point{X: 1, Y: 2}),
		Pair:  [2]Named{named("left"), named("right", Point{X: -1, Y: -2}, Point{X: 300, Y: -300})},
		Grid:  [2][2]Point{{{X: 1, Y: 2}, {X: 3, Y: 4}}, {{X: 5, Y: 6}, {X: 7, Y: 8}}},
		Tags:  []Tagged{tag, {Mode: ModeSlow, At: Point{X: 9, Y: 10}}},
		Names: []Named{named("a", Point{X: 11, Y: 12}), named("bb")},
		Bigs:  []Big{{Value: 0x1234}, {Value: 0xABCD}},
		Deep:  Deep{Level: Deeper{Tags: []Tagged{tag}, Mode: ModeSlow}},
	}
}

// frame is the SetFrame the C++ test builds: pixel i is 3i+1.
func frame() *SetFrame {
	frame := &SetFrame{}
	for index := range frame.Pixels {
		frame.Pixels[index] = uint8(index*3 + 1)
	}
	return frame
}

var toggle = &Toggle{On: true, Channel: 9, Duration: 0xBEEF}

type taggedSample struct {
	name    string
	message any
	encoded []byte
	// size is the length encoded must have: the tag costs a byte only
	// where the variant's first bit run has no room for it.
	size   int
	decode func([]byte) (any, error)
}

func decodeCommand(bytes []byte) (any, error) { return DecodeCommand(bytes) }
func decodeSignal(bytes []byte) (any, error)  { return DecodeSignal(bytes) }
func decodePulse(bytes []byte) (any, error)   { return DecodePulse(bytes) }

// taggedSamples are the messages of the sets with TagBits; the Kotlin and
// Swift tests build the same ones.
func taggedSamples() []taggedSample {
	rename := &Rename{Name: "pump"}
	dial := &Dial{Level: -20, Fine: true, Severity: SeverityHigh}
	beacon := &Beacon{Urgent: true, Level: -7, Signed: SignedPositive, Count: 0x1234}
	heartbeat := &Heartbeat{Count: 42}

	return []taggedSample{
		{"command-toggle", toggle, EncodeCommand(toggle), 3, decodeCommand},
		{"command-rename", rename, EncodeCommand(rename), 6, decodeCommand},
		{"command-dial", dial, EncodeCommand(dial), 2, decodeCommand},
		{"command-halt", &Halt{}, EncodeCommand(&Halt{}), 1, decodeCommand},
		{"signal-beacon", beacon, EncodeSignal(beacon), 4, decodeSignal},
		{"signal-quiet", &Quiet{}, EncodeSignal(&Quiet{}), 1, decodeSignal},
		{"pulse-heartbeat", heartbeat, EncodePulse(heartbeat), 4, decodePulse},
	}
}

func TestTagBits(t *testing.T) {
	for _, sample := range taggedSamples() {
		if len(sample.encoded) != sample.size {
			t.Errorf("%s: %d bytes, want %d", sample.name, len(sample.encoded), sample.size)
		}

		decoded, err := sample.decode(sample.encoded)
		if err != nil || !reflect.DeepEqual(decoded, sample.message) {
			t.Errorf("%s: decoded %+v, %v, want %+v", sample.name, decoded, err, sample.message)
		}

		for length := range len(sample.encoded) {
			if _, err := sample.decode(sample.encoded[:length:length]); err == nil {
				t.Errorf("%s: decoded %d of its %d bytes", sample.name, length, len(sample.encoded))
			}
		}
	}

	failures := []struct {
		bytes  []byte
		decode func([]byte) (any, error)
		want   string
	}{
		{nil, decodeCommand, "packed: empty Command message"},
		{[]byte{0x01}, decodeCommand, "packed: unknown Command type 1"},
		{[]byte{0xC0}, decodeSignal, "packed: unknown Signal type 3"},
		{EncodeCommand(toggle)[:2:2], decodeCommand, "packed: truncated Command type 0"},
		{append(EncodeCommand(toggle), 0), decodeCommand, "packed: Command type 0 has 1 trailing bytes"},
		{[]byte{1, 2}, decodePulse, "packed: truncated Pulse type 0"},
		{[]byte{1, 2, 3, 4, 5}, decodePulse, "packed: Pulse type 0 has 1 trailing bytes"},
	}
	for _, failure := range failures {
		if _, err := failure.decode(failure.bytes); err == nil || err.Error() != failure.want {
			t.Errorf("% x: got %v, want %s", failure.bytes, err, failure.want)
		}
	}

	prefixed := AppendCommand([]byte{0xAB}, toggle)
	if decoded, err := DecodeCommand(prefixed[1:]); prefixed[0] != 0xAB || err != nil || !reflect.DeepEqual(decoded, toggle) {
		t.Errorf("prefixed: % x decoded to %+v, %v", prefixed, decoded, err)
	}
}

// TestMaximum checks that a list over its Maximum panics when encoded and
// when decoded, the bytes on the wire being those of any list.
func TestMaximum(t *testing.T) {
	panics := func(name string, want string, run func()) {
		t.Helper()
		defer func() {
			t.Helper()
			if recovered := recover(); recovered == nil || !strings.Contains(fmt.Sprint(recovered), want) {
				t.Errorf("%s: got %v, want a panic with %q", name, recovered, want)
			}
		}()
		run()
	}

	want := "packed: Chunks holds more than its maximum of 4 items"
	blob := &Blob{Chunks: [][]byte{{1}, {2}, {3}, {4}, {5}}}
	panics("encode", want, func() { blob.ToBytes(make([]byte, 64), 0) })
	// an empty Payload and Small, then 5 chunks
	panics("decode", want, func() { (&Blob{}).FromBytes([]byte{0, 0, 0, 0, 0, 5}, 0) })

	full := &Blob{Chunks: [][]byte{{1}, {2}, {3}, {4}}}
	if bytes := encode(full); !reflect.DeepEqual(bytes[:6], []byte{0, 0, 0, 0, 0, 4}) {
		t.Errorf("a full list encoded as % x", bytes)
	}
}

func encode(message interface {
	Size() int
	ToBytes([]byte, int) int
}) []byte {
	bytes := make([]byte, message.Size())
	if end := message.ToBytes(bytes, 0); end != len(bytes) {
		panic(fmt.Sprintf("ToBytes returned %d for size %d", end, len(bytes)))
	}
	return bytes
}

func TestVectors(t *testing.T) {
	lines := []string{}
	for _, sample := range samples() {
		bytes := encode(sample.message)
		lines = append(lines, sample.name+" "+hex.EncodeToString(bytes))

		back := reflect.New(reflect.TypeOf(sample.message).Elem()).Interface().(interface {
			FromBytes([]byte, int) int
		})
		if end := back.FromBytes(bytes, 0); end != len(bytes) {
			t.Fatalf("%s: FromBytes returned %d, want %d", sample.name, end, len(bytes))
		}
		if !reflect.DeepEqual(back, sample.message) {
			t.Fatalf("%s: round trip got %+v want %+v", sample.name, back, sample.message)
		}
	}

	record := samples()[2].message.(*Record)
	lines = append(lines, "envelope-record "+hex.EncodeToString(EncodeEnvelope(record)))
	lines = append(lines, "envelope-ping "+hex.EncodeToString(EncodeEnvelope(&Ping{Sequence: 7})))

	for _, tagged := range taggedSamples() {
		lines = append(lines, tagged.name+" "+hex.EncodeToString(tagged.encoded))
	}
	lines = append(lines, "prefixed-toggle "+hex.EncodeToString(AppendCommand([]byte{0xAB}, toggle)))
	lines = append(lines, "display-setframe "+hex.EncodeToString(EncodeDisplay(frame())))
	lines = append(lines, "display-read "+hex.EncodeToString(EncodeDisplay(&Read{})))

	content := strings.Join(lines, "\n") + "\n"

	if *update {
		if err := os.WriteFile("testdata/vectors.txt", []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	existing, err := os.ReadFile("testdata/vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(existing) != content {
		t.Fatalf("testdata/vectors.txt is stale, run go test -update\n got:\n%s\nwant:\n%s", existing, content)
	}
}

// generated are the files generate.go writes, relative to this folder.
var generated = []string{
	"output.go",
	"kotlin/Output.kt",
	"swift/Sources/Compliance/Messages.swift",
	"swift/Sources/Compliance/PackedError.swift",
	"typescript/output.ts",
	"header/messages.h",
}

// strict are the flags every host build of the header's tests takes:
// warnings as errors, and the undefined behaviour sanitizer.
var strict = []string{"-O2", "-Wall", "-Wextra", "-Wpedantic", "-Werror", "-fsanitize=undefined", "-fno-sanitize-recover=undefined"}

// TestHeader builds the header's tests with each host compiler on PATH and
// runs them against testdata/vectors.txt: header/compliance.c, which checks
// every sample through the C functions, as C11 and as C++20, and
// header/compliance.cpp, which checks the C++ layer. It is skipped when
// there is no compiler.
func TestHeader(t *testing.T) {
	vectors, err := filepath.Abs("testdata/vectors.txt")
	if err != nil {
		t.Fatal(err)
	}

	builds := []struct {
		compiler  string
		arguments []string
	}{
		{"clang", []string{"-std=c11", "header/compliance.c"}},
		{"gcc", []string{"-std=c11", "header/compliance.c"}},
		{"clang++", []string{"-std=c++20", "-x", "c++", "header/compliance.c"}},
		{"g++", []string{"-std=c++20", "-x", "c++", "header/compliance.c"}},
		{"clang++", []string{"-std=c++20", "header/compliance.cpp"}},
		{"g++", []string{"-std=c++20", "header/compliance.cpp"}},
	}

	ran := false
	for _, build := range builds {
		compiler, err := exec.LookPath(build.compiler)
		if err != nil {
			continue
		}
		ran = true
		name := build.compiler + " " + filepath.Base(build.arguments[len(build.arguments)-1])
		t.Run(name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "compliance")
			arguments := append(append(append([]string{}, strict...), build.arguments...), "-o", binary)
			if output, err := exec.Command(compiler, arguments...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", build.compiler, err, output)
			}
			output, err := exec.Command(binary, vectors).CombinedOutput()
			if err != nil {
				t.Fatalf("compliance: %v\n%s", err, output)
			}
			t.Logf("%s", output)
		})
	}
	if !ran {
		t.Skip("no C or C++ compiler is on PATH")
	}
}

// TestAVR builds header/firmware.c for an ATtiny88 as C with avr-gcc and
// as C++ with avr-g++ (C++20 needs GCC 10 or newer), and requires each to
// fit the chip's 8 KB of flash. It is skipped when no such avr-gcc is on
// PATH.
func TestAVR(t *testing.T) {
	gcc, err := exec.LookPath("avr-gcc")
	if err != nil {
		t.Skip("avr-gcc is not on PATH")
	}
	version, err := exec.Command(gcc, "-dumpversion").Output()
	if err != nil {
		t.Skipf("avr-gcc -dumpversion: %v", err)
	}
	major, _, _ := strings.Cut(strings.TrimSpace(string(version)), ".")
	if number, err := strconv.Atoi(major); err != nil || number < 10 {
		t.Skipf("avr-gcc %s has no C++20", strings.TrimSpace(string(version)))
	}

	builds := []struct {
		compiler string
		language []string
	}{
		{"avr-gcc", []string{"-std=c11"}},
		{"avr-g++", []string{"-std=c++20", "-fno-exceptions", "-x", "c++"}},
	}
	for _, build := range builds {
		t.Run(build.compiler, func(t *testing.T) {
			compiler, err := exec.LookPath(build.compiler)
			if err != nil {
				t.Skipf("%s is not on PATH", build.compiler)
			}
			binary := filepath.Join(t.TempDir(), "firmware.elf")
			// The compliance header holds Float64 fields, which need a 64-bit double.
			arguments := append(append([]string{"-mmcu=attiny88", "-mdouble=64", "-Os", "-Wall", "-Wextra", "-Wpedantic", "-Werror"},
				build.language...), "header/firmware.c", "-o", binary)
			if output, err := exec.Command(compiler, arguments...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", build.compiler, err, output)
			}

			size, err := exec.LookPath("avr-size")
			if err != nil {
				return
			}
			output, err := exec.Command(size, binary).CombinedOutput()
			if err != nil {
				t.Fatalf("avr-size: %v\n%s", err, output)
			}
			t.Logf("%s", output)

			fields := strings.Fields(strings.Split(strings.TrimSpace(string(output)), "\n")[1])
			text, data := 0, 0
			fmt.Sscan(fields[0], &text)
			fmt.Sscan(fields[1], &data)
			if text+data > 8192 {
				t.Errorf("the firmware takes %d bytes of flash, more than the ATtiny88's 8192", text+data)
			}
		})
	}
}

// TestTypeScript runs the TypeScript compliance test, which encodes the
// samples with the generated TypeScript (and ../../typescript/packed.ts)
// and compares with testdata/vectors.txt, in Node, which strips the types.
// It is skipped when node is not on PATH.
func TestTypeScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH")
	}
	output, err := exec.Command(node, "--test", "typescript/compliance.test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("node --test: %v\n%s", err, output)
	}
}

// TestGeneratedFilesAreCurrent runs generate.go into a temporary folder and
// requires the checked-in Go, Kotlin and Swift output to match it.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	directory := t.TempDir()

	output, err := exec.Command("go", "run", "generate.go", directory).CombinedOutput()
	if err != nil {
		t.Fatalf("go run generate.go: %v\n%s", err, output)
	}

	for _, path := range generated {
		fresh, err := os.ReadFile(filepath.Join(directory, path))
		if err != nil {
			t.Fatal(err)
		}

		if *update {
			if err := os.WriteFile(path, fresh, 0644); err != nil {
				t.Fatal(err)
			}
		}

		existing, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(existing, fresh) {
			t.Errorf("%s is stale, run go test -update (or go run generate.go)", path)
		}
	}
}

// TestSwift runs the Swift compliance package, which encodes the samples
// with the generated Swift and compares with testdata/vectors.txt. It is
// skipped when swift is not on PATH. The build goes to a temporary folder
// so nothing is left in the repository.
func TestSwift(t *testing.T) {
	swift, err := exec.LookPath("swift")
	if err != nil {
		t.Skip("swift is not on PATH")
	}

	command := exec.Command(swift, "test", "--package-path", "swift", "--scratch-path", t.TempDir())
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("swift test: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}
