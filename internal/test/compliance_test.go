package packed

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

// The expected byte slices below are produced by a real C
// __attribute__((__packed__)) struct with the same fields and values, compiled
// with clang for a little-endian (x86-64) and a big-endian (aarch64_be) target
// and dumped from the object file. They are the ground truth this generator must
// match byte-for-byte. See internal scratchpad oracle for how they were derived.

func crossCheck(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("%s: ToBytes = %v\n      want C __packed__ = %v", name, got, want)
	}
}

func TestPackedBitfieldStraddleLittleEndian(t *testing.T) {
	// O: field C (signed) straddles the 64-bit word boundary.
	o := O{A: 876543, B: 0x712345678, C: -2500, D: 99, E: true}
	got := make([]byte, o.Size())
	o.ToBytes(got, 0)
	crossCheck(t, "O", got, []byte{255, 95, 141, 103, 69, 35, 113, 30, 59, 14})

	var back O
	back.FromBytes(got, 0)
	if !reflect.DeepEqual(o, back) {
		t.Errorf("O round-trip: got %+v want %+v", back, o)
	}
}

func TestPackedBitfieldStraddleBigEndian(t *testing.T) {
	// P: big-endian mirror of O.
	p := P{A: 1000000, B: 0x5AAAABBBB, C: -1234, D: 100, E: false}
	got := make([]byte, p.Size())
	p.ToBytes(got, 0)
	crossCheck(t, "P", got, []byte{244, 36, 11, 85, 85, 119, 119, 178, 236, 128})

	var back P
	back.FromBytes(got, 0)
	if !reflect.DeepEqual(p, back) {
		t.Errorf("P round-trip: got %+v want %+v", back, p)
	}
}

func TestPackedBitfieldThreeWordsLittleEndian(t *testing.T) {
	// Q: 145-bit run over three words, straddling the 64- and 128-bit boundaries.
	q := Q{A: 0x3FFFFFFFFFFFF, B: 0x2AAAAAAAAAAAA, C: 0x99887766, D: -9}
	got := make([]byte, q.Size())
	q.ToBytes(got, 0)
	crossCheck(t, "Q", got, []byte{
		255, 255, 255, 255, 255, 255, 171, 170, 170, 170,
		170, 170, 106, 118, 135, 152, 9, 112, 1,
	})

	var back Q
	back.FromBytes(got, 0)
	if !reflect.DeepEqual(q, back) {
		t.Errorf("Q round-trip: got %+v want %+v", back, q)
	}
}

func TestPackedBitfieldThreeWordsBigEndian(t *testing.T) {
	// R: big-endian mirror of Q.
	r := R{A: 0x3123456789AB, B: 0x1FEDCBA987654, C: 0xABCDEF12, D: -3}
	got := make([]byte, r.Size())
	r.ToBytes(got, 0)
	crossCheck(t, "R", got, []byte{
		12, 72, 209, 89, 226, 106, 223, 237, 203, 169,
		135, 101, 64, 10, 188, 222, 241, 46, 128,
	})

	var back R
	back.FromBytes(got, 0)
	if !reflect.DeepEqual(r, back) {
		t.Errorf("R round-trip: got %+v want %+v", back, r)
	}
}

func TestPackedBitfield77BitReproducer(t *testing.T) {
	// S/T: the original reproducer - 21 contiguous bit-fields totalling 77 bits.
	// Under C __packed__ this is 10 bytes; the old splitter produced 11.
	if s := (S{}); s.Size() != 10 {
		t.Fatalf("S.Size() = %d, want 10 (C __packed__)", s.Size())
	}
	if tt := (T{}); tt.Size() != 10 {
		t.Fatalf("T.Size() = %d, want 10 (C __packed__)", tt.Size())
	}

	s := S{
		F0: 21, F1: 30, F2: 7, F3: 19, F4: 11, F5: 28, F6: 3, F7: 25, F8: 14,
		F9: 9, F10: 31, F11: 1, F12: 100, F13: 1, F14: 0, F15: 1, F16: 1,
		F17: 2, F18: 0, F19: 1, F20: 3,
	}
	gotS := make([]byte, s.Size())
	s.ToBytes(gotS, 0)
	crossCheck(t, "S", gotS, []byte{213, 159, 185, 248, 200, 46, 253, 64, 110, 29})

	var backS S
	backS.FromBytes(gotS, 0)
	if !reflect.DeepEqual(s, backS) {
		t.Errorf("S round-trip: got %+v want %+v", backS, s)
	}

	tv := T{
		F0: 21, F1: 30, F2: 7, F3: 19, F4: 11, F5: 28, F6: 3, F7: 25, F8: 14,
		F9: 9, F10: 31, F11: 1, F12: 100, F13: 1, F14: 0, F15: 1, F16: 1,
		F17: 2, F18: 0, F19: 1, F20: 3,
	}
	gotT := make([]byte, tv.Size())
	tv.ToBytes(gotT, 0)
	crossCheck(t, "T", gotT, []byte{175, 143, 53, 240, 121, 114, 126, 28, 151, 56})

	var backT T
	backT.FromBytes(gotT, 0)
	if !reflect.DeepEqual(tv, backT) {
		t.Errorf("T round-trip: got %+v want %+v", backT, tv)
	}
}

func TestPackedBitfieldNonZeroIndex(t *testing.T) {
	// ToBytes/FromBytes must honour a non-zero base index for straddling runs.
	o := O{A: 876543, B: 0x712345678, C: -2500, D: 99, E: true}
	buffer := make([]byte, o.Size()+5)
	o.ToBytes(buffer, 3)

	want := []byte{255, 95, 141, 103, 69, 35, 113, 30, 59, 14}
	crossCheck(t, "O@3", buffer[3:3+o.Size()], want)

	var back O
	back.FromBytes(buffer, 3)
	if !reflect.DeepEqual(o, back) {
		t.Errorf("O@3 round-trip: got %+v want %+v", back, o)
	}
}

func TestPackedBitfieldRoundTripFuzz(t *testing.T) {
	random := rand.New(rand.NewSource(1))

	for iteration := 0; iteration < 5000; iteration++ {
		// O / P: uint32:20, uint64:35, int16:13 (straddles 64), uint8:7, bool
		a := uint32(random.Intn(1 << 20))
		b := uint64(random.Int63n(1 << 35))
		c := int16(random.Intn(1<<13) - (1 << 12))
		d := uint8(random.Intn(1 << 7))
		e := random.Intn(2) == 0

		o := O{A: a, B: b, C: c, D: d, E: e}
		buffer := make([]byte, o.Size())
		o.ToBytes(buffer, 0)
		var ro O
		ro.FromBytes(buffer, 0)
		if ro != o {
			t.Fatalf("O fuzz round-trip: %+v -> %+v", o, ro)
		}

		p := P{A: a, B: b, C: c, D: d, E: e}
		p.ToBytes(buffer, 0)
		var rp P
		rp.FromBytes(buffer, 0)
		if rp != p {
			t.Fatalf("P fuzz round-trip: %+v -> %+v", p, rp)
		}

		// Q / R: uint64:50, uint64:50 (straddles 64), uint64:40 (straddles 128), int8:5
		qa := uint64(random.Int63()) & ((1 << 50) - 1)
		qb := uint64(random.Int63()) & ((1 << 50) - 1)
		qc := uint64(random.Int63()) & ((1 << 40) - 1)
		qd := int8(random.Intn(1<<5) - (1 << 4))

		q := Q{A: qa, B: qb, C: qc, D: qd}
		wide := make([]byte, q.Size())
		q.ToBytes(wide, 0)
		var rq Q
		rq.FromBytes(wide, 0)
		if rq != q {
			t.Fatalf("Q fuzz round-trip: %+v -> %+v", q, rq)
		}

		r := R{A: qa, B: qb, C: qc, D: qd}
		r.ToBytes(wide, 0)
		var rr R
		rr.FromBytes(wide, 0)
		if rr != r {
			t.Fatalf("R fuzz round-trip: %+v -> %+v", r, rr)
		}
	}
}
