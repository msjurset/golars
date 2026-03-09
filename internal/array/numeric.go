package array

import (
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// NewInt8Array creates a new array of int8 values.
func NewInt8Array(data []int8, validity *bitmap.Bitmap) *TypedArray[int8] {
	return NewTypedArray(data, dtype.Int8, validity)
}

// NewInt16Array creates a new array of int16 values.
func NewInt16Array(data []int16, validity *bitmap.Bitmap) *TypedArray[int16] {
	return NewTypedArray(data, dtype.Int16, validity)
}

// NewInt32Array creates a new array of int32 values.
func NewInt32Array(data []int32, validity *bitmap.Bitmap) *TypedArray[int32] {
	return NewTypedArray(data, dtype.Int32, validity)
}

// NewInt64Array creates a new array of int64 values.
func NewInt64Array(data []int64, validity *bitmap.Bitmap) *TypedArray[int64] {
	return NewTypedArray(data, dtype.Int64, validity)
}

// NewUInt8Array creates a new array of uint8 values.
func NewUInt8Array(data []uint8, validity *bitmap.Bitmap) *TypedArray[uint8] {
	return NewTypedArray(data, dtype.UInt8, validity)
}

// NewUInt16Array creates a new array of uint16 values.
func NewUInt16Array(data []uint16, validity *bitmap.Bitmap) *TypedArray[uint16] {
	return NewTypedArray(data, dtype.UInt16, validity)
}

// NewUInt32Array creates a new array of uint32 values.
func NewUInt32Array(data []uint32, validity *bitmap.Bitmap) *TypedArray[uint32] {
	return NewTypedArray(data, dtype.UInt32, validity)
}

// NewUInt64Array creates a new array of uint64 values.
func NewUInt64Array(data []uint64, validity *bitmap.Bitmap) *TypedArray[uint64] {
	return NewTypedArray(data, dtype.UInt64, validity)
}

// NewFloat32Array creates a new array of float32 values.
func NewFloat32Array(data []float32, validity *bitmap.Bitmap) *TypedArray[float32] {
	return NewTypedArray(data, dtype.Float32, validity)
}

// NewFloat64Array creates a new array of float64 values.
func NewFloat64Array(data []float64, validity *bitmap.Bitmap) *TypedArray[float64] {
	return NewTypedArray(data, dtype.Float64, validity)
}
