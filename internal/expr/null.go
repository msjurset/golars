package expr

import (
	"fmt"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// isNullExpr checks if values are null or not null.
type isNullExpr struct {
	exprBase
	inner  Expr
	negate bool // if true, IsNotNull
}

func (e *isNullExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if e.negate {
		return s.IsNotNullSeries(), nil
	}
	return s.IsNullSeries(), nil
}

func (e *isNullExpr) String() string {
	if e.negate {
		return fmt.Sprintf("%s.is_not_null()", e.inner.String())
	}
	return fmt.Sprintf("%s.is_null()", e.inner.String())
}

// fillNullExpr replaces null values with a fill expression.
type fillNullExpr struct {
	exprBase
	inner Expr
	fill  Expr
}

func (e *fillNullExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if !s.HasNulls() {
		return s, nil
	}
	fillS, err := e.fill.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	switch s.DataType() {
	case dtype.Int64:
		if fillS.DataType() == dtype.Int64 {
			v, _ := fillS.GetInt64(0)
			return s.FillNullInt64(v), nil
		}
	case dtype.Float64:
		if fillS.DataType() == dtype.Float64 {
			v, _ := fillS.GetFloat64(0)
			return s.FillNullFloat64(v), nil
		}
	case dtype.String:
		if fillS.DataType() == dtype.String {
			v, _ := fillS.GetString(0)
			return s.FillNullString(v), nil
		}
	}
	return nil, fmt.Errorf("golars: fill_null: type mismatch: %s vs %s", s.DataType(), fillS.DataType())
}

func (e *fillNullExpr) String() string {
	return fmt.Sprintf("%s.fill_null(%s)", e.inner.String(), e.fill.String())
}
