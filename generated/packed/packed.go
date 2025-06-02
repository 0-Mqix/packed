package packed

import (
	"packed"
	"packed/test"
)

var (
	MonkeyConverter = &test.Monkey{}
	Int8Converter   = &packed.Int8{}
)

type A struct {
	A int8
	B int8
	C int8
	D int16
}

func (reciever *A) ToBytesLittleEndian(bytes []byte, index int) {
	Int8Converter.ToBytesLittleEndian(&reciever.A, bytes, index+0)
	Int8Converter.ToBytesLittleEndian(&reciever.B, bytes, index+1)
	Int8Converter.ToBytesLittleEndian(&reciever.C, bytes, index+2)
	packed.Int16Converter.ToBytesLittleEndian(&reciever.D, bytes, index+3)
}

func (reciever *A) FromBytesLittleEndian(bytes []byte, index int) {
	Int8Converter.FromBytesLittleEndian(&reciever.A, bytes, index+0)
	Int8Converter.FromBytesLittleEndian(&reciever.B, bytes, index+1)
	Int8Converter.FromBytesLittleEndian(&reciever.C, bytes, index+2)
	packed.Int16Converter.FromBytesLittleEndian(&reciever.D, bytes, index+3)
}

func (reciever *A) ToBytesBigEndian(bytes []byte, index int) {
	Int8Converter.ToBytesBigEndian(&reciever.A, bytes, index+0)
	Int8Converter.ToBytesBigEndian(&reciever.B, bytes, index+1)
	Int8Converter.ToBytesBigEndian(&reciever.C, bytes, index+2)
	packed.Int16Converter.ToBytesBigEndian(&reciever.D, bytes, index+3)
}

func (reciever *A) FromBytesBigEndian(bytes []byte, index int) {
	Int8Converter.FromBytesBigEndian(&reciever.A, bytes, index+0)
	Int8Converter.FromBytesBigEndian(&reciever.B, bytes, index+1)
	Int8Converter.FromBytesBigEndian(&reciever.C, bytes, index+2)
	packed.Int16Converter.FromBytesBigEndian(&reciever.D, bytes, index+3)
}

type D struct {
	A         int8
	B         int8
	Converter test.MonkeyEnum
	Custom    test.Custom
}

func (reciever *D) ToBytesLittleEndian(bytes []byte, index int) {
	Int8Converter.ToBytesLittleEndian(&reciever.A, bytes, index+0)
	Int8Converter.ToBytesLittleEndian(&reciever.B, bytes, index+1)
	MonkeyConverter.ToBytesLittleEndian(&reciever.Converter, bytes, index+2)
	reciever.Custom.ToBytesLittleEndian(bytes, index+2)
}

func (reciever *D) FromBytesLittleEndian(bytes []byte, index int) {
	Int8Converter.FromBytesLittleEndian(&reciever.A, bytes, index+0)
	Int8Converter.FromBytesLittleEndian(&reciever.B, bytes, index+1)
	MonkeyConverter.FromBytesLittleEndian(&reciever.Converter, bytes, index+2)
	reciever.Custom.FromBytesLittleEndian(bytes, index+2)
}

func (reciever *D) ToBytesBigEndian(bytes []byte, index int) {
	Int8Converter.ToBytesBigEndian(&reciever.A, bytes, index+0)
	Int8Converter.ToBytesBigEndian(&reciever.B, bytes, index+1)
	MonkeyConverter.ToBytesBigEndian(&reciever.Converter, bytes, index+2)
	reciever.Custom.ToBytesBigEndian(bytes, index+2)
}

func (reciever *D) FromBytesBigEndian(bytes []byte, index int) {
	Int8Converter.FromBytesBigEndian(&reciever.A, bytes, index+0)
	Int8Converter.FromBytesBigEndian(&reciever.B, bytes, index+1)
	MonkeyConverter.FromBytesBigEndian(&reciever.Converter, bytes, index+2)
	reciever.Custom.FromBytesBigEndian(bytes, index+2)
}

type B struct {
	A A
	B int8 `json:"b"`
	C D
}

func (reciever *B) ToBytesLittleEndian(bytes []byte, index int) {
	reciever.A.ToBytesLittleEndian(bytes, index+0)
	Int8Converter.ToBytesLittleEndian(&reciever.B, bytes, index+5)
	reciever.C.ToBytesLittleEndian(bytes, index+6)
}

func (reciever *B) FromBytesLittleEndian(bytes []byte, index int) {
	reciever.A.FromBytesLittleEndian(bytes, index+0)
	Int8Converter.FromBytesLittleEndian(&reciever.B, bytes, index+5)
	reciever.C.FromBytesLittleEndian(bytes, index+6)
}

func (reciever *B) ToBytesBigEndian(bytes []byte, index int) {
	reciever.A.ToBytesBigEndian(bytes, index+0)
	Int8Converter.ToBytesBigEndian(&reciever.B, bytes, index+5)
	reciever.C.ToBytesBigEndian(bytes, index+6)
}

func (reciever *B) FromBytesBigEndian(bytes []byte, index int) {
	reciever.A.FromBytesBigEndian(bytes, index+0)
	Int8Converter.FromBytesBigEndian(&reciever.B, bytes, index+5)
	reciever.C.FromBytesBigEndian(bytes, index+6)
}
