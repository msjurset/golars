package expr

import (
	"fmt"
	"math"
	"sort"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

type aggOp int

const (
	aggSum aggOp = iota
	aggMean
	aggMin
	aggMax
	aggCount
	aggStd
	aggVar
	aggNUnique
	aggMedian
	aggFirst
	aggLast
)

var aggOpNames = [...]string{
	aggSum:     "sum",
	aggMean:    "mean",
	aggMin:     "min",
	aggMax:     "max",
	aggCount:   "count",
	aggStd:     "std",
	aggVar:     "var",
	aggNUnique: "n_unique",
	aggMedian:  "median",
	aggFirst:   "first",
	aggLast:    "last",
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

	case aggVar:
		v, ok := s.Var()
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggNUnique:
		n := s.NUnique()
		return series.NewInt64(name, []int64{int64(n)}), nil

	case aggMedian:
		v, ok := medianForSeries(s)
		if !ok {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		return series.NewFloat64(name, []float64{v}), nil

	case aggFirst:
		if s.Len() == 0 {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		// Return the first non-null value, or the first value if all are null
		for i := 0; i < s.Len(); i++ {
			if s.IsValid(i) {
				return s.Slice(i, i+1), nil
			}
		}
		return s.Slice(0, 1), nil

	case aggLast:
		if s.Len() == 0 {
			return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
		}
		// Return the last non-null value, or the last value if all are null
		for i := s.Len() - 1; i >= 0; i-- {
			if s.IsValid(i) {
				return s.Slice(i, i+1), nil
			}
		}
		return s.Slice(s.Len()-1, s.Len()), nil

	default:
		return nil, fmt.Errorf("golars: unknown aggregation op %d", op)
	}
}

// collectFloat64Values extracts non-null float64 values from a numeric series.
func collectFloat64Values(s *series.Series) ([]float64, bool) {
	var vals []float64
	switch s.DataType() {
	case dtype.Int64:
		ta, ok := s.Array().(*array.TypedArray[int64])
		if !ok {
			return nil, false
		}
		for i := 0; i < s.Len(); i++ {
			if s.IsValid(i) {
				vals = append(vals, float64(ta.Value(i)))
			}
		}
	case dtype.Float64:
		ta, ok := s.Array().(*array.TypedArray[float64])
		if !ok {
			return nil, false
		}
		for i := 0; i < s.Len(); i++ {
			if s.IsValid(i) {
				vals = append(vals, ta.Value(i))
			}
		}
	case dtype.Float32:
		ta, ok := s.Array().(*array.TypedArray[float32])
		if !ok {
			return nil, false
		}
		for i := 0; i < s.Len(); i++ {
			if s.IsValid(i) {
				vals = append(vals, float64(ta.Value(i)))
			}
		}
	default:
		return nil, false
	}
	if len(vals) == 0 {
		return nil, false
	}
	return vals, true
}

func medianForSeries(s *series.Series) (float64, bool) {
	vals, ok := collectFloat64Values(s)
	if !ok {
		return 0, false
	}
	sort.Float64s(vals)
	n := len(vals)
	if n%2 == 1 {
		return vals[n/2], true
	}
	return (vals[n/2-1] + vals[n/2]) / 2.0, true
}

func quantileForSeries(s *series.Series, p float64) (float64, bool) {
	if p < 0 || p > 1 {
		return 0, false
	}
	vals, ok := collectFloat64Values(s)
	if !ok {
		return 0, false
	}
	sort.Float64s(vals)
	n := len(vals)
	if n == 1 {
		return vals[0], true
	}
	// Linear interpolation
	pos := p * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return vals[lo], true
	}
	frac := pos - float64(lo)
	return vals[lo]*(1-frac) + vals[hi]*frac, true
}

// quantileExpr evaluates a quantile aggregation on the inner expression.
type quantileExpr struct {
	exprBase
	inner      Expr
	percentile float64
}

func (q *quantileExpr) ensureSelf() {
	if q.exprBase.self == nil {
		q.exprBase.self = q
	}
}

func (q *quantileExpr) Evaluate(ctx *Context) (*series.Series, error) {
	q.ensureSelf()
	s, err := q.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	name := s.Name()
	v, ok := quantileForSeries(s, q.percentile)
	if !ok {
		return series.NewFloat64WithValidity(name, []float64{0}, []bool{false}), nil
	}
	return series.NewFloat64(name, []float64{v}), nil
}

func (q *quantileExpr) Alias(name string) Expr {
	q.ensureSelf()
	return q.exprBase.Alias(name)
}

func (q *quantileExpr) String() string {
	q.ensureSelf()
	return fmt.Sprintf("%s.quantile(%g)", q.inner.String(), q.percentile)
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
