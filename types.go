package packed

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

var (
	Boolean = BooleanConverter{}
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

type BooleanConverter struct{}

func (BooleanConverter) Size() int { return 1 }

func (BooleanConverter) ToBytesLittleEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (BooleanConverter) FromBytesLittleEndian(receiver *bool, bytes []byte, index int) {
	*receiver = bytes[index] > 0
}

func (BooleanConverter) ToBytesBigEndian(value *bool, bytes []byte, index int) {
	if *value {
		bytes[index] = 1
	} else {
		bytes[index] = 0
	}
}

func (BooleanConverter) FromBytesBigEndian(receiver *bool, bytes []byte, index int) {
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
	binary.LittleEndian.PutUint16(bytes[index:index+2], uint16(*value))
}

func (Int16Converter) FromBytesLittleEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(binary.LittleEndian.Uint16(bytes[index : index+2]))
}

func (Int16Converter) ToBytesBigEndian(value *int16, bytes []byte, index int) {
	binary.BigEndian.PutUint16(bytes[index:index+2], uint16(*value))
}

func (Int16Converter) FromBytesBigEndian(receiver *int16, bytes []byte, index int) {
	*receiver = int16(binary.BigEndian.Uint16(bytes[index : index+2]))
}

type Int32Converter struct{}

func (Int32Converter) Size() int { return 4 }

func (Int32Converter) ToBytesLittleEndian(value *int32, bytes []byte, index int) {
	binary.LittleEndian.PutUint32(bytes[index:index+4], uint32(*value))
}

func (Int32Converter) FromBytesLittleEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(binary.LittleEndian.Uint32(bytes[index : index+4]))
}

func (Int32Converter) ToBytesBigEndian(value *int32, bytes []byte, index int) {
	binary.BigEndian.PutUint32(bytes[index:index+4], uint32(*value))
}

func (Int32Converter) FromBytesBigEndian(receiver *int32, bytes []byte, index int) {
	*receiver = int32(binary.BigEndian.Uint32(bytes[index : index+4]))
}

type Int64Converter struct{}

func (Int64Converter) Size() int { return 8 }

func (Int64Converter) ToBytesLittleEndian(value *int64, bytes []byte, index int) {
	binary.LittleEndian.PutUint64(bytes[index:index+8], uint64(*value))
}

func (Int64Converter) FromBytesLittleEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(binary.LittleEndian.Uint64(bytes[index : index+8]))
}

func (Int64Converter) ToBytesBigEndian(value *int64, bytes []byte, index int) {
	binary.BigEndian.PutUint64(bytes[index:index+8], uint64(*value))
}

func (Int64Converter) FromBytesBigEndian(receiver *int64, bytes []byte, index int) {
	*receiver = int64(binary.BigEndian.Uint64(bytes[index : index+8]))
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
	binary.LittleEndian.PutUint16(bytes[index:index+2], *value)
}

func (Uint16Converter) FromBytesLittleEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = binary.LittleEndian.Uint16(bytes[index : index+2])
}

func (Uint16Converter) ToBytesBigEndian(value *uint16, bytes []byte, index int) {
	binary.BigEndian.PutUint16(bytes[index:index+2], *value)
}

func (Uint16Converter) FromBytesBigEndian(receiver *uint16, bytes []byte, index int) {
	*receiver = binary.BigEndian.Uint16(bytes[index : index+2])
}

type Uint32Converter struct{}

func (Uint32Converter) Size() int { return 4 }

func (Uint32Converter) ToBytesLittleEndian(value *uint32, bytes []byte, index int) {
	binary.LittleEndian.PutUint32(bytes[index:index+4], *value)
}

func (Uint32Converter) FromBytesLittleEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = binary.LittleEndian.Uint32(bytes[index : index+4])
}

func (Uint32Converter) ToBytesBigEndian(value *uint32, bytes []byte, index int) {
	binary.BigEndian.PutUint32(bytes[index:index+4], *value)
}

func (Uint32Converter) FromBytesBigEndian(receiver *uint32, bytes []byte, index int) {
	*receiver = binary.BigEndian.Uint32(bytes[index : index+4])
}

type Uint64Converter struct{}

func (Uint64Converter) Size() int { return 8 }

func (Uint64Converter) ToBytesLittleEndian(value *uint64, bytes []byte, index int) {
	binary.LittleEndian.PutUint64(bytes[index:index+8], *value)
}

func (Uint64Converter) FromBytesLittleEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = binary.LittleEndian.Uint64(bytes[index : index+8])
}

func (Uint64Converter) ToBytesBigEndian(value *uint64, bytes []byte, index int) {
	binary.BigEndian.PutUint64(bytes[index:index+8], *value)
}

func (Uint64Converter) FromBytesBigEndian(receiver *uint64, bytes []byte, index int) {
	*receiver = binary.BigEndian.Uint64(bytes[index : index+8])
}

type Float32Converter struct{}

func (Float32Converter) Size() int { return 4 }

func (Float32Converter) ToBytesLittleEndian(value *float32, bytes []byte, index int) {
	binary.LittleEndian.PutUint32(bytes[index:index+4], math.Float32bits(*value))
}

func (Float32Converter) FromBytesLittleEndian(receiver *float32, bytes []byte, index int) {
	*receiver = math.Float32frombits(binary.LittleEndian.Uint32(bytes[index : index+4]))
}

func (Float32Converter) ToBytesBigEndian(value *float32, bytes []byte, index int) {
	binary.BigEndian.PutUint32(bytes[index:index+4], math.Float32bits(*value))
}

func (Float32Converter) FromBytesBigEndian(receiver *float32, bytes []byte, index int) {
	*receiver = math.Float32frombits(binary.BigEndian.Uint32(bytes[index : index+4]))
}

type Float64Converter struct{}

func (Float64Converter) Size() int { return 8 }

func (Float64Converter) ToBytesLittleEndian(value *float64, bytes []byte, index int) {
	binary.LittleEndian.PutUint64(bytes[index:index+8], math.Float64bits(*value))
}

func (Float64Converter) FromBytesLittleEndian(receiver *float64, bytes []byte, index int) {
	*receiver = math.Float64frombits(binary.LittleEndian.Uint64(bytes[index : index+8]))
}

func (Float64Converter) ToBytesBigEndian(value *float64, bytes []byte, index int) {
	binary.BigEndian.PutUint64(bytes[index:index+8], math.Float64bits(*value))
}

func (Float64Converter) FromBytesBigEndian(receiver *float64, bytes []byte, index int) {
	*receiver = math.Float64frombits(binary.BigEndian.Uint64(bytes[index : index+8]))
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
