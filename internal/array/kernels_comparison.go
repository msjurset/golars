package array

import (
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/pool"
)

// Ordered is a constraint that permits types supporting the < operator.
type Ordered interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~string
}

// Equal returns a BooleanArray indicating element-wise equality between a and b.
// Null values propagate: if either operand is null, the result is null.
func Equal[T comparable](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] == bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// NotEqual returns a BooleanArray indicating element-wise inequality between a and b.
// Null values propagate: if either operand is null, the result is null.
func NotEqual[T comparable](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] != bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// LessThan returns a BooleanArray indicating where a < b element-wise.
// Null values propagate: if either operand is null, the result is null.
func LessThan[T Ordered](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] < bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// LessThanEqual returns a BooleanArray indicating where a <= b element-wise.
// Null values propagate: if either operand is null, the result is null.
func LessThanEqual[T Ordered](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] <= bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// GreaterThan returns a BooleanArray indicating where a > b element-wise.
// Null values propagate: if either operand is null, the result is null.
func GreaterThan[T Ordered](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] > bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// GreaterThanEqual returns a BooleanArray indicating where a >= b element-wise.
// Null values propagate: if either operand is null, the result is null.
func GreaterThanEqual[T Ordered](a, b *TypedArray[T]) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] >= bv[i] {
				data.Set(i)
			}
		}
	})
	return NewBooleanArrayFromBitmap(data, mergeValidity(a.Validity(), b.Validity()))
}

// EqualScalar returns a BooleanArray indicating where each element of a equals
// the given scalar. Null values are preserved.
func EqualScalar[T comparable](a *TypedArray[T], scalar T) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] == scalar {
				data.Set(i)
			}
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewBooleanArrayFromBitmap(data, validity)
}

// LessThanScalar returns a BooleanArray indicating where each element of a is
// less than the given scalar. Null values are preserved.
func LessThanScalar[T Ordered](a *TypedArray[T], scalar T) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] < scalar {
				data.Set(i)
			}
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewBooleanArrayFromBitmap(data, validity)
}

// GreaterThanScalar returns a BooleanArray indicating where each element of a
// is greater than the given scalar. Null values are preserved.
func GreaterThanScalar[T Ordered](a *TypedArray[T], scalar T) *BooleanArray {
	n := a.Len()
	data := bitmap.NewEmpty(n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			if av[i] > scalar {
				data.Set(i)
			}
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewBooleanArrayFromBitmap(data, validity)
}
