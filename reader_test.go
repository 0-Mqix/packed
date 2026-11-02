package packed

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func message(schema uint64, body ...byte) []byte {
	return append(AppendSchema(nil, schema), body...)
}

func TestReaderSchema(t *testing.T) {
	if reader := NewReader(message(7), 7); reader.Finish() != nil {
		t.Fatalf("its own schema: %v", reader.Err())
	}
	if err := NewReader(message(8), 7).Err(); !errors.Is(err, ErrOutOfDate) {
		t.Fatalf("another schema: %v, want ErrOutOfDate", err)
	}
	if err := NewReader([]byte{7, 0, 0}, 7).Err(); err == nil || errors.Is(err, ErrOutOfDate) {
		t.Fatalf("a message shorter than its schema: %v, want truncated", err)
	}
}

func TestReaderAfterFailure(t *testing.T) {
	reader := NewReader(message(1, 5, 0, 0, 0, 'a'), 1)

	if value := reader.ReadString(); value != "" || reader.Err() == nil {
		t.Fatalf("a string longer than the message: %q, %v", value, reader.Err())
	}
	first := reader.Err()

	// every read after the first failure is a zero value, and keeps it
	if _, _, ok := reader.Take(1); ok {
		t.Fatal("Take after a failure")
	}
	if count, present := reader.Count(1, 1); count != 0 || present {
		t.Fatal("Count after a failure")
	}
	if reader.Present() || reader.Allocate(1) || reader.Enter() || reader.Bytes() != nil {
		t.Fatal("a read after a failure")
	}
	if reader.Finish() != first {
		t.Fatalf("Finish is %v, want the first failure %v", reader.Finish(), first)
	}
}

func TestReaderCount(t *testing.T) {
	nilCount := message(1, 0xFF, 0xFF, 0xFF, 0xFF)
	if count, present := NewReader(nilCount, 1).Count(1, 1); count != 0 || present {
		t.Fatalf("0xFFFFFFFF is %d, %v, want nil", count, present)
	}

	// three elements of at least 2 bytes need 6
	for left, fits := range map[int]bool{5: false, 6: true} {
		reader := NewReader(message(1, append([]byte{3, 0, 0, 0}, make([]byte, left)...)...), 1)
		count, present := reader.Count(2, 0)
		if fits != (present && count == 3 && reader.Err() == nil) {
			t.Errorf("3 elements of 2 bytes in %d: %d, %v, %v", left, count, present, reader.Err())
		}
	}

	// a minimum of 0 still needs a byte each
	reader := NewReader(message(1, 9, 0, 0, 0), 1)
	if _, present := reader.Count(0, 0); present {
		t.Fatal("9 elements of no bytes in none")
	}
}

func TestReaderBytes(t *testing.T) {
	for _, value := range [][]byte{nil, {}, {1, 2, 3}} {
		encoded := AppendBytes(AppendSchema(nil, 1), value)
		reader := NewReader(encoded, 1)
		decoded := reader.Bytes()
		if reader.Finish() != nil || !bytes.Equal(decoded, value) || (decoded == nil) != (value == nil) {
			t.Fatalf("%v decoded to %v, %v", value, decoded, reader.Err())
		}
	}
}

func TestReaderDepth(t *testing.T) {
	limits := DefaultLimits
	defer func() { DefaultLimits = limits }()
	DefaultLimits.Depth = 2

	reader := NewReader(message(1), 1)
	if !reader.Enter() || !reader.Enter() {
		t.Fatal("two levels")
	}
	if reader.Enter() || !strings.Contains(reader.Err().Error(), "deeper than 2") {
		t.Fatalf("three levels: %v", reader.Err())
	}
}

func TestReaderKeys(t *testing.T) {
	keys := AppendString(AppendString(AppendString(nil, "a"), "b"), "b")
	reader := NewReader(message(1, keys...), 1)

	var previous []byte
	for index, want := range []bool{true, true, false} {
		start := reader.Position()
		reader.ReadString()
		reader.Ascending(&previous, start)
		if (reader.Err() == nil) != want {
			t.Fatalf("key %d: %v", index, reader.Err())
		}
	}

	reader = NewReader(message(1), 1)
	if reader.Unique(2, 2); reader.Err() != nil {
		t.Fatal(reader.Err())
	}
	if reader.Unique(1, 2); reader.Err() == nil {
		t.Fatal("a map of 2 entries and 1 key")
	}
}

func TestSortedKeys(t *testing.T) {
	entries := map[string]int{"bb": 1, "a": 2, "c": 3, "": 4}
	var keys []string
	for _, entry := range SortedKeys(entries, AppendString) {
		keys = append(keys, entry.Key)
		if !bytes.Equal(entry.Bytes, AppendString(nil, entry.Key)) {
			t.Fatalf("%q has the bytes % x", entry.Key, entry.Bytes)
		}
	}
	// by the encoded bytes: the length first
	if want := []string{"", "a", "c", "bb"}; strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("sorted %q, want %q", keys, want)
	}
}

func TestTimeConverter(t *testing.T) {
	converter := TimeConverter{}
	buffer := make([]byte, 8)

	for _, value := range []time.Time{
		{},
		time.Unix(0, 0).UTC(),
		time.Date(2026, time.October, 25, 12, 0, 0, 1, time.UTC),
		time.Unix(0, math.MinInt64+1).UTC(),
		time.Unix(0, math.MaxInt64).UTC(),
	} {
		converter.ToBytesLittleEndian(&value, buffer, 0)
		var decoded time.Time
		converter.FromBytesLittleEndian(&decoded, buffer, 0)
		if decoded != value {
			t.Errorf("%v decoded to %v", value, decoded)
		}
	}

	// another location reads back as the same instant in UTC
	amsterdam := time.FixedZone("CET", 3600)
	local := time.Date(2026, time.March, 1, 13, 0, 0, 0, amsterdam)
	converter.ToBytesLittleEndian(&local, buffer, 0)
	var decoded time.Time
	converter.FromBytesLittleEndian(&decoded, buffer, 0)
	if !decoded.Equal(local) || decoded.Location() != time.UTC {
		t.Errorf("%v decoded to %v", local, decoded)
	}

	defer func() {
		if recover() == nil {
			t.Error("a time before 1678 encoded")
		}
	}()
	early := time.Date(1600, time.January, 1, 0, 0, 0, 0, time.UTC)
	converter.ToBytesLittleEndian(&early, buffer, 0)
}

func TestAppendCount(t *testing.T) {
	if got := AppendCount(nil, 0, true); !bytes.Equal(got, []byte{0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Fatalf("nil is % x", got)
	}
	if got := AppendCount(nil, 258, false); !bytes.Equal(got, []byte{2, 1, 0, 0}) {
		t.Fatalf("258 is % x", got)
	}
	defer func() {
		if recover() == nil {
			t.Error("a count of 0xFFFFFFFF, nil's, was written")
		}
	}()
	AppendCount(nil, math.MaxUint32, false)
}
