// Package series provides the Series type, a named typed column of data.
//
// A Series wraps an array with a name and data type, providing a high-level
// API for column operations. Series are immutable after construction; all
// mutation operations return new Series values.
//
// All read operations on a Series are safe for concurrent use by multiple
// goroutines.
package series

import (
	"fmt"
	"strings"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
)

// Series is a named, typed column of data. It wraps an underlying columnar
// array with a name and provides a rich API for data manipulation.
type Series struct {
	name  string
	arr   array.Array
	dtype dtype.DataType
}

// New creates a new Series with the given name and underlying array.
func New(name string, arr array.Array) *Series {
	return &Series{
		name:  name,
		arr:   arr,
		dtype: arr.DataType(),
	}
}

// NewInt8 creates a new Series of int8 values.
func NewInt8(name string, data []int8) *Series {
	return New(name, array.NewInt8Array(data, nil))
}

// NewInt16 creates a new Series of int16 values.
func NewInt16(name string, data []int16) *Series {
	return New(name, array.NewInt16Array(data, nil))
}

// NewInt32 creates a new Series of int32 values.
func NewInt32(name string, data []int32) *Series {
	return New(name, array.NewInt32Array(data, nil))
}

// NewInt64 creates a new Series of int64 values.
func NewInt64(name string, data []int64) *Series {
	return New(name, array.NewInt64Array(data, nil))
}

// NewUInt8 creates a new Series of uint8 values.
func NewUInt8(name string, data []uint8) *Series {
	return New(name, array.NewUInt8Array(data, nil))
}

// NewUInt16 creates a new Series of uint16 values.
func NewUInt16(name string, data []uint16) *Series {
	return New(name, array.NewUInt16Array(data, nil))
}

// NewUInt32 creates a new Series of uint32 values.
func NewUInt32(name string, data []uint32) *Series {
	return New(name, array.NewUInt32Array(data, nil))
}

// NewUInt64 creates a new Series of uint64 values.
func NewUInt64(name string, data []uint64) *Series {
	return New(name, array.NewUInt64Array(data, nil))
}

// NewFloat32 creates a new Series of float32 values.
func NewFloat32(name string, data []float32) *Series {
	return New(name, array.NewFloat32Array(data, nil))
}

// NewFloat64 creates a new Series of float64 values.
func NewFloat64(name string, data []float64) *Series {
	return New(name, array.NewFloat64Array(data, nil))
}

// NewBoolean creates a new Series of boolean values.
func NewBoolean(name string, data []bool) *Series {
	return New(name, array.NewBooleanArray(data, nil))
}

// NewString creates a new Series of string values.
func NewString(name string, data []string) *Series {
	return New(name, array.NewStringArray(data, nil))
}

// NewInt64WithValidity creates an Int64 Series with explicit null tracking.
// Positions where valid[i] is false are marked as null.
func NewInt64WithValidity(name string, data []int64, valid []bool) *Series {
	v := bitmap.New(len(data))
	for i, ok := range valid {
		if !ok {
			v.Clear(i)
		}
	}
	return New(name, array.NewInt64Array(data, v))
}

// NewFloat64WithValidity creates a Float64 Series with explicit null tracking.
func NewFloat64WithValidity(name string, data []float64, valid []bool) *Series {
	v := bitmap.New(len(data))
	for i, ok := range valid {
		if !ok {
			v.Clear(i)
		}
	}
	return New(name, array.NewFloat64Array(data, v))
}

// NewStringWithValidity creates a String Series with explicit null tracking.
func NewStringWithValidity(name string, data []string, valid []bool) *Series {
	v := bitmap.New(len(data))
	for i, ok := range valid {
		if !ok {
			v.Clear(i)
		}
	}
	return New(name, array.NewStringArray(data, v))
}

// NewBooleanWithValidity creates a Boolean Series with explicit null tracking.
func NewBooleanWithValidity(name string, data []bool, valid []bool) *Series {
	v := bitmap.New(len(data))
	for i, ok := range valid {
		if !ok {
			v.Clear(i)
		}
	}
	return New(name, array.NewBooleanArray(data, v))
}

// Name returns the name of the series.
func (s *Series) Name() string { return s.name }

// DataType returns the data type of the series.
func (s *Series) DataType() dtype.DataType { return s.dtype }

// Len returns the number of elements in the series.
func (s *Series) Len() int { return s.arr.Len() }

// IsNull returns true if the element at index i is null.
func (s *Series) IsNull(i int) bool { return s.arr.IsNull(i) }

// IsValid returns true if the element at index i is not null.
func (s *Series) IsValid(i int) bool { return s.arr.IsValid(i) }

// NullCount returns the number of null elements.
func (s *Series) NullCount() int { return s.arr.NullCount() }

// HasNulls returns true if the series contains any null values.
func (s *Series) HasNulls() bool { return s.NullCount() > 0 }

// Array returns the underlying array storage.
func (s *Series) Array() array.Array { return s.arr }

// Rename returns a new Series with the given name.
func (s *Series) Rename(name string) *Series {
	return &Series{name: name, arr: s.arr, dtype: s.dtype}
}

// Slice returns a new Series for the range [start, end).
func (s *Series) Slice(start, end int) *Series {
	return New(s.name, s.arr.Slice(start, end))
}

// Head returns the first n elements.
func (s *Series) Head(n int) *Series {
	if n > s.Len() {
		n = s.Len()
	}
	return s.Slice(0, n)
}

// Tail returns the last n elements.
func (s *Series) Tail(n int) *Series {
	l := s.Len()
	if n > l {
		n = l
	}
	return s.Slice(l-n, l)
}

// GetInt64 returns the int64 value at index i and whether it's valid.
func (s *Series) GetInt64(i int) (int64, bool) {
	if s.arr.IsNull(i) {
		return 0, false
	}
	if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
		return ta.Value(i), true
	}
	return 0, false
}

// GetFloat64 returns the float64 value at index i and whether it's valid.
func (s *Series) GetFloat64(i int) (float64, bool) {
	if s.arr.IsNull(i) {
		return 0, false
	}
	if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
		return ta.Value(i), true
	}
	return 0, false
}

// GetString returns the string value at index i and whether it's valid.
func (s *Series) GetString(i int) (string, bool) {
	if s.arr.IsNull(i) {
		return "", false
	}
	if sa, ok := s.arr.(*array.StringArray); ok {
		return sa.Value(i), true
	}
	return "", false
}

// GetBool returns the boolean value at index i and whether it's valid.
func (s *Series) GetBool(i int) (bool, bool) {
	if s.arr.IsNull(i) {
		return false, false
	}
	if ba, ok := s.arr.(*array.BooleanArray); ok {
		return ba.Value(i), true
	}
	return false, false
}

// Int64Values returns the underlying int64 slice, or nil if wrong type.
func (s *Series) Int64Values() []int64 {
	if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
		return ta.Values()
	}
	return nil
}

// Float64Values returns the underlying float64 slice, or nil if wrong type.
func (s *Series) Float64Values() []float64 {
	if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
		return ta.Values()
	}
	return nil
}

// BooleanArray returns the underlying BooleanArray, or nil if wrong type.
func (s *Series) BooleanArray() *array.BooleanArray {
	if ba, ok := s.arr.(*array.BooleanArray); ok {
		return ba
	}
	return nil
}

// StringArray returns the underlying StringArray, or nil if wrong type.
func (s *Series) StringArray() *array.StringArray {
	if sa, ok := s.arr.(*array.StringArray); ok {
		return sa
	}
	return nil
}

// Validity returns the null bitmap, or nil if there are no nulls.
func (s *Series) Validity() *bitmap.Bitmap { return s.arr.Validity() }

// Equal checks if two series have the same name, type, length, and values.
func (s *Series) Equal(other *Series) bool {
	if s.name != other.name || s.dtype != other.dtype || s.Len() != other.Len() {
		return false
	}
	for i := 0; i < s.Len(); i++ {
		sNull := s.IsNull(i)
		oNull := other.IsNull(i)
		if sNull != oNull {
			return false
		}
		if sNull {
			continue
		}
		// Compare based on type
		switch s.dtype {
		case dtype.Int64:
			a, _ := s.GetInt64(i)
			b, _ := other.GetInt64(i)
			if a != b {
				return false
			}
		case dtype.Float64:
			a, _ := s.GetFloat64(i)
			b, _ := other.GetFloat64(i)
			if a != b {
				return false
			}
		case dtype.String:
			a, _ := s.GetString(i)
			b, _ := other.GetString(i)
			if a != b {
				return false
			}
		case dtype.Boolean:
			a, _ := s.GetBool(i)
			b, _ := other.GetBool(i)
			if a != b {
				return false
			}
		default:
			// For other types, compare string representation
			return s.arr.String() == other.arr.String()
		}
	}
	return true
}

// String returns a human-readable representation of the series.
func (s *Series) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Series: '%s' [%s]\n", s.name, s.dtype))
	b.WriteString("[\n")

	n := s.Len()
	maxShow := 10
	showHead := maxShow / 2
	showTail := maxShow / 2

	if n <= maxShow {
		for i := 0; i < n; i++ {
			b.WriteString("\t")
			b.WriteString(s.formatValue(i))
			b.WriteString("\n")
		}
	} else {
		for i := 0; i < showHead; i++ {
			b.WriteString("\t")
			b.WriteString(s.formatValue(i))
			b.WriteString("\n")
		}
		b.WriteString("\t...\n")
		for i := n - showTail; i < n; i++ {
			b.WriteString("\t")
			b.WriteString(s.formatValue(i))
			b.WriteString("\n")
		}
	}

	b.WriteString("]\n")
	return b.String()
}

func (s *Series) formatValue(i int) string {
	if s.arr.IsNull(i) {
		return "null"
	}
	switch s.dtype {
	case dtype.Int64:
		v, _ := s.GetInt64(i)
		return fmt.Sprintf("%d", v)
	case dtype.Float64:
		v, _ := s.GetFloat64(i)
		return fmt.Sprintf("%g", v)
	case dtype.String:
		v, _ := s.GetString(i)
		return fmt.Sprintf("%q", v)
	case dtype.Boolean:
		v, _ := s.GetBool(i)
		return fmt.Sprintf("%t", v)
	default:
		return "?"
	}
}
