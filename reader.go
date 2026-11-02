package packed

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

// The dynamic layer's runtime: what the code generated for a type that
// embeds Dynamic calls to write and to read its messages.

// ErrOutOfDate is the error of a message written for another version of
// its type: its schema hash is not the reader's. errors.Is finds it in the
// error Decode<Type> returns.
var ErrOutOfDate = errors.New("packed: out of date")

// EncodeError is a value the dynamic layer can't write: a time Unix
// nanoseconds can't hold, or a string, slice or map longer than its Uint32
// length holds. AppendPacked returns it.
type EncodeError struct {
	Message string
}

func (err *EncodeError) Error() string { return "packed: " + err.Message }

// RecoverEncode, deferred by a generated AppendPacked, turns the
// EncodeError writing a value panicked with into err, so one bad value
// fails its message instead of the program. Any other panic goes on.
func RecoverEncode(err *error) {
	if value := recover(); value != nil {
		encodeError, ok := value.(*EncodeError)
		if !ok {
			panic(value)
		}
		*err = encodeError
	}
}

// nilCount is the count a nil slice or map is written with, kept apart from
// an empty one.
const nilCount = math.MaxUint32

// Limits bound what decoding one message may take.
type Limits struct {
	// Depth is how deeply structs may nest.
	Depth int
	// Size is the longest message, and the most memory the decoded values
	// may take: a count is charged its elements' Go size before they are
	// allocated, a string its length, a pointer its value's size.
	Size int
}

// DefaultLimits are the limits of Decode<Type>.
var DefaultLimits = Limits{Depth: 100, Size: 64 << 20}

// Reader reads one message of the dynamic layer. Every read is
// bounds-checked; after the first failure every read returns a zero value
// and Err reports that failure, so the generated decoder runs to its end
// without a check after each read, and never panics.
type Reader struct {
	bytes  []byte
	index  int
	depth  int
	memory int
	limits Limits
	err    error
}

// NewReader starts reading the message in bytes, which must start with
// schema; a message that starts with another is ErrOutOfDate.
func NewReader(bytes []byte, schema uint64) *Reader {
	reader := &Reader{bytes: bytes, limits: DefaultLimits, memory: DefaultLimits.Size}

	if len(bytes) > reader.limits.Size {
		reader.fail(fmt.Errorf("packed: a message of %d bytes is longer than the limit of %d", len(bytes), reader.limits.Size))
		return reader
	}

	if found, ok := reader.uint64(); ok && found != schema {
		reader.fail(fmt.Errorf("%w: the message's schema is %016x, this build reads %016x", ErrOutOfDate, found, schema))
	}

	return reader
}

// Err is the first failure, or nil.
func (r *Reader) Err() error { return r.err }

// Finish is the first failure, or the bytes left after the message.
func (r *Reader) Finish() error {
	if r.err == nil && r.index != len(r.bytes) {
		r.err = fmt.Errorf("packed: %d bytes after the message", len(r.bytes)-r.index)
	}
	return r.err
}

func (r *Reader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

func (r *Reader) left() int { return len(r.bytes) - r.index }

func (r *Reader) truncated(size int) {
	r.fail(fmt.Errorf("packed: the message ends %d bytes into the %d at byte %d", r.left(), size, r.index))
}

// Take consumes size bytes: the message's bytes and the index they start
// at, which the caller reads with packed's converters and bit runs.
func (r *Reader) Take(size int) (bytes []byte, index int, ok bool) {
	if r.err != nil {
		return nil, 0, false
	}
	if size > r.left() {
		r.truncated(size)
		return nil, 0, false
	}
	index = r.index
	r.index += size
	return r.bytes, index, true
}

func (r *Reader) uint32() (uint32, bool) {
	bytes, index, ok := r.Take(4)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint32(bytes[index:]), true
}

func (r *Reader) uint64() (uint64, bool) {
	bytes, index, ok := r.Take(8)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint64(bytes[index:]), true
}

// charge takes memory bytes from what the decoded values may still take.
func (r *Reader) charge(count int, memory uintptr) bool {
	if memory != 0 && uint64(count) > uint64(r.memory)/uint64(memory) {
		r.fail(fmt.Errorf("packed: the message's values at byte %d take more than the limit of %d bytes", r.index, r.limits.Size))
		return false
	}
	r.memory -= count * int(memory)
	return true
}

// Count reads a slice's or a map's count: present is false for nil. It
// refuses a count whose elements, each at least minimum bytes, can't fit in
// the bytes left, and charges memory bytes for each, before the caller
// allocates them.
func (r *Reader) Count(minimum int, memory uintptr) (count int, present bool) {
	start := r.index
	raw, ok := r.uint32()
	if !ok || raw == nilCount {
		return 0, false
	}
	if uint64(raw) > uint64(r.left())/uint64(max(minimum, 1)) {
		r.fail(fmt.Errorf("packed: the count %d at byte %d has elements of at least %d bytes, and %d bytes are left", raw, start, minimum, r.left()))
		return 0, false
	}
	if !r.charge(int(raw), memory) {
		return 0, false
	}
	return int(raw), true
}

// ReadString reads a string: its length, a Uint32, and its bytes.
func (r *Reader) ReadString() string {
	start := r.index
	length, ok := r.uint32()
	if !ok {
		return ""
	}
	if uint64(length) > uint64(r.left()) {
		r.fail(fmt.Errorf("packed: the string of %d bytes at byte %d is longer than the %d bytes left", length, start, r.left()))
		return ""
	}
	if !r.charge(int(length), 1) {
		return ""
	}
	value := string(r.bytes[r.index : r.index+int(length)])
	r.index += int(length)
	return value
}

// Bytes reads a []byte: its count, a Uint32 that is 0xFFFFFFFF for nil, and
// its bytes.
func (r *Reader) Bytes() []byte {
	count, present := r.Count(1, 1)
	if !present {
		return nil
	}
	value := make([]byte, count)
	r.index += copy(value, r.bytes[r.index:])
	return value
}

// Present reads the presence of a pointer that is no struct's field: a run
// of one bit.
func (r *Reader) Present() bool {
	bytes, index, ok := r.Take(1)
	return ok && bytes[index]&1 != 0
}

// Allocate charges memory bytes for a pointer's value.
func (r *Reader) Allocate(memory uintptr) bool {
	return r.err == nil && r.charge(1, memory)
}

// Enter starts a struct, false when it nests deeper than the limit.
func (r *Reader) Enter() bool {
	if r.err != nil {
		return false
	}
	r.depth++
	if r.depth > r.limits.Depth {
		r.fail(fmt.Errorf("packed: the message nests deeper than %d at byte %d", r.limits.Depth, r.index))
		return false
	}
	return true
}

// Leave ends a struct Enter started.
func (r *Reader) Leave() { r.depth-- }

// Position is where the next read starts.
func (r *Reader) Position() int { return r.index }

// Ascending checks that the map key read since start comes after the
// previous key in its encoded bytes, as the encoder sorts them.
func (r *Reader) Ascending(previous *[]byte, start int) {
	if r.err != nil {
		return
	}
	key := r.bytes[start:r.index]
	if *previous != nil && bytes.Compare(*previous, key) >= 0 {
		r.fail(fmt.Errorf("packed: the map key at byte %d does not come after the one before it", start))
		return
	}
	*previous = key
}

// Unique checks that a map of count entries holds count keys: two
// encodings of one key make it smaller.
func (r *Reader) Unique(length int, count int) {
	if r.err == nil && length != count {
		r.fail(fmt.Errorf("packed: a map before byte %d holds a key twice", r.index))
	}
}

// Overflow fails on a number that does not fit its Go type: an int or a
// uint on a platform where they are 32 bits.
func (r *Reader) Overflow() {
	r.fail(fmt.Errorf("packed: a number before byte %d does not fit this platform's int", r.index))
}

// AppendSchema starts a message with its schema hash.
func AppendSchema(bytes []byte, schema uint64) []byte {
	return binary.LittleEndian.AppendUint64(bytes, schema)
}

// AppendCount appends a slice's or a map's count, or 0xFFFFFFFF when it is
// nil. It panics with an EncodeError on a count the Uint32 can't hold.
func AppendCount(bytes []byte, count int, isNil bool) []byte {
	if isNil {
		return binary.LittleEndian.AppendUint32(bytes, nilCount)
	}
	if uint64(count) >= nilCount {
		panic(&EncodeError{Message: fmt.Sprintf("%d elements are more than a Uint32 count holds", count)})
	}
	return binary.LittleEndian.AppendUint32(bytes, uint32(count))
}

// AppendString appends value's length, a Uint32, and its bytes. It panics
// with an EncodeError on a string the Uint32 can't hold.
func AppendString(bytes []byte, value string) []byte {
	if uint64(len(value)) > math.MaxUint32 {
		panic(&EncodeError{Message: fmt.Sprintf("a string of %d bytes is longer than a Uint32 length holds", len(value))})
	}
	bytes = binary.LittleEndian.AppendUint32(bytes, uint32(len(value)))
	return append(bytes, value...)
}

// AppendBytes appends a []byte: its count, or 0xFFFFFFFF for nil, and its
// bytes.
func AppendBytes(bytes []byte, value []byte) []byte {
	bytes = AppendCount(bytes, len(value), value == nil)
	return append(bytes, value...)
}

// Entry is a map's key and its encoded bytes.
type Entry[Key comparable] struct {
	Key   Key
	Bytes []byte
}

// SortedKeys returns the keys of entries sorted by their encoded bytes, as
// appendKey writes them, so a map always gives the same bytes.
func SortedKeys[Key comparable, Value any](entries map[Key]Value, appendKey func([]byte, Key) []byte) []Entry[Key] {
	keys := make([]Entry[Key], 0, len(entries))
	ends := make([]int, 0, len(entries))
	var encoded []byte

	for key := range entries {
		encoded = appendKey(encoded, key)
		keys = append(keys, Entry[Key]{Key: key})
		ends = append(ends, len(encoded))
	}

	start := 0
	for index, end := range ends {
		keys[index].Bytes = encoded[start:end:end]
		start = end
	}

	slices.SortFunc(keys, func(a, b Entry[Key]) int { return bytes.Compare(a.Bytes, b.Bytes) })
	return keys
}

// TimeConverter writes a time.Time as its Unix nanoseconds, an Int64, and
// reads it back in UTC. The zero Time is math.MinInt64, so it reads back as
// the zero Time; writing a time outside the years 1678 to 2262, which Unix
// nanoseconds can't hold, panics with an EncodeError, which the dynamic
// layer's AppendPacked returns.
type TimeConverter struct{}

func (TimeConverter) Size() int { return 8 }

func timeNanoseconds(value time.Time) int64 {
	if value.IsZero() {
		return math.MinInt64
	}
	if value.Before(earliestTime) || value.After(latestTime) {
		panic(&EncodeError{Message: fmt.Sprintf("%s is outside the years 1678 to 2262 that Unix nanoseconds hold", value)})
	}
	return value.UnixNano()
}

var (
	earliestTime = time.Unix(0, math.MinInt64+1)
	latestTime   = time.Unix(0, math.MaxInt64)
)

func timeFromNanoseconds(nanoseconds int64) time.Time {
	if nanoseconds == math.MinInt64 {
		return time.Time{}
	}
	return time.Unix(0, nanoseconds).UTC()
}

func (TimeConverter) ToBytesLittleEndian(value *time.Time, bytes []byte, index int) {
	binary.LittleEndian.PutUint64(bytes[index:index+8], uint64(timeNanoseconds(*value)))
}

func (TimeConverter) FromBytesLittleEndian(receiver *time.Time, bytes []byte, index int) {
	*receiver = timeFromNanoseconds(int64(binary.LittleEndian.Uint64(bytes[index : index+8])))
}

func (TimeConverter) ToBytesBigEndian(value *time.Time, bytes []byte, index int) {
	binary.BigEndian.PutUint64(bytes[index:index+8], uint64(timeNanoseconds(*value)))
}

func (TimeConverter) FromBytesBigEndian(receiver *time.Time, bytes []byte, index int) {
	*receiver = timeFromNanoseconds(int64(binary.BigEndian.Uint64(bytes[index : index+8])))
}
