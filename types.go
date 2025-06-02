package packed

type Int8 struct{}
type Int16 struct{}
type Int32 struct{}
type Int64 struct{}
type Uint8 struct{}
type Uint16 struct{}
type Uint32 struct{}
type Uint64 struct{}
type Float32 struct{}
type Float64 struct{}

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
	RegisterConverter[Int16]("Int16Converter")
}

func (t *Int8) Size() int {
	return 1
}

func (t *Int8) ToBytesLittleEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (t *Int8) FromBytesLittleEndian(reciever *int8, bytes []byte, index int) {
	*reciever = int8(bytes[index])
}

func (t *Int8) ToBytesBigEndian(value *int8, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (t *Int8) FromBytesBigEndian(reciever *int8, bytes []byte, index int) {
	*reciever = int8(bytes[index])
}

func (t *Int16) Size() int {
	return 2
}

func (t *Int16) ToBytesLittleEndian(value *int16, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (t *Int16) FromBytesLittleEndian(reciever *int16, bytes []byte, index int) {
	*reciever = int16(bytes[index])
}

func (t *Int16) ToBytesBigEndian(value *int16, bytes []byte, index int) {
	bytes[index] = byte(*value)
}

func (t *Int16) FromBytesBigEndian(reciever *int16, bytes []byte, index int) {
	*reciever = int16(bytes[index])
}
