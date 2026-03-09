package series

import (
	"fmt"
	"math"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
)

// Sum returns the sum of all non-null values as a float64.
// The second return value is false if the series is empty or all null.
func (s *Series) Sum() (float64, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.Int16:
		if ta, ok := s.arr.(*array.TypedArray[int16]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.Int8:
		if ta, ok := s.arr.(*array.TypedArray[int8]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.UInt64:
		if ta, ok := s.arr.(*array.TypedArray[uint64]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.UInt32:
		if ta, ok := s.arr.(*array.TypedArray[uint32]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.UInt16:
		if ta, ok := s.arr.(*array.TypedArray[uint16]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.UInt8:
		if ta, ok := s.arr.(*array.TypedArray[uint8]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			v, valid := array.Sum(ta)
			return v, valid
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			v, valid := array.Sum(ta)
			return float64(v), valid
		}
	}
	return 0, false
}

// Mean returns the arithmetic mean of all non-null values.
// The second return value is false if the series is empty or all null.
func (s *Series) Mean() (float64, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return array.Mean(ta)
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return array.Mean(ta)
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return array.Mean(ta)
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return array.Mean(ta)
		}
	}
	return 0, false
}

// Min returns the minimum non-null value as a float64.
// The second return value is false if the series is empty or all null.
func (s *Series) Min() (float64, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			v, valid := array.Min(ta)
			return float64(v), valid
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			v, valid := array.Min(ta)
			return float64(v), valid
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			v, valid := array.Min(ta)
			return v, valid
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			v, valid := array.Min(ta)
			return float64(v), valid
		}
	}
	return 0, false
}

// Max returns the maximum non-null value as a float64.
// The second return value is false if the series is empty or all null.
func (s *Series) Max() (float64, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			v, valid := array.Max(ta)
			return float64(v), valid
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			v, valid := array.Max(ta)
			return float64(v), valid
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			v, valid := array.Max(ta)
			return v, valid
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			v, valid := array.Max(ta)
			return float64(v), valid
		}
	}
	return 0, false
}

// Std returns the standard deviation of all non-null values.
// Uses ddof=1 (sample standard deviation) by default.
func (s *Series) Std() (float64, bool) {
	return s.StdDDOF(1)
}

// StdDDOF returns the standard deviation with the specified delta degrees of freedom.
func (s *Series) StdDDOF(ddof int) (float64, bool) {
	v, ok := s.VarDDOF(ddof)
	if !ok {
		return 0, false
	}
	return math.Sqrt(v), true
}

// Var returns the variance of all non-null values.
// Uses ddof=1 (sample variance) by default.
func (s *Series) Var() (float64, bool) {
	return s.VarDDOF(1)
}

// VarDDOF returns the variance with the specified delta degrees of freedom.
func (s *Series) VarDDOF(ddof int) (float64, bool) {
	switch s.dtype {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return array.Variance(ta, ddof)
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return array.Variance(ta, ddof)
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return array.Variance(ta, ddof)
		}
	}
	return 0, false
}

// Count returns the number of non-null values.
func (s *Series) Count() int {
	return s.Len() - s.NullCount()
}

// NUnique returns the number of unique non-null values.
// This is an O(n) operation that uses a hash set internally.
func (s *Series) NUnique() int {
	switch s.dtype {
	case dtype.Int64:
		return countUnique[int64](s)
	case dtype.Float64:
		return countUnique[float64](s)
	case dtype.String:
		return countUniqueString(s)
	case dtype.Boolean:
		return countUniqueBool(s)
	}
	return 0
}

func countUnique[T comparable](s *Series) int {
	ta, ok := s.arr.(*array.TypedArray[T])
	if !ok {
		return 0
	}
	seen := make(map[T]struct{})
	for i := 0; i < s.Len(); i++ {
		if !s.IsNull(i) {
			seen[ta.Value(i)] = struct{}{}
		}
	}
	return len(seen)
}

func countUniqueString(s *Series) int {
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return 0
	}
	seen := make(map[string]struct{})
	for i := 0; i < s.Len(); i++ {
		if !s.IsNull(i) {
			seen[sa.Value(i)] = struct{}{}
		}
	}
	return len(seen)
}

func countUniqueBool(s *Series) int {
	ba, ok := s.arr.(*array.BooleanArray)
	if !ok {
		return 0
	}
	hasTrue, hasFalse := false, false
	for i := 0; i < s.Len(); i++ {
		if !s.IsNull(i) {
			if ba.Value(i) {
				hasTrue = true
			} else {
				hasFalse = true
			}
			if hasTrue && hasFalse {
				return 2
			}
		}
	}
	if hasTrue || hasFalse {
		return 1
	}
	return 0
}

// Describe returns a summary string with count, mean, std, min, max.
func (s *Series) Describe() string {
	if !dtype.IsNumeric(s.dtype) {
		return fmt.Sprintf("Series '%s': count=%d, null_count=%d, unique=%d",
			s.name, s.Count(), s.NullCount(), s.NUnique())
	}
	mean, _ := s.Mean()
	std, _ := s.Std()
	min, _ := s.Min()
	max, _ := s.Max()
	return fmt.Sprintf("Series '%s': count=%d, null_count=%d, mean=%.4f, std=%.4f, min=%.4f, max=%.4f",
		s.name, s.Count(), s.NullCount(), mean, std, min, max)
}
