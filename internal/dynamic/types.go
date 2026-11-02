// Package dynamic holds the dynamic layer's fixtures: Go types that travel,
// the file generated for them (packed_dynamic.go, checked in) and their
// tests.
package dynamic

import (
	"time"

	"github.com/0-Mqix/packed"
	"github.com/0-Mqix/packed/internal/dynamic/other"
)

//go:generate go run github.com/0-Mqix/packed/dynamic

type (
	Status     uint8
	Identifier int
	Label      string
	Tags       []string
	Celsius    float64
	Deadline   time.Time
	Flag       bool
)

// Device reaches every kind of field the layer lays out.
type Device struct {
	packed.Dynamic

	ID       Identifier
	Name     string
	Online   bool
	Muted    Flag
	Owner    *User
	Backup   *User
	Archived bool
	Status   Status

	Tags        Tags
	Labels      []Label
	Readings    []float32
	Raw         []byte
	Flags       []bool
	Counts      []int
	Matrix      [2][3]int16
	Temperature Celsius
	Unsigned    uint

	Scores map[string]int64
	Groups map[Point][]string
	Lookup map[Status]*Part

	Created time.Time
	Due     Deadline
	Seen    *time.Time

	Children []Device
	Parts    [2]Part
	Nested   [][]Point
	Window   *[4]uint8
	Optional **int32

	Location other.Location
	Pages    other.Page[Point]
	Meta     struct {
		Version uint16
		Beta    bool
		Notes   []string
	}

	secret  int
	Skipped string `packed:"-"`
}

type User struct {
	Name  string
	Admin bool
	Since time.Time
	Email *string
}

type Point struct {
	X, Y int32
}

type Part struct {
	Serial string
	Count  uint32
	Spare  bool
}

// Small is a struct the DSL can express too (package dsl): the two give
// the same bytes.
type Small struct {
	packed.Dynamic

	Active  bool
	Visible bool
	Count   uint16
	Total   int64
	Name    string
	Ratio   float32
	Level   int8
	Values  []int32
	Inner   Inner
	Codes   [2]uint16
}

type Inner struct {
	On    bool
	Value uint32
}

// Tree holds itself through a slice, as deep as a message nests it.
type Tree struct {
	packed.Dynamic

	Value    int32
	Children []Tree
}

// Empty has no fields: its message is its schema.
type Empty struct {
	packed.Dynamic
}
