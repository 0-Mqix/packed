package types

type ExampleEnum int8

const (
	ExampleEnumValueA ExampleEnum = iota + 1
	ExampleEnumValueB
	ExampleEnumValueC
)

type ExampleTypeInterface struct {
	A int
}

func (o *ExampleTypeInterface) Size() int {
	return 1
}

func (o *ExampleTypeInterface) ToBytesLittleEndian(bytes []byte, index int) {
	bytes[index] = byte(o.A)
}

func (o *ExampleTypeInterface) FromBytesLittleEndian(bytes []byte, index int) {
	o.A = int(bytes[index])
}

func (o *ExampleTypeInterface) ToBytesBigEndian(bytes []byte, index int) {
	bytes[index] = byte(o.A)
}

func (o *ExampleTypeInterface) FromBytesBigEndian(bytes []byte, index int) {
	o.A = int(bytes[index])
}

type ExampleRecieverType struct {
	A int
}

type ExampleConverter struct{}

func (o *ExampleConverter) Size() int {
	return 1
}

func (o *ExampleConverter) ToBytesLittleEndian(reciever *ExampleRecieverType, bytes []byte, index int) {
	bytes[index] = byte(reciever.A)
}

func (o *ExampleConverter) FromBytesLittleEndian(reciever *ExampleRecieverType, bytes []byte, index int) {
	reciever.A = int(bytes[index])
}

func (o *ExampleConverter) ToBytesBigEndian(reciever *ExampleRecieverType, bytes []byte, index int) {
	bytes[index] = byte(reciever.A)
}

func (o *ExampleConverter) FromBytesBigEndian(reciever *ExampleRecieverType, bytes []byte, index int) {
	reciever.A = int(bytes[index])
}
