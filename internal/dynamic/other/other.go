// Package other holds types of another package that the dynamic fixtures
// reach: the generated file in package dynamic names them and writes
// their helpers.
package other

type Zone int16

type Location struct {
	City        string
	Coordinates [2]float64
	Zone        Zone
	Tags        map[string]string
	internal    int
}

// Page is generic: package dynamic holds a Page[Point].
type Page[Item any] struct {
	Items []Item
	Total int
	Next  *Page[Item] `packed:"-"`
}
