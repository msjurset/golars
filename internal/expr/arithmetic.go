package expr

import (
	"fmt"
	"math"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

type binaryOp int

const (
	opAdd binaryOp = iota
	opSub
	opMul
	opDiv
	opMod
	opPow
)

var binaryOpNames = [...]string{
	opAdd: "+",
	opSub: "-",
	opMul: "*",
	opDiv: "/",
	opMod: "%",
	opPow: "**",
}

// binaryExpr evaluates a binary arithmetic operation on two expressions.
type binaryExpr struct {
	exprBase
	left  Expr
	right Expr
	op    binaryOp
}

func init() {
	// Cannot set self in struct literal because exprBase needs a pointer
	// to the containing type. We do this lazily in Evaluate/Alias/String instead.
}

func (b *binaryExpr) ensureSelf() {
	if b.exprBase.self == nil {
		b.exprBase.self = b
	}
}

func (b *binaryExpr) Evaluate(ctx *Context) (*series.Series, error) {
	b.ensureSelf()

	// Fast path: scalar right operand (Col op Lit)
	if litR, ok := b.right.(*litExpr); ok {
		left, err := b.left.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		result, err := applyArithmeticScalar(left, litR.value, b.op, false)
		if err == nil {
			return result, nil
		}
		// Fall through to generic path
	}

	// Fast path: scalar left operand (Lit op Col)
	if litL, ok := b.left.(*litExpr); ok {
		right, err := b.right.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		result, err := applyArithmeticScalar(right, litL.value, b.op, true)
		if err == nil {
			return result, nil
		}
		// Fall through to generic path
	}

	left, err := b.left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	right, err := b.right.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return applyArithmetic(left, right, b.op)
}

func (b *binaryExpr) Alias(name string) Expr {
	b.ensureSelf()
	return b.exprBase.Alias(name)
}

func (b *binaryExpr) String() string {
	b.ensureSelf()
	return fmt.Sprintf("(%s %s %s)", b.left.String(), binaryOpNames[b.op], b.right.String())
}

// applyArithmeticScalar handles arithmetic where one operand is a scalar.
// If reversed is true, the scalar is the left operand (scalar op col).
func applyArithmeticScalar(col *series.Series, scalar any, op binaryOp, reversed bool) (*series.Series, error) {
	dt := col.DataType()

	switch dt {
	case dtype.Float64:
		var sv float64
		switch v := scalar.(type) {
		case float64:
			sv = v
		case float32:
			sv = float64(v)
		case int:
			sv = float64(v)
		case int64:
			sv = float64(v)
		case int32:
			sv = float64(v)
		default:
			return nil, fmt.Errorf("golars: cannot apply arithmetic on %s with %T", dt, scalar)
		}
		return arithmeticFloat64Scalar(col, sv, op, reversed)

	case dtype.Int64:
		switch v := scalar.(type) {
		case int:
			return arithmeticInt64Scalar(col, int64(v), op, reversed)
		case int64:
			return arithmeticInt64Scalar(col, v, op, reversed)
		case int32:
			return arithmeticInt64Scalar(col, int64(v), op, reversed)
		case float64:
			promoted, err := promoteToFloat64(col)
			if err != nil {
				return nil, err
			}
			return arithmeticFloat64Scalar(promoted, v, op, reversed)
		default:
			return nil, fmt.Errorf("golars: cannot apply arithmetic on %s with %T", dt, scalar)
		}

	default:
		return nil, fmt.Errorf("golars: scalar arithmetic not supported for type %s", dt)
	}
}

func arithmeticFloat64Scalar(s *series.Series, scalar float64, op binaryOp, reversed bool) (*series.Series, error) {
	a := s.Array().(*array.TypedArray[float64])
	var result *array.TypedArray[float64]

	switch op {
	case opAdd:
		result = array.AddScalar(a, scalar)
	case opSub:
		if reversed {
			// scalar - col: negate then add
			neg := array.Neg(a)
			result = array.AddScalar(neg, scalar)
		} else {
			result = array.SubScalar(a, scalar)
		}
	case opMul:
		result = array.MulScalar(a, scalar)
	case opDiv:
		if reversed {
			return nil, fmt.Errorf("golars: scalar/col not supported in scalar fast path")
		}
		result = array.DivScalar(a, scalar)
	default:
		return nil, fmt.Errorf("golars: op %d not supported in scalar fast path", op)
	}
	return series.New(s.Name(), result), nil
}

func arithmeticInt64Scalar(s *series.Series, scalar int64, op binaryOp, reversed bool) (*series.Series, error) {
	a := s.Array().(*array.TypedArray[int64])
	var result *array.TypedArray[int64]

	switch op {
	case opAdd:
		result = array.AddScalar(a, scalar)
	case opSub:
		if reversed {
			neg := array.Neg(a)
			result = array.AddScalar(neg, scalar)
		} else {
			result = array.SubScalar(a, scalar)
		}
	case opMul:
		result = array.MulScalar(a, scalar)
	case opDiv:
		if reversed {
			return nil, fmt.Errorf("golars: scalar/col not supported in scalar fast path")
		}
		result = array.DivScalar(a, scalar)
	default:
		return nil, fmt.Errorf("golars: op %d not supported in scalar fast path", op)
	}
	return series.New(s.Name(), result), nil
}

func applyArithmetic(left, right *series.Series, op binaryOp) (*series.Series, error) {
	lt, rt := left.DataType(), right.DataType()

	// Pow always operates on float64
	if op == opPow {
		lf, err := promoteToFloat64(left)
		if err != nil {
			return nil, err
		}
		rf, err := promoteToFloat64(right)
		if err != nil {
			return nil, err
		}
		return arithmeticFloat64(lf, rf, op)
	}

	// Same type fast paths
	if lt == rt {
		switch lt {
		case dtype.Int64:
			return arithmeticInt64(left, right, op)
		case dtype.Float64:
			return arithmeticFloat64(left, right, op)
		}
		return nil, fmt.Errorf("golars: arithmetic not supported for type %s", lt)
	}

	// Mixed numeric: promote to float64
	if dtype.IsNumeric(lt) && dtype.IsNumeric(rt) {
		lf, err := promoteToFloat64(left)
		if err != nil {
			return nil, err
		}
		rf, err := promoteToFloat64(right)
		if err != nil {
			return nil, err
		}
		return arithmeticFloat64(lf, rf, op)
	}

	return nil, fmt.Errorf("golars: cannot perform arithmetic on %s and %s", lt, rt)
}

func arithmeticInt64(left, right *series.Series, op binaryOp) (*series.Series, error) {
	la := left.Array().(*array.TypedArray[int64])
	ra := right.Array().(*array.TypedArray[int64])
	var result *array.TypedArray[int64]
	switch op {
	case opAdd:
		result = array.Add(la, ra)
	case opSub:
		result = array.Sub(la, ra)
	case opMul:
		result = array.Mul(la, ra)
	case opDiv:
		result = array.Div(la, ra)
	case opMod:
		result = array.Mod(la, ra)
	default:
		return nil, fmt.Errorf("golars: unknown arithmetic op %d", op)
	}
	return series.New(left.Name(), result), nil
}

func arithmeticFloat64(left, right *series.Series, op binaryOp) (*series.Series, error) {
	la := left.Array().(*array.TypedArray[float64])
	ra := right.Array().(*array.TypedArray[float64])
	var result *array.TypedArray[float64]
	switch op {
	case opAdd:
		result = array.Add(la, ra)
	case opSub:
		result = array.Sub(la, ra)
	case opMul:
		result = array.Mul(la, ra)
	case opDiv:
		result = array.Div(la, ra)
	case opMod:
		av, bv := la.Values(), ra.Values()
		n := la.Len()
		res := make([]float64, n)
		for i := 0; i < n; i++ {
			res[i] = math.Mod(av[i], bv[i])
		}
		var validity *bitmap.Bitmap
		if la.Validity() != nil && ra.Validity() != nil {
			validity = la.Validity().And(ra.Validity())
		} else if la.Validity() != nil {
			validity = la.Validity().Clone()
		} else if ra.Validity() != nil {
			validity = ra.Validity().Clone()
		}
		result = array.NewTypedArray(res, dtype.Float64, validity)
	case opPow:
		result = array.Pow(la, ra)
	default:
		return nil, fmt.Errorf("golars: unknown arithmetic op %d", op)
	}
	return series.New(left.Name(), result), nil
}
