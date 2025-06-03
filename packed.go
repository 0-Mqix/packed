package packed

import (
	"bytes"
	"fmt"
	"os"
	"reflect"

	"golang.org/x/tools/imports"
)

type converter struct {
	variable string
	external bool
}

var (
	structs    = map[string]PackedStruct{}
	converters = map[reflect.Type]converter{}
	imported   = map[string]bool{"packed": true}
)

type TypeInterface interface {
	Size() int
	ToBytesLittleEndian(bytes []byte, index int)
	FromBytesLittleEndian(bytes []byte, index int)
	ToBytesBigEndian(bytes []byte, index int)
	FromBytesBigEndian(bytes []byte, index int)
}

type ConverterInterface[Reciever any] interface {
	Size() int
	ToBytesLittleEndian(reciever *Reciever, bytes []byte, index int)
	FromBytesLittleEndian(reciever *Reciever, bytes []byte, index int)
	ToBytesBigEndian(reciever *Reciever, bytes []byte, index int)
	FromBytesBigEndian(reciever *Reciever, bytes []byte, index int)
}

func RegisterConverter[T any](variable string) {
	converters[reflect.TypeOf(new(T))] = converter{variable: variable, external: true}
}

func Generate() {
	buffer := bytes.Buffer{}

	buffer.WriteString("package packed\n\n")
	buffer.WriteString("import (\n")
	for importPath := range imported {
		buffer.WriteString(fmt.Sprintf("\"%s\"\n", importPath))
	}
	buffer.WriteString(")\n\n")

	buffer.WriteString("var (\n")
	for reflection, variable := range converters {

		if variable.external {
			continue
		}

		buffer.WriteString(fmt.Sprintf("%s = &%s{}\n", variable.variable, reflection.Elem().String()))

	}
	buffer.WriteString(")\n")

	for _, packed := range structs {
		buffer.Write(packed.StructDefinition())
		buffer.WriteString("\n")
		buffer.Write(packed.SizeDefinition())
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("ToBytes"))
		buffer.WriteString("\n")
		buffer.Write(packed.ConversionDefinition("FromBytes"))
		buffer.WriteString("\n")
	}

	result, err := imports.Process("", buffer.Bytes(), &imports.Options{
		AllErrors:  true,
		FormatOnly: false,
	})

	os.MkdirAll("./generated/packed", 0755)

	if err != nil {
		os.WriteFile("./generated/packed/packed.go", buffer.Bytes(), 0644)
		fmt.Println(err)
		return
	}

	os.WriteFile("./generated/packed/packed.go", result, 0644)
}
