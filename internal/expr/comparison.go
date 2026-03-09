package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

type cmpOp int

const (
	cmpEq cmpOp = iota
	cmpNeq
	cmpLt
	cmpGt
	cmpLte
	cmpGte
)

var cmpOpNames = [...]string{
	cmpEq:  "==",
	cmpNeq: "!=",
	cmpLt:  "<",
	cmpGt:  ">",
	cmpLte: "<=",
	cmpGte: ">=",
}

// comparisonExpr evaluates a comparison between two expressions, producing a boolean Series.
type comparisonExpr struct {
	exprBase
	left  Expr
	right Expr
	op    cmpOp
}

func (c *comparisonExpr) ensureSelf() {
	if c.exprBase.self == nil {
		c.exprBase.self = c
	}
}

func (c *comparisonExpr) Evaluate(ctx *Context) (*series.Series, error) {
	c.ensureSelf()
	left, err := c.left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	right, err := c.right.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return applyComparison(left, right, c.op)
}

func (c *comparisonExpr) Alias(name string) Expr {
	c.ensureSelf()
	return c.exprBase.Alias(name)
}

func (c *comparisonExpr) String() string {
	c.ensureSelf()
	return fmt.Sprintf("(%s %s %s)", c.left.String(), cmpOpNames[c.op], c.right.String())
}

func applyComparison(left, right *series.Series, op cmpOp) (*series.Series, error) {
	lt, rt := left.DataType(), right.DataType()

	// Same type fast paths
	if lt == rt {
		switch lt {
		case dtype.Int64:
			return compareInt64(left, right, op)
		case dtype.Float64:
			return compareFloat64(left, right, op)
		case dtype.String:
			return compareString(left, right, op)
		case dtype.Boolean:
			return compareBool(left, right, op)
		}
		return nil, fmt.Errorf("golars: comparison not supported for type %s", lt)
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
		return compareFloat64(lf, rf, op)
	}

	return nil, fmt.Errorf("golars: cannot compare %s and %s", lt, rt)
}

func compareInt64(left, right *series.Series, op cmpOp) (*series.Series, error) {
	la := left.Array().(*array.TypedArray[int64])
	ra := right.Array().(*array.TypedArray[int64])
	var result *array.BooleanArray
	switch op {
	case cmpEq:
		result = array.Equal(la, ra)
	case cmpNeq:
		result = array.NotEqual(la, ra)
	case cmpLt:
		result = array.LessThan(la, ra)
	case cmpGt:
		result = array.GreaterThan(la, ra)
	case cmpLte:
		result = array.LessThanEqual(la, ra)
	case cmpGte:
		result = array.GreaterThanEqual(la, ra)
	default:
		return nil, fmt.Errorf("golars: unknown comparison op %d", op)
	}
	return series.New(left.Name(), result), nil
}

func compareFloat64(left, right *series.Series, op cmpOp) (*series.Series, error) {
	la := left.Array().(*array.TypedArray[float64])
	ra := right.Array().(*array.TypedArray[float64])
	var result *array.BooleanArray
	switch op {
	case cmpEq:
		result = array.Equal(la, ra)
	case cmpNeq:
		result = array.NotEqual(la, ra)
	case cmpLt:
		result = array.LessThan(la, ra)
	case cmpGt:
		result = array.GreaterThan(la, ra)
	case cmpLte:
		result = array.LessThanEqual(la, ra)
	case cmpGte:
		result = array.GreaterThanEqual(la, ra)
	default:
		return nil, fmt.Errorf("golars: unknown comparison op %d", op)
	}
	return series.New(left.Name(), result), nil
}

func compareString(left, right *series.Series, op cmpOp) (*series.Series, error) {
	la := left.StringArray()
	ra := right.StringArray()
	if la == nil || ra == nil {
		return nil, fmt.Errorf("golars: expected string arrays for comparison")
	}
	n := la.Len()
	result := make([]bool, n)
	for i := 0; i < n; i++ {
		if la.IsNull(i) || ra.IsNull(i) {
			continue
		}
		lv, rv := la.Value(i), ra.Value(i)
		switch op {
		case cmpEq:
			result[i] = lv == rv
		case cmpNeq:
			result[i] = lv != rv
		case cmpLt:
			result[i] = lv < rv
		case cmpGt:
			result[i] = lv > rv
		case cmpLte:
			result[i] = lv <= rv
		case cmpGte:
			result[i] = lv >= rv
		}
	}
	validity := mergeArrayValidity(la.Validity(), ra.Validity(), n)
	return series.New(left.Name(), array.NewBooleanArray(result, validity)), nil
}

func compareBool(left, right *series.Series, op cmpOp) (*series.Series, error) {
	la := left.BooleanArray()
	ra := right.BooleanArray()
	if la == nil || ra == nil {
		return nil, fmt.Errorf("golars: expected boolean arrays for comparison")
	}
	n := la.Len()
	result := make([]bool, n)
	for i := 0; i < n; i++ {
		if la.IsNull(i) || ra.IsNull(i) {
			continue
		}
		lv, rv := la.Value(i), ra.Value(i)
		switch op {
		case cmpEq:
			result[i] = lv == rv
		case cmpNeq:
			result[i] = lv != rv
		default:
			return nil, fmt.Errorf("golars: boolean comparison only supports == and !=")
		}
	}
	validity := mergeArrayValidity(la.Validity(), ra.Validity(), n)
	return series.New(left.Name(), array.NewBooleanArray(result, validity)), nil
}

// mergeArrayValidity combines two validity bitmaps for a binary operation result.
func mergeArrayValidity(va, vb *bitmap.Bitmap, _ int) *bitmap.Bitmap {
	if va != nil && vb != nil {
		return va.And(vb)
	}
	if va != nil {
		return va.Clone()
	}
	if vb != nil {
		return vb.Clone()
	}
	return nil
}
