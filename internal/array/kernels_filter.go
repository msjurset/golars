package array

import (
	"github.com/msjurseth/golars/internal/bitmap"
)

// FilterTyped returns a new TypedArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is
// filtered accordingly.
func FilterTyped[T any](arr *TypedArray[T], mask *bitmap.Bitmap) *TypedArray[T] {
	n := mask.PopCount()
	data := make([]T, 0, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil

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

	return NewTypedArray(data, arr.DataType(), validity)
}

// TakeTyped returns a new TypedArray with elements gathered from arr at the
// given indices. The validity bitmap is carried over for each gathered element.
func TakeTyped[T any](arr *TypedArray[T], indices []int) *TypedArray[T] {
	n := len(indices)
	data := make([]T, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	for j, idx := range indices {
		data[j] = arr.Value(idx)
		if hasNulls && arr.IsNull(idx) {
			validity.Clear(j)
		}
	}

	return NewTypedArray(data, arr.DataType(), validity)
}

// FilterBoolean returns a new BooleanArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is filtered
// accordingly.
func FilterBoolean(arr *BooleanArray, mask *bitmap.Bitmap) *BooleanArray {
	n := mask.PopCount()
	data := bitmap.NewEmpty(n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	j := 0
	for i := 0; i < arr.Len(); i++ {
		if mask.IsSet(i) {
			if arr.Value(i) {
				data.Set(j)
			}
			if hasNulls && arr.IsNull(i) {
				validity.Clear(j)
			}
			j++
		}
	}

	return NewBooleanArrayFromBitmap(data, validity)
}

// FilterString returns a new StringArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is filtered
// accordingly.
func FilterString(arr *StringArray, mask *bitmap.Bitmap) *StringArray {
	n := mask.PopCount()
	values := make([]string, 0, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	j := 0
	for i := 0; i < arr.Len(); i++ {
		if mask.IsSet(i) {
			values = append(values, arr.Value(i))
			if hasNulls && arr.IsNull(i) {
				validity.Clear(j)
			}
			j++
		}
	}

	return NewStringArray(values, validity)
}
