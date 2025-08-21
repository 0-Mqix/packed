package packed

import (
	"reflect"
	"testing"
)

func TestSimple(t *testing.T) {

	b := C{
		A: [2][2]B{
			{{A: [2][2][2]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}}, B: [2]A{{A: 1, B: 2}, {A: 3, B: 4}}, C: 1.0, D: "1"}, {A: [2][2][2]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}}, B: [2]A{{A: 1, B: 2}, {A: 3, B: 4}}, C: 1.0, D: "1"}},
			{{A: [2][2][2]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}}, B: [2]A{{A: 1, B: 2}, {A: 3, B: 4}}, C: 1.0, D: "1"}, {A: [2][2][2]int32{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}}, B: [2]A{{A: 1, B: 2}, {A: 3, B: 4}}, C: 1.0, D: "1"}},
		},
	}

	bytes := make([]byte, b.Size()+10)

	b.ToBytes(bytes, 0)
	var result C
	result.FromBytes(bytes, 5)

	if !reflect.DeepEqual(b, result) {
		t.Errorf("C: expected %v, got %v", b, result)
	}
}
