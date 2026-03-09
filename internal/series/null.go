package series

import (
	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// IsNullSeries returns a Boolean Series where true indicates a null in the original.
func (s *Series) IsNullSeries() *Series {
	n := s.Len()
	result := make([]bool, n)
	for i := 0; i < n; i++ {
		result[i] = s.IsNull(i)
	}
	return NewBoolean(s.name, result)
}

// IsNotNullSeries returns a Boolean Series where true indicates a non-null in the original.
func (s *Series) IsNotNullSeries() *Series {
	n := s.Len()
	result := make([]bool, n)
	for i := 0; i < n; i++ {
		result[i] = s.IsValid(i)
	}
	return NewBoolean(s.name, result)
}

// DropNulls returns a new Series with null values removed.
func (s *Series) DropNulls() *Series {
	if !s.HasNulls() {
		return s
	}
	n := s.Len()
	mask := bitmap.New(n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			mask.Clear(i)
		}
	}
	return s.Filter(mask)
}

// FillNullInt64 returns a new Int64 Series with nulls replaced by the given value.
func (s *Series) FillNullInt64(fillValue int64) *Series {
	if !s.HasNulls() || s.dtype != dtype.Int64 {
		return s
	}
	ta, ok := s.arr.(*array.TypedArray[int64])
	if !ok {
		return s
	}
	n := s.Len()
	data := make([]int64, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			data[i] = fillValue
		} else {
			data[i] = ta.Value(i)
		}
	}
	return NewInt64(s.name, data)
}

// FillNullFloat64 returns a new Float64 Series with nulls replaced by the given value.
func (s *Series) FillNullFloat64(fillValue float64) *Series {
	if !s.HasNulls() || s.dtype != dtype.Float64 {
		return s
	}
	ta, ok := s.arr.(*array.TypedArray[float64])
	if !ok {
		return s
	}
	n := s.Len()
	data := make([]float64, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			data[i] = fillValue
		} else {
			data[i] = ta.Value(i)
		}
	}
	return NewFloat64(s.name, data)
}

// FillNullString returns a new String Series with nulls replaced by the given value.
func (s *Series) FillNullString(fillValue string) *Series {
	if !s.HasNulls() || s.dtype != dtype.String {
		return s
	}
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return s
	}
	n := s.Len()
	data := make([]string, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			data[i] = fillValue
		} else {
			data[i] = sa.Value(i)
		}
	}
	return NewString(s.name, data)
}

// Filter returns a new Series containing only elements where mask bits are set.
func (s *Series) Filter(mask *bitmap.Bitmap) *Series {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return New(s.name, array.FilterTyped(ta, mask))
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return New(s.name, array.FilterTyped(ta, mask))
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return New(s.name, array.FilterTyped(ta, mask))
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return New(s.name, array.FilterTyped(ta, mask))
		}
	case dtype.Boolean:
		if ba, ok := s.arr.(*array.BooleanArray); ok {
			return New(s.name, array.FilterBoolean(ba, mask))
		}
	case dtype.String:
		if sa, ok := s.arr.(*array.StringArray); ok {
			return New(s.name, array.FilterString(sa, mask))
		}
	}
	// Fallback for other types - shouldn't reach here for supported types
	return s
}

// Take returns a new Series with elements at the given indices.
func (s *Series) Take(indices []int) *Series {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return New(s.name, array.TakeTyped(ta, indices))
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return New(s.name, array.TakeTyped(ta, indices))
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return New(s.name, array.TakeTyped(ta, indices))
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return New(s.name, array.TakeTyped(ta, indices))
		}
	case dtype.String:
		if sa, ok := s.arr.(*array.StringArray); ok {
			return New(s.name, array.TakeString(sa, indices))
		}
	}
	return s
}
