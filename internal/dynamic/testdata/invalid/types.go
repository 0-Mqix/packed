// Package invalid holds fields that can't travel: generating its dynamic
// layer fails, with one error for each.
package invalid

import (
	"time"

	"github.com/0-Mqix/packed"
)

type Invalid struct {
	packed.Dynamic

	Any      any
	Events   chan int
	Callback func()
	Ratios   map[float64]string
	Owners   map[*int]string
	Times    map[time.Time]string
	Keys     map[Key]string
	Complex  complex128
	Address  uintptr
	Tagged   string `packed:"skip"`
	Next     *Invalid
	Nothing  []struct{ hidden int }
	Total    count
	Fine     string
}

// count is a variable of the generated code.
type count int

type Key struct {
	Name   string
	hidden int
}

type Node struct {
	packed.Dynamic

	Children []*Node
}
