package array

import (
	"fmt"

	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// StringArray stores string values using an offset-based layout for cache
// locality and reduced GC pressure. All string bytes are concatenated in a
// single byte slice, with int32 offsets marking boundaries.
type StringArray struct {
	data     []byte
	offsets  []int32
	validity *bitmap.Bitmap
	length   int
}

// NewStringArray creates a new StringArray from a slice of Go strings.
// If validity is nil, all elements are considered valid.
func NewStringArray(values []string, validity *bitmap.Bitmap) *StringArray {
	totalBytes := 0
	for _, s := range values {
		totalBytes += len(s)
	}

	data := make([]byte, 0, totalBytes)
	offsets := make([]int32, len(values)+1)
	offsets[0] = 0

	for i, s := range values {
		data = append(data, s...)
		offsets[i+1] = int32(len(data))
	}

	return &StringArray{
		data:     data,
		offsets:  offsets,
		validity: validity,
		length:   len(values),
	}
}

// NewStringArrayFromBytes creates a StringArray from raw offset-based storage.
func NewStringArrayFromBytes(data []byte, offsets []int32, validity *bitmap.Bitmap) *StringArray {
	return &StringArray{
		data:     data,
		offsets:  offsets,
		validity: validity,
		length:   len(offsets) - 1,
	}
}

// DataType returns String.
func (a *StringArray) DataType() dtype.DataType { return dtype.String }

// Len returns the number of elements.
func (a *StringArray) Len() int { return a.length }

// IsNull returns true if the element at index i is null.
func (a *StringArray) IsNull(i int) bool {
	if a.validity == nil {
		return false
	}
	return !a.validity.IsSet(i)
}

// IsValid returns true if the element at index i is not null.
func (a *StringArray) IsValid(i int) bool {
	if a.validity == nil {
		return true
	}
	return a.validity.IsSet(i)
}

// NullCount returns the number of null elements.
func (a *StringArray) NullCount() int {
	if a.validity == nil {
		return 0
	}
	return a.length - a.validity.PopCount()
}

// Validity returns the null bitmap.
func (a *StringArray) Validity() *bitmap.Bitmap { return a.validity }

// Value returns the string at index i. The caller should check IsNull first.
func (a *StringArray) Value(i int) string {
	start := a.offsets[i]
	end := a.offsets[i+1]
	return string(a.data[start:end])
}

// ValueBytes returns the raw bytes of the string at index i without allocation.
func (a *StringArray) ValueBytes(i int) []byte {
	start := a.offsets[i]
	end := a.offsets[i+1]
	return a.data[start:end]
}

// Slice returns a new StringArray for the range [start, end).
func (a *StringArray) Slice(start, end int) Array {
	byteStart := a.offsets[start]
	newOffsets := make([]int32, end-start+1)
	for i := start; i <= end; i++ {
		newOffsets[i-start] = a.offsets[i] - byteStart
	}
	byteEnd := a.offsets[end]

	var v *bitmap.Bitmap
	if a.validity != nil {
		v = a.validity.Slice(start, end)
	}

	return &StringArray{
		data:     a.data[byteStart:byteEnd],
		offsets:  newOffsets,
		validity: v,
		length:   end - start,
	}
}

// String returns a human-readable representation.
func (a *StringArray) String() string {
	return fmt.Sprintf("String[%d]", a.length)
}

// TotalBytes returns the total number of bytes used by all string values.
func (a *StringArray) TotalBytes() int {
	return len(a.data)
}

// Offsets returns the raw offsets slice.
func (a *StringArray) Offsets() []int32 { return a.offsets }

// Data returns the raw byte data.
func (a *StringArray) Data() []byte { return a.data }

// BinaryArray stores binary (arbitrary bytes) values using the same offset-based
// layout as StringArray.
type BinaryArray = StringArray

// NewBinaryArray creates a new BinaryArray from byte slices.
func NewBinaryArray(values [][]byte, validity *bitmap.Bitmap) *BinaryArray {
	totalBytes := 0
	for _, b := range values {
		totalBytes += len(b)
	}

	data := make([]byte, 0, totalBytes)
	offsets := make([]int32, len(values)+1)
	offsets[0] = 0

	for i, b := range values {
		data = append(data, b...)
		offsets[i+1] = int32(len(data))
	}

	arr := &StringArray{
		data:     data,
		offsets:  offsets,
		validity: validity,
		length:   len(values),
	}
	return arr
}
