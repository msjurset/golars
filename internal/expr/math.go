package expr

import (
	"fmt"
	"math"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

type mathOp int

const (
	mathAbs mathOp = iota
	mathSqrt
	mathLog
	mathExp
	mathFloor
	mathCeil
)

var mathOpNames = [...]string{"abs", "sqrt", "log", "exp", "floor", "ceil"}

// mathExpr applies a unary math function element-wise.
type mathExpr struct {
	exprBase
	inner Expr
	op    mathOp
}

func (e *mathExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return applyMathOp(s, e.op)
}

func (e *mathExpr) String() string {
	return fmt.Sprintf("%s.%s()", e.inner.String(), mathOpNames[e.op])
}

func applyMathOp(s *series.Series, op mathOp) (*series.Series, error) {
	fn := mathFunc(op)

	switch s.DataType() {
	case dtype.Float64:
		ta, ok := s.Array().(*array.TypedArray[float64])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Float64")
		}
		vals := ta.Values()
		result := make([]float64, len(vals))
		for i, v := range vals {
			result[i] = fn(v)
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Float64, ta.Validity().Clone())), nil
		}
		return series.NewFloat64(s.Name(), result), nil

	case dtype.Float32:
		ta, ok := s.Array().(*array.TypedArray[float32])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Float32")
		}
		vals := ta.Values()
		result := make([]float64, len(vals))
		for i, v := range vals {
			result[i] = fn(float64(v))
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Float64, ta.Validity().Clone())), nil
		}
		return series.NewFloat64(s.Name(), result), nil

	case dtype.Int64:
		ta, ok := s.Array().(*array.TypedArray[int64])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Int64")
		}
		vals := ta.Values()
		if op == mathAbs {
			// Abs on integers stays integer
			result := make([]int64, len(vals))
			for i, v := range vals {
				if v < 0 {
					result[i] = -v
				} else {
					result[i] = v
				}
			}
			if ta.Validity() != nil {
				return series.New(s.Name(), array.NewTypedArray(result, dtype.Int64, ta.Validity().Clone())), nil
			}
			return series.NewInt64(s.Name(), result), nil
		}
		result := make([]float64, len(vals))
		for i, v := range vals {
			result[i] = fn(float64(v))
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Float64, ta.Validity().Clone())), nil
		}
		return series.NewFloat64(s.Name(), result), nil

	default:
		return nil, fmt.Errorf("golars: %s not supported for %s", mathOpNames[op], s.DataType())
	}
}

func mathFunc(op mathOp) func(float64) float64 {
	switch op {
	case mathAbs:
		return math.Abs
	case mathSqrt:
		return math.Sqrt
	case mathLog:
		return math.Log
	case mathExp:
		return math.Exp
	case mathFloor:
		return math.Floor
	case mathCeil:
		return math.Ceil
	default:
		return func(v float64) float64 { return v }
	}
}

// roundExpr rounds values to a given number of decimal places.
type roundExpr struct {
	exprBase
	inner    Expr
	decimals int
}

func (e *roundExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return applyRound(s, e.decimals)
}

func (e *roundExpr) String() string {
	return fmt.Sprintf("%s.round(%d)", e.inner.String(), e.decimals)
}

func applyRound(s *series.Series, decimals int) (*series.Series, error) {
	factor := math.Pow(10, float64(decimals))

	switch s.DataType() {
	case dtype.Float64:
		ta, ok := s.Array().(*array.TypedArray[float64])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Float64")
		}
		vals := ta.Values()
		result := make([]float64, len(vals))
		for i, v := range vals {
			result[i] = math.Round(v*factor) / factor
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Float64, ta.Validity().Clone())), nil
		}
		return series.NewFloat64(s.Name(), result), nil

	case dtype.Float32:
		ta, ok := s.Array().(*array.TypedArray[float32])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Float32")
		}
		vals := ta.Values()
		result := make([]float64, len(vals))
		for i, v := range vals {
			result[i] = math.Round(float64(v)*factor) / factor
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Float64, ta.Validity().Clone())), nil
		}
		return series.NewFloat64(s.Name(), result), nil

	case dtype.Int64:
		// Rounding integers is a no-op for decimals >= 0
		if decimals >= 0 {
			return s, nil
		}
		ta, ok := s.Array().(*array.TypedArray[int64])
		if !ok {
			return nil, fmt.Errorf("golars: unexpected array type for Int64")
		}
		vals := ta.Values()
		result := make([]int64, len(vals))
		f := int64(factor)
		if f == 0 {
			f = 1
		}
		for i, v := range vals {
			result[i] = int64(math.Round(float64(v)/float64(f))) * f
		}
		if ta.Validity() != nil {
			return series.New(s.Name(), array.NewTypedArray(result, dtype.Int64, ta.Validity().Clone())), nil
		}
		return series.NewInt64(s.Name(), result), nil

	default:
		return nil, fmt.Errorf("golars: round not supported for %s", s.DataType())
	}
}
