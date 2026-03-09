package array

import "github.com/msjurset/golars/internal/bitmap"

// CumSum returns a new array where each element is the cumulative sum of all
// preceding valid elements. Null positions remain null and do not contribute
// to the running total.
func CumSum[T Numeric](a *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	vals := a.Values()
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}

	var running T
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			running += vals[i]
			result[i] = running
		}
	}
	return NewTypedArray(result, a.DataType(), validity)
}

// CumProd returns a new array where each element is the cumulative product of
// all preceding valid elements. Null positions remain null.
func CumProd[T Numeric](a *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	vals := a.Values()
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}

	running := T(1)
	first := true
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			if first {
				running = vals[i]
				first = false
			} else {
				running *= vals[i]
			}
			result[i] = running
		}
	}
	return NewTypedArray(result, a.DataType(), validity)
}

// CumMin returns a new array where each element is the running minimum of all
// preceding valid elements. Null positions remain null.
func CumMin[T Ordered](a *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	vals := a.Values()
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}

	var running T
	first := true
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if first || v < running {
				running = v
				first = false
			}
			result[i] = running
		}
	}
	return NewTypedArray(result, a.DataType(), validity)
}

// CumMax returns a new array where each element is the running maximum of all
// preceding valid elements. Null positions remain null.
func CumMax[T Ordered](a *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	vals := a.Values()
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}

	var running T
	first := true
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if first || v > running {
				running = v
				first = false
			}
			result[i] = running
		}
	}
	return NewTypedArray(result, a.DataType(), validity)
}
