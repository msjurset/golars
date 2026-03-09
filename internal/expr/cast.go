package expr

import (
	"fmt"
	"strconv"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// castExpr casts a series to a different data type.
type castExpr struct {
	exprBase
	inner  Expr
	target dtype.DataType
}

func (e *castExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() == e.target {
		return s, nil
	}
	return castSeries(s, e.target)
}

func (e *castExpr) String() string {
	return fmt.Sprintf("%s.cast(%s)", e.inner.String(), e.target)
}

// castSeries converts a series to the target data type.
func castSeries(s *series.Series, target dtype.DataType) (*series.Series, error) {
	n := s.Len()

	switch {
	case s.DataType() == dtype.Int64 && target == dtype.Float64:
		return int64ToFloat64(s), nil

	case s.DataType() == dtype.Float64 && target == dtype.Int64:
		ta, ok := s.Array().(*array.TypedArray[float64])
		if !ok {
			return nil, fmt.Errorf("golars: cast: unexpected array type")
		}
		vals := ta.Values()
		data := make([]int64, n)
		for i, v := range vals {
			data[i] = int64(v)
		}
		if ta.Validity() != nil {
			return series.NewInt64WithValidity(s.Name(), data, validBools(s)), nil
		}
		return series.NewInt64(s.Name(), data), nil

	case s.DataType() == dtype.Int64 && target == dtype.String:
		data := make([]string, n)
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			v, _ := s.GetInt64(i)
			data[i] = strconv.FormatInt(v, 10)
		}
		if s.HasNulls() {
			return series.NewStringWithValidity(s.Name(), data, validBools(s)), nil
		}
		return series.NewString(s.Name(), data), nil

	case s.DataType() == dtype.Float64 && target == dtype.String:
		data := make([]string, n)
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			v, _ := s.GetFloat64(i)
			data[i] = strconv.FormatFloat(v, 'g', -1, 64)
		}
		if s.HasNulls() {
			return series.NewStringWithValidity(s.Name(), data, validBools(s)), nil
		}
		return series.NewString(s.Name(), data), nil

	case s.DataType() == dtype.String && target == dtype.Int64:
		sa := s.StringArray()
		if sa == nil {
			return nil, fmt.Errorf("golars: cast: not a string series")
		}
		data := make([]int64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			v := sa.Value(i)
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("golars: cast: cannot parse %q as Int64", v)
			}
			data[i] = parsed
			valid[i] = true
		}
		return series.NewInt64WithValidity(s.Name(), data, valid), nil

	case s.DataType() == dtype.String && target == dtype.Float64:
		sa := s.StringArray()
		if sa == nil {
			return nil, fmt.Errorf("golars: cast: not a string series")
		}
		data := make([]float64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			v := sa.Value(i)
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("golars: cast: cannot parse %q as Float64", v)
			}
			data[i] = parsed
			valid[i] = true
		}
		return series.NewFloat64WithValidity(s.Name(), data, valid), nil

	case s.DataType() == dtype.Boolean && target == dtype.Int64:
		ba := s.BooleanArray()
		if ba == nil {
			return nil, fmt.Errorf("golars: cast: not a boolean series")
		}
		data := make([]int64, n)
		for i := 0; i < n; i++ {
			if !s.IsNull(i) && ba.Value(i) {
				data[i] = 1
			}
		}
		if s.HasNulls() {
			return series.NewInt64WithValidity(s.Name(), data, validBools(s)), nil
		}
		return series.NewInt64(s.Name(), data), nil

	default:
		return nil, fmt.Errorf("golars: cast from %s to %s not supported", s.DataType(), target)
	}
}

// validBools returns a bool slice indicating which elements are valid (non-null).
func validBools(s *series.Series) []bool {
	n := s.Len()
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		valid[i] = s.IsValid(i)
	}
	return valid
}
