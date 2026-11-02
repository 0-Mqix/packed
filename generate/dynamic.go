package generate

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/0-Mqix/packed"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
	"golang.org/x/tools/imports"
)

// DynamicFile is the file Dynamic writes into the package.
const DynamicFile = "packed_dynamic.go"

const packedPath = "github.com/0-Mqix/packed"

// local matches the names of the generated code's variables, which would
// hide a type or a package of the same name.
var local = regexp.MustCompile(`^(bytes|index|ok|count|present|start|reader|value|err|(key|element|entry|previous|present|r|b|o|i)[0-9]+|s[0-9]+_[0-9]+)$`)

// reserved are the import names the generated file uses itself.
var reserved = map[string]string{
	"packed": packedPath,
	"binary": "encoding/binary",
	"unsafe": "unsafe",
	"time":   "time",
}

// Dynamic writes the dynamic layer of the package in directory into its
// DynamicFile: for each type that embeds packed.Dynamic its Schema,
// AppendPacked and Decode<Type>, and helpers for every type reachable from
// it.
func Dynamic(directory string) error {
	source, err := DynamicSource(directory)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, DynamicFile), source, 0644)
}

// DynamicSource loads the package in directory with go/types, ignoring the
// DynamicFile it holds, and returns the formatted source of its dynamic
// layer. The error lists every field that can't travel, by its path, with
// a suggestion.
func DynamicSource(directory string) ([]byte, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	// The file from the last run may no longer compile against the types.
	overlay := map[string][]byte{}
	output := filepath.Join(directory, DynamicFile)
	if existing, err := parser.ParseFile(token.NewFileSet(), output, nil, parser.PackageClauseOnly); err == nil {
		overlay[output] = []byte("package " + existing.Name.Name + "\n")
	}

	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:     directory,
		Overlay: overlay,
	}
	loaded, err := packages.Load(config, ".")
	if err != nil {
		return nil, err
	}
	if len(loaded) != 1 || loaded[0].Types == nil || loaded[0].Name == "" {
		return nil, fmt.Errorf("packed: %s holds no Go package", directory)
	}

	return dynamicSource(loaded[0])
}

func dynamicSource(loaded *packages.Package) ([]byte, error) {
	w := &walker{
		files:        loaded.Fset,
		pkg:          loaded.Types,
		functions:    map[string]bool{},
		imports:      map[string]string{},
		names:        map[string]string{},
		packageNames: map[string]string{},
	}
	if len(loaded.Errors) > 0 {
		w.compileError = loaded.Errors[0]
	}

	scope := loaded.Types.Scope()
	var roots []*packed.Layout

	for _, name := range scope.Names() {
		object, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() {
			continue
		}
		named, ok := object.Type().(*types.Named)
		if !ok {
			continue
		}
		structure, ok := named.Underlying().(*types.Struct)
		if !ok || !embedsDynamic(structure) {
			continue
		}
		if named.TypeParams().Len() > 0 {
			w.fail(object.Pos(), name, "a generic type can't be marked, as the generator needs its type arguments; mark a struct that holds an instantiation instead")
			continue
		}

		for _, generated := range []string{name + "Schema", "Decode" + name} {
			if scope.Lookup(generated) != nil {
				w.fail(object.Pos(), name, fmt.Sprintf("the package declares %s, which the generated file declares; rename it", generated))
			}
		}
		if found, _, _ := types.LookupFieldOrMethod(named, true, w.pkg, "AppendPacked"); found != nil {
			w.fail(object.Pos(), name, "the type has a field or method AppendPacked, which the generated file declares; rename it")
		}

		if layout := w.walk(named, name, object.Pos()); layout != nil {
			roots = append(roots, layout)
		}
	}

	w.checkCycles()

	for _, function := range w.functionNames() {
		for _, name := range []string{"appendPacked" + function, "decodePacked" + function} {
			if scope.Lookup(name) != nil {
				w.fail(token.NoPos, name, "the package declares "+name+", which the generated file declares; rename it")
			}
		}
	}

	paths := make([]string, 0, len(w.imports))
	for path := range w.imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	importNames := []string{"packed", "binary", "unsafe"}
	var importList []packed.Import
	for _, path := range paths {
		if path == packedPath {
			continue
		}
		name := w.imports[path]
		importNames = append(importNames, name)
		if name == w.packageNames[path] {
			name = ""
		}
		importList = append(importList, packed.Import{Path: path, Name: name})
	}
	for _, name := range importNames {
		if scope.Lookup(name) != nil {
			w.fail(token.NoPos, name, "the package declares "+name+", the name of a package the generated file imports; rename it")
		}
	}

	if len(w.errors) > 0 {
		return nil, errors.Join(w.errors...)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("packed: no type of %s embeds packed.Dynamic", loaded.PkgPath)
	}

	source := packed.DynamicSource(loaded.Name, importList, roots)

	formatted, err := imports.Process(DynamicFile, source, &imports.Options{AllErrors: true, Comments: true})
	if err != nil {
		return source, fmt.Errorf("packed: the generated source does not format: %w", err)
	}
	return formatted, nil
}

func embedsDynamic(structure *types.Struct) bool {
	for index := range structure.NumFields() {
		field := structure.Field(index)
		if field.Embedded() && isDynamic(field.Type()) {
			return true
		}
	}
	return false
}

func isDynamic(t types.Type) bool {
	return isNamed(t, packedPath, "Dynamic")
}

func isTime(t types.Type) bool {
	return isNamed(t, "time", "Time")
}

func isNamed(t types.Type, path, name string) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == path && named.Obj().Name() == name
}

type walker struct {
	files *token.FileSet
	pkg   *types.Package
	// structs holds the layout of every struct type reached, named or not.
	structs typeutil.Map
	// functions are the helper names taken.
	functions map[string]bool
	// imports maps a path to its name in the generated file, names back,
	// and packageNames a path to its package's own name.
	imports      map[string]string
	names        map[string]string
	packageNames map[string]string
	// edges lead from each struct to the structs its fields reach.
	edges []edge
	// current is the struct whose fields are being walked, pointers the
	// pointers on the way from its field to here.
	current  *packed.Layout
	pointers int

	compileError error
	errors       []error
}

type edge struct {
	from, to *packed.Layout
	pointer  bool
	path     string
	position token.Pos
}

func (w *walker) fail(position token.Pos, path string, message string) {
	if position.IsValid() {
		location := w.files.Position(position)
		if working, err := os.Getwd(); err == nil {
			if relative, err := filepath.Rel(working, location.Filename); err == nil {
				location.Filename = relative
			}
		}
		w.errors = append(w.errors, fmt.Errorf("%s: %s: %s", location, path, message))
		return
	}
	w.errors = append(w.errors, fmt.Errorf("%s: %s", path, message))
}

// walk lays out t, reached by path (the root's name and the fields to it).
// It returns nil after reporting a type that can't travel.
func (w *walker) walk(t types.Type, path string, position token.Pos) *packed.Layout {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		object := t.Obj()

		if isTime(t) {
			return &packed.Layout{Kind: packed.LayoutTime, Type: w.typeString(t)}
		}
		if object.Pkg() != nil && object.Pkg() != w.pkg && !object.Exported() {
			w.fail(position, path, fmt.Sprintf("%s is unexported, so the generated file of package %s can't name it; export it", w.typeString(t), w.pkg.Name()))
			return nil
		}
		if object.Pkg() == w.pkg && local.MatchString(object.Name()) {
			w.fail(position, path, fmt.Sprintf("the type's name %s is a variable of the generated code, which would hide it; rename the type", object.Name()))
			return nil
		}

		if structure, ok := t.Underlying().(*types.Struct); ok {
			if definedAsTime(structure) {
				return &packed.Layout{Kind: packed.LayoutTime, Type: w.typeString(t), Name: object.Name()}
			}
			return w.structure(t, structure, path, position)
		}

		layout := w.walk(t.Underlying(), path, position)
		if layout == nil {
			return nil
		}
		layout.Type = w.typeString(t)
		layout.Name = object.Name()
		return layout

	case *types.Basic:
		return w.basic(t, path, position)

	case *types.Slice:
		element := w.walk(t.Elem(), path+"[]", position)
		if element == nil {
			return nil
		}
		if element.MinimumSize() == 0 {
			w.fail(position, path, "its elements encode to no bytes, so their count can't be checked against the message's length; give the element type an exported field")
			return nil
		}
		return &packed.Layout{Kind: packed.LayoutSlice, Type: w.typeString(t), Element: element}

	case *types.Array:
		element := w.walk(t.Elem(), path+"[]", position)
		if element == nil {
			return nil
		}
		return &packed.Layout{Kind: packed.LayoutArray, Type: w.typeString(t), Element: element, Length: int(t.Len())}

	case *types.Map:
		if problem := keyProblem(t.Key()); problem != "" {
			w.fail(position, path+"[key]", problem)
			return nil
		}
		key := w.walk(t.Key(), path+"[key]", position)
		element := w.walk(t.Elem(), path+"[]", position)
		if key == nil || element == nil {
			return nil
		}
		if key.MinimumSize() == 0 {
			w.fail(position, path+"[key]", "the key encodes to no bytes, so its order can't be checked; use a key with an exported field")
			return nil
		}
		return &packed.Layout{Kind: packed.LayoutMap, Type: w.typeString(t), Key: key, Element: element}

	case *types.Pointer:
		w.pointers++
		element := w.walk(t.Elem(), path, position)
		w.pointers--
		if element == nil {
			return nil
		}
		return &packed.Layout{Kind: packed.LayoutPointer, Type: w.typeString(t), Element: element}

	case *types.Struct:
		for index := range t.NumFields() {
			if field := t.Field(index); !field.Exported() && field.Pkg() != w.pkg {
				w.fail(position, path, fmt.Sprintf("an unnamed struct of package %s with unexported fields can't be written outside it; give the struct a name there", field.Pkg().Name()))
				return nil
			}
		}
		return w.structure(t, t, path, position)

	case *types.Interface:
		w.fail(position, path, "an interface can't travel, as the decoder can't know which type it holds; use a concrete type, or skip the field with `packed:\"-\"`")
	case *types.Chan:
		w.fail(position, path, "a channel can't travel; skip the field with `packed:\"-\"`")
	case *types.Signature:
		w.fail(position, path, "a function can't travel; skip the field with `packed:\"-\"`")
	default:
		w.fail(position, path, fmt.Sprintf("%s can't travel; skip the field with `packed:\"-\"`", t))
	}
	return nil
}

var basicKinds = map[types.BasicKind]packed.LayoutKind{
	types.Bool:    packed.LayoutBool,
	types.Int:     packed.LayoutInt,
	types.Int8:    packed.LayoutInt8,
	types.Int16:   packed.LayoutInt16,
	types.Int32:   packed.LayoutInt32,
	types.Int64:   packed.LayoutInt64,
	types.Uint:    packed.LayoutUint,
	types.Uint8:   packed.LayoutUint8,
	types.Uint16:  packed.LayoutUint16,
	types.Uint32:  packed.LayoutUint32,
	types.Uint64:  packed.LayoutUint64,
	types.Float32: packed.LayoutFloat32,
	types.Float64: packed.LayoutFloat64,
	types.String:  packed.LayoutString,
}

func (w *walker) basic(t *types.Basic, path string, position token.Pos) *packed.Layout {
	if kind, ok := basicKinds[t.Kind()]; ok {
		return &packed.Layout{Kind: kind, Type: t.Name()}
	}

	switch t.Kind() {
	case types.Complex64, types.Complex128:
		w.fail(position, path, "a complex number has no layout; use two float64 fields, or skip the field with `packed:\"-\"`")
	case types.Uintptr, types.UnsafePointer:
		w.fail(position, path, "an address means nothing in another process; use a uint64, or skip the field with `packed:\"-\"`")
	case types.Invalid:
		message := "the type does not compile"
		if w.compileError != nil {
			message += ": " + w.compileError.Error()
		}
		w.fail(position, path, message)
	default:
		w.fail(position, path, fmt.Sprintf("%s can't travel", t))
	}
	return nil
}

// structure lays out a struct, named or not, once: a struct reached again
// is the same layout, which is how a type holds itself through a slice.
func (w *walker) structure(t types.Type, structure *types.Struct, path string, position token.Pos) *packed.Layout {
	if existing, ok := w.structs.At(t).(*packed.Layout); ok {
		w.edge(existing, path, position)
		return existing
	}

	layout := &packed.Layout{Kind: packed.LayoutStruct, Type: w.typeString(t), Function: w.function(t, path)}
	if named, ok := t.(*types.Named); ok {
		layout.Name = named.Obj().Name()
	}
	w.structs.Set(t, layout)
	w.edge(layout, path, position)

	current, pointers := w.current, w.pointers
	w.current, w.pointers = layout, 0
	defer func() { w.current, w.pointers = current, pointers }()

	for index := range structure.NumFields() {
		field := structure.Field(index)
		if field.Embedded() && isDynamic(field.Type()) || !field.Exported() {
			continue
		}

		fieldPath := path + "." + field.Name()
		if tag, ok := reflect.StructTag(structure.Tag(index)).Lookup("packed"); ok {
			if tag == "-" {
				continue
			}
			w.fail(field.Pos(), fieldPath, fmt.Sprintf("the packed tag %q is unknown; `packed:\"-\"` skips a field", tag))
			continue
		}

		if fieldLayout := w.walk(field.Type(), fieldPath, field.Pos()); fieldLayout != nil {
			layout.Fields = append(layout.Fields, packed.LayoutField{Name: field.Name(), Layout: fieldLayout})
		}
	}

	return layout
}

func (w *walker) edge(to *packed.Layout, path string, position token.Pos) {
	if w.current != nil {
		w.edges = append(w.edges, edge{from: w.current, to: to, pointer: w.pointers > 0, path: path, position: position})
	}
}

// checkCycles reports every pointer on a cycle of structs: a value of such
// a type can hold itself, which one pass can't write.
func (w *walker) checkCycles() {
	following := map[*packed.Layout][]*packed.Layout{}
	for _, edge := range w.edges {
		following[edge.from] = append(following[edge.from], edge.to)
	}

	// Tarjan's strongly connected components.
	index := 0
	indices := map[*packed.Layout]int{}
	lowest := map[*packed.Layout]int{}
	onStack := map[*packed.Layout]bool{}
	component := map[*packed.Layout]int{}
	var stack []*packed.Layout
	components := 0

	var connect func(layout *packed.Layout)
	connect = func(layout *packed.Layout) {
		indices[layout], lowest[layout] = index, index
		index++
		stack = append(stack, layout)
		onStack[layout] = true

		for _, next := range following[layout] {
			if _, seen := indices[next]; !seen {
				connect(next)
				lowest[layout] = min(lowest[layout], lowest[next])
			} else if onStack[next] {
				lowest[layout] = min(lowest[layout], indices[next])
			}
		}

		if lowest[layout] == indices[layout] {
			for {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[top] = false
				component[top] = components
				if top == layout {
					break
				}
			}
			components++
		}
	}

	for _, edge := range w.edges {
		if _, seen := indices[edge.from]; !seen {
			connect(edge.from)
		}
	}

	for _, edge := range w.edges {
		if edge.pointer && component[edge.from] == component[edge.to] {
			name := edge.to.Type
			w.fail(edge.position, edge.path, fmt.Sprintf("%s holds itself through this pointer, so a value of it can be cyclic, which one pass can't write; hold the values in a slice ([]%s), or skip the field with `packed:\"-\"`", name, name))
		}
	}
}

// keyProblem says why a map key's type can't travel, or "".
func keyProblem(t types.Type) string {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if isTime(t) {
			return "a time.Time key compares its location and monotonic clock as well as its instant; use its Unix nanoseconds (an int64)"
		}
		return keyProblem(t.Underlying())
	case *types.Basic:
		if t.Info()&(types.IsFloat|types.IsComplex) != 0 {
			return "a key holding a float has no single encoding (NaN is unequal to itself, -0 equals 0); use an integer or a string"
		}
	case *types.Pointer:
		return "a key holding a pointer compares by address; use the value it points to"
	case *types.Interface:
		return "a key holding an interface can't travel; use a concrete type"
	case *types.Chan:
		return "a key holding a channel can't travel"
	case *types.Array:
		return keyProblem(t.Elem())
	case *types.Struct:
		for index := range t.NumFields() {
			field := t.Field(index)
			tag := reflect.StructTag(t.Tag(index)).Get("packed")
			if !field.Exported() || tag == "-" || field.Embedded() && isDynamic(field.Type()) {
				return fmt.Sprintf("the key's field %s isn't encoded, so keys that differ only there would collide; encode it or use another key", field.Name())
			}
			if problem := keyProblem(field.Type()); problem != "" {
				return problem
			}
		}
	}
	return ""
}

// definedAsTime reports whether structure is time.Time's: the underlying
// struct of a type defined as time.Time, whose unexported fields only
// package time can declare.
func definedAsTime(structure *types.Struct) bool {
	if structure.NumFields() != 3 {
		return false
	}
	for index, name := range []string{"wall", "ext", "loc"} {
		field := structure.Field(index)
		if field.Name() != name || field.Pkg() == nil || field.Pkg().Path() != "time" {
			return false
		}
	}
	return true
}

func (w *walker) typeString(t types.Type) string {
	return types.TypeString(t, w.qualifier)
}

// qualifier names a package in the generated file, importing it.
func (w *walker) qualifier(other *types.Package) string {
	if other == w.pkg {
		return ""
	}
	path := other.Path()
	if name, ok := w.imports[path]; ok {
		return name
	}

	base := other.Name()
	if local.MatchString(base) {
		base += "Package"
	}
	name := base
	for number := 2; ; number++ {
		reservedPath, isReserved := reserved[name]
		takenPath, isTaken := w.names[name]
		if (!isReserved || reservedPath == path) && (!isTaken || takenPath == path) {
			break
		}
		name = base + strconv.Itoa(number)
	}

	w.imports[path] = name
	w.names[name] = path
	w.packageNames[path] = other.Name()
	return name
}

// function names a struct's helpers: its name, with its package's for
// another package's type, or the path to an unnamed struct.
func (w *walker) function(t types.Type, path string) string {
	name := path
	if named, ok := t.(*types.Named); ok {
		name = w.typeString(named)
	}

	// "other.Page[Item]" becomes "OtherPageItem"
	var builder strings.Builder
	capital := true
	for _, character := range name {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			capital = true
			continue
		}
		if capital {
			character = unicode.ToUpper(character)
			capital = false
		}
		builder.WriteRune(character)
	}
	name = builder.String()

	function := name
	for number := 2; w.functions[function]; number++ {
		function = name + strconv.Itoa(number)
	}
	w.functions[function] = true
	return function
}

func (w *walker) functionNames() []string {
	names := make([]string, 0, len(w.functions))
	for name := range w.functions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
