package expr

import (
	"fmt"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

type logOp int

const (
	logAnd logOp = iota
	logOr
)

var logOpNames = [...]string{
	logAnd: "and",
	logOr:  "or",
}

// logicalExpr evaluates a logical AND/OR between two boolean expressions.
type logicalExpr struct {
	exprBase
	left  Expr
	right Expr
	op    logOp
}

func (l *logicalExpr) ensureSelf() {
	if l.exprBase.self == nil {
		l.exprBase.self = l
	}
}

func (l *logicalExpr) Evaluate(ctx *Context) (*series.Series, error) {
	l.ensureSelf()
	left, err := l.left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	right, err := l.right.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if left.DataType() != dtype.Boolean || right.DataType() != dtype.Boolean {
		return nil, fmt.Errorf("golars: logical operations require boolean operands, got %s and %s", left.DataType(), right.DataType())
	}
	la := left.BooleanArray()
	ra := right.BooleanArray()
	if la == nil || ra == nil {
		return nil, fmt.Errorf("golars: expected boolean arrays for logical operation")
	}

	var dataBM *bitmap.Bitmap
	switch l.op {
	case logAnd:
		dataBM = la.DataBitmap().And(ra.DataBitmap())
	case logOr:
		dataBM = la.DataBitmap().Or(ra.DataBitmap())
	default:
		return nil, fmt.Errorf("golars: unknown logical op %d", l.op)
	}

	validity := mergeArrayValidity(la.Validity(), ra.Validity(), la.Len())
	return series.New(left.Name(), array.NewBooleanArrayFromBitmap(dataBM, validity)), nil
}

func (l *logicalExpr) Alias(name string) Expr {
	l.ensureSelf()
	return l.exprBase.Alias(name)
}

func (l *logicalExpr) String() string {
	l.ensureSelf()
	return fmt.Sprintf("(%s %s %s)", l.left.String(), logOpNames[l.op], l.right.String())
}

// notExpr evaluates logical NOT on a boolean expression.
type notExpr struct {
	exprBase
	inner Expr
}

func (n *notExpr) ensureSelf() {
	if n.exprBase.self == nil {
		n.exprBase.self = n
	}
}

func (n *notExpr) Evaluate(ctx *Context) (*series.Series, error) {
	n.ensureSelf()
	s, err := n.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() != dtype.Boolean {
		return nil, fmt.Errorf("golars: Not() requires boolean operand, got %s", s.DataType())
	}
	ba := s.BooleanArray()
	if ba == nil {
		return nil, fmt.Errorf("golars: expected boolean array for Not()")
	}
	dataBM := ba.DataBitmap().Not()
	var validity *bitmap.Bitmap
	if ba.Validity() != nil {
		validity = ba.Validity().Clone()
	}
	return series.New(s.Name(), array.NewBooleanArrayFromBitmap(dataBM, validity)), nil
}

func (n *notExpr) Alias(name string) Expr {
	n.ensureSelf()
	return n.exprBase.Alias(name)
}

func (n *notExpr) String() string {
	n.ensureSelf()
	return fmt.Sprintf("not(%s)", n.inner.String())
}
