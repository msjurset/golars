package series

import (
	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dtype"
)

// Unique returns a new Series containing only unique values. Order is preserved
// (first occurrence is kept). Null appears at most once in the result.
func (s *Series) Unique() *Series {
	switch s.dtype {
	case dtype.Int64:
		return uniqueTyped[int64](s)
	case dtype.Int32:
		return uniqueTyped[int32](s)
	case dtype.Float64:
		return uniqueTyped[float64](s)
	case dtype.Float32:
		return uniqueTyped[float32](s)
	case dtype.String:
		return uniqueString(s)
	case dtype.Boolean:
		return uniqueBool(s)
	}
	return s
}

func uniqueTyped[T comparable](s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[T])
	if !ok {
		return s
	}
	seen := make(map[T]struct{})
	seenNull := false
	indices := make([]int, 0)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			if !seenNull {
				seenNull = true
				indices = append(indices, i)
			}
			continue
		}
		v := ta.Value(i)
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

func uniqueString(s *Series) *Series {
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return s
	}
	seen := make(map[string]struct{})
	seenNull := false
	indices := make([]int, 0)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			if !seenNull {
				seenNull = true
				indices = append(indices, i)
			}
			continue
		}
		v := sa.Value(i)
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

func uniqueBool(s *Series) *Series {
	ba, ok := s.arr.(*array.BooleanArray)
	if !ok {
		return s
	}
	hasTrue, hasFalse, hasNull := false, false, false
	indices := make([]int, 0, 3)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			if !hasNull {
				hasNull = true
				indices = append(indices, i)
			}
			continue
		}
		v := ba.Value(i)
		if v && !hasTrue {
			hasTrue = true
			indices = append(indices, i)
		} else if !v && !hasFalse {
			hasFalse = true
			indices = append(indices, i)
		}
		if hasTrue && hasFalse && hasNull {
			break
		}
	}
	return s.Take(indices)
}

// IsDuplicated returns a Boolean Series where true indicates the value at that
// position appears more than once in the Series.
func (s *Series) IsDuplicated() *Series {
	switch s.dtype {
	case dtype.Int64:
		return isDuplicatedTyped[int64](s)
	case dtype.Float64:
		return isDuplicatedTyped[float64](s)
	case dtype.String:
		return isDuplicatedString(s)
	}
	n := s.Len()
	result := make([]bool, n)
	return NewBoolean(s.name, result)
}

func isDuplicatedTyped[T comparable](s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[T])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	counts := make(map[T]int)
	nullCount := 0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			nullCount++
		} else {
			counts[ta.Value(i)]++
		}
	}
	result := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			result[i] = nullCount > 1
		} else {
			result[i] = counts[ta.Value(i)] > 1
		}
	}
	return NewBoolean(s.name, result)
}

func isDuplicatedString(s *Series) *Series {
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	counts := make(map[string]int)
	nullCount := 0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			nullCount++
		} else {
			counts[sa.Value(i)]++
		}
	}
	result := make([]bool, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			result[i] = nullCount > 1
		} else {
			result[i] = counts[sa.Value(i)] > 1
		}
	}
	return NewBoolean(s.name, result)
}
