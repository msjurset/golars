package series

import (
	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// Shift returns a new Series with values shifted by n positions.
// Positive n shifts down (introduces nulls at the top).
// Negative n shifts up (introduces nulls at the bottom).
func (s *Series) Shift(n int) *Series {
	length := s.Len()
	if length == 0 || n == 0 {
		return s
	}

	absN := n
	if absN < 0 {
		absN = -absN
	}
	if absN >= length {
		return s.nullSeries(length)
	}

	switch s.dtype {
	case dtype.Int64:
		return shiftTyped[int64](s, n, length)
	case dtype.Float64:
		return shiftTyped[float64](s, n, length)
	case dtype.String:
		return shiftString(s, n, length)
	case dtype.Boolean:
		return shiftBool(s, n, length)
	default:
		return s
	}
}

func shiftTyped[T int64 | float64](s *Series, n, length int) *Series {
	ta := s.arr.(*array.TypedArray[T])
	src := ta.Values()
	data := make([]T, length)
	valid := make([]bool, length)

	if n > 0 {
		copy(data[n:], src[:length-n])
		for i := n; i < length; i++ {
			valid[i] = s.IsValid(i - n)
		}
	} else {
		absN := -n
		copy(data[:length-absN], src[absN:])
		for i := 0; i < length-absN; i++ {
			valid[i] = s.IsValid(i + absN)
		}
	}

	var result *Series
	switch any(data).(type) {
	case []int64:
		result = NewInt64WithValidity(s.name, any(data).([]int64), valid)
	case []float64:
		result = NewFloat64WithValidity(s.name, any(data).([]float64), valid)
	}
	return result
}

func shiftString(s *Series, n, length int) *Series {
	data := make([]string, length)
	valid := make([]bool, length)

	if n > 0 {
		for i := n; i < length; i++ {
			if s.IsValid(i - n) {
				v, _ := s.GetString(i - n)
				data[i] = v
				valid[i] = true
			}
		}
	} else {
		absN := -n
		for i := 0; i < length-absN; i++ {
			if s.IsValid(i + absN) {
				v, _ := s.GetString(i + absN)
				data[i] = v
				valid[i] = true
			}
		}
	}

	return NewStringWithValidity(s.name, data, valid)
}

func shiftBool(s *Series, n, length int) *Series {
	data := make([]bool, length)
	valid := make([]bool, length)

	if n > 0 {
		for i := n; i < length; i++ {
			if s.IsValid(i - n) {
				v, _ := s.GetBool(i - n)
				data[i] = v
				valid[i] = true
			}
		}
	} else {
		absN := -n
		for i := 0; i < length-absN; i++ {
			if s.IsValid(i + absN) {
				v, _ := s.GetBool(i + absN)
				data[i] = v
				valid[i] = true
			}
		}
	}

	return NewBooleanWithValidity(s.name, data, valid)
}

func (s *Series) nullSeries(length int) *Series {
	valid := make([]bool, length)
	switch s.dtype {
	case dtype.Int64:
		return NewInt64WithValidity(s.name, make([]int64, length), valid)
	case dtype.Float64:
		return NewFloat64WithValidity(s.name, make([]float64, length), valid)
	case dtype.String:
		return NewStringWithValidity(s.name, make([]string, length), valid)
	case dtype.Boolean:
		return NewBooleanWithValidity(s.name, make([]bool, length), valid)
	default:
		return NewFloat64WithValidity(s.name, make([]float64, length), valid)
	}
}

// Diff returns a new Series with the difference between each element and the
// element n positions before it. The first n values will be null.
// Only supports Int64 and Float64 types.
func (s *Series) Diff(n int) *Series {
	if n <= 0 {
		n = 1
	}
	length := s.Len()
	if length == 0 {
		return s
	}

	switch s.dtype {
	case dtype.Int64:
		return diffInt64(s, n, length)
	case dtype.Float64:
		return diffFloat64(s, n, length)
	default:
		return nil
	}
}

func diffInt64(s *Series, n, length int) *Series {
	data := make([]int64, length)
	valid := make([]bool, length)

	for i := n; i < length; i++ {
		if s.IsValid(i) && s.IsValid(i-n) {
			curr, _ := s.GetInt64(i)
			prev, _ := s.GetInt64(i - n)
			data[i] = curr - prev
			valid[i] = true
		}
	}

	return NewInt64WithValidity(s.name, data, valid)
}

func diffFloat64(s *Series, n, length int) *Series {
	data := make([]float64, length)
	valid := make([]bool, length)

	for i := n; i < length; i++ {
		if s.IsValid(i) && s.IsValid(i-n) {
			curr, _ := s.GetFloat64(i)
			prev, _ := s.GetFloat64(i - n)
			data[i] = curr - prev
			valid[i] = true
		}
	}

	return NewFloat64WithValidity(s.name, data, valid)
}

// PctChange returns the percentage change between each element and the element
// n positions before it. Results are Float64. The first n values will be null.
func (s *Series) PctChange(n int) *Series {
	if n <= 0 {
		n = 1
	}
	length := s.Len()
	if length == 0 {
		return NewFloat64(s.name, nil)
	}

	data := make([]float64, length)
	valid := make([]bool, length)

	for i := n; i < length; i++ {
		if !s.IsValid(i) || !s.IsValid(i-n) {
			continue
		}
		var curr, prev float64
		switch s.dtype {
		case dtype.Int64:
			c, _ := s.GetInt64(i)
			p, _ := s.GetInt64(i - n)
			curr, prev = float64(c), float64(p)
		case dtype.Float64:
			curr, _ = s.GetFloat64(i)
			prev, _ = s.GetFloat64(i - n)
		default:
			continue
		}
		if prev != 0 {
			data[i] = (curr - prev) / prev
			valid[i] = true
		}
	}

	return NewFloat64WithValidity(s.name, data, valid)
}

// Ensure bitmap import is used.
var _ = bitmap.New
