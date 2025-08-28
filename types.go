package packed

import (
	"fmt"
	"math"
	"strings"
)

var (
	Bool    = BoolConverter{}
	Int8    = Int8Converter{}
	Int16   = Int16Converter{}
	Int32   = Int32Converter{}
	Int64   = Int64Converter{}
	Uint8   = Uint8Converter{}
	Uint16  = Uint16Converter{}
	Uint32  = Uint32Converter{}
	Uint64  = Uint64Converter{}
	Float32 = Float32Converter{}
	Float64 = Float64Converter{}
)

type BoolConverter struct{}

func (BoolConverter) Size() int { return 1 }

func (BoolConverter) ToBytesLittleEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (BoolConverter) FromBytesLittleEndian(receiver *bool, bytes []byte, index int) {
	*receiver = bytes[index] > 0
}

func (BoolConverter) ToBytesBigEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (BoolConverter) FromBytesBigEndian(receiver *bool, bytes []byte, index int) {
	*receiver = bytes[index] > 0
}

type Int8Converter struct{}

func (Int8Converter) Size() int { return 1 }

func (Int8Converter) ToBytesLittleEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (Int8Converter) FromBytesLittleEndian(receiver *int8, bytes []byte, index int) {
	*receiver = int8(bytes[index])
}

func (Int8Converter) ToBytesBigEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (Int8Converter) FromBytesBigEndian(receiver *int8, bytes []byte, index int) {
	*receiver = int8(bytes[index])
}

type Int16Converter struct{}

func (Int16Converter) Size() int { return 2 }

func (Int16Converter) ToBytesLittleEndian(value *int16, bytes []byte, index int) {
	v := uint16(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
}

func (Int16Converter) FromBytesLittleEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(uint16(bytes[index]) | uint16(bytes[index+1])<<8)
}

func (Int16Converter) ToBytesBigEndian(value *int16, bytes []byte, index int) {
	v := uint16(*value)
	bytes[index] = byte(v >> 8)
	bytes[index+1] = byte(v)
}

func (Int16Converter) FromBytesBigEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(uint16(bytes[index])<<8 | uint16(bytes[index+1]))
}

type Int32Converter struct{}

func (Int32Converter) Size() int { return 4 }

func (Int32Converter) ToBytesLittleEndian(value *int32, bytes []byte, index int) {
	v := uint32(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
}

func (Int32Converter) FromBytesLittleEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24)
}

func (Int32Converter) ToBytesBigEndian(value *int32, bytes []byte, index int) {
	v := uint32(*value)
	bytes[index] = byte(v >> 24)
	bytes[index+1] = byte(v >> 16)
	bytes[index+2] = byte(v >> 8)
	bytes[index+3] = byte(v)
}

func (Int32Converter) FromBytesBigEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3]))
}

type Int64Converter struct{}

func (Int64Converter) Size() int { return 8 }

func (Int64Converter) ToBytesLittleEndian(value *int64, bytes []byte, index int) {
	v := uint64(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
	bytes[index+4] = byte(v >> 32)
	bytes[index+5] = byte(v >> 40)
	bytes[index+6] = byte(v >> 48)
	bytes[index+7] = byte(v >> 56)
}

func (Int64Converter) FromBytesLittleEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56)
}

func (Int64Converter) ToBytesBigEndian(value *int64, bytes []byte, index int) {
	v := uint64(*value)
	bytes[index] = byte(v >> 56)
	bytes[index+1] = byte(v >> 48)
	bytes[index+2] = byte(v >> 40)
	bytes[index+3] = byte(v >> 32)
	bytes[index+4] = byte(v >> 24)
	bytes[index+5] = byte(v >> 16)
	bytes[index+6] = byte(v >> 8)
	bytes[index+7] = byte(v)
}

func (Int64Converter) FromBytesBigEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7]))
}

type Uint8Converter struct{}

func (Uint8Converter) Size() int { return 1 }

func (Uint8Converter) ToBytesLittleEndian(value *uint8, bytes []byte, index int) {
	bytes[index] = *value
}

func (Uint8Converter) FromBytesLittleEndian(receiver *uint8, bytes []byte, index int) {
	*receiver = bytes[index]
}

func (Uint8Converter) ToBytesBigEndian(value *uint8, bytes []byte, index int) {
	bytes[index] = *value
}

func (Uint8Converter) FromBytesBigEndian(receiver *uint8, bytes []byte, index int) {
	*receiver = bytes[index]
}

type Uint16Converter struct{}

func (Uint16Converter) Size() int { return 2 }

func (Uint16Converter) ToBytesLittleEndian(value *uint16, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
}

func (Uint16Converter) FromBytesLittleEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = uint16(bytes[index]) | uint16(bytes[index+1])<<8
}

func (Uint16Converter) ToBytesBigEndian(value *uint16, bytes []byte, index int) {
	bytes[index] = byte(*value >> 8)
	bytes[index+1] = byte(*value)
}

func (Uint16Converter) FromBytesBigEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = uint16(bytes[index])<<8 | uint16(bytes[index+1])
}

type Uint32Converter struct{}

func (Uint32Converter) Size() int { return 4 }

func (Uint32Converter) ToBytesLittleEndian(value *uint32, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
	bytes[index+2] = byte(*value >> 16)
	bytes[index+3] = byte(*value >> 24)
}

func (Uint32Converter) FromBytesLittleEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24
}

func (Uint32Converter) ToBytesBigEndian(value *uint32, bytes []byte, index int) {
	bytes[index] = byte(*value >> 24)
	bytes[index+1] = byte(*value >> 16)
	bytes[index+2] = byte(*value >> 8)
	bytes[index+3] = byte(*value)
}

func (Uint32Converter) FromBytesBigEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3])
}

type Uint64Converter struct{}

func (Uint64Converter) Size() int { return 8 }

func (Uint64Converter) ToBytesLittleEndian(value *uint64, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
	bytes[index+2] = byte(*value >> 16)
	bytes[index+3] = byte(*value >> 24)
	bytes[index+4] = byte(*value >> 32)
	bytes[index+5] = byte(*value >> 40)
	bytes[index+6] = byte(*value >> 48)
	bytes[index+7] = byte(*value >> 56)
}

func (Uint64Converter) FromBytesLittleEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56
}

func (Uint64Converter) ToBytesBigEndian(value *uint64, bytes []byte, index int) {
	bytes[index] = byte(*value >> 56)
	bytes[index+1] = byte(*value >> 48)
	bytes[index+2] = byte(*value >> 40)
	bytes[index+3] = byte(*value >> 32)
	bytes[index+4] = byte(*value >> 24)
	bytes[index+5] = byte(*value >> 16)
	bytes[index+6] = byte(*value >> 8)
	bytes[index+7] = byte(*value)
}

func (Uint64Converter) FromBytesBigEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7])
}

type Float32Converter struct{}

func (Float32Converter) Size() int { return 4 }

func (Float32Converter) ToBytesLittleEndian(value *float32, bytes []byte, index int) {
	v := math.Float32bits(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
}

func (Float32Converter) FromBytesLittleEndian(receiver *float32, bytes []byte, index int) {
	v := uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24
	*receiver = math.Float32frombits(v)
}

func (Float32Converter) ToBytesBigEndian(value *float32, bytes []byte, index int) {
	v := math.Float32bits(*value)
	bytes[index] = byte(v >> 24)
	bytes[index+1] = byte(v >> 16)
	bytes[index+2] = byte(v >> 8)
	bytes[index+3] = byte(v)
}

func (Float32Converter) FromBytesBigEndian(receiver *float32, bytes []byte, index int) {
	v := uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3])
	*receiver = math.Float32frombits(v)
}

type Float64Converter struct{}

func (Float64Converter) Size() int { return 8 }

func (Float64Converter) ToBytesLittleEndian(value *float64, bytes []byte, index int) {
	v := math.Float64bits(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
	bytes[index+4] = byte(v >> 32)
	bytes[index+5] = byte(v >> 40)
	bytes[index+6] = byte(v >> 48)
	bytes[index+7] = byte(v >> 56)
}

func (Float64Converter) FromBytesLittleEndian(receiver *float64, bytes []byte, index int) {
	v := uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56
	*receiver = math.Float64frombits(v)
}

func (Float64Converter) ToBytesBigEndian(value *float64, bytes []byte, index int) {
	v := math.Float64bits(*value)
	bytes[index] = byte(v >> 56)
	bytes[index+1] = byte(v >> 48)
	bytes[index+2] = byte(v >> 40)
	bytes[index+3] = byte(v >> 32)
	bytes[index+4] = byte(v >> 24)
	bytes[index+5] = byte(v >> 16)
	bytes[index+6] = byte(v >> 8)
	bytes[index+7] = byte(v)
}

func (Float64Converter) FromBytesBigEndian(receiver *float64, bytes []byte, index int) {
	v := uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7])
	*receiver = math.Float64frombits(v)
}

func String(length int) StringConverter {
	return StringConverter{Length: length}
}

type StringConverter struct {
	Length int `packed_hash_field:"length"`
}

func (s *StringConverter) InitializeConverterFields() map[string]string {
	return map[string]string{
		"Length": fmt.Sprintf("%v", s.Length),
	}
}

func (s *StringConverter) Size() int { return s.Length }

func (s *StringConverter) ToBytesLittleEndian(value *string, bytes []byte, index int) {
	copy(bytes[index:index+s.Length], []byte(*value))
}

func (s *StringConverter) FromBytesLittleEndian(receiver *string, bytes []byte, index int) {
	*receiver = strings.TrimRight(string(bytes[index:index+s.Length]), "\x00")
}

func (s *StringConverter) ToBytesBigEndian(value *string, bytes []byte, index int) {
	copy(bytes[index:index+s.Length], []byte(*value))
}

func (s *StringConverter) FromBytesBigEndian(receiver *string, bytes []byte, index int) {
	*receiver = strings.TrimRight(string(bytes[index:index+s.Length]), "\x00")
}
