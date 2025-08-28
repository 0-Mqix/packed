package packed

import (
	"maps"
	"regexp"
	"testing"
)

var endianRegex = regexp.MustCompile(`(?m)(LittleEndian|BigEndian)\(&reciever\.([^,]+)`)

type EndianProperties map[string]bool

func getEndianProperties(source []byte) EndianProperties {

	matches := endianRegex.FindAllSubmatch(source, -1)

	properties := make(EndianProperties)

	for _, match := range matches {
		properties[string(match[2])] = string(match[1]) == "LittleEndian"
	}

	return properties
}

func TestEndianSettings(t *testing.T) {

	var a = Struct("A", true,
		Field("A", Int8),
		Field("B", Int16, LittleEndian(false)),
	)

	intput := EndianProperties{
		"A": true,
		"B": false,
	}

	output := getEndianProperties(a.conversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of a not equal")
	}

	var b = Struct("D", false,
		Field("A", a, LittleEndian(false)),
		Field("B", a, LittleEndian(true)),
		Field("C", a),
	)

	intput = EndianProperties{
		"A.A": false,
		"A.B": false,
		"B.A": true,
		"B.B": true,
		"C.A": false,
		"C.B": false,
	}

	output = getEndianProperties(b.conversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of b not equal")
	}

	var c = Struct("C", true,
		Field("A", Uint16, LittleEndian(false)),
		Field("B", Uint16),
		Field("C", b, LittleEndian(false)),
		Field("D", b, LittleEndian(true)),
		Field("E", a),
	)

	intput = EndianProperties{
		"A":     false,
		"B":     true,
		"C.A.A": false,
		"C.A.B": false,
		"C.B.A": false,
		"C.B.B": false,
		"C.C.A": false,
		"C.C.B": false,
		"D.A.A": true,
		"D.A.B": true,
		"D.B.A": true,
		"D.B.B": true,
		"D.C.A": true,
		"D.C.B": true,
		"E.A":   true,
		"E.B":   false,
	}

	output = getEndianProperties(c.conversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of c not equal")
	}
}
