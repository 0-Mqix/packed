package packed

// Dynamic marks a type that travels in packed's dynamic layer. Embedded in
// a struct, it makes `go run github.com/0-Mqix/packed/dynamic` give
// the struct `<Type>Schema`, `AppendPacked` and `Decode<Type>`, laid out
// from its Go fields. It takes no space and is no field of the encoding.
type Dynamic struct{}

func (Dynamic) dynamic() {}

// Marked is the interface of the types that embed Dynamic: a function that
// takes or returns one only compiles with a type that travels.
type Marked interface {
	dynamic()
}
