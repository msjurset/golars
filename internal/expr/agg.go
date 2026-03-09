package expr

import (
	"fmt"
	"math"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

type aggOp int

const (
	aggSum aggOp = iota
	aggMean
	aggMin
	aggMax
	aggCount
	aggStd
)

var aggOpNames = [...]string{
	aggSum:   "sum",
	aggMean:  "mean",
	aggMin:   "min",
	aggMax:   "max",
	aggCount: "count",
	aggStd:   "std",
}

// aggExpr evaluates an aggregation on the inner expression, producing a single-element Series.
type aggExpr struct {
	exprBase
	inner Expr
	op    aggOp
}

func (a *aggExpr) ensureSelf() {
	if a.exprBase.self == nil {
		a.exprBase.self = a
	}
}

func (a *aggExpr) Evaluate(ctx *Context) (*series.Series, error) {
	a.ensureSelf()
	s, err := a.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return applyAgg(s, a.op)
}

func (a *aggExpr) Alias(name string) Expr {
	a.ensureSelf()
	return a.exprBase.Alias(name)
}

func (a *aggExpr) String() string {
	a.ensureSelf()
	return fmt.Sprintf("%s.%s()", a.inner.String(), aggOpNames[a.op])
}

func applyAgg(s *series.Series, op aggOp) (*series.Series, error) {
	name := s.Name()

	switch op {
	case aggSum:
		v, ok := s.Sum()
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggMean:
		v, ok := s.Mean()
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggMin:
		v, ok := s.Min()
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggMax:
		v, ok := s.Max()
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggCount:
		count := s.Count()
		return series.NewInt64(name, []int64{int64(count)}), nil

	case aggStd:
		v, ok := stdForSeries(s)
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	default:
		return nil, fmt.Errorf("golars: unknown aggregation op %d", op)
	}
}

func stdForSeries(s *series.Series) (float64, bool) {
	switch s.DataType() {
	case dtype.Int64:
		if ta, ok := s.Array().(*array.TypedArray[int64]); ok {
			return array.Std(ta, 1)
		}
	case dtype.Float64:
		if ta, ok := s.Array().(*array.TypedArray[float64]); ok {
			return array.Std(ta, 1)
		}
	case dtype.Float32:
		if ta, ok := s.Array().(*array.TypedArray[float32]); ok {
			return array.Std(ta, 1)
		}
	}
	return math.NaN(), false
}
