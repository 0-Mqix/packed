package packed

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// The header generator: one header that C11 and C++20 both include, for
// hosts and microcontrollers alike (an ATtiny with avr-gcc), so no heap and
// nothing beyond <stdint.h>, <stddef.h>, <stdbool.h> and <string.h>. The
// layout and the code are C: a fixed struct is a struct
// __attribute__((packed)) whose bit fields are C bit fields, so its sizeof
// is its Size, and every struct gets <Struct>Size, <Struct>ToBytes and
// <Struct>FromBytes. An enum is its wire type with a constant per value
// (LevelWarning); a message set a struct of its type and a union of its
// variants. A variable-length field is native C: text, bytes and a list
// whose elements are in the input as they are in memory are a pointer and
// a count, which decoding points into the input; any other list, with a
// Maximum, is an array of that many and a count, which decoding copies
// into. C++ gets the same structs with methods and enum classes on top,
// under #ifdef __cplusplus.

var headerKeywords = map[string]bool{
	"alignas": true, "alignof": true, "and": true, "asm": true, "auto": true, "bool": true,
	"break": true, "case": true, "catch": true, "char": true, "class": true, "concept": true,
	"const": true, "consteval": true, "constexpr": true, "constinit": true, "continue": true,
	"default": true, "delete": true, "do": true, "double": true, "else": true, "enum": true,
	"explicit": true, "export": true, "extern": true, "false": true, "float": true, "for": true,
	"friend": true, "goto": true, "if": true, "inline": true, "int": true, "long": true,
	"mutable": true, "namespace": true, "new": true, "noexcept": true, "not": true, "nullptr": true,
	"operator": true, "or": true, "private": true, "protected": true, "public": true,
	"register": true, "requires": true, "restrict": true, "return": true, "short": true,
	"signed": true, "sizeof": true, "static": true, "static_assert": true, "struct": true,
	"switch": true, "template": true, "this": true, "throw": true, "true": true, "try": true,
	"typedef": true, "typename": true, "typeof": true, "union": true, "unsigned": true,
	"using": true, "virtual": true, "void": true, "volatile": true, "while": true, "xor": true,
}

// headerStructMembers are the names a struct's C++ members and its tag
// take: a field of the same name gets an underscore, as a keyword does.
var headerStructMembers = map[string]bool{
	"size": true, "toBytes": true, "fromBytes": true, "view": true, "fixedSize": true, "tag": true,
	"unused": true,
}

// headerSetMembers are the names a message set's members take.
var headerSetMembers = map[string]bool{
	"type": true, "size": true, "encode": true, "decode": true, "Type": true,
}

// headerName turns a Go name into a member name.
func headerName(name string, reserved map[string]bool) string {
	result := camelName(name)
	if headerKeywords[result] || reserved[result] {
		return result + "_"
	}
	return result
}

// headerPrimitiveType is the C type of a built-in converter's value.
func headerPrimitiveType(primitive kotlinPrimitive) string {
	switch primitive.method {
	case "boolean":
		return "bool"
	case "float32":
		return "float"
	case "float64":
		return "double"
	}
	return primitive.method + "_t"
}

// headerUnsigned is the narrowest unsigned type that holds size bytes.
func headerUnsigned(size int) string {
	switch {
	case size <= 1:
		return "uint8_t"
	case size <= 2:
		return "uint16_t"
	case size <= 4:
		return "uint32_t"
	}
	return "uint64_t"
}

// headerCountType is the narrowest unsigned type that counts to maximum.
func headerCountType(maximum int) string {
	switch {
	case maximum <= 0xFF:
		return "uint8_t"
	case maximum <= 0xFFFF:
		return "uint16_t"
	}
	return "uint32_t"
}

// headerHex is an unsigned literal that fits the value.
func headerHex(value uint64) string {
	if value > 0xFFFFFFFF {
		return fmt.Sprintf("0x%Xull", value)
	}
	return fmt.Sprintf("0x%Xu", value)
}

func headerEndian(littleEndian bool) string {
	if littleEndian {
		return "true"
	}
	return "false"
}

// headerPut writes value as size bytes.
func headerPut(size int, value string, littleEndian bool) string {
	switch {
	case size == 1:
		return fmt.Sprintf("PackedWriterPut8(writer, (uint8_t)(%s));", value)
	case size == 2:
		return fmt.Sprintf("PackedWriterPut16(writer, (uint16_t)(%s), %s);", value, headerEndian(littleEndian))
	case size <= 4:
		return fmt.Sprintf("PackedWriterPut32(writer, (uint32_t)(%s), %d, %s);", value, size, headerEndian(littleEndian))
	}
	return fmt.Sprintf("PackedWriterPut64(writer, (uint64_t)(%s), %d, %s);", value, size, headerEndian(littleEndian))
}

// headerGet reads size bytes as the narrowest unsigned type that holds them.
func headerGet(size int, littleEndian bool) string {
	switch {
	case size == 1:
		return "PackedReaderGet8(reader)"
	case size == 2:
		return fmt.Sprintf("PackedReaderGet16(reader, %s)", headerEndian(littleEndian))
	case size <= 4:
		return fmt.Sprintf("PackedReaderGet32(reader, %d, %s)", size, headerEndian(littleEndian))
	}
	return fmt.Sprintf("PackedReaderGet64(reader, %d, %s)", size, headerEndian(littleEndian))
}

// headerMember is a struct's member: value's in its functions, or one of
// prefix, a nested struct or an element.
func headerMember(prefix, name string) string {
	if prefix == "" {
		return "value->" + name
	}
	return prefix + "." + name
}

// headerDirective writes a preprocessor line at the start of the line,
// wherever in a struct it is.
func headerDirective(w *sourceWriter, text string) {
	w.buffer.WriteString(text + "\n")
}

func headerConverter(kind kind, packed any) any {
	if kind == kindConverterCast {
		return packed.(converterCast).converter.instance
	}
	return packed
}

// headerVariable is a variable-length value as C holds it: items, a pointer
// or an array, and count, how many there are.
type headerVariable struct {
	items string
	count string
}

// fieldVariable is a variable-length field's pair of members: name and
// name<Length> for text and bytes, name and name<Count> for a list.
func fieldVariable(prefix string, property packedProperty) headerVariable {
	suffix := "Count"
	if property.kind == kindVariableString {
		suffix = "Length"
	}
	return headerVariable{
		items: headerMember(prefix, headerName(property.name, headerStructMembers)),
		count: headerMember(prefix, headerName(property.name+suffix, headerStructMembers)),
	}
}

// elementVariable is a variable-length element's members: a PackedText's
// text and length, a PackedBytes' bytes and length.
func elementVariable(kind kind, packed any, element string) headerVariable {
	if kind == kindVariableString && !packed.(packedVariableString).Bytes {
		return headerVariable{items: element + ".text", count: element + ".length"}
	}
	return headerVariable{items: element + ".bytes", count: element + ".length"}
}

// headerByteElements reports whether a list's elements are single bytes
// that are any value of their type.
func headerByteElements(kind kind, element any) bool {
	if kind != kindConverter {
		return false
	}
	primitive, ok := converterPrimitive(element, "", "C")
	return ok && (primitive.method == "uint8" || primitive.method == "int8")
}

// headerInPlace reports whether a list's elements are in the input as they
// are in memory, so a decoded list points at them: bytes, and fixed structs
// that are little endian throughout (on a little-endian machine), whose
// may_alias lets C++ read them there. Wider numbers are not: they need not
// be aligned. Nor are enums: C++ may not read bytes as an enum class.
func headerInPlace(kind kind, element any) bool {
	switch kind {
	case kindConverter:
		return headerByteElements(kind, element)
	case kindStruct:
		structure := element.(packedStruct)
		return !structure.dynamic && structure.size > 0 && headerLittleEndianLayout(structure)
	}
	return false
}

// headerHasEnums reports whether a value holds an enum, whose value a
// decoded list pointing at it still has to check.
func headerHasEnums(kind kind, packed any) bool {
	switch kind {
	case kindEnum:
		return true
	case kindStruct:
		for _, property := range packed.(packedStruct).properties {
			if property.kind == kindBitFieldGroup {
				for _, field := range property.packed.(packedBitFieldGroup).fields {
					if field.bitFieldKind == bitFieldKindEnum {
						return true
					}
				}
				continue
			}
			if headerHasEnums(property.kind, property.packed) {
				return true
			}
		}
	case kindArray:
		array := packed.(packedArray)
		return headerHasEnums(array.ElementKind, array.Element)
	}
	return false
}

// headerLittleEndianLayout reports whether a fixed struct's memory on a
// little endian machine is its wire layout: every multi-byte value and bit
// run little endian, and no Bool byte, which could hold a value a bool
// can't.
func headerLittleEndianLayout(structure packedStruct) bool {
	for _, property := range structure.properties {
		if !headerLittleEndianValue(property.kind, property.packed, property.littleEndian) {
			return false
		}
	}
	return true
}

func headerLittleEndianValue(kind kind, packed any, littleEndian bool) bool {
	switch kind {
	case kindBitFieldGroup:
		return littleEndian
	case kindStruct:
		return headerLittleEndianLayout(packed.(packedStruct))
	case kindArray:
		array := packed.(packedArray)
		return headerLittleEndianValue(array.ElementKind, array.Element, littleEndian)
	case kindEnum:
		return littleEndian || packed.(packedEnum).size == 1
	case kindConverter, kindConverterCast:
		primitive, ok := converterPrimitive(headerConverter(kind, packed), "", "C")
		if !ok {
			return true
		}
		return primitive.method != "boolean" && (littleEndian || primitive.size == 1)
	}
	return false
}

type headerGenerator struct {
	// double is set when a field is a Float64, which needs a 64-bit double.
	double bool
}

// HeaderSource renders every registered enum, struct and message set as one
// header for C11 and C++20 with the same byte layout as the Go output.
func HeaderSource() []byte {
	generator := &headerGenerator{}
	body := &sourceWriter{buffer: &bytes.Buffer{}}

	for _, name := range sortedKeys(enums) {
		enum := enums[name]
		values := make([]headerEnumValue, len(enum.values))
		for index, value := range enum.values {
			values[index] = headerEnumValue{value, uint64(index)}
		}
		generator.enum(body, enum.name, headerPrimitiveType(kotlinIntegers[enum.reflection.Kind()]), values)
		body.line("")
	}

	for _, name := range headerOrder() {
		generator.structure(body, structs[name])
		body.line("")
	}

	for _, name := range sortedKeys(messageSets) {
		generator.messageSet(body, messageSets[name])
		body.line("")
	}

	result := &bytes.Buffer{}
	fmt.Fprintf(result, "// Code generated by github.com/0-Mqix/packed; DO NOT EDIT.\n\n")
	result.WriteString("// C11 and C++20, no heap. GCC on x86 notes that packed bit fields moved in\n")
	result.WriteString("// GCC 4.4; the layout is C's __packed__ one either way, and\n")
	result.WriteString("// -Wno-packed-bitfield-compat silences the note.\n\n")
	result.WriteString("#pragma once\n\n")
	result.WriteString("#include <stdbool.h>\n")
	result.WriteString("#include <stddef.h>\n")
	result.WriteString("#include <stdint.h>\n")
	result.WriteString("#include <string.h>\n\n")
	result.WriteString(headerRuntime)
	result.WriteString("\n")
	if generator.double {
		result.WriteString("PACKED_STATIC_ASSERT(sizeof(double) == 8, \"packed: a Float64 field needs a 64-bit double; with avr-gcc, build with -mdouble=64\");\n\n")
	}
	result.Write(bytes.TrimRight(body.buffer.Bytes(), "\n"))

	return result.Bytes()
}

// headerOrder is every struct's name, each after the structs its fields
// hold, as C needs a type declared before a field holds it.
func headerOrder() []string {
	visited := map[string]bool{}
	order := []string{}

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		for _, property := range structs[name].properties {
			for _, dependency := range headerDependencies(property.kind, property.packed) {
				visit(dependency)
			}
		}
		order = append(order, name)
	}

	for _, name := range sortedKeys(structs) {
		visit(name)
	}
	return order
}

func headerDependencies(kind kind, packed any) []string {
	switch kind {
	case kindStruct:
		return []string{packed.(packedStruct).name}
	case kindArray:
		array := packed.(packedArray)
		return headerDependencies(array.ElementKind, array.Element)
	case kindSlice:
		slice := packed.(packedSlice)
		return headerDependencies(slice.ElementKind, slice.Element)
	}
	return nil
}

type headerEnumValue struct {
	name  string
	value uint64
}

// enum declares a C++ enum class, and for C the wire type and a constant
// per value, which C++ gets too: LevelWarning is Level::Warning.
func (g *headerGenerator) enum(w *sourceWriter, name string, wire string, values []headerEnumValue) {
	w.line("#ifdef __cplusplus")
	w.open("enum class %s : %s {", name, wire)
	for _, value := range values {
		enumerator := value.name
		if headerKeywords[enumerator] {
			enumerator += "_"
		}
		w.line("%s = %d,", enumerator, value.value)
	}
	w.close("};")
	for _, value := range values {
		enumerator := value.name
		if headerKeywords[enumerator] {
			enumerator += "_"
		}
		w.line("constexpr %s %s%s = %s::%s;", name, name, value.name, name, enumerator)
	}
	w.line("#else")
	w.line("typedef %s %s;", wire, name)
	w.open("enum {")
	for _, value := range values {
		w.line("%s%s = %d,", name, value.name, value.value)
	}
	w.close("};")
	w.line("#endif")
}

// elementType is the C type of an element of an array or a list.
func (g *headerGenerator) elementType(kind kind, packed any, location string) string {
	switch kind {
	case kindStruct:
		return packed.(packedStruct).name
	case kindEnum:
		return packed.(packedEnum).name
	case kindVariableString:
		if packed.(packedVariableString).Bytes {
			return "PackedBytes"
		}
		return "PackedText"
	case kindConverter, kindConverterCast:
		if primitive, ok := converterPrimitive(headerConverter(kind, packed), location, "C"); ok {
			if primitive.method == "float64" {
				g.double = true
			}
			return headerPrimitiveType(primitive)
		}
		panic(fmt.Sprintf("%s is a Slice of fixed Strings, which has no C mapping: put the String in a Struct", location))
	case kindArray:
		panic(fmt.Sprintf("%s is a Slice of Arrays, which has no C mapping: put the Array in a Struct", location))
	case kindSlice:
		slice := packed.(packedSlice)
		if slice.ElementKind == kindConverter && slice.maximum == 0 {
			if primitive, ok := converterPrimitive(slice.Element, location, "C"); ok && primitive.method == "uint8" {
				return "PackedBytes"
			}
		}
		panic(fmt.Sprintf("%s is a list of lists other than bytes, which has no C mapping: put the inner Slice in a Struct", location))
	}
	panic(fmt.Sprintf("%s uses a custom Go type, which has no C mapping", location))
}

// declarations declares a field: one member, or a variable-length field's
// two.
func (g *headerGenerator) declarations(property packedProperty, location string) []string {
	name := headerName(property.name, headerStructMembers)
	switch property.kind {
	case kindVariableString:
		variable := fieldVariable("", property)
		count := strings.TrimPrefix(variable.count, "value->")
		if property.packed.(packedVariableString).Bytes {
			return []string{"const uint8_t* " + name, "size_t " + count}
		}
		return []string{"const char* " + name, "size_t " + count}
	case kindSlice:
		slice := property.packed.(packedSlice)
		count := strings.TrimPrefix(fieldVariable("", property).count, "value->")
		element := g.elementType(slice.ElementKind, slice.Element, location)
		if slice.maximum > 0 {
			return []string{fmt.Sprintf("%s %s[%d]", element, name, slice.maximum), headerCountType(slice.maximum) + " " + count}
		}
		if !headerInPlace(slice.ElementKind, slice.Element) {
			panic(fmt.Sprintf("%s is a Slice of %s, which is not in the input as it is in memory, so C reads it into an array: give it a Maximum, Slice(...).Maximum(n)", location, element))
		}
		return []string{fmt.Sprintf("const %s* %s", element, name), "size_t " + count}
	}
	return []string{g.declaration(property.kind, property.packed, name, location)}
}

// declaration declares a member: arrays and fixed strings as C arrays.
func (g *headerGenerator) declaration(kind kind, packed any, name string, location string) string {
	switch kind {
	case kindArray:
		array := packed.(packedArray)
		return g.declaration(array.ElementKind, array.Element, fmt.Sprintf("%s[%d]", name, array.Length), location)
	case kindConverter, kindConverterCast:
		converter := headerConverter(kind, packed)
		if _, ok := converterPrimitive(converter, location, "C"); !ok {
			length := reflect.ValueOf(converter).Elem().Interface().(StringConverter).Length
			return fmt.Sprintf("char %s[%d]", name, length)
		}
	}
	return g.elementType(kind, packed, location) + " " + name
}

func (g *headerGenerator) bitFieldDeclaration(field packedBitField) string {
	name := headerName(field.packedProperty.name, headerStructMembers)
	switch field.bitFieldKind {
	case bitFieldKindTag:
		return fmt.Sprintf("__extension__ uint8_t tag : %d", field.bitSize)
	case bitFieldKindBoolean:
		return fmt.Sprintf("bool %s : 1", name)
	case bitFieldKindInteger:
		return fmt.Sprintf("__extension__ %s %s : %d", headerPrimitiveType(kotlinIntegers[field.reflection.Kind()]), name, field.bitSize)
	case bitFieldKindEnum:
		return fmt.Sprintf("__extension__ %s %s : %d", headerBitFieldEnumType(field), name, field.bitSize)
	}
	panic(fmt.Sprintf("bit field %s uses a custom Go type, which has no C mapping", field.packedProperty.name))
}

// headerBitFieldEnumType is the type an enum's bit field is declared with:
// the enum, or for an enum of a signed type the unsigned type of its size,
// as a signed bit field can't hold the highest values its bits can.
func headerBitFieldEnumType(field packedBitField) string {
	if kotlinIntegers[field.reflection.Kind()].signed {
		return headerUnsigned(field.enum.size)
	}
	return field.enum.name
}

// hasNamedFields reports whether a struct has a field the generated code
// reads or writes through value.
func hasNamedFields(structure packedStruct) bool {
	for _, property := range structure.properties {
		if property.kind != kindBitFieldGroup {
			return true
		}
		for _, field := range property.packed.(packedBitFieldGroup).fields {
			if field.bitFieldKind != bitFieldKindTag {
				return true
			}
		}
	}
	return false
}

func (g *headerGenerator) structure(w *sourceWriter, structure packedStruct) {
	name := structure.name
	fixed := !structure.dynamic
	view := fixed && structure.size > 0 && headerLittleEndianLayout(structure)

	members := []string{}
	pointsAtStructs := false
	for _, property := range structure.properties {
		if property.kind == kindBitFieldGroup {
			for _, field := range property.packed.(packedBitFieldGroup).fields {
				members = append(members, g.bitFieldDeclaration(field))
			}
			continue
		}
		members = append(members, g.declarations(property, name+"."+property.name)...)
		if slice, ok := property.packed.(packedSlice); ok && slice.maximum == 0 && slice.ElementKind == kindStruct {
			pointsAtStructs = true
		}
	}

	names := map[string]bool{}
	for _, member := range members {
		fields := strings.Fields(strings.TrimSuffix(strings.Split(member, ":")[0], " "))
		memberName := strings.TrimLeft(strings.Split(fields[len(fields)-1], "[")[0], "*")
		if names[memberName] {
			panic(fmt.Sprintf("struct %s has two members named %s in C, one of them a variable-length field's count: rename a field", name, memberName))
		}
		names[memberName] = true
	}

	w.line("typedef struct %s %s;", name, name)
	w.line("")
	w.line("static inline size_t %sSize(const %s* value);", name, name)
	w.line("static inline PackedError %sToBytes(const %s* value, uint8_t* bytes, size_t capacity, size_t index, size_t* end);", name, name)
	w.line("static inline PackedError %sFromBytes(%s* value, const uint8_t* bytes, size_t length, size_t index, size_t* end);", name, name)
	if view {
		w.line("#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__")
		w.line("static inline const %s* %sView(const uint8_t* bytes, size_t length, size_t index);", name, name)
		w.line("#endif")
	}
	w.line("")

	switch {
	case view:
		w.open("struct __attribute__((packed, may_alias)) %s {", name)
	case fixed:
		w.open("struct __attribute__((packed)) %s {", name)
	default:
		w.open("struct %s {", name)
	}
	for _, member := range members {
		w.line("%s;", member)
	}
	if len(structure.properties) == 0 {
		w.line("// C has no empty structs; this is never written or read.")
		w.line("char unused;")
	}

	w.line("")
	headerDirective(w, "#ifdef __cplusplus")
	if fixed {
		w.line("static constexpr size_t fixedSize = %d;", structure.size)
		w.line("")
	}
	w.line("size_t size() const { return %sSize(this); }", name)
	w.line("")
	w.line("// Writes the struct at index of bytes, capacity long; end is the index just past it.")
	w.open("PackedError toBytes(uint8_t* bytes, size_t capacity, size_t index, size_t& end) const {")
	w.line("return %sToBytes(this, bytes, capacity, index, &end);", name)
	w.close("}")
	w.line("")
	w.line("// Reads the struct at index of bytes, length long; end is the index just past it.")
	w.line("// Text, bytes and lists point into bytes, which must outlive them.")
	w.open("PackedError fromBytes(const uint8_t* bytes, size_t length, size_t index, size_t& end) {")
	w.line("return %sFromBytes(this, bytes, length, index, &end);", name)
	w.close("}")
	if view {
		w.line("")
		headerDirective(w, "#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__")
		w.open("static const %s* view(const uint8_t* bytes, size_t length, size_t index = 0) {", name)
		w.line("return %sView(bytes, length, index);", name)
		w.close("}")
		headerDirective(w, "#endif")
	}
	headerDirective(w, "#endif")
	w.close("};")
	w.line("")

	if fixed && structure.size > 0 {
		w.line("PACKED_STATIC_ASSERT(sizeof(%s) == %d, \"packed: %s is not %d bytes\");", name, structure.size, name, structure.size)
		w.line("")
		w.line("#define %sFixedSize ((size_t)%d)", name, structure.size)
		w.line("")
	}
	if pointsAtStructs {
		w.line("PACKED_STATIC_ASSERT(__BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__, \"packed: %s points at structs in the input, which needs a little-endian machine\");", name)
		w.line("")
	}

	w.open("static inline void %sWrite(const %s* value, PackedWriter* writer) {", name, name)
	if !hasNamedFields(structure) {
		w.line("(void)value;")
	}
	for _, property := range structure.properties {
		g.writeProperty(w, property, "", 0)
	}
	if len(structure.properties) == 0 {
		w.line("(void)writer;")
	}
	w.close("}")
	w.line("")

	w.open("static inline void %sRead(%s* value, PackedReader* reader) {", name, name)
	if !hasNamedFields(structure) {
		w.line("(void)value;")
	}
	if len(structure.properties) == 0 {
		w.line("(void)reader;")
	}
	for _, property := range structure.properties {
		g.readProperty(w, property, "", 0)
	}
	w.close("}")
	w.line("")

	w.open("static inline size_t %sSize(const %s* value) {", name, name)
	if fixed {
		w.line("(void)value;")
		w.line("return %d;", structure.size)
	} else {
		w.line("size_t size = %d;", structure.size)
		for _, property := range structure.properties {
			if property.dynamic {
				g.propertySize(w, property, "", 0)
			}
		}
		w.line("return size;")
	}
	w.close("}")
	w.line("")

	w.open("static inline PackedError %sToBytes(const %s* value, uint8_t* bytes, size_t capacity, size_t index, size_t* end) {", name, name)
	w.line("PackedWriter writer = PackedWriterOf(bytes, capacity, index);")
	w.line("%sWrite(value, &writer);", name)
	w.line("*end = writer.index;")
	w.line("return writer.error;")
	w.close("}")
	w.line("")

	w.open("static inline PackedError %sFromBytes(%s* value, const uint8_t* bytes, size_t length, size_t index, size_t* end) {", name, name)
	w.line("PackedReader reader = PackedReaderOf(bytes, length, index);")
	w.line("%sRead(value, &reader);", name)
	w.line("*end = reader.index;")
	w.line("return reader.error;")
	w.close("}")

	if view {
		w.line("")
		w.line("#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__")
		w.line("// The struct at index of bytes as it is, without a copy, or NULL when")
		w.line("// bytes is too short. Unlike FromBytes, it does not check the enums' values.")
		w.open("static inline const %s* %sView(const uint8_t* bytes, size_t length, size_t index) {", name, name)
		w.open("if (index > length || length - index < %d) {", structure.size)
		w.line("return NULL;")
		w.close("}")
		w.line("return (const %s*)(bytes + index);", name)
		w.close("}")
		w.line("#endif")
	}
}

// propertySize adds a dynamic property's size to size.
func (g *headerGenerator) propertySize(w *sourceWriter, property packedProperty, prefix string, depth int) {
	switch property.kind {
	case kindVariableString:
		variable := fieldVariable(prefix, property)
		w.line("size += %d + %s;", property.packed.(packedVariableString).Length.size, variable.count)
	case kindSlice:
		g.sliceSize(w, property.packed.(packedSlice), fieldVariable(prefix, property), depth)
	default:
		g.valueSize(w, property.kind, property.packed, headerMember(prefix, headerName(property.name, headerStructMembers)), depth)
	}
}

func (g *headerGenerator) sliceSize(w *sourceWriter, slice packedSlice, variable headerVariable, depth int) {
	if !isDynamic(slice.ElementKind, slice.Element) {
		w.line("size += %d + %s * %d;", slice.Length.size, variable.count, slice.ElementSize)
		return
	}
	w.line("size += %d;", slice.Length.size)
	index := fmt.Sprintf("i%d", depth)
	w.open("for (size_t %s = 0; %s < %s; %s++) {", index, index, variable.count, index)
	g.valueSize(w, slice.ElementKind, slice.Element, fmt.Sprintf("%s[%s]", variable.items, index), depth+1)
	w.close("}")
}

// valueSize adds the size of a dynamic value, an element or a struct, to
// size.
func (g *headerGenerator) valueSize(w *sourceWriter, kind kind, packed any, value string, depth int) {
	switch kind {
	case kindVariableString:
		w.line("size += %d + %s;", packed.(packedVariableString).Length.size, elementVariable(kind, packed, value).count)
	case kindSlice:
		w.line("size += %d + %s;", packed.(packedSlice).Length.size, elementVariable(kind, packed, value).count)
	case kindStruct:
		w.line("size += %sSize(&%s);", packed.(packedStruct).name, value)
	case kindArray:
		array := packed.(packedArray)
		index := fmt.Sprintf("i%d", depth)
		w.open("for (size_t %s = 0; %s < %d; %s++) {", index, index, array.Length, index)
		g.valueSize(w, array.ElementKind, array.Element, fmt.Sprintf("%s[%s]", value, index), depth+1)
		w.close("}")
	default:
		panic("invalid dynamic property kind")
	}
}

func (g *headerGenerator) writeProperty(w *sourceWriter, property packedProperty, prefix string, depth int) {
	switch property.kind {
	case kindBitFieldGroup:
		g.writeBits(w, property.packed.(packedBitFieldGroup), property.littleEndian, prefix)
	case kindVariableString:
		variable := fieldVariable(prefix, property)
		g.writeBytes(w, variable, property.packed.(packedVariableString).Length.size, property.littleEndian)
	case kindSlice:
		g.writeSlice(w, property.packed.(packedSlice), fieldVariable(prefix, property), property.littleEndian, depth)
	default:
		g.writeValue(w, property.kind, property.packed, headerMember(prefix, headerName(property.name, headerStructMembers)), property.littleEndian, depth)
	}
}

// writeBytes writes text or bytes: their length, then them.
func (g *headerGenerator) writeBytes(w *sourceWriter, variable headerVariable, prefix int, littleEndian bool) {
	w.line("PackedWriterItems(writer, %s, %s, 1, %d, %s);", variable.items, variable.count, prefix, headerEndian(littleEndian))
}

func (g *headerGenerator) writeSlice(w *sourceWriter, slice packedSlice, variable headerVariable, littleEndian bool, depth int) {
	if slice.maximum == 0 {
		// in place: the elements in memory are the bytes on the wire
		w.line("PackedWriterItems(writer, %s, %s, %d, %d, %s);", variable.items, variable.count, slice.ElementSize, slice.Length.size, headerEndian(littleEndian))
		return
	}
	w.open("if (%s > %d) {", variable.count, slice.maximum)
	w.line("PackedWriterFail(writer, PackedErrorTooLong);")
	w.indent--
	w.open("} else {")
	if headerByteElements(slice.ElementKind, slice.Element) {
		w.line("PackedWriterItems(writer, %s, %s, 1, %d, %s);", variable.items, variable.count, slice.Length.size, headerEndian(littleEndian))
	} else {
		w.line("PackedWriterLength(writer, %s, %d, %s);", variable.count, slice.Length.size, headerEndian(littleEndian))
		index := fmt.Sprintf("i%d", depth)
		w.open("for (size_t %s = 0; %s < %s; %s++) {", index, index, variable.count, index)
		g.writeValue(w, slice.ElementKind, slice.Element, fmt.Sprintf("%s[%s]", variable.items, index), littleEndian, depth+1)
		w.close("}")
	}
	w.close("}")
}

func (g *headerGenerator) writeValue(w *sourceWriter, kind kind, packed any, value string, littleEndian bool, depth int) {
	switch kind {

	case kindStruct:
		for _, child := range packed.(packedStruct).properties {
			g.writeProperty(w, child, value, depth)
		}

	case kindConverter, kindConverterCast:
		converter := headerConverter(kind, packed)
		primitive, ok := converterPrimitive(converter, value, "C")
		if !ok {
			length := reflect.ValueOf(converter).Elem().Interface().(StringConverter).Length
			w.line("PackedWriterCopy(writer, %s, %d);", value, length)
			return
		}
		switch primitive.method {
		case "boolean":
			w.line("PackedWriterPut8(writer, %s ? 1 : 0);", value)
		case "float32":
			w.line("%s", headerPut(4, fmt.Sprintf("PackedFloat32Bits(%s)", value), littleEndian))
		case "float64":
			g.double = true
			w.line("%s", headerPut(8, fmt.Sprintf("PackedFloat64Bits(%s)", value), littleEndian))
		default:
			w.line("%s", headerPut(primitive.size, value, littleEndian))
		}

	case kindEnum:
		w.line("%s", headerPut(packed.(packedEnum).size, value, littleEndian))

	case kindVariableString:
		g.writeBytes(w, elementVariable(kind, packed, value), packed.(packedVariableString).Length.size, littleEndian)

	case kindSlice:
		g.writeBytes(w, elementVariable(kind, packed, value), packed.(packedSlice).Length.size, littleEndian)

	case kindArray:
		array := packed.(packedArray)
		index := fmt.Sprintf("i%d", depth)
		w.open("for (size_t %s = 0; %s < %d; %s++) {", index, index, array.Length, index)
		g.writeValue(w, array.ElementKind, array.Element, fmt.Sprintf("%s[%s]", value, index), littleEndian, depth+1)
		w.close("}")

	default:
		panic(fmt.Sprintf("%s uses a custom Go type, which has no C mapping", value))
	}
}

// writeBits assembles each word of a bit run in the narrowest unsigned type
// that holds it, as writeBits in kotlin.go does, then writes its bytes. A
// message set's tag is written as the constant it is, whatever tag holds.
func (g *headerGenerator) writeBits(w *sourceWriter, group packedBitFieldGroup, littleEndian bool, prefix string) {
	placements := group.placements(littleEndian)

	w.open("{")
	for word := 0; word < wordCount(group.size); word++ {
		size := wordBytes(group.size, word)
		unsigned := headerUnsigned(size)
		variable := fmt.Sprintf("word%d", word)
		w.line("%s %s = 0;", unsigned, variable)

		for _, placement := range placements {
			field := placement.field
			value := headerMember(prefix, headerName(field.packedProperty.name, headerStructMembers))
			if field.bitFieldKind == bitFieldKindTag {
				value = fmt.Sprintf("%d", field.tagValue)
			}

			for _, deposit := range placement.deposits {
				if deposit.word != word {
					continue
				}
				var part string
				if placement.straddle {
					source := fmt.Sprintf("(uint64_t)(%s)", value)
					if deposit.sourceShift != 0 {
						source = fmt.Sprintf("(%s >> %d)", source, deposit.sourceShift)
					}
					part = fmt.Sprintf("(%s & %s)", source, headerHex(bitMask(deposit.bitCount)))
				} else {
					part = fmt.Sprintf("((%s)(%s) & %s)", unsigned, value, headerHex(bitMask(deposit.bitCount)))
				}
				if deposit.destinationShift != 0 {
					part = fmt.Sprintf("(%s << %d)", part, deposit.destinationShift)
				}
				w.line("%s |= (%s)%s;", variable, unsigned, part)
			}
		}

		w.line("%s", headerPut(size, variable, littleEndian))
	}
	w.close("}")
}

func (g *headerGenerator) readProperty(w *sourceWriter, property packedProperty, prefix string, depth int) {
	switch property.kind {
	case kindBitFieldGroup:
		g.readBits(w, property.packed.(packedBitFieldGroup), property.littleEndian, prefix)
	case kindVariableString:
		variable := property.packed.(packedVariableString)
		element := "char"
		if variable.Bytes {
			element = "uint8_t"
		}
		g.readBytes(w, fieldVariable(prefix, property), element, variable.Length.size, property.littleEndian, depth)
	case kindSlice:
		g.readSlice(w, property.packed.(packedSlice), fieldVariable(prefix, property), property.littleEndian, depth, prefix+"."+property.name)
	default:
		g.readValue(w, property.kind, property.packed, headerMember(prefix, headerName(property.name, headerStructMembers)), property.littleEndian, depth)
	}
}

// readBytes reads a length prefix and points items at that many bytes.
func (g *headerGenerator) readBytes(w *sourceWriter, variable headerVariable, element string, prefix int, littleEndian bool, depth int) {
	count := fmt.Sprintf("count%d", depth)
	at := fmt.Sprintf("at%d", depth)
	w.open("{")
	w.line("size_t %s = 0;", count)
	w.line("const uint8_t* %s = PackedReaderItems(reader, %d, %s, 1, &%s);", at, prefix, headerEndian(littleEndian), count)
	w.line("%s = (const %s*)%s;", variable.items, element, at)
	w.line("%s = %s;", variable.count, count)
	w.close("}")
}

func (g *headerGenerator) readSlice(w *sourceWriter, slice packedSlice, variable headerVariable, littleEndian bool, depth int, location string) {
	count := fmt.Sprintf("count%d", depth)
	at := fmt.Sprintf("at%d", depth)
	index := fmt.Sprintf("i%d", depth)
	element := g.elementType(slice.ElementKind, slice.Element, location)

	w.open("{")
	if slice.maximum == 0 {
		// in place: point at the elements, and check their structs' enums
		w.line("size_t %s = 0;", count)
		w.line("const uint8_t* %s = PackedReaderItems(reader, %d, %s, %d, &%s);", at, slice.Length.size, headerEndian(littleEndian), slice.ElementSize, count)
		w.line("%s = (const %s*)%s;", variable.items, element, at)
		w.line("%s = %s;", variable.count, count)
		if slice.ElementKind == kindStruct && headerHasEnums(slice.ElementKind, slice.Element) {
			elements := fmt.Sprintf("elements%d", depth)
			w.line("PackedReader %s = PackedReaderOf(%s, %s * %d, 0);", elements, at, count, slice.ElementSize)
			w.open("for (size_t %s = 0; %s < %s && %s.error == PackedErrorNone; %s++) {", index, index, count, elements, index)
			w.line("%s item;", element)
			w.line("%sRead(&item, &%s);", element, elements)
			w.close("}")
			w.open("if (%s.error != PackedErrorNone) {", elements)
			w.line("PackedReaderFail(reader, %s.error);", elements)
			w.close("}")
		}
		w.close("}")
		return
	}

	w.line("size_t %s = PackedReaderCount(reader, %d, %s);", count, slice.Length.size, headerEndian(littleEndian))
	w.open("if (%s > %d) {", count, slice.maximum)
	w.line("PackedReaderFail(reader, PackedErrorTooLong);")
	w.line("%s = 0;", count)
	w.close("}")
	w.line("%s = (%s)%s;", variable.count, headerCountType(slice.maximum), count)
	if headerByteElements(slice.ElementKind, slice.Element) {
		w.line("PackedReaderCopy(reader, %s, %s);", variable.items, count)
	} else {
		w.open("for (size_t %s = 0; %s < %s && reader->error == PackedErrorNone; %s++) {", index, index, count, index)
		g.readValue(w, slice.ElementKind, slice.Element, fmt.Sprintf("%s[%s]", variable.items, index), littleEndian, depth+1)
		w.close("}")
	}
	w.close("}")
}

// readPrimitive is the expression that reads a built-in converter's value.
func (g *headerGenerator) readPrimitive(primitive kotlinPrimitive, littleEndian bool) string {
	switch primitive.method {
	case "boolean":
		return "PackedReaderGet8(reader) != 0"
	case "float32":
		return fmt.Sprintf("PackedFloat32Value(%s)", headerGet(4, littleEndian))
	case "float64":
		g.double = true
		return fmt.Sprintf("PackedFloat64Value(%s)", headerGet(8, littleEndian))
	}
	read := headerGet(primitive.size, littleEndian)
	if primitive.signed {
		return fmt.Sprintf("(%s)%s", headerPrimitiveType(primitive), read)
	}
	return read
}

func (g *headerGenerator) readValue(w *sourceWriter, kind kind, packed any, target string, littleEndian bool, depth int) {
	switch kind {

	case kindStruct:
		for _, child := range packed.(packedStruct).properties {
			g.readProperty(w, child, target, depth)
		}

	case kindConverter, kindConverterCast:
		converter := headerConverter(kind, packed)
		if primitive, ok := converterPrimitive(converter, target, "C"); ok {
			w.line("%s = %s;", target, g.readPrimitive(primitive, littleEndian))
			return
		}
		length := reflect.ValueOf(converter).Elem().Interface().(StringConverter).Length
		w.line("PackedReaderCopy(reader, %s, %d);", target, length)

	case kindEnum:
		enum := packed.(packedEnum)
		w.open("{")
		w.line("%s raw = %s;", headerUnsigned(enum.size), headerGet(enum.size, littleEndian))
		w.open("if (raw >= %d) {", len(enum.values))
		w.line("PackedReaderFail(reader, PackedErrorUnknownValue);")
		w.close("}")
		w.line("%s = (%s)raw;", target, enum.name)
		w.close("}")

	case kindVariableString:
		element := "char"
		if packed.(packedVariableString).Bytes {
			element = "uint8_t"
		}
		g.readBytes(w, elementVariable(kind, packed, target), element, packed.(packedVariableString).Length.size, littleEndian, depth)

	case kindSlice:
		g.readBytes(w, elementVariable(kind, packed, target), "uint8_t", packed.(packedSlice).Length.size, littleEndian, depth)

	case kindArray:
		array := packed.(packedArray)
		index := fmt.Sprintf("i%d", depth)
		w.open("for (size_t %s = 0; %s < %d && reader->error == PackedErrorNone; %s++) {", index, index, array.Length, index)
		g.readValue(w, array.ElementKind, array.Element, fmt.Sprintf("%s[%s]", target, index), littleEndian, depth+1)
		w.close("}")

	default:
		panic(fmt.Sprintf("%s uses a custom Go type, which has no C mapping", target))
	}
}

func (g *headerGenerator) readBits(w *sourceWriter, group packedBitFieldGroup, littleEndian bool, prefix string) {
	placements := group.placements(littleEndian)
	words := wordCount(group.size)

	w.open("{")
	for word := 0; word < words; word++ {
		size := wordBytes(group.size, word)
		w.line("%s word%d = %s;", headerUnsigned(size), word, headerGet(size, littleEndian))
	}

	for _, placement := range placements {
		field := placement.field

		term := func(deposit bitDeposit) string {
			text := fmt.Sprintf("word%d", deposit.word)
			if deposit.destinationShift != 0 {
				text = fmt.Sprintf("(%s >> %d)", text, deposit.destinationShift)
			}
			text = fmt.Sprintf("(%s & %s)", text, headerHex(bitMask(deposit.bitCount)))
			if placement.straddle {
				text = fmt.Sprintf("(uint64_t)%s", text)
			}
			if deposit.sourceShift != 0 {
				text = fmt.Sprintf("(%s << %d)", text, deposit.sourceShift)
			}
			return text
		}

		raw := term(placement.deposits[0])
		if len(placement.deposits) == 2 {
			raw = fmt.Sprintf("(%s | %s)", raw, term(placement.deposits[1]))
		}

		if field.bitFieldKind == bitFieldKindTag {
			w.line("%s = (uint8_t)%s;", headerMember(prefix, "tag"), raw)
			continue
		}

		target := headerMember(prefix, headerName(field.packedProperty.name, headerStructMembers))
		switch field.bitFieldKind {
		case bitFieldKindBoolean:
			w.line("%s = %s != 0;", target, raw)
		case bitFieldKindEnum:
			w.open("if (%s >= %d) {", raw, len(field.enum.values))
			w.line("PackedReaderFail(reader, PackedErrorUnknownValue);")
			w.close("}")
			w.line("%s = (%s)%s;", target, headerBitFieldEnumType(field), raw)
		case bitFieldKindInteger:
			primitive := kotlinIntegers[field.reflection.Kind()]
			if primitive.signed {
				sign := headerHex(uint64(1) << (field.bitSize - 1))
				w.line("%s = (%s)((%s ^ %s) - %s);", target, headerPrimitiveType(primitive), raw, sign, sign)
			} else {
				w.line("%s = (%s)%s;", target, headerPrimitiveType(primitive), raw)
			}
		default:
			panic(fmt.Sprintf("bit field %s uses a custom Go type, which has no C mapping", field.packedProperty.name))
		}
	}
	w.close("}")
}

func (g *headerGenerator) messageSet(w *sourceWriter, set packedMessageSet) {
	name := set.name
	members := make([]string, len(set.variants))
	values := make([]headerEnumValue, len(set.variants))
	for index, variant := range set.variants {
		members[index] = headerName(variant.structure.name, headerSetMembers)
		values[index] = headerEnumValue{variant.structure.name, uint64(variant.typeByte)}
	}

	g.enum(w, name+"Type", "uint8_t", values)
	w.line("")

	w.line("typedef struct %s %s;", name, name)
	w.line("")
	w.line("static inline size_t %sSize(const %s* message);", name, name)
	w.line("static inline PackedError %sEncode(const %s* message, uint8_t* bytes, size_t capacity, size_t* end);", name, name)
	w.line("static inline PackedError %sDecode(%s* message, const uint8_t* bytes, size_t length);", name, name)
	w.line("")

	w.open("struct %s {", name)
	w.line("%sType type;", name)
	w.open("union {")
	for index, variant := range set.variants {
		w.line("%s %s;", variant.structure.name, members[index])
	}
	w.close("};")
	w.line("")
	headerDirective(w, "#ifdef __cplusplus")
	w.line("using Type = %sType;", name)
	w.line("")
	w.line("// The number of bytes encode writes.")
	w.line("size_t size() const { return %sSize(this); }", name)
	w.line("")
	w.line("// Writes the message at the start of bytes, capacity long; end is the index just past it.")
	w.open("PackedError encode(uint8_t* bytes, size_t capacity, size_t& end) const {")
	w.line("return %sEncode(this, bytes, capacity, &end);", name)
	w.close("}")
	w.line("")
	w.line("// Reads the message that fills bytes, length long. Text, bytes and lists")
	w.line("// point into bytes, which must outlive them.")
	w.open("static PackedError decode(const uint8_t* bytes, size_t length, %s& message) {", name)
	w.line("return %sDecode(&message, bytes, length);", name)
	w.close("}")
	headerDirective(w, "#endif")
	w.close("};")
	w.line("")

	typeByte := 1
	if set.tagInBits {
		typeByte = 0
	}

	w.open("static inline size_t %sSize(const %s* message) {", name, name)
	w.open("switch ((uint8_t)message->type) {")
	for index, variant := range set.variants {
		w.line("case %d: return %d + %sSize(&message->%s);", variant.typeByte, typeByte, variant.structure.name, members[index])
	}
	w.close("}")
	w.line("return 0;")
	w.close("}")
	w.line("")

	w.open("static inline PackedError %sEncode(const %s* message, uint8_t* bytes, size_t capacity, size_t* end) {", name, name)
	w.line("PackedWriter writer = PackedWriterOf(bytes, capacity, 0);")
	w.open("switch ((uint8_t)message->type) {")
	for index, variant := range set.variants {
		w.open("case %d:", variant.typeByte)
		if typeByte == 1 {
			w.line("PackedWriterPut8(&writer, %d);", variant.typeByte)
		}
		w.line("%sWrite(&message->%s, &writer);", variant.structure.name, members[index])
		w.line("break;")
		w.indent--
	}
	w.open("default:")
	w.line("PackedWriterFail(&writer, PackedErrorUnknownType);")
	w.line("break;")
	w.indent--
	w.close("}")
	w.line("*end = writer.index;")
	w.line("return writer.error;")
	w.close("}")
	w.line("")

	w.open("static inline PackedError %sDecode(%s* message, const uint8_t* bytes, size_t length) {", name, name)
	decodeVariant := func(index int) {
		variant := set.variants[index]
		w.line("memset(message, 0, sizeof *message);")
		w.line("message->type = (%sType)%d;", name, variant.typeByte)
		w.line("%sRead(&message->%s, &reader);", variant.structure.name, members[index])
	}
	if set.tagInBits && set.tagBits == 0 {
		w.line("PackedReader reader = PackedReaderOf(bytes, length, 0);")
		decodeVariant(0)
	} else {
		w.open("if (length == 0) {")
		w.line("return PackedErrorEmpty;")
		w.close("}")
		switch {
		case !set.tagInBits || set.tagBits == 8:
			w.line("uint8_t tag = bytes[0];")
		case set.littleEndian:
			w.line("uint8_t tag = bytes[0] & %s;", headerHex(bitMask(set.tagBits)))
		default:
			w.line("uint8_t tag = bytes[0] >> %d;", 8-set.tagBits)
		}
		w.line("PackedReader reader = PackedReaderOf(bytes, length, %d);", typeByte)
		w.open("switch (tag) {")
		for index, variant := range set.variants {
			w.open("case %d:", variant.typeByte)
			decodeVariant(index)
			w.line("break;")
			w.indent--
		}
		w.open("default:")
		w.line("return PackedErrorUnknownType;")
		w.indent--
		w.close("}")
	}
	w.open("if (reader.error != PackedErrorNone) {")
	w.line("return reader.error;")
	w.close("}")
	w.open("if (reader.index != length) {")
	w.line("return PackedErrorTrailingBytes;")
	w.close("}")
	w.line("return PackedErrorNone;")
	w.close("}")
}

// headerRuntime is the reader, writer and element types every generated
// header shares, once per program however many headers include it.
var headerRuntime = strings.TrimLeft(`
#ifndef PACKED_RUNTIME
#define PACKED_RUNTIME

// What a write or a read failed with: the first failure sticks.
typedef enum PackedError {
    PackedErrorNone,
    // The input ended before the value did.
    PackedErrorTruncated,
    // The output has no room for the value.
    PackedErrorFull,
    // A list or text holds more than its length prefix or its maximum can say.
    PackedErrorTooLong,
    // An enum holds a value that has no case.
    PackedErrorUnknownValue,
    // A message set was decoded from no bytes.
    PackedErrorEmpty,
    // A message set's type has no variant.
    PackedErrorUnknownType,
    // A message set's input continues after the variant.
    PackedErrorTrailingBytes,
} PackedError;

#ifdef __cplusplus
#define PACKED_STATIC_ASSERT(condition, message) static_assert(condition, message)
#else
#define PACKED_STATIC_ASSERT(condition, message) _Static_assert(condition, message)
#endif

// C++20 where the standard library has them (not avr-gcc) gets text as a
// std::string_view and bytes as a std::span.
#if defined(__cplusplus) && __cplusplus >= 202002L && defined(__has_include)
#if __has_include(<string_view>) && __has_include(<span>)
#include <span>
#include <string_view>
#define PACKED_STANDARD_LIBRARY 1
#endif
#endif

// Text as an element of an array or a list: UTF-8, without a terminating
// zero. A field of text is a pointer and a length of its own.
typedef struct PackedText {
    const char* text;
    size_t length;
#ifdef PACKED_STANDARD_LIBRARY
    operator std::string_view() const { return std::string_view(text, length); }
#endif
} PackedText;

// Bytes as an element of an array or a list. A field of bytes is a pointer
// and a length of its own.
typedef struct PackedBytes {
    const uint8_t* bytes;
    size_t length;
#ifdef PACKED_STANDARD_LIBRARY
    operator std::span<const uint8_t>() const { return std::span<const uint8_t>(bytes, length); }
#endif
} PackedBytes;

static inline uint32_t PackedFloat32Bits(float value) {
    uint32_t bits = 0;
    memcpy(&bits, &value, sizeof value);
    return bits;
}

static inline float PackedFloat32Value(uint32_t bits) {
    float value;
    memcpy(&value, &bits, sizeof value);
    return value;
}

static inline uint64_t PackedFloat64Bits(double value) {
    uint64_t bits = 0;
    memcpy(&bits, &value, sizeof value);
    return bits;
}

static inline double PackedFloat64Value(uint64_t bits) {
    double value;
    memcpy(&value, &bits, sizeof value);
    return value;
}

typedef struct PackedWriter {
    uint8_t* bytes;
    size_t capacity;
    size_t index;
    PackedError error;
} PackedWriter;

static inline PackedWriter PackedWriterOf(uint8_t* bytes, size_t capacity, size_t index) {
    PackedWriter writer;
    writer.bytes = bytes;
    writer.capacity = capacity;
    writer.index = index;
    writer.error = PackedErrorNone;
    return writer;
}

static inline void PackedWriterFail(PackedWriter* writer, PackedError error) {
    if (writer->error == PackedErrorNone) {
        writer->error = error;
    }
}

// Claims count bytes and returns where they start, or NULL when they don't
// fit or an earlier write failed.
static inline uint8_t* PackedWriterClaim(PackedWriter* writer, size_t count) {
    if (writer->error != PackedErrorNone) {
        return NULL;
    }
    if (writer->index > writer->capacity || count > writer->capacity - writer->index) {
        PackedWriterFail(writer, PackedErrorFull);
        return NULL;
    }
    uint8_t* at = writer->bytes + writer->index;
    writer->index += count;
    return at;
}

static inline void PackedWriterPut8(PackedWriter* writer, uint8_t value) {
    uint8_t* at = PackedWriterClaim(writer, 1);
    if (at != NULL) {
        at[0] = value;
    }
}

static inline void PackedWriterPut16(PackedWriter* writer, uint16_t value, bool little) {
    uint8_t* at = PackedWriterClaim(writer, 2);
    if (at != NULL) {
        at[little ? 0 : 1] = (uint8_t)value;
        at[little ? 1 : 0] = (uint8_t)(value >> 8);
    }
}

// Writes the low size bytes of value, 1 to 4.
static inline void PackedWriterPut32(PackedWriter* writer, uint32_t value, uint8_t size, bool little) {
    uint8_t* at = PackedWriterClaim(writer, size);
    if (at == NULL) {
        return;
    }
    for (uint8_t byte = 0; byte < size; byte++) {
        at[little ? byte : size - 1 - byte] = (uint8_t)(value >> (8 * byte));
    }
}

// Writes the low size bytes of value, 1 to 8.
static inline void PackedWriterPut64(PackedWriter* writer, uint64_t value, uint8_t size, bool little) {
    uint8_t* at = PackedWriterClaim(writer, size);
    if (at == NULL) {
        return;
    }
    for (uint8_t byte = 0; byte < size; byte++) {
        at[little ? byte : size - 1 - byte] = (uint8_t)(value >> (8 * byte));
    }
}

static inline void PackedWriterCopy(PackedWriter* writer, const void* source, size_t count) {
    uint8_t* at = PackedWriterClaim(writer, count);
    if (at != NULL && count > 0) {
        memcpy(at, source, count);
    }
}

// Writes a length prefix of size bytes. Always inlined, so a prefix's
// constant size leaves only its own branch, and no 64-bit arithmetic on a
// machine whose size_t is smaller.
static inline __attribute__((always_inline)) void PackedWriterLength(PackedWriter* writer, size_t count, uint8_t size, bool little) {
    switch (size) {
    case 1:
        if (count > UINT8_MAX) {
            PackedWriterFail(writer, PackedErrorTooLong);
            return;
        }
        PackedWriterPut8(writer, (uint8_t)count);
        break;
    case 2:
#if SIZE_MAX > UINT16_MAX
        if (count > UINT16_MAX) {
            PackedWriterFail(writer, PackedErrorTooLong);
            return;
        }
#endif
        PackedWriterPut16(writer, (uint16_t)count, little);
        break;
    case 4:
#if SIZE_MAX > UINT32_MAX
        if (count > UINT32_MAX) {
            PackedWriterFail(writer, PackedErrorTooLong);
            return;
        }
#endif
        PackedWriterPut32(writer, (uint32_t)count, 4, little);
        break;
    default:
        PackedWriterPut64(writer, (uint64_t)count, 8, little);
        break;
    }
}

// Writes a length prefix of size bytes, then count items of itemSize bytes
// as they are in memory.
static inline __attribute__((always_inline)) void PackedWriterItems(PackedWriter* writer, const void* items, size_t count, size_t itemSize, uint8_t size, bool little) {
    size_t total;
    if (__builtin_mul_overflow(count, itemSize, &total)) {
        PackedWriterFail(writer, PackedErrorTooLong);
        return;
    }
    PackedWriterLength(writer, count, size, little);
    PackedWriterCopy(writer, items, total);
}

typedef struct PackedReader {
    const uint8_t* bytes;
    size_t length;
    size_t index;
    PackedError error;
} PackedReader;

static inline PackedReader PackedReaderOf(const uint8_t* bytes, size_t length, size_t index) {
    PackedReader reader;
    reader.bytes = bytes;
    reader.length = length;
    reader.index = index;
    reader.error = PackedErrorNone;
    return reader;
}

static inline void PackedReaderFail(PackedReader* reader, PackedError error) {
    if (reader->error == PackedErrorNone) {
        reader->error = error;
    }
}

// Takes count bytes and returns where they start, or NULL when the input
// ends first or an earlier read failed.
static inline const uint8_t* PackedReaderTake(PackedReader* reader, size_t count) {
    if (reader->error != PackedErrorNone) {
        return NULL;
    }
    if (reader->index > reader->length || count > reader->length - reader->index) {
        PackedReaderFail(reader, PackedErrorTruncated);
        return NULL;
    }
    const uint8_t* at = reader->bytes + reader->index;
    reader->index += count;
    return at;
}

static inline uint8_t PackedReaderGet8(PackedReader* reader) {
    const uint8_t* at = PackedReaderTake(reader, 1);
    return at == NULL ? 0 : at[0];
}

static inline uint16_t PackedReaderGet16(PackedReader* reader, bool little) {
    const uint8_t* at = PackedReaderTake(reader, 2);
    if (at == NULL) {
        return 0;
    }
    return (uint16_t)(at[little ? 0 : 1] | ((uint16_t)at[little ? 1 : 0] << 8));
}

// Reads size bytes, 1 to 4.
static inline uint32_t PackedReaderGet32(PackedReader* reader, uint8_t size, bool little) {
    const uint8_t* at = PackedReaderTake(reader, size);
    uint32_t value = 0;
    if (at == NULL) {
        return value;
    }
    for (uint8_t byte = 0; byte < size; byte++) {
        value |= (uint32_t)at[little ? byte : size - 1 - byte] << (8 * byte);
    }
    return value;
}

// Reads size bytes, 1 to 8.
static inline uint64_t PackedReaderGet64(PackedReader* reader, uint8_t size, bool little) {
    const uint8_t* at = PackedReaderTake(reader, size);
    uint64_t value = 0;
    if (at == NULL) {
        return value;
    }
    for (uint8_t byte = 0; byte < size; byte++) {
        value |= (uint64_t)at[little ? byte : size - 1 - byte] << (8 * byte);
    }
    return value;
}

static inline void PackedReaderCopy(PackedReader* reader, void* target, size_t count) {
    const uint8_t* at = PackedReaderTake(reader, count);
    if (at != NULL && count > 0) {
        memcpy(target, at, count);
    }
}

// Reads a length prefix of size bytes; one that size_t can't hold is
// longer than any input. Always inlined, as PackedWriterLength is.
static inline __attribute__((always_inline)) size_t PackedReaderCount(PackedReader* reader, uint8_t size, bool little) {
    switch (size) {
    case 1:
        return PackedReaderGet8(reader);
    case 2:
        return PackedReaderGet16(reader, little);
    case 4: {
        uint32_t value = PackedReaderGet32(reader, 4, little);
#if SIZE_MAX < UINT32_MAX
        if (value > SIZE_MAX) {
            PackedReaderFail(reader, PackedErrorTruncated);
            return 0;
        }
#endif
        return (size_t)value;
    }
    default: {
        uint64_t value = PackedReaderGet64(reader, 8, little);
#if SIZE_MAX < UINT64_MAX
        if (value > SIZE_MAX) {
            PackedReaderFail(reader, PackedErrorTruncated);
            return 0;
        }
#endif
        return (size_t)value;
    }
    }
}

// Reads a length prefix of size bytes and takes that many items of
// itemSize bytes, returning where they start with their count, which is 0
// when the input ends first.
static inline __attribute__((always_inline)) const uint8_t* PackedReaderItems(PackedReader* reader, uint8_t size, bool little, size_t itemSize, size_t* count) {
    *count = PackedReaderCount(reader, size, little);
    size_t total;
    if (__builtin_mul_overflow(*count, itemSize, &total)) {
        PackedReaderFail(reader, PackedErrorTruncated);
        *count = 0;
        return NULL;
    }
    const uint8_t* at = PackedReaderTake(reader, total);
    if (at == NULL) {
        *count = 0;
    }
    return at;
}

#endif
`, "\n")
