// Package array provides columnar storage arrays for the Golars DataFrame library.
//
// Arrays are the fundamental storage unit, holding typed columnar data with
// optional null bitmaps. All arrays implement the Array interface.
package array

import (
	"fmt"

	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
)

// Array is the interface that all columnar array types implement.
// Arrays are immutable after construction. All read operations are safe
// for concurrent use by multiple goroutines.
type Array interface {
	// DataType returns the logical data type of this array.
	DataType() dtype.DataType

	// Len returns the number of elements in the array.
	Len() int

	// IsNull returns true if the element at index i is null.
	IsNull(i int) bool

	// IsValid returns true if the element at index i is not null.
	IsValid(i int) bool

	// NullCount returns the number of null elements.
	NullCount() int

	// Validity returns the underlying null bitmap, or nil if there are no nulls.
	Validity() *bitmap.Bitmap

	// Slice returns a zero-copy slice of this array from start to end.
	Slice(start, end int) Array

	// String returns a human-readable representation of the array.
	String() string
}

// TypedArray is a generic columnar array holding values of type T.
// It stores data in a contiguous slice with an optional null bitmap.
type TypedArray[T any] struct {
	data     []T
	validity *bitmap.Bitmap
	dt       dtype.DataType
	offset   int
	length   int
}

// NewTypedArray creates a new TypedArray from the given data and data type.
// If validity is nil, all elements are considered valid (non-null).
func NewTypedArray[T any](data []T, dt dtype.DataType, validity *bitmap.Bitmap) *TypedArray[T] {
	return &TypedArray[T]{
		data:     data,
		validity: validity,
		dt:       dt,
		offset:   0,
		length:   len(data),
	}
}

// DataType returns the logical data type of this array.
func (a *TypedArray[T]) DataType() dtype.DataType { return a.dt }

// Len returns the number of elements in the array.
func (a *TypedArray[T]) Len() int { return a.length }

// IsNull returns true if the element at index i is null.
func (a *TypedArray[T]) IsNull(i int) bool {
	if a.validity == nil {
		return false
	}
	return !a.validity.IsSet(a.offset + i)
}

// IsValid returns true if the element at index i is not null.
func (a *TypedArray[T]) IsValid(i int) bool {
	if a.validity == nil {
		return true
	}
	return a.validity.IsSet(a.offset + i)
}

// NullCount returns the number of null elements.
func (a *TypedArray[T]) NullCount() int {
	if a.validity == nil {
		return 0
	}
	// Count nulls in our range
	count := 0
	for i := 0; i < a.length; i++ {
		if !a.validity.IsSet(a.offset + i) {
			count++
		}
	}
	return count
}

// Validity returns the underlying null bitmap, or nil if there are no nulls.
func (a *TypedArray[T]) Validity() *bitmap.Bitmap { return a.validity }

// Slice returns a zero-copy slice of this array from start to end.
func (a *TypedArray[T]) Slice(start, end int) Array {
	var v *bitmap.Bitmap
	if a.validity != nil {
		v = a.validity.Slice(a.offset+start, a.offset+end)
	}
	return &TypedArray[T]{
		data:     a.data[a.offset+start : a.offset+end],
		validity: v,
		dt:       a.dt,
		offset:   0,
		length:   end - start,
	}
}

// Value returns the value at index i. The caller should check IsNull first.
func (a *TypedArray[T]) Value(i int) T { return a.data[a.offset+i] }

// Values returns the raw underlying data slice for the active range.
func (a *TypedArray[T]) Values() []T { return a.data[a.offset : a.offset+a.length] }

// SetValidity sets the validity bitmap for this array.
func (a *TypedArray[T]) SetValidity(v *bitmap.Bitmap) { a.validity = v }

// String returns a human-readable representation of the array.
func (a *TypedArray[T]) String() string {
	return fmt.Sprintf("%s[%d]", a.dt, a.length)
}

// Filter returns a new array containing only elements where mask is true.
func Filter[T any](arr *TypedArray[T], mask *bitmap.Bitmap) *TypedArray[T] {
	n := mask.PopCount()
	data := make([]T, 0, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.validity != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	j := 0
	for i := 0; i < arr.Len(); i++ {
		if mask.IsSet(i) {
			data = append(data, arr.Value(i))
			if hasNulls && arr.IsNull(i) {
				validity.Clear(j)
			}
			j++
		}
	}

	return NewTypedArray(data, arr.dt, validity)
}

// Take returns a new array with elements at the given indices.
func Take[T any](arr *TypedArray[T], indices []int) *TypedArray[T] {
	n := len(indices)
	data := make([]T, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.validity != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	for j, idx := range indices {
		data[j] = arr.Value(idx)
		if hasNulls && arr.IsNull(idx) {
			validity.Clear(j)
		}
	}

	return NewTypedArray(data, arr.dt, validity)
}
