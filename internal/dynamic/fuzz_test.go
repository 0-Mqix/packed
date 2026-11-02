package dynamic

import (
	"bytes"
	"reflect"
	"testing"
)

// The fuzz tests decode hostile messages: decoding never panics, a failure
// gives the zero value, and what decodes encodes to bytes that decode to
// the same value and encode to the same bytes again.
//
//	go test ./internal/dynamic -fuzz FuzzDevice -fuzztime 1m

func FuzzDevice(f *testing.F) {
	for _, device := range samples {
		f.Add(must(device.AppendPacked(nil)))
	}
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, message []byte) {
		device, err := DecodeDevice(message)
		if err != nil {
			if !reflect.DeepEqual(device, Device{}) {
				t.Fatalf("a failed decode returned %+v", device)
			}
			return
		}
		checkStable(t, must(device.AppendPacked(nil)), func(message []byte) ([]byte, error) {
			device, err := DecodeDevice(message)
			return must(device.AppendPacked(nil)), err
		})
	})
}

func FuzzTree(f *testing.F) {
	f.Add(must(Tree{Value: 1, Children: []Tree{{Value: 2, Children: []Tree{{}}}, {}}}.AppendPacked(nil)))
	f.Add(must(Tree{}.AppendPacked(nil)))

	f.Fuzz(func(t *testing.T, message []byte) {
		tree, err := DecodeTree(message)
		if err != nil {
			if !reflect.DeepEqual(tree, Tree{}) {
				t.Fatalf("a failed decode returned %+v", tree)
			}
			return
		}
		checkStable(t, must(tree.AppendPacked(nil)), func(message []byte) ([]byte, error) {
			tree, err := DecodeTree(message)
			return must(tree.AppendPacked(nil)), err
		})
	})
}

func FuzzSmall(f *testing.F) {
	f.Add(must(Small{Active: true, Name: "ab", Values: []int32{1}}.AppendPacked(nil)))

	f.Fuzz(func(t *testing.T, message []byte) {
		small, err := DecodeSmall(message)
		if err != nil {
			return
		}
		checkStable(t, must(small.AppendPacked(nil)), func(message []byte) ([]byte, error) {
			small, err := DecodeSmall(message)
			return must(small.AppendPacked(nil)), err
		})
	})
}

// must is a message of a value the tests know encodes.
func must(message []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return message
}

// checkStable decodes encoded and encodes it again: the same bytes. (The
// value itself may hold a NaN, which no value equals.)
func checkStable(t *testing.T, encoded []byte, roundTrip func([]byte) ([]byte, error)) {
	t.Helper()
	again, err := roundTrip(encoded)
	if err != nil {
		t.Fatalf("an encoded value does not decode: %v\n% x", err, encoded)
	}
	if !bytes.Equal(again, encoded) {
		t.Fatalf("an encoded value encodes to other bytes after decoding:\n% x\n% x", encoded, again)
	}
}
