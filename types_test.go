package packed

import (
	"math"
	"testing"
)

func TestBoolConverter(t *testing.T) {
	converter := BoolConverter{}
	values := []bool{true, false}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian bool
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Bool FromBytesLittleEndian: expected %v, got %v", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian bool
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Bool FromBytesBigEndian: expected %v, got %v", original, resultBigEndian)
		}
	}

	bytes := []byte{2}
	var result bool
	converter.FromBytesLittleEndian(&result, bytes, 0)
	if !result {
		t.Errorf("Bool FromBytesLittleEndian: expected true for byte value > 1, got false")
	}
	converter.FromBytesBigEndian(&result, bytes, 0)
	if !result {
		t.Errorf("Bool FromBytesBigEndian: expected true for byte value > 1, got false")
	}
}

func TestInt8Converter(t *testing.T) {
	converter := Int8Converter{}
	values := []int8{0, 1, -1, 127, -128}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian int8
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Int8 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian int8
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Int8 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestInt16Converter(t *testing.T) {
	converter := Int16Converter{}
	values := []int16{0, 1, -1, 255, -255, 32767, -32768}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian int16
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Int16 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian int16
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Int16 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestInt32Converter(t *testing.T) {
	converter := Int32Converter{}
	values := []int32{0, 1, -1, 65535, -65535, 2147483647, -2147483648}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian int32
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Int32 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian int32
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Int32 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestInt64Converter(t *testing.T) {
	converter := Int64Converter{}
	values := []int64{0, 1, -1, 4294967295, -4294967295, 9223372036854775807, -9223372036854775808}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian int64
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Int64 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian int64
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Int64 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestUint8Converter(t *testing.T) {
	converter := Uint8Converter{}
	values := []uint8{0, 1, 127, 128, 255}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian uint8
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Uint8 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian uint8
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Uint8 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestUint16Converter(t *testing.T) {
	converter := Uint16Converter{}
	values := []uint16{0, 1, 255, 256, 32767, 32768, 65535}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian uint16
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Uint16 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian uint16
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Uint16 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestUint32Converter(t *testing.T) {
	converter := Uint32Converter{}
	values := []uint32{0, 1, 65535, 65536, 2147483647, 2147483648, 4294967295}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian uint32
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Uint32 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian uint32
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Uint32 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestUint64Converter(t *testing.T) {
	converter := Uint64Converter{}
	values := []uint64{0, 1, 4294967295, 4294967296, 9223372036854775807, 9223372036854775808, 18446744073709551615}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian uint64
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Uint64 FromBytesLittleEndian: expected %d, got %d", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian uint64
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Uint64 FromBytesBigEndian: expected %d, got %d", original, resultBigEndian)
		}
	}
}

func TestFloat32Converter(t *testing.T) {
	converter := Float32Converter{}
	values := []float32{0.0, 1.0, -1.0, 3.14159, -3.14159, math.MaxFloat32, -math.MaxFloat32, math.SmallestNonzeroFloat32}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian float32
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Float32 FromBytesLittleEndian: expected %f, got %f", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian float32
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Float32 FromBytesBigEndian: expected %f, got %f", original, resultBigEndian)
		}
	}
}

func TestFloat64Converter(t *testing.T) {
	converter := Float64Converter{}
	values := []float64{0.0, 1.0, -1.0, 3.141592653589793, -3.141592653589793, math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64}

	for _, original := range values {
		bytes := make([]byte, converter.Size())

		converter.ToBytesLittleEndian(&original, bytes, 0)
		var resultLittleEndian float64
		converter.FromBytesLittleEndian(&resultLittleEndian, bytes, 0)
		if resultLittleEndian != original {
			t.Errorf("Float64 FromBytesLittleEndian: expected %f, got %f", original, resultLittleEndian)
		}

		converter.ToBytesBigEndian(&original, bytes, 0)
		var resultBigEndian float64
		converter.FromBytesBigEndian(&resultBigEndian, bytes, 0)
		if resultBigEndian != original {
			t.Errorf("Float64 FromBytesBigEndian: expected %f, got %f", original, resultBigEndian)
		}
	}
}

func TestStringConverter(t *testing.T) {
	converter := String(1)
	original := "A"

	bytes := make([]byte, converter.Size())

	converter.ToBytesLittleEndian(&original, bytes, 0)
	var result string
	converter.FromBytesLittleEndian(&result, bytes, 0)

	if result != original {
		t.Errorf("StringConverter: expected %s, got %s", original, result)
	}
}
