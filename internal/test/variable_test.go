package packed

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/0-Mqix/packed/internal/test/types"
)

func sampleRecord() *Record {
	return &Record{
		ID:       0x01020304,
		Name:     "pump · west",
		Severity: SeverityHigh,
		Flags:    RecordFlags{Active: true, Level: 5},
		Labels: []Label{
			{Name: "a", Weight: 1.5},
			{Name: "", Weight: -2},
		},
		Values:     []int16{-1, 300},
		Trailer:    0xBEEF,
		Pair:       [2]string{"x", "yz"},
		Nested:     [][]uint8{{1, 2}, {}, {3}},
		Severities: []Severity{SeverityLow, SeverityMedium},
		Codes:      [2]types.ExampleEnum{1, -2},
	}
}

func TestRecordWireLayout(t *testing.T) {
	record := &Record{Name: "ab", Severity: SeverityMedium, Flags: RecordFlags{Active: true, Level: 2}, Trailer: 0x0102}
	got := make([]byte, record.Size())
	end := record.ToBytes(got, 0)

	want := []byte{
		0, 0, 0, 0, // ID
		2, 0, 'a', 'b', // Name
		1,       // Severity
		0b00101, // Flags: Active in bit 0, Level in bits 1-3
		0,       // Labels length
		0, 0,    // Values length
		1, 2, // Trailer, big-endian
		0, 0, // Pair: two empty strings
		0,    // Nested length
		0,    // Severities length
		0, 0, // Codes
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("ToBytes = %v\n   want %v", got, want)
	}
	if end != len(want) {
		t.Fatalf("ToBytes returned %d, want %d", end, len(want))
	}
}

func TestRecordRoundTrip(t *testing.T) {
	record := sampleRecord()

	buffer := make([]byte, 3+record.Size()+5)
	end := record.ToBytes(buffer, 3)
	if end != 3+record.Size() {
		t.Fatalf("ToBytes returned %d, want %d", end, 3+record.Size())
	}

	var back Record
	if read := back.FromBytes(buffer, 3); read != end {
		t.Fatalf("FromBytes returned %d, want %d", read, end)
	}

	if !reflect.DeepEqual(*record, back) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", back, *record)
	}
}

func TestFixedStructReturnsEnd(t *testing.T) {
	ping := Ping{Sequence: 9}
	buffer := make([]byte, 10)
	if end := ping.ToBytes(buffer, 2); end != 6 {
		t.Fatalf("ToBytes returned %d, want 6", end)
	}
	if !bytes.Equal(buffer[2:6], []byte{0, 0, 0, 9}) {
		t.Fatalf("Ping is big-endian, got %v", buffer[2:6])
	}
}

func TestEnvelopeRoundTrip(t *testing.T) {
	for _, message := range []Envelope{sampleRecord(), &Ping{Sequence: 42}} {
		encoded := EncodeEnvelope(message)

		if encoded[0] != message.EnvelopeType() {
			t.Fatalf("type byte %d, want %d", encoded[0], message.EnvelopeType())
		}

		decoded, err := DecodeEnvelope(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(decoded, message) {
			t.Fatalf("decoded %+v, want %+v", decoded, message)
		}
	}
}

func TestEnvelopeRejectsBadInput(t *testing.T) {
	encoded := EncodeEnvelope(sampleRecord())

	cases := map[string][]byte{
		"empty":     {},
		"unknown":   {200, 1, 2},
		"truncated": encoded[:len(encoded)-3],
		"trailing":  append(append([]byte{}, encoded...), 0),
	}

	for name, input := range cases {
		if _, err := DecodeEnvelope(input); err == nil {
			t.Errorf("%s: DecodeEnvelope accepted %v", name, input)
		}
	}
}

func TestLengthPrefixOverflowPanics(t *testing.T) {
	label := Label{Name: strings.Repeat("x", 256)}

	defer func() {
		if recover() == nil {
			t.Fatal("ToBytes accepted a 256 byte name with a uint8 length prefix")
		}
	}()

	label.ToBytes(make([]byte, 300), 0)
}

func TestEnumString(t *testing.T) {
	if SeverityHigh.String() != "High" || Severity(7).String() != "Severity(7)" {
		t.Fatalf("String() = %q, %q", SeverityHigh.String(), Severity(7).String())
	}
}
