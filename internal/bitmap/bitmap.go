// Package bitmap provides a compact bitset implementation for null tracking
// in columnar data structures. Each bit represents whether a value is valid
// (set/1) or null (clear/0).
package bitmap

import (
	"math/bits"
)

const wordSize = 64

// wordsNeeded returns the number of uint64 words needed to store n bits.
func wordsNeeded(n int) int {
	if n <= 0 {
		return 0
	}
	return (n + wordSize - 1) / wordSize
}

// Bitmap is a compact bitset that stores bits in []uint64 words.
// It is used to track null values in columnar data, where a set bit
// means "valid/not-null" and a clear bit means "null".
type Bitmap struct {
	data []uint64
	len  int
}

// New creates a Bitmap of the given length with all bits initially set,
// meaning all values are valid (not null).
func New(length int) *Bitmap {
	if length < 0 {
		length = 0
	}
	n := wordsNeeded(length)
	data := make([]uint64, n)
	for i := range data {
		data[i] = ^uint64(0)
	}
	b := &Bitmap{data: data, len: length}
	b.clearTrailingBits()
	return b
}

// NewEmpty creates a Bitmap of the given length with all bits initially clear,
// meaning all values are null.
func NewEmpty(length int) *Bitmap {
	if length < 0 {
		length = 0
	}
	n := wordsNeeded(length)
	return &Bitmap{
		data: make([]uint64, n),
		len:  length,
	}
}

// clearTrailingBits zeroes out any bits beyond the logical length in the last word.
func (b *Bitmap) clearTrailingBits() {
	if b.len == 0 {
		return
	}
	rem := b.len % wordSize
	if rem != 0 && len(b.data) > 0 {
		mask := (uint64(1) << rem) - 1
		b.data[len(b.data)-1] &= mask
	}
}

// Len returns the number of bits in the bitmap.
func (b *Bitmap) Len() int {
	return b.len
}

// Set sets bit i, marking the value at position i as valid (not null).
// Panics if i is out of range.
func (b *Bitmap) Set(i int) {
	if i < 0 || i >= b.len {
		panic("bitmap: index out of range")
	}
	word := i / wordSize
	bit := uint(i % wordSize)
	b.data[word] |= 1 << bit
}

// Clear clears bit i, marking the value at position i as null.
// Panics if i is out of range.
func (b *Bitmap) Clear(i int) {
	if i < 0 || i >= b.len {
		panic("bitmap: index out of range")
	}
	word := i / wordSize
	bit := uint(i % wordSize)
	b.data[word] &^= 1 << bit
}

// IsSet returns true if bit i is set (valid/not-null).
// Panics if i is out of range.
func (b *Bitmap) IsSet(i int) bool {
	if i < 0 || i >= b.len {
		panic("bitmap: index out of range")
	}
	word := i / wordSize
	bit := uint(i % wordSize)
	return b.data[word]&(1<<bit) != 0
}

// Clone returns an independent copy of the bitmap.
func (b *Bitmap) Clone() *Bitmap {
	data := make([]uint64, len(b.data))
	copy(data, b.data)
	return &Bitmap{
		data: data,
		len:  b.len,
	}
}

// And returns a new bitmap that is the bitwise AND of b and other.
// Both bitmaps must have the same length; panics otherwise.
func (b *Bitmap) And(other *Bitmap) *Bitmap {
	if b.len != other.len {
		panic("bitmap: And requires bitmaps of equal length")
	}
	data := make([]uint64, len(b.data))
	for i := range data {
		data[i] = b.data[i] & other.data[i]
	}
	return &Bitmap{data: data, len: b.len}
}

// Or returns a new bitmap that is the bitwise OR of b and other.
// Both bitmaps must have the same length; panics otherwise.
func (b *Bitmap) Or(other *Bitmap) *Bitmap {
	if b.len != other.len {
		panic("bitmap: Or requires bitmaps of equal length")
	}
	data := make([]uint64, len(b.data))
	for i := range data {
		data[i] = b.data[i] | other.data[i]
	}
	return &Bitmap{data: data, len: b.len}
}

// Not returns a new bitmap that is the bitwise NOT of b.
// Trailing bits beyond the logical length are kept clear.
func (b *Bitmap) Not() *Bitmap {
	data := make([]uint64, len(b.data))
	for i := range data {
		data[i] = ^b.data[i]
	}
	result := &Bitmap{data: data, len: b.len}
	result.clearTrailingBits()
	return result
}

// PopCount returns the number of set bits (valid/not-null values) in the bitmap.
func (b *Bitmap) PopCount() int {
	count := 0
	for _, w := range b.data {
		count += bits.OnesCount64(w)
	}
	return count
}

// NullCount returns the number of clear bits (null values) in the bitmap.
func (b *Bitmap) NullCount() int {
	return b.len - b.PopCount()
}

// AllSet returns true if every bit is set, meaning there are no null values.
func (b *Bitmap) AllSet() bool {
	return b.PopCount() == b.len
}

// NoneSet returns true if every bit is clear, meaning all values are null.
func (b *Bitmap) NoneSet() bool {
	return b.PopCount() == 0
}

// Slice extracts bits in the range [start, end) into a new bitmap of length end-start.
// Panics if start or end are out of range or start > end.
func (b *Bitmap) Slice(start, end int) *Bitmap {
	if start < 0 || end < 0 || start > end || end > b.len {
		panic("bitmap: slice bounds out of range")
	}
	newLen := end - start
	result := NewEmpty(newLen)
	for i := 0; i < newLen; i++ {
		if b.IsSet(start + i) {
			result.Set(i)
		}
	}
	return result
}

// SetRange sets all bits in the range [start, end).
// Panics if start or end are out of range or start > end.
func (b *Bitmap) SetRange(start, end int) {
	if start < 0 || end < 0 || start > end || end > b.len {
		panic("bitmap: range bounds out of range")
	}
	for i := start; i < end; {
		word := i / wordSize
		bit := i % wordSize
		// If we're at the start of a word and the remaining range covers the full word, set the whole word.
		if bit == 0 && i+wordSize <= end {
			b.data[word] = ^uint64(0)
			i += wordSize
		} else {
			// Set individual bits up to the end of this word or end of range.
			upper := (word + 1) * wordSize
			if upper > end {
				upper = end
			}
			// Build a mask for bits [bit, upper - word*wordSize).
			lo := uint(bit)
			hi := uint(upper - word*wordSize)
			mask := (uint64(1)<<hi - 1) &^ (uint64(1)<<lo - 1)
			b.data[word] |= mask
			i = upper
		}
	}
	b.clearTrailingBits()
}

// ClearRange clears all bits in the range [start, end).
// Panics if start or end are out of range or start > end.
func (b *Bitmap) ClearRange(start, end int) {
	if start < 0 || end < 0 || start > end || end > b.len {
		panic("bitmap: range bounds out of range")
	}
	for i := start; i < end; {
		word := i / wordSize
		bit := i % wordSize
		if bit == 0 && i+wordSize <= end {
			b.data[word] = 0
			i += wordSize
		} else {
			upper := (word + 1) * wordSize
			if upper > end {
				upper = end
			}
			lo := uint(bit)
			hi := uint(upper - word*wordSize)
			mask := (uint64(1)<<hi - 1) &^ (uint64(1)<<lo - 1)
			b.data[word] &^= mask
			i = upper
		}
	}
}

// Grow extends the bitmap to newLen bits. New bits are clear (null).
// If newLen is less than or equal to the current length, Grow is a no-op.
func (b *Bitmap) Grow(newLen int) {
	if newLen <= b.len {
		return
	}
	needed := wordsNeeded(newLen)
	if needed > len(b.data) {
		newData := make([]uint64, needed)
		copy(newData, b.data)
		b.data = newData
	}
	// Clear any bits that were trailing in the old last word beyond old len
	// (they should already be clear, but be safe).
	b.len = newLen
	b.clearTrailingBits()
}
