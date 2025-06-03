package packed

import (
	"packed"
)

var ()

type A struct {
	A int8
	B int16
}

func (reciever *A) Size() int {
	return 0
}

func (reciever *A) ToBytes(bytes []byte, index int) {
	packed.Int8Converter.ToBytesLittleEndian(&reciever.A, bytes, index+0)
	packed.Int16Converter.ToBytesBigEndian(&reciever.B, bytes, index+1)
}

func (reciever *A) FromBytes(bytes []byte, index int) {
	packed.Int8Converter.FromBytesLittleEndian(&reciever.A, bytes, index+0)
	packed.Int16Converter.FromBytesBigEndian(&reciever.B, bytes, index+1)
}

type D struct {
	A A
	B A
	C A
}

func (reciever *D) Size() int {
	return 0
}

func (reciever *D) ToBytes(bytes []byte, index int) {
	packed.Int8Converter.ToBytesBigEndian(&reciever.A.A, bytes, index+0)
	packed.Int16Converter.ToBytesBigEndian(&reciever.A.B, bytes, index+1)
	packed.Int8Converter.ToBytesLittleEndian(&reciever.B.A, bytes, index+3)
	packed.Int16Converter.ToBytesLittleEndian(&reciever.B.B, bytes, index+4)
	packed.Int8Converter.ToBytesBigEndian(&reciever.C.A, bytes, index+6)
	packed.Int16Converter.ToBytesBigEndian(&reciever.C.B, bytes, index+7)
}

func (reciever *D) FromBytes(bytes []byte, index int) {
	packed.Int8Converter.FromBytesBigEndian(&reciever.A.A, bytes, index+0)
	packed.Int16Converter.FromBytesBigEndian(&reciever.A.B, bytes, index+1)
	packed.Int8Converter.FromBytesLittleEndian(&reciever.B.A, bytes, index+3)
	packed.Int16Converter.FromBytesLittleEndian(&reciever.B.B, bytes, index+4)
	packed.Int8Converter.FromBytesBigEndian(&reciever.C.A, bytes, index+6)
	packed.Int16Converter.FromBytesBigEndian(&reciever.C.B, bytes, index+7)
}

type C struct {
	A uint16
	B uint16
	C D
	D D
	E A
}

func (reciever *C) Size() int {
	return 0
}

func (reciever *C) ToBytes(bytes []byte, index int) {
	packed.Uint16Converter.ToBytesBigEndian(&reciever.A, bytes, index+0)
	packed.Uint16Converter.ToBytesLittleEndian(&reciever.B, bytes, index+2)
	packed.Int8Converter.ToBytesBigEndian(&reciever.C.A.A, bytes, index+4)
	packed.Int16Converter.ToBytesBigEndian(&reciever.C.A.B, bytes, index+5)
	packed.Int8Converter.ToBytesBigEndian(&reciever.C.B.A, bytes, index+7)
	packed.Int16Converter.ToBytesBigEndian(&reciever.C.B.B, bytes, index+8)
	packed.Int8Converter.ToBytesBigEndian(&reciever.C.C.A, bytes, index+10)
	packed.Int16Converter.ToBytesBigEndian(&reciever.C.C.B, bytes, index+11)
	packed.Int8Converter.ToBytesLittleEndian(&reciever.D.A.A, bytes, index+13)
	packed.Int16Converter.ToBytesLittleEndian(&reciever.D.A.B, bytes, index+14)
	packed.Int8Converter.ToBytesLittleEndian(&reciever.D.B.A, bytes, index+16)
	packed.Int16Converter.ToBytesLittleEndian(&reciever.D.B.B, bytes, index+17)
	packed.Int8Converter.ToBytesLittleEndian(&reciever.D.C.A, bytes, index+19)
	packed.Int16Converter.ToBytesLittleEndian(&reciever.D.C.B, bytes, index+20)
	packed.Int8Converter.ToBytesLittleEndian(&reciever.E.A, bytes, index+22)
	packed.Int16Converter.ToBytesBigEndian(&reciever.E.B, bytes, index+23)
}

func (reciever *C) FromBytes(bytes []byte, index int) {
	packed.Uint16Converter.FromBytesBigEndian(&reciever.A, bytes, index+0)
	packed.Uint16Converter.FromBytesLittleEndian(&reciever.B, bytes, index+2)
	packed.Int8Converter.FromBytesBigEndian(&reciever.C.A.A, bytes, index+4)
	packed.Int16Converter.FromBytesBigEndian(&reciever.C.A.B, bytes, index+5)
	packed.Int8Converter.FromBytesBigEndian(&reciever.C.B.A, bytes, index+7)
	packed.Int16Converter.FromBytesBigEndian(&reciever.C.B.B, bytes, index+8)
	packed.Int8Converter.FromBytesBigEndian(&reciever.C.C.A, bytes, index+10)
	packed.Int16Converter.FromBytesBigEndian(&reciever.C.C.B, bytes, index+11)
	packed.Int8Converter.FromBytesLittleEndian(&reciever.D.A.A, bytes, index+13)
	packed.Int16Converter.FromBytesLittleEndian(&reciever.D.A.B, bytes, index+14)
	packed.Int8Converter.FromBytesLittleEndian(&reciever.D.B.A, bytes, index+16)
	packed.Int16Converter.FromBytesLittleEndian(&reciever.D.B.B, bytes, index+17)
	packed.Int8Converter.FromBytesLittleEndian(&reciever.D.C.A, bytes, index+19)
	packed.Int16Converter.FromBytesLittleEndian(&reciever.D.C.B, bytes, index+20)
	packed.Int8Converter.FromBytesLittleEndian(&reciever.E.A, bytes, index+22)
	packed.Int16Converter.FromBytesBigEndian(&reciever.E.B, bytes, index+23)
}
