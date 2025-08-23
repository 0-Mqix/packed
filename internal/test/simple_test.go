package packed

import (
	"reflect"
	"testing"
)

func TestSimple(t *testing.T) {

	b := A{
		A: 1,
		B: 2,
		C: "3",
		D: 4,
		E: 5,
		F: "6",
	}

	bytes := make([]byte, b.Size())

	b.ToBytes(bytes, 0)

	var result A
	result.FromBytes(bytes, 0)

	if !reflect.DeepEqual(b, result) {
		t.Errorf("A: expected %v, got %v", b, result)
	}
}
