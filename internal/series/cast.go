package series

import (
	"fmt"
	"strconv"

	"github.com/msjurset/golars/internal/dtype"
)

// Cast converts the Series to the target data type.
// Supported casts:
//   - Int64 <-> Float64 (numeric conversion)
//   - Int64/Float64 -> String (formatting)
//   - String -> Int64/Float64 (parsing, invalid values become null)
//   - Boolean -> Int64 (true=1, false=0)
//   - Int64 -> Boolean (0=false, nonzero=true)
func (s *Series) Cast(target dtype.DataType) (*Series, error) {
	if s.dtype == target {
		return s, nil
	}

	switch {
	case s.dtype == dtype.Int64 && target == dtype.Float64:
		return s.int64ToFloat64(), nil
	case s.dtype == dtype.Float64 && target == dtype.Int64:
		return s.float64ToInt64(), nil
	case s.dtype == dtype.Int64 && target == dtype.String:
		return s.int64ToString(), nil
	case s.dtype == dtype.Float64 && target == dtype.String:
		return s.float64ToString(), nil
	case s.dtype == dtype.String && target == dtype.Int64:
		return s.stringToInt64(), nil
	case s.dtype == dtype.String && target == dtype.Float64:
		return s.stringToFloat64(), nil
	case s.dtype == dtype.Boolean && target == dtype.Int64:
		return s.boolToInt64(), nil
	case s.dtype == dtype.Int64 && target == dtype.Boolean:
		return s.int64ToBool(), nil
	case s.dtype == dtype.Boolean && target == dtype.String:
		return s.boolToString(), nil
	default:
		return nil, fmt.Errorf("golars: cast: cannot cast %s to %s", s.dtype, target)
	}
}

func (s *Series) int64ToFloat64() *Series {
	n := s.Len()
	data := make([]float64, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetInt64(i)
			data[i] = float64(v)
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewFloat64WithValidity(s.name, data, valid)
	}
	return NewFloat64(s.name, data)
}

func (s *Series) float64ToInt64() *Series {
	n := s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetFloat64(i)
			data[i] = int64(v)
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewInt64WithValidity(s.name, data, valid)
	}
	return NewInt64(s.name, data)
}

func (s *Series) int64ToString() *Series {
	n := s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetInt64(i)
			data[i] = strconv.FormatInt(v, 10)
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewStringWithValidity(s.name, data, valid)
	}
	return NewString(s.name, data)
}

func (s *Series) float64ToString() *Series {
	n := s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetFloat64(i)
			data[i] = strconv.FormatFloat(v, 'f', -1, 64)
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewStringWithValidity(s.name, data, valid)
	}
	return NewString(s.name, data)
}

func (s *Series) stringToInt64() *Series {
	n := s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if !s.IsValid(i) {
			continue
		}
		v, _ := s.GetString(i)
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			data[i] = parsed
			valid[i] = true
		}
	}
	return NewInt64WithValidity(s.name, data, valid)
}

func (s *Series) stringToFloat64() *Series {
	n := s.Len()
	data := make([]float64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if !s.IsValid(i) {
			continue
		}
		v, _ := s.GetString(i)
		parsed, err := strconv.ParseFloat(v, 64)
		if err == nil {
			data[i] = parsed
			valid[i] = true
		}
	}
	return NewFloat64WithValidity(s.name, data, valid)
}

func (s *Series) boolToInt64() *Series {
	n := s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetBool(i)
			if v {
				data[i] = 1
			}
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewInt64WithValidity(s.name, data, valid)
	}
	return NewInt64(s.name, data)
}

func (s *Series) int64ToBool() *Series {
	n := s.Len()
	data := make([]bool, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetInt64(i)
			data[i] = v != 0
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewBooleanWithValidity(s.name, data, valid)
	}
	return NewBoolean(s.name, data)
}

func (s *Series) boolToString() *Series {
	n := s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	hasNulls := false
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			v, _ := s.GetBool(i)
			if v {
				data[i] = "true"
			} else {
				data[i] = "false"
			}
			valid[i] = true
		} else {
			hasNulls = true
		}
	}
	if hasNulls {
		return NewStringWithValidity(s.name, data, valid)
	}
	return NewString(s.name, data)
}
