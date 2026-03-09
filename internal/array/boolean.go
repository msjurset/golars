package array

import (
	"fmt"

	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
)

// BooleanArray stores boolean values in a packed bitmap for memory efficiency.
// Each value occupies a single bit. Null tracking uses a separate validity bitmap.
type BooleanArray struct {
	data     *bitmap.Bitmap
	validity *bitmap.Bitmap
	length   int
}

// NewBooleanArray creates a new BooleanArray from a slice of booleans.
// If validity is nil, all elements are considered valid.
func NewBooleanArray(data []bool, validity *bitmap.Bitmap) *BooleanArray {
	bm := bitmap.NewEmpty(len(data))
	for i, v := range data {
		if v {
			bm.Set(i)
		}
	}
	return &BooleanArray{
		data:     bm,
		validity: validity,
		length:   len(data),
	}
}

// NewBooleanArrayFromBitmap creates a BooleanArray from an existing bitmap.
func NewBooleanArrayFromBitmap(data *bitmap.Bitmap, validity *bitmap.Bitmap) *BooleanArray {
	return &BooleanArray{
		data:     data,
		validity: validity,
		length:   data.Len(),
	}
}

// DataType returns Boolean.
func (a *BooleanArray) DataType() dtype.DataType { return dtype.Boolean }

// Len returns the number of elements.
func (a *BooleanArray) Len() int { return a.length }

// IsNull returns true if the element at index i is null.
func (a *BooleanArray) IsNull(i int) bool {
	if a.validity == nil {
		return false
	}
	return !a.validity.IsSet(i)
}

// IsValid returns true if the element at index i is not null.
func (a *BooleanArray) IsValid(i int) bool {
	if a.validity == nil {
		return true
	}
	return a.validity.IsSet(i)
}

// NullCount returns the number of null elements.
func (a *BooleanArray) NullCount() int {
	if a.validity == nil {
		return 0
	}
	return a.length - a.validity.PopCount()
}

// Validity returns the null bitmap.
func (a *BooleanArray) Validity() *bitmap.Bitmap { return a.validity }

// Value returns the boolean value at index i.
func (a *BooleanArray) Value(i int) bool { return a.data.IsSet(i) }

// DataBitmap returns the underlying data bitmap.
func (a *BooleanArray) DataBitmap() *bitmap.Bitmap { return a.data }

// Slice returns a zero-copy slice from start to end.
func (a *BooleanArray) Slice(start, end int) Array {
	var v *bitmap.Bitmap
	if a.validity != nil {
		v = a.validity.Slice(start, end)
	}
	return &BooleanArray{
		data:     a.data.Slice(start, end),
		validity: v,
		length:   end - start,
	}
}

// String returns a human-readable representation.
func (a *BooleanArray) String() string {
	return fmt.Sprintf("Boolean[%d]", a.length)
}

// TrueCount returns the number of true values (excluding nulls).
func (a *BooleanArray) TrueCount() int {
	if a.validity == nil {
		return a.data.PopCount()
	}
	combined := a.data.And(a.validity)
	return combined.PopCount()
}

// FalseCount returns the number of false values (excluding nulls).
func (a *BooleanArray) FalseCount() int {
	return a.length - a.NullCount() - a.TrueCount()
}
