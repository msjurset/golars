package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

type binaryOp int

const (
	opAdd binaryOp = iota
	opSub
	opMul
	opDiv
)

var binaryOpNames = [...]string{
	opAdd: "+",
	opSub: "-",
	opMul: "*",
	opDiv: "/",
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

func applyArithmetic(left, right *series.Series, op binaryOp) (*series.Series, error) {
	lt, rt := left.DataType(), right.DataType()

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
	default:
		return nil, fmt.Errorf("golars: unknown arithmetic op %d", op)
	}
	return series.New(left.Name(), result), nil
}
