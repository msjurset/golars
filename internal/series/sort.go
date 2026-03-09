package series

import (
	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dtype"
)

// Sort returns a new Series with elements sorted. Nulls are placed at the end.
func (s *Series) Sort(descending bool) *Series {
	indices := s.ArgSort(descending)
	return s.Take(indices)
}

// ArgSort returns the indices that would sort this Series. Nulls are placed
// at the end.
func (s *Series) ArgSort(descending bool) []int {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return array.ArgSort(ta, descending)
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return array.ArgSort(ta, descending)
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return array.ArgSort(ta, descending)
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return array.ArgSort(ta, descending)
		}
	case dtype.String:
		if sa, ok := s.arr.(*array.StringArray); ok {
			return array.ArgSortString(sa, descending)
		}
	}
	// Identity permutation as fallback
	indices := make([]int, s.Len())
	for i := range indices {
		indices[i] = i
	}
	return indices
}

// ArgMin returns the index of the minimum non-null value.
// Returns -1, false if the series is empty or all null.
func (s *Series) ArgMin() (int, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return array.ArgMin(ta)
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return array.ArgMin(ta)
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return array.ArgMin(ta)
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return array.ArgMin(ta)
		}
	}
	return -1, false
}

// ArgMax returns the index of the maximum non-null value.
// Returns -1, false if the series is empty or all null.
func (s *Series) ArgMax() (int, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return array.ArgMax(ta)
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return array.ArgMax(ta)
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return array.ArgMax(ta)
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return array.ArgMax(ta)
		}
	}
	return -1, false
}
