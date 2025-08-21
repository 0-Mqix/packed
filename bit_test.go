package packed

import "testing"

func TestBit(t *testing.T) {

	var x uint32 = 0

	x = (x & 0xFFFFFFF0) | 6

	for i := range 32 {
		print(((x >> i) & 0x1))
	}

	println()
}
