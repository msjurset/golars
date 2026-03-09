package array

import (
	"math"

	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/pool"
)

// Numeric is a constraint that permits all integer and floating-point types.
type Numeric interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// Integer is a constraint that permits all integer types (signed and unsigned).
type Integer interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// SignedNumeric is a constraint that permits signed integers and floating-point types.
type SignedNumeric interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

// mergeValidity computes the combined validity bitmap for a binary operation.
// If both inputs have validity bitmaps, the result is their bitwise AND.
// If only one has a validity bitmap, its clone is returned. If neither has
// one, nil is returned.
func mergeValidity(va, vb *bitmap.Bitmap) *bitmap.Bitmap {
	if va != nil && vb != nil {
		return va.And(vb)
	}
	if va != nil {
		return va.Clone()
	}
	if vb != nil {
		return vb.Clone()
	}
	return nil
}

// Add returns a new array whose elements are the element-wise sum of a and b.
// Null values propagate: if either operand is null at a position, the result
// is null at that position.
func Add[T Numeric](a, b *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] + bv[i]
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Sub returns a new array whose elements are the element-wise difference of a and b.
// Null values propagate through the operation.
func Sub[T Numeric](a, b *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] - bv[i]
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Mul returns a new array whose elements are the element-wise product of a and b.
// Null values propagate through the operation.
func Mul[T Numeric](a, b *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] * bv[i]
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Div returns a new array whose elements are the element-wise quotient of a and b.
// For integer types this performs integer division. Null values propagate
// through the operation.
func Div[T Numeric](a, b *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] / bv[i]
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Mod returns a new array whose elements are the element-wise remainder of a
// divided by b. This operation is only defined for integer types. Null values
// propagate through the operation.
func Mod[T Integer](a, b *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] % bv[i]
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Pow returns a new float64 array whose elements are a[i] raised to the power of b[i].
// Null values propagate through the operation.
func Pow(a, b *TypedArray[float64]) *TypedArray[float64] {
	n := a.Len()
	result := make([]float64, n)
	av, bv := a.Values(), b.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = math.Pow(av[i], bv[i])
		}
	})
	return NewTypedArray(result, a.DataType(), mergeValidity(a.Validity(), b.Validity()))
}

// Neg returns a new array whose elements are the negation of each element in a.
// This operation is only defined for signed numeric types. Null values are
// preserved.
func Neg[T SignedNumeric](a *TypedArray[T]) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = -av[i]
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewTypedArray(result, a.DataType(), validity)
}

// AddScalar returns a new array whose elements are each element of a plus
// the given scalar value. Null values are preserved.
func AddScalar[T Numeric](a *TypedArray[T], scalar T) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] + scalar
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewTypedArray(result, a.DataType(), validity)
}

// MulScalar returns a new array whose elements are each element of a
// multiplied by the given scalar value. Null values are preserved.
func MulScalar[T Numeric](a *TypedArray[T], scalar T) *TypedArray[T] {
	n := a.Len()
	result := make([]T, n)
	av := a.Values()
	pool.ParallelDo(n, pool.DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			result[i] = av[i] * scalar
		}
	})
	var validity *bitmap.Bitmap
	if a.Validity() != nil {
		validity = a.Validity().Clone()
	}
	return NewTypedArray(result, a.DataType(), validity)
}
