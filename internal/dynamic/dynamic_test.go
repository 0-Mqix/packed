package dynamic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0-Mqix/packed"
	"github.com/0-Mqix/packed/generate"
	"github.com/0-Mqix/packed/internal/dynamic/dsl"
	"github.com/0-Mqix/packed/internal/dynamic/other"
)

var update = flag.Bool("update", false, "rewrite the generated files")

// TestGeneratedFilesAreCurrent generates the dynamic layer of this package
// and the DSL's twin of Small, and requires the checked-in files to match.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	fresh, err := generate.DynamicSource(".")
	if err != nil {
		t.Fatal(err)
	}
	compare(t, generate.DynamicFile, fresh)

	directory := t.TempDir()
	command := exec.Command("go", "run", "generate.go", directory)
	command.Dir = "dsl"
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go run generate.go: %v\n%s", err, output)
	}
	fresh, err = os.ReadFile(filepath.Join(directory, "output.go"))
	if err != nil {
		t.Fatal(err)
	}
	compare(t, "dsl/output.go", fresh)
}

func compare(t *testing.T, path string, fresh []byte) {
	t.Helper()
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
		t.Errorf("%s is stale, run go test -update", path)
	}
}

func date(day int) time.Time {
	return time.Date(2026, time.October, day, 9, 30, 15, 123456789, time.UTC)
}

func sampleDevice() Device {
	email := "ada@example.com"
	seen := date(3)
	level := int32(-42)
	optional := &level

	return Device{
		ID:       math.MaxInt,
		Name:     "Pump 3 · west 🚰",
		Online:   true,
		Muted:    true,
		Owner:    &User{Name: "Ada", Admin: true, Since: date(1), Email: &email},
		Archived: true,
		Status:   7,

		Tags:        Tags{"inlet", "", "north"},
		Labels:      []Label{},
		Readings:    []float32{1.5, float32(math.Inf(-1)), 0},
		Raw:         []byte{0, 1, 254, 255},
		Flags:       []bool{true, false, true},
		Counts:      []int{-1, math.MaxInt, math.MinInt},
		Matrix:      [2][3]int16{{1, -2, 3}, {-4, 5, math.MinInt16}},
		Temperature: -12.5,
		Unsigned:    math.MaxUint,

		Scores: map[string]int64{"b": 2, "a": 1, "": -1, "long key": math.MaxInt64},
		Groups: map[Point][]string{{X: 1, Y: 2}: {"x"}, {X: -1, Y: 2}: nil, {X: 0}: {}},
		Lookup: map[Status]*Part{3: {Serial: "s3", Count: 3}, 1: nil},

		Created: date(2),
		Seen:    &seen,

		Children: []Device{{Name: "child", Children: []Device{{Name: "grandchild"}}}},
		Parts:    [2]Part{{Serial: "a", Count: 1, Spare: true}, {}},
		Nested:   [][]Point{{{X: 1, Y: 1}}, nil, {}},
		Window:   &[4]uint8{1, 2, 3, 4},
		Optional: &optional,

		Location: other.Location{City: "Delft", Coordinates: [2]float64{52.01, 4.36}, Zone: -3, Tags: map[string]string{"k": "v"}},
		Pages:    other.Page[Point]{Items: []Point{{X: 5}}, Total: 1},
	}
}

func init() {
	sample := sampleDevice()
	sample.Meta.Version = 3
	sample.Meta.Beta = true
	sample.Meta.Notes = []string{"one"}
	samples = append(samples, sample, Device{}, Device{Optional: new(*int32), Lookup: map[Status]*Part{}})
}

var samples []Device

func TestRoundTrip(t *testing.T) {
	for index, device := range samples {
		message := must(device.AppendPacked(nil))
		decoded, err := DecodeDevice(message)
		if err != nil {
			t.Fatalf("sample %d: %v", index, err)
		}
		if !reflect.DeepEqual(decoded, device) {
			t.Fatalf("sample %d:\n got %+v\nwant %+v", index, decoded, device)
		}
		if again := must(decoded.AppendPacked(nil)); !bytes.Equal(again, message) {
			t.Fatalf("sample %d: the decoded value encodes to other bytes", index)
		}
	}

	tree := Tree{Value: 1, Children: []Tree{{Value: 2}, {Value: 3, Children: []Tree{}}}}
	decoded, err := DecodeTree(must(tree.AppendPacked(nil)))
	if err != nil || !reflect.DeepEqual(decoded, tree) {
		t.Fatalf("tree: %+v, %v", decoded, err)
	}

	if message := must((Empty{}).AppendPacked(nil)); len(message) != 8 {
		t.Fatalf("an empty struct's message is %d bytes, want its schema's 8", len(message))
	} else if _, err := DecodeEmpty(message); err != nil {
		t.Fatal(err)
	}
}

// TestSameBytes encodes one value many times, with its maps built in
// different orders: a map's entries are sorted by their keys' bytes.
func TestSameBytes(t *testing.T) {
	first := must(sampleDevice().AppendPacked(nil))

	for range 50 {
		device := sampleDevice()
		scores := map[string]int64{}
		for _, key := range []string{"long key", "", "b", "a"} {
			scores[key] = device.Scores[key]
		}
		device.Scores = scores
		if message := must(device.AppendPacked(nil)); !bytes.Equal(message, first) {
			t.Fatal("one value gave two encodings")
		}
	}
}

// TestLayout pins the bytes of Small and Tree.
func TestLayout(t *testing.T) {
	small := Small{Active: true, Count: 0x0102, Total: -2, Name: "ab", Ratio: 1, Level: -1, Inner: Inner{On: true, Value: 5}, Codes: [2]uint16{7, 8}}

	schema := binary.LittleEndian.AppendUint64(nil, SmallSchema)
	want := append(schema,
		0b01,       // Active and Visible: one bit run
		0x02, 0x01, // Count
		0xFE, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // Total
		2, 0, 0, 0, 'a', 'b', // Name: a Uint32 length and UTF-8
		0, 0, 0x80, 0x3F, // Ratio
		0xFF,                   // Level
		0xFF, 0xFF, 0xFF, 0xFF, // Values: nil
		0b1, 5, 0, 0, 0, // Inner
		7, 0, 8, 0, // Codes
	)
	if got := must(small.AppendPacked(nil)); !bytes.Equal(got, want) {
		t.Fatalf("Small:\n got % x\nwant % x", got, want)
	}

	tree := Tree{Value: 1, Children: []Tree{{Value: 2}}}
	want = append(binary.LittleEndian.AppendUint64(nil, TreeSchema),
		1, 0, 0, 0, // Value
		1, 0, 0, 0, // one child
		2, 0, 0, 0, // its Value
		0xFF, 0xFF, 0xFF, 0xFF, // its Children: nil
	)
	if got := must(tree.AppendPacked(nil)); !bytes.Equal(got, want) {
		t.Fatalf("Tree:\n got % x\nwant % x", got, want)
	}
}

// TestSmallMatchesTheDSL encodes Small and the DSL's twin of it: the bit
// runs, integers, strings with a Uint32 length, slices, nested structs and
// arrays both can express are one encoding.
func TestSmallMatchesTheDSL(t *testing.T) {
	values := []Small{
		{},
		{Active: true, Visible: true, Count: 65535, Total: math.MinInt64, Name: "Pump · west", Ratio: -0.25, Level: 100, Values: []int32{1, -1, math.MaxInt32}, Inner: Inner{On: true, Value: 0xDEADBEEF}, Codes: [2]uint16{1, 65535}},
		{Visible: true, Name: "🚰", Values: []int32{}},
	}

	for _, value := range values {
		twin := dsl.Small{
			Active: value.Active, Visible: value.Visible, Count: value.Count, Total: value.Total,
			Name: value.Name, Ratio: value.Ratio, Level: value.Level, Values: value.Values,
			Inner: dsl.Inner{On: value.Inner.On, Value: value.Inner.Value}, Codes: value.Codes,
		}
		if twin.Values == nil {
			// the DSL's slices have no nil; the dynamic layer writes it as 0xFFFFFFFF
			twin.Values = []int32{}
			value.Values = []int32{}
		}

		want := make([]byte, twin.Size())
		twin.ToBytes(want, 0)

		got := must(value.AppendPacked(nil))[8:]
		if !bytes.Equal(got, want) {
			t.Fatalf("%+v:\ndynamic % x\n    DSL % x", value, got, want)
		}
	}
}

// TestUnencodable writes a time Unix nanoseconds can't hold: an error, not
// a panic, so one bad date fails one message.
func TestUnencodable(t *testing.T) {
	for _, device := range []Device{
		{Created: time.Date(1600, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{Owner: &User{Since: time.Date(2300, time.January, 1, 0, 0, 0, 0, time.UTC)}},
	} {
		message, err := device.AppendPacked(nil)
		var encodeError *packed.EncodeError
		if !errors.As(err, &encodeError) || !strings.Contains(err.Error(), "outside the years 1678 to 2262") || message != nil {
			t.Fatalf("%v: %q, %v", device.Created, message, err)
		}
	}
}

func TestOutOfDate(t *testing.T) {
	message := must(sampleDevice().AppendPacked(nil))

	if _, err := DecodeSmall(message); !errors.Is(err, packed.ErrOutOfDate) {
		t.Fatalf("a Device read as a Small: %v, want ErrOutOfDate", err)
	}

	message[0] ^= 1
	if _, err := DecodeDevice(message); !errors.Is(err, packed.ErrOutOfDate) {
		t.Fatalf("another schema: %v, want ErrOutOfDate", err)
	}

	if SmallSchema == DeviceSchema || TreeSchema == EmptySchema {
		t.Fatal("two layouts share a schema")
	}
}

func TestRejects(t *testing.T) {
	message := must(samples[0].AppendPacked(nil))

	// every truncation fails, never panics, and gives the zero value
	for length := range len(message) {
		device, err := DecodeDevice(message[:length:length])
		if err == nil {
			t.Fatalf("decoded %d of the message's %d bytes", length, len(message))
		}
		if !reflect.DeepEqual(device, Device{}) {
			t.Fatalf("a failed decode returned %+v", device)
		}
	}

	if _, err := DecodeDevice(append(message, 0)); err == nil || !strings.Contains(err.Error(), "1 bytes after the message") {
		t.Fatalf("a trailing byte: %v", err)
	}

	tree := func(children ...byte) []byte {
		return append(binary.LittleEndian.AppendUint64(nil, TreeSchema), append([]byte{1, 0, 0, 0}, children...)...)
	}

	// a count whose elements can't fit is refused before allocating
	if _, err := DecodeTree(tree(0xFE, 0xFF, 0xFF, 0xFF)); err == nil || !strings.Contains(err.Error(), "the count 4294967294") {
		t.Fatalf("a huge count: %v", err)
	}

	// as deep as the limit, and one deeper
	deep := func(depth int) Tree {
		root := Tree{}
		node := &root
		for range depth - 1 {
			node.Children = []Tree{{}}
			node = &node.Children[0]
		}
		return root
	}
	if _, err := DecodeTree(must(deep(packed.DefaultLimits.Depth).AppendPacked(nil))); err != nil {
		t.Fatalf("a tree as deep as the limit: %v", err)
	}
	if _, err := DecodeTree(must(deep(packed.DefaultLimits.Depth + 1).AppendPacked(nil))); err == nil || !strings.Contains(err.Error(), "nests deeper") {
		t.Fatalf("a tree deeper than the limit: %v", err)
	}

	// a map's keys out of order, and twice
	scores := func(keys ...string) []byte {
		value := Device{Scores: map[string]int64{"a": 1, "b": 2}}
		message := must(value.AppendPacked(nil))
		a := packed.AppendString(nil, "a")
		b := packed.AppendString(nil, "b")
		index := bytes.Index(message, a)
		entries := [][]byte{}
		for _, key := range keys {
			entries = append(entries, append(packed.AppendString(nil, key), make([]byte, 8)...))
		}
		rest := message[index+2*(len(a)+8):]
		if !bytes.Equal(message[index+len(a)+8:index+len(a)+8+len(b)], b) {
			t.Fatal("the scores aren't where the test looks")
		}
		return append(append(message[:index:index], bytes.Join(entries, nil)...), rest...)
	}
	if _, err := DecodeDevice(scores("a", "b")); err != nil {
		t.Fatalf("the scores in order: %v", err)
	}
	if _, err := DecodeDevice(scores("b", "a")); err == nil || !strings.Contains(err.Error(), "does not come after") {
		t.Fatalf("the scores out of order: %v", err)
	}
	if _, err := DecodeDevice(scores("a", "a")); err == nil || !strings.Contains(err.Error(), "does not come after") {
		t.Fatalf("a score twice: %v", err)
	}

	// fixed keys out of order
	lookup := func(first, second byte) []byte {
		value := Device{Lookup: map[Status]*Part{1: nil, 2: nil}}
		message := must(value.AppendPacked(nil))
		index := bytes.Index(message, []byte{2, 0, 0, 0, 1, 0, 2, 0})
		if index < 0 {
			t.Fatal("the lookup isn't where the test looks")
		}
		message[index+4], message[index+6] = first, second
		return message
	}
	if _, err := DecodeDevice(lookup(1, 2)); err != nil {
		t.Fatalf("the lookup as written: %v", err)
	}
	if _, err := DecodeDevice(lookup(2, 1)); err == nil || !strings.Contains(err.Error(), "does not come after") {
		t.Fatalf("the lookup out of order: %v", err)
	}
}

func TestSizeLimit(t *testing.T) {
	limits := packed.DefaultLimits
	defer func() { packed.DefaultLimits = limits }()

	message := must(Device{Name: strings.Repeat("x", 1000)}.AppendPacked(nil))

	packed.DefaultLimits.Size = len(message) - 1
	if _, err := DecodeDevice(message); err == nil || !strings.Contains(err.Error(), "longer than the limit") {
		t.Fatalf("a message longer than the limit: %v", err)
	}

	// memory: 100 empty strings take 4 bytes each in the message and 16 as
	// Go strings
	message = must(Device{Tags: make(Tags, 100)}.AppendPacked(nil))
	if len(message) >= 100*16 {
		t.Fatalf("the message is %d bytes", len(message))
	}
	packed.DefaultLimits.Size = 100*16 - 1
	if _, err := DecodeDevice(message); err == nil || !strings.Contains(err.Error(), "take more than the limit") {
		t.Fatalf("values larger than the limit: %v", err)
	}
	packed.DefaultLimits.Size = 100 * 16
	if _, err := DecodeDevice(message); err != nil {
		t.Fatalf("values as large as the limit: %v", err)
	}
}

func TestBuildErrors(t *testing.T) {
	_, err := generate.DynamicSource("testdata/invalid")
	if err == nil {
		t.Fatal("the invalid types generated")
	}
	message := err.Error()

	for _, want := range []string{
		"Invalid.Any: an interface can't travel",
		"Invalid.Events: a channel can't travel",
		"Invalid.Callback: a function can't travel",
		"Invalid.Ratios[key]: a key holding a float",
		"Invalid.Owners[key]: a key holding a pointer",
		"Invalid.Times[key]: a time.Time key",
		"Invalid.Keys[key]: the key's field hidden isn't encoded",
		"Invalid.Complex: a complex number has no layout",
		"Invalid.Address: an address means nothing",
		"Invalid.Tagged: the packed tag \"skip\" is unknown",
		"Invalid.Next: Invalid holds itself through this pointer",
		"Invalid.Nothing: its elements encode to no bytes",
		"Invalid.Total: the type's name count is a variable of the generated code",
		"Node.Children[]: Node holds itself through this pointer",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("the errors lack %q", want)
		}
	}
	if strings.Contains(message, "Fine") {
		t.Errorf("a field that travels has an error")
	}
	if !strings.Contains(message, filepath.Join("testdata", "invalid", "types.go")+":") {
		t.Errorf("the errors have no positions:\n%s", message)
	}
	if lines := strings.Count(message, "\n") + 1; lines != 14 {
		t.Errorf("%d errors, want 14:\n%s", lines, message)
	}
}
