package packed

import (
	"math"
)

var BoolConverter = &Bool{}
var Int8Converter = &Int8{}
var Int16Converter = &Int16{}
var Int32Converter = &Int32{}
var Int64Converter = &Int64{}
var Uint8Converter = &Uint8{}
var Uint16Converter = &Uint16{}
var Uint32Converter = &Uint32{}
var Uint64Converter = &Uint64{}
var Float32Converter = &Float32{}
var Float64Converter = &Float64{}

func init() {
	RegisterConverter[Bool]("BoolConverter")
	RegisterConverter[Int8]("Int8Converter")
	RegisterConverter[Int16]("Int16Converter")
	RegisterConverter[Int32]("Int32Converter")
	RegisterConverter[Int64]("Int64Converter")
	RegisterConverter[Uint8]("Uint8Converter")
	RegisterConverter[Uint16]("Uint16Converter")
	RegisterConverter[Uint32]("Uint32Converter")
	RegisterConverter[Uint64]("Uint64Converter")
	RegisterConverter[Float32]("Float32Converter")
	RegisterConverter[Float64]("Float64Converter")
}

type Bool struct{}

func (Bool) Size() int { return 1 }

func (Bool) ToBytesLittleEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (Bool) FromBytesLittleEndian(receiver *bool, bytes []byte, index int) {
	*receiver = bytes[index] > 0
}

func (Bool) ToBytesBigEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (Bool) FromBytesBigEndian(receiver *bool, bytes []byte, index int) {
	*receiver = bytes[index] > 0
}

type Int8 struct{}

func (Int8) Size() int { return 1 }

func (Int8) ToBytesLittleEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (Int8) FromBytesLittleEndian(receiver *int8, bytes []byte, index int) {
	*receiver = int8(bytes[index])
}

func (Int8) ToBytesBigEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (Int8) FromBytesBigEndian(receiver *int8, bytes []byte, index int) {
	*receiver = int8(bytes[index])
}

type Int16 struct{}

func (Int16) Size() int { return 2 }

func (Int16) ToBytesLittleEndian(value *int16, bytes []byte, index int) {
	v := uint16(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
}

func (Int16) FromBytesLittleEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(uint16(bytes[index]) | uint16(bytes[index+1])<<8)
}

func (Int16) ToBytesBigEndian(value *int16, bytes []byte, index int) {
	v := uint16(*value)
	bytes[index] = byte(v >> 8)
	bytes[index+1] = byte(v)
}

func (Int16) FromBytesBigEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(uint16(bytes[index])<<8 | uint16(bytes[index+1]))
}

type Int32 struct{}

func (Int32) Size() int { return 4 }

func (Int32) ToBytesLittleEndian(value *int32, bytes []byte, index int) {
	v := uint32(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
}

func (Int32) FromBytesLittleEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24)
}

func (Int32) ToBytesBigEndian(value *int32, bytes []byte, index int) {
	v := uint32(*value)
	bytes[index] = byte(v >> 24)
	bytes[index+1] = byte(v >> 16)
	bytes[index+2] = byte(v >> 8)
	bytes[index+3] = byte(v)
}

func (Int32) FromBytesBigEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3]))
}

type Int64 struct{}

func (Int64) Size() int { return 8 }

func (Int64) ToBytesLittleEndian(value *int64, bytes []byte, index int) {
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

func (Int64) FromBytesLittleEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56)
}

func (Int64) ToBytesBigEndian(value *int64, bytes []byte, index int) {
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

func (Int64) FromBytesBigEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7]))
}

type Uint8 struct{}

func (Uint8) Size() int { return 1 }

func (Uint8) ToBytesLittleEndian(value *uint8, bytes []byte, index int) {
	bytes[index] = *value
}

func (Uint8) FromBytesLittleEndian(receiver *uint8, bytes []byte, index int) {
	*receiver = bytes[index]
}

func (Uint8) ToBytesBigEndian(value *uint8, bytes []byte, index int) {
	bytes[index] = *value
}

func (Uint8) FromBytesBigEndian(receiver *uint8, bytes []byte, index int) {
	*receiver = bytes[index]
}

type Uint16 struct{}

func (Uint16) Size() int { return 2 }

func (Uint16) ToBytesLittleEndian(value *uint16, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
}

func (Uint16) FromBytesLittleEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = uint16(bytes[index]) | uint16(bytes[index+1])<<8
}

func (Uint16) ToBytesBigEndian(value *uint16, bytes []byte, index int) {
	bytes[index] = byte(*value >> 8)
	bytes[index+1] = byte(*value)
}

func (Uint16) FromBytesBigEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = uint16(bytes[index])<<8 | uint16(bytes[index+1])
}

type Uint32 struct{}

func (Uint32) Size() int { return 4 }

func (Uint32) ToBytesLittleEndian(value *uint32, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
	bytes[index+2] = byte(*value >> 16)
	bytes[index+3] = byte(*value >> 24)
}

func (Uint32) FromBytesLittleEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24
}

func (Uint32) ToBytesBigEndian(value *uint32, bytes []byte, index int) {
	bytes[index] = byte(*value >> 24)
	bytes[index+1] = byte(*value >> 16)
	bytes[index+2] = byte(*value >> 8)
	bytes[index+3] = byte(*value)
}

func (Uint32) FromBytesBigEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3])
}

type Uint64 struct{}

func (Uint64) Size() int { return 8 }

func (Uint64) ToBytesLittleEndian(value *uint64, bytes []byte, index int) {
	bytes[index] = byte(*value)
	bytes[index+1] = byte(*value >> 8)
	bytes[index+2] = byte(*value >> 16)
	bytes[index+3] = byte(*value >> 24)
	bytes[index+4] = byte(*value >> 32)
	bytes[index+5] = byte(*value >> 40)
	bytes[index+6] = byte(*value >> 48)
	bytes[index+7] = byte(*value >> 56)
}

func (Uint64) FromBytesLittleEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56
}

func (Uint64) ToBytesBigEndian(value *uint64, bytes []byte, index int) {
	bytes[index] = byte(*value >> 56)
	bytes[index+1] = byte(*value >> 48)
	bytes[index+2] = byte(*value >> 40)
	bytes[index+3] = byte(*value >> 32)
	bytes[index+4] = byte(*value >> 24)
	bytes[index+5] = byte(*value >> 16)
	bytes[index+6] = byte(*value >> 8)
	bytes[index+7] = byte(*value)
}

func (Uint64) FromBytesBigEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7])
}

type Float32 struct{}

func (Float32) Size() int { return 4 }

func (Float32) ToBytesLittleEndian(value *float32, bytes []byte, index int) {
	v := math.Float32bits(*value)
	bytes[index] = byte(v)
	bytes[index+1] = byte(v >> 8)
	bytes[index+2] = byte(v >> 16)
	bytes[index+3] = byte(v >> 24)
}

func (Float32) FromBytesLittleEndian(receiver *float32, bytes []byte, index int) {
	v := uint32(bytes[index]) | uint32(bytes[index+1])<<8 | uint32(bytes[index+2])<<16 | uint32(bytes[index+3])<<24
	*receiver = math.Float32frombits(v)
}

func (Float32) ToBytesBigEndian(value *float32, bytes []byte, index int) {
	v := math.Float32bits(*value)
	bytes[index] = byte(v >> 24)
	bytes[index+1] = byte(v >> 16)
	bytes[index+2] = byte(v >> 8)
	bytes[index+3] = byte(v)
}

func (Float32) FromBytesBigEndian(receiver *float32, bytes []byte, index int) {
	v := uint32(bytes[index])<<24 | uint32(bytes[index+1])<<16 | uint32(bytes[index+2])<<8 | uint32(bytes[index+3])
	*receiver = math.Float32frombits(v)
}

type Float64 struct{}

func (Float64) Size() int { return 8 }

func (Float64) ToBytesLittleEndian(value *float64, bytes []byte, index int) {
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

func (Float64) FromBytesLittleEndian(receiver *float64, bytes []byte, index int) {
	v := uint64(bytes[index]) | uint64(bytes[index+1])<<8 | uint64(bytes[index+2])<<16 | uint64(bytes[index+3])<<24 | uint64(bytes[index+4])<<32 | uint64(bytes[index+5])<<40 | uint64(bytes[index+6])<<48 | uint64(bytes[index+7])<<56
	*receiver = math.Float64frombits(v)
}

func (Float64) ToBytesBigEndian(value *float64, bytes []byte, index int) {
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

func (Float64) FromBytesBigEndian(receiver *float64, bytes []byte, index int) {
	v := uint64(bytes[index])<<56 | uint64(bytes[index+1])<<48 | uint64(bytes[index+2])<<40 | uint64(bytes[index+3])<<32 | uint64(bytes[index+4])<<24 | uint64(bytes[index+5])<<16 | uint64(bytes[index+6])<<8 | uint64(bytes[index+7])
	*receiver = math.Float64frombits(v)
}
