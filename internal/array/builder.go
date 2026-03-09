package array

import (
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// Builder is the interface for building arrays incrementally.
// Builders allow appending values one at a time or in bulk, with optional
// null tracking. Call Build to finalize the builder into an immutable array.
type Builder interface {
	// Len returns the number of elements appended so far.
	Len() int
	// Cap returns the current capacity of the builder.
	Cap() int
	// Reserve ensures there is capacity for at least additional more elements.
	Reserve(additional int)
	// AppendNull appends a null value.
	AppendNull()
	// IsNull returns true if the element at index i is null.
	IsNull(i int) bool
	// Build finalizes the builder into an Array and resets the builder.
	Build() Array
}

// TypedBuilder is a generic builder for constructing TypedArray values
// incrementally. It supports numeric and temporal element types.
type TypedBuilder[T any] struct {
	data     []T
	validity *bitmap.Bitmap
	dt       dtype.DataType
	hasNulls bool
	length   int
}

// NewTypedBuilder creates a new TypedBuilder for the given data type with
// the specified initial capacity. The builder starts empty and grows as
// values are appended.
func NewTypedBuilder[T any](dt dtype.DataType, initialCap int) *TypedBuilder[T] {
	if initialCap < 0 {
		initialCap = 0
	}
	return &TypedBuilder[T]{
		data: make([]T, 0, initialCap),
		dt:   dt,
	}
}

// Append appends a valid (non-null) value to the builder.
func (b *TypedBuilder[T]) Append(v T) {
	b.data = append(b.data, v)
	if b.hasNulls {
		b.growValidity(b.length + 1)
		b.validity.Set(b.length)
	}
	b.length++
}

// AppendNull appends a null value to the builder.
func (b *TypedBuilder[T]) AppendNull() {
	var zero T
	b.data = append(b.data, zero)
	if !b.hasNulls {
		b.hasNulls = true
		b.validity = bitmap.New(b.length + 1)
		// All previous elements were valid, so bits [0, b.length) are already set
		// by bitmap.New. Clear the new element.
		b.validity.Clear(b.length)
	} else {
		b.growValidity(b.length + 1)
		// New bit is clear by default after Grow, which is what we want for null.
	}
	b.length++
}

// AppendValues appends multiple values in bulk. The valid slice, if non-nil,
// indicates which values are valid (true) or null (false). If valid is nil,
// all values are treated as valid. When valid is provided, its length must
// match the length of values.
func (b *TypedBuilder[T]) AppendValues(values []T, valid []bool) {
	if len(values) == 0 {
		return
	}

	if valid == nil {
		b.data = append(b.data, values...)
		if b.hasNulls {
			b.growValidity(b.length + len(values))
			for i := 0; i < len(values); i++ {
				b.validity.Set(b.length + i)
			}
		}
		b.length += len(values)
		return
	}

	// Check if any nulls are present in the batch.
	batchHasNulls := false
	for _, v := range valid {
		if !v {
			batchHasNulls = true
			break
		}
	}

	b.data = append(b.data, values...)

	if batchHasNulls && !b.hasNulls {
		b.hasNulls = true
		b.validity = bitmap.New(b.length + len(values))
		// Previous elements [0, b.length) are already set by bitmap.New.
		for i, v := range valid {
			if !v {
				b.validity.Clear(b.length + i)
			}
		}
	} else if b.hasNulls {
		b.growValidity(b.length + len(values))
		for i, v := range valid {
			if v {
				b.validity.Set(b.length + i)
			}
		}
	}

	b.length += len(values)
}

// Len returns the number of elements appended so far.
func (b *TypedBuilder[T]) Len() int { return b.length }

// Cap returns the current capacity of the underlying data slice.
func (b *TypedBuilder[T]) Cap() int { return cap(b.data) }

// Reserve ensures there is capacity for at least additional more elements
// beyond the current length without further allocation.
func (b *TypedBuilder[T]) Reserve(additional int) {
	needed := b.length + additional
	if needed <= cap(b.data) {
		return
	}
	newData := make([]T, b.length, needed)
	copy(newData, b.data)
	b.data = newData
}

// IsNull returns true if the element at index i is null.
func (b *TypedBuilder[T]) IsNull(i int) bool {
	if !b.hasNulls {
		return false
	}
	return !b.validity.IsSet(i)
}

// Build finalizes the builder, returning the constructed TypedArray.
// The builder is reset to an empty state after this call and may be reused.
func (b *TypedBuilder[T]) Build() *TypedArray[T] {
	var validity *bitmap.Bitmap
	if b.hasNulls {
		validity = b.validity
	}
	arr := NewTypedArray(b.data[:b.length], b.dt, validity)

	// Reset the builder.
	b.data = b.data[:0]
	b.validity = nil
	b.hasNulls = false
	b.length = 0

	return arr
}

// growValidity grows the validity bitmap to accommodate at least n elements.
func (b *TypedBuilder[T]) growValidity(n int) {
	if b.validity.Len() < n {
		b.validity.Grow(n)
	}
}

// StringBuilder is a builder for constructing StringArray values incrementally.
// Internally it accumulates string bytes into a contiguous byte buffer with
// int32 offsets, matching the StringArray storage layout.
type StringBuilder struct {
	data     []byte
	offsets  []int32
	validity *bitmap.Bitmap
	hasNulls bool
	length   int
}

// NewStringBuilder creates a new StringBuilder with the specified initial
// capacity hint for the number of strings.
func NewStringBuilder(initialCap int) *StringBuilder {
	if initialCap < 0 {
		initialCap = 0
	}
	offsets := make([]int32, 1, initialCap+1)
	offsets[0] = 0
	return &StringBuilder{
		data:    make([]byte, 0, initialCap*8), // rough estimate of avg string length
		offsets: offsets,
	}
}

// Append appends a valid (non-null) string value.
func (b *StringBuilder) Append(s string) {
	b.data = append(b.data, s...)
	b.offsets = append(b.offsets, int32(len(b.data)))
	if b.hasNulls {
		b.growValidity(b.length + 1)
		b.validity.Set(b.length)
	}
	b.length++
}

// AppendNull appends a null string value.
func (b *StringBuilder) AppendNull() {
	b.offsets = append(b.offsets, int32(len(b.data)))
	if !b.hasNulls {
		b.hasNulls = true
		b.validity = bitmap.New(b.length + 1)
		b.validity.Clear(b.length)
	} else {
		b.growValidity(b.length + 1)
		// New bit is clear by default after Grow.
	}
	b.length++
}

// AppendValues appends multiple string values in bulk. The valid slice, if
// non-nil, indicates which values are valid (true) or null (false). If valid
// is nil, all values are treated as valid.
func (b *StringBuilder) AppendValues(values []string, valid []bool) {
	if len(values) == 0 {
		return
	}

	if valid == nil {
		for _, s := range values {
			b.data = append(b.data, s...)
			b.offsets = append(b.offsets, int32(len(b.data)))
		}
		if b.hasNulls {
			b.growValidity(b.length + len(values))
			for i := 0; i < len(values); i++ {
				b.validity.Set(b.length + i)
			}
		}
		b.length += len(values)
		return
	}

	batchHasNulls := false
	for _, v := range valid {
		if !v {
			batchHasNulls = true
			break
		}
	}

	for _, s := range values {
		b.data = append(b.data, s...)
		b.offsets = append(b.offsets, int32(len(b.data)))
	}

	if batchHasNulls && !b.hasNulls {
		b.hasNulls = true
		b.validity = bitmap.New(b.length + len(values))
		for i, v := range valid {
			if !v {
				b.validity.Clear(b.length + i)
			}
		}
	} else if b.hasNulls {
		b.growValidity(b.length + len(values))
		for i, v := range valid {
			if v {
				b.validity.Set(b.length + i)
			}
		}
	}

	b.length += len(values)
}

// Len returns the number of string elements appended so far.
func (b *StringBuilder) Len() int { return b.length }

// Cap returns the current capacity in terms of number of strings.
func (b *StringBuilder) Cap() int { return cap(b.offsets) - 1 }

// Reserve ensures there is capacity for at least additional more string
// elements without further allocation of the offsets slice.
func (b *StringBuilder) Reserve(additional int) {
	needed := b.length + 1 + additional
	if needed <= cap(b.offsets) {
		return
	}
	newOffsets := make([]int32, len(b.offsets), needed)
	copy(newOffsets, b.offsets)
	b.offsets = newOffsets
}

// IsNull returns true if the string element at index i is null.
func (b *StringBuilder) IsNull(i int) bool {
	if !b.hasNulls {
		return false
	}
	return !b.validity.IsSet(i)
}

// Build finalizes the builder, returning the constructed StringArray.
// The builder is reset to an empty state after this call and may be reused.
func (b *StringBuilder) Build() *StringArray {
	var validity *bitmap.Bitmap
	if b.hasNulls {
		validity = b.validity
	}
	arr := NewStringArrayFromBytes(b.data, b.offsets, validity)

	// Reset the builder.
	b.data = b.data[:0]
	b.offsets = b.offsets[:1]
	b.offsets[0] = 0
	b.validity = nil
	b.hasNulls = false
	b.length = 0

	return arr
}

// growValidity grows the validity bitmap to accommodate at least n elements.
func (b *StringBuilder) growValidity(n int) {
	if b.validity.Len() < n {
		b.validity.Grow(n)
	}
}

// BooleanBuilder is a builder for constructing BooleanArray values
// incrementally. Boolean values are packed into a bitmap for storage
// efficiency.
type BooleanBuilder struct {
	data     *bitmap.Bitmap
	validity *bitmap.Bitmap
	hasNulls bool
	length   int
	capacity int
}

// NewBooleanBuilder creates a new BooleanBuilder with the specified initial
// capacity hint.
func NewBooleanBuilder(initialCap int) *BooleanBuilder {
	if initialCap < 0 {
		initialCap = 0
	}
	return &BooleanBuilder{
		data:     bitmap.NewEmpty(initialCap),
		capacity: initialCap,
	}
}

// Append appends a valid (non-null) boolean value.
func (b *BooleanBuilder) Append(v bool) {
	b.ensureDataCapacity(b.length + 1)
	if v {
		b.data.Set(b.length)
	}
	if b.hasNulls {
		b.growValidity(b.length + 1)
		b.validity.Set(b.length)
	}
	b.length++
}

// AppendNull appends a null boolean value.
func (b *BooleanBuilder) AppendNull() {
	b.ensureDataCapacity(b.length + 1)
	if !b.hasNulls {
		b.hasNulls = true
		b.validity = bitmap.New(b.length + 1)
		b.validity.Clear(b.length)
	} else {
		b.growValidity(b.length + 1)
		// New bit is clear by default after Grow.
	}
	b.length++
}

// Len returns the number of boolean elements appended so far.
func (b *BooleanBuilder) Len() int { return b.length }

// Cap returns the current capacity of the builder.
func (b *BooleanBuilder) Cap() int { return b.capacity }

// Reserve ensures there is capacity for at least additional more boolean
// elements without further reallocation.
func (b *BooleanBuilder) Reserve(additional int) {
	needed := b.length + additional
	if needed <= b.capacity {
		return
	}
	b.data.Grow(needed)
	b.capacity = needed
}

// IsNull returns true if the boolean element at index i is null.
func (b *BooleanBuilder) IsNull(i int) bool {
	if !b.hasNulls {
		return false
	}
	return !b.validity.IsSet(i)
}

// Build finalizes the builder, returning the constructed BooleanArray.
// The builder is reset to an empty state after this call and may be reused.
func (b *BooleanBuilder) Build() *BooleanArray {
	var validity *bitmap.Bitmap
	if b.hasNulls {
		validity = b.validity
	}

	// Extract the data bitmap trimmed to the actual length.
	dataBitmap := b.data.Slice(0, b.length)
	arr := NewBooleanArrayFromBitmap(dataBitmap, validity)

	// Reset the builder.
	b.data = bitmap.NewEmpty(b.capacity)
	b.validity = nil
	b.hasNulls = false
	b.length = 0

	return arr
}

// ensureDataCapacity ensures the data bitmap can hold at least n elements.
func (b *BooleanBuilder) ensureDataCapacity(n int) {
	if n > b.capacity {
		newCap := b.capacity * 2
		if newCap < n {
			newCap = n
		}
		b.data.Grow(newCap)
		b.capacity = newCap
	}
}

// growValidity grows the validity bitmap to accommodate at least n elements.
func (b *BooleanBuilder) growValidity(n int) {
	if b.validity.Len() < n {
		b.validity.Grow(n)
	}
}
