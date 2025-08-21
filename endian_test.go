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
		Field[Int8]("A"),
		Field[Int16]("B", LittleEndian(false)),
	)

	intput := EndianProperties{
		"A": true,
		"B": false,
	}

	output := getEndianProperties(a.ConversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of a not equal")
	}

	var b = Struct("D", false,
		Field[any]("A", Type(a), LittleEndian(false)),
		Field[any]("B", Type(a), LittleEndian(true)),
		Field[any]("C", Type(a)),
	)

	intput = EndianProperties{
		"A.A": false,
		"A.B": false,
		"B.A": true,
		"B.B": true,
		"C.A": false,
		"C.B": false,
	}

	output = getEndianProperties(b.ConversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of b not equal")
	}

	var c = Struct("C", true,
		Field[Uint16]("A", LittleEndian(false)),
		Field[Uint16]("B"),
		Field[any]("C", Type(b), LittleEndian(false)),
		Field[any]("D", Type(b), LittleEndian(true)),
		Field[any]("E", Type(a)),
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

	output = getEndianProperties(c.ConversionDefinition("ToBytes"))

	if !maps.Equal(intput, output) {
		t.Fatal("endian properties of c not equal")
	}
}
