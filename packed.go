package packed

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"sort"

	"golang.org/x/tools/imports"
)

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type converterHashField struct {
	reflect reflect.StructField
	hash    bool
	value   any
}

type converterHash struct {
	instance                    any
	reflection                  string
	hasInitializeConverterField bool
	fields                      map[string]converterHashField
	hash                        string
}

var (
	structs                 = map[string]packedStruct{}
	converters              = map[string]converterHash{}
	imported                = map[string]bool{}
	converterIdentifiers    = map[string]string{}
	converterLastIdentifier = 0
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

type InitializeConverterFieldInterface interface {
	InitializeConverterFields() map[string]string
}

type OverwriteConverterReflectionInterface interface {
	OverwriteConverterReflection(reflection reflect.Type) reflect.Type
}

type OverwriteConverterReciverReflectionInterface interface {
	OverwriteConverterReciverReflection(reflection reflect.Type) reflect.Type
}

func createConverterHash(converter any) converterHash {
	reflection := reflect.TypeOf(converter)
	value := reflect.ValueOf(converter)

	if reflection.Kind() == reflect.Ptr {
		reflection = reflection.Elem()
		value = value.Elem()
	}

	hash := reflection.String()

	result := converterHash{instance: converter, reflection: reflection.String(), fields: map[string]converterHashField{}}

	if _, ok := converter.(InitializeConverterFieldInterface); ok {
		result.hasInitializeConverterField = true
	}

	if _, ok := converter.(OverwriteConverterReflectionInterface); ok {
		result.reflection = converter.(OverwriteConverterReflectionInterface).OverwriteConverterReflection(reflection).String()
	}

	if reflection.Kind() == reflect.Struct {
		for i := 0; i < reflection.NumField(); i++ {
			field := reflection.Field(i)
			fieldValue := value.Field(i)

			if tagValue := field.Tag.Get("packed_hash_field"); tagValue != "" {
				hash += fmt.Sprintf("%s:%v", tagValue, fieldValue.Interface())
				result.fields[tagValue] = converterHashField{reflect: field, value: fieldValue.Interface(), hash: true}
			}
		}
	}

	result.hash = "_" + base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(hash))

	return result
}

func getConverterName(hash string) string {

	if len(hash) > 0 && hash[0] != '_' {
		panic("invalid hash")
	}

	if name, ok := converterIdentifiers[hash]; ok {
		return name
	}

	name := fmt.Sprintf("c%d", converterLastIdentifier)

	converterIdentifiers[hash] = name

	converterLastIdentifier++

	return name
}

func Load(structures ...packedStruct) {
	for _, structure := range structures {
		structs[structure.name] = structure
	}
}

type Property struct {
	Name string
	Type reflect.Type
}

type GenerateHook func(buffer *bytes.Buffer, structName string, properties []Property)

func (p *packedStruct) collectProperties() []Property {
	var result []Property

	for _, property := range p.properties {
		switch property.kind {

		case kindBitFieldGroup:
			group := property.packed.(packedBitFieldGroup)
			for _, field := range group.fields {
				var t reflect.Type
				switch field.bitFieldKind {
				case bitFieldKindBitsType, bitFieldKindBitsConverter:
					t = field.bitsTargetReflection.Elem()
				default:
					t = field.reflection
				}
				result = append(result, Property{Name: field.packedProperty.name, Type: t})
			}

		case kindStruct:
			result = append(result, Property{Name: property.name, Type: reflect.TypeFor[*packedStruct]()})

		case kindConverter:
			if overwrite, ok := property.packed.(OverwriteConverterReciverReflectionInterface); ok {
				result = append(result, Property{Name: property.name, Type: overwrite.OverwriteConverterReciverReflection(property.recieverType)})
			} else {
				result = append(result, Property{Name: property.name, Type: property.recieverType})
			}

		case kindConverterCast:
			cast := property.packed.(converterCast)
			result = append(result, Property{Name: property.name, Type: cast.target})

		case kindType:
			result = append(result, Property{Name: property.name, Type: property.propertyType.Elem()})

		case kindArray:
			continue
		}
	}

	return result
}

func Generate(outputFile string, packageName string, hooks ...GenerateHook) {

	buffer := &bytes.Buffer{}

	fmt.Fprintf(buffer, "// Code generated by github.com/0-mqix/packed; DO NOT EDIT.\n\n")
	fmt.Fprintf(buffer, "package %s\n\n", packageName)
	fmt.Fprintf(buffer, "import (\n")
	for importPath := range imported {
		fmt.Fprintf(buffer, "\"%s\"\n", importPath)
	}
	fmt.Fprintf(buffer, ")\n\n")

	fmt.Fprintf(buffer, "var (\n")

	for _, hash := range sortedKeys(converters) {
		converter := converters[hash]

		var initialize InitializeConverterFieldInterface

		if converter.hasInitializeConverterField {
			initialize = converter.instance.(InitializeConverterFieldInterface)
		}

		fmt.Fprintf(buffer, "// %s", converter.reflection)

		for _, name := range sortedKeys(converter.fields) {
			field := converter.fields[name]

			if !field.hash {
				continue
			}

			fmt.Fprintf(buffer, " %s: %v", name, field.value)
		}

		fmt.Fprintf(buffer, "\n%s = &%s", getConverterName(converter.hash), converter.reflection)

		if converter.hasInitializeConverterField {
			fmt.Fprintf(buffer, "{")

			fields := initialize.InitializeConverterFields()

			for _, name := range sortedKeys(fields) {
				fmt.Fprintf(buffer, " %s: %v,", name, fields[name])
			}

			fmt.Fprintf(buffer, "}")

		} else {
			fmt.Fprintf(buffer, "{}")
		}

		fmt.Fprintf(buffer, "\n")
	}

	fmt.Fprintf(buffer, ")\n")

	for _, name := range sortedKeys(structs) {
		packed := structs[name]
		buffer.Write(packed.structDefinition())
		fmt.Fprintf(buffer, "\n")
		buffer.Write(packed.sizeDefinition())
		fmt.Fprintf(buffer, "\n")
		buffer.Write(packed.conversionDefinition("ToBytes"))
		fmt.Fprintf(buffer, "\n")
		buffer.Write(packed.conversionDefinition("FromBytes"))
		fmt.Fprintf(buffer, "\n")

		if len(hooks) > 0 {
			properties := packed.collectProperties()
			for _, hook := range hooks {
				hook(buffer, packed.name, properties)
			}
		}
	}

	result, err := imports.Process("", buffer.Bytes(), &imports.Options{
		AllErrors:  true,
		FormatOnly: false,
		Comments:   true,
	})

	if err != nil {
		os.WriteFile(outputFile, buffer.Bytes(), 0644)
		panic("failed to generate code")
	}

	os.WriteFile(outputFile, result, 0644)
}
