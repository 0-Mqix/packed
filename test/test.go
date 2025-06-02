package test

type Custom struct{}

func (p *Custom) Size() int                                     { return 0 }
func (p *Custom) ToBytesLittleEndian(bytes []byte, index int)   {}
func (p *Custom) FromBytesLittleEndian(bytes []byte, index int) {}
func (p *Custom) ToBytesBigEndian(bytes []byte, index int)      {}
func (p *Custom) FromBytesBigEndian(bytes []byte, index int)    {}

type Monkey struct{}

type MonkeyEnum int

const (
	MonkeyEnumA MonkeyEnum = iota
	MonkeyEnumB
	MonkeyEnumC
)

func (c *Monkey) Size() int {
	return 0
}

func (c *Monkey) ToBytesLittleEndian(value *MonkeyEnum, bytes []byte, index int)      {}
func (c *Monkey) FromBytesLittleEndian(receiver *MonkeyEnum, bytes []byte, index int) {}
func (c *Monkey) ToBytesBigEndian(value *MonkeyEnum, bytes []byte, index int)         {}
func (c *Monkey) FromBytesBigEndian(receiver *MonkeyEnum, bytes []byte, index int)    {}
