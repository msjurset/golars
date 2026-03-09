package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// isInExpr checks if each element is in a set of values.
type isInExpr struct {
	exprBase
	inner  Expr
	values []any
}

func (e *isInExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	n := s.Len()
	result := make([]bool, n)
	valid := make([]bool, n)

	switch s.DataType() {
	case dtype.Int64:
		set := make(map[int64]struct{})
		for _, v := range e.values {
			switch val := v.(type) {
			case int:
				set[int64(val)] = struct{}{}
			case int64:
				set[val] = struct{}{}
			case int32:
				set[int64(val)] = struct{}{}
			}
		}
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			valid[i] = true
			v, _ := s.GetInt64(i)
			_, result[i] = set[v]
		}
	case dtype.Float64:
		set := make(map[float64]struct{})
		for _, v := range e.values {
			switch val := v.(type) {
			case float64:
				set[val] = struct{}{}
			case float32:
				set[float64(val)] = struct{}{}
			case int:
				set[float64(val)] = struct{}{}
			case int64:
				set[float64(val)] = struct{}{}
			}
		}
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			valid[i] = true
			v, _ := s.GetFloat64(i)
			_, result[i] = set[v]
		}
	case dtype.String:
		set := make(map[string]struct{})
		for _, v := range e.values {
			if val, ok := v.(string); ok {
				set[val] = struct{}{}
			}
		}
		for i := 0; i < n; i++ {
			if s.IsNull(i) {
				continue
			}
			valid[i] = true
			v, _ := s.GetString(i)
			_, result[i] = set[v]
		}
	default:
		return nil, fmt.Errorf("golars: is_in not supported for type %s", s.DataType())
	}

	if s.HasNulls() {
		return series.NewBooleanWithValidity(s.Name(), result, valid), nil
	}
	return series.NewBoolean(s.Name(), result), nil
}

func (e *isInExpr) String() string {
	return fmt.Sprintf("%s.is_in(%v)", e.inner.String(), e.values)
}

// isBetweenExpr checks if each element is between lower and upper (inclusive).
type isBetweenExpr struct {
	exprBase
	inner Expr
	lower Expr
	upper Expr
}

func (e *isBetweenExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	lo, err := e.lower.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	hi, err := e.upper.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	// inner >= lower AND inner <= upper
	geResult, err := applyComparison(s, lo, cmpGte)
	if err != nil {
		return nil, fmt.Errorf("golars: is_between lower: %w", err)
	}
	leResult, err := applyComparison(s, hi, cmpLte)
	if err != nil {
		return nil, fmt.Errorf("golars: is_between upper: %w", err)
	}

	// AND the two boolean results
	geBool := geResult.BooleanArray()
	leBool := leResult.BooleanArray()
	if geBool == nil || leBool == nil {
		return nil, fmt.Errorf("golars: expected boolean arrays from comparison")
	}

	dataBM := geBool.DataBitmap().And(leBool.DataBitmap())
	validity := mergeArrayValidity(geBool.Validity(), leBool.Validity(), geBool.Len())
	return series.New(s.Name(), array.NewBooleanArrayFromBitmap(dataBM, validity)), nil
}

func (e *isBetweenExpr) String() string {
	return fmt.Sprintf("%s.is_between(%s, %s)", e.inner.String(), e.lower.String(), e.upper.String())
}
