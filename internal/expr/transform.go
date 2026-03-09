package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/series"
)

// --- shiftExpr ---

type shiftExpr struct {
	exprBase
	inner Expr
	n     int
}

func (e *shiftExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Shift(e.n), nil
}

func (e *shiftExpr) String() string { return fmt.Sprintf("%s.shift(%d)", e.inner.String(), e.n) }

// --- diffExpr ---

type diffExpr struct {
	exprBase
	inner Expr
	n     int
}

func (e *diffExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Diff(e.n), nil
}

func (e *diffExpr) String() string { return fmt.Sprintf("%s.diff(%d)", e.inner.String(), e.n) }

// --- pctChangeExpr ---

type pctChangeExpr struct {
	exprBase
	inner Expr
	n     int
}

func (e *pctChangeExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.PctChange(e.n), nil
}

func (e *pctChangeExpr) String() string {
	return fmt.Sprintf("%s.pct_change(%d)", e.inner.String(), e.n)
}

// --- CumNamespace ---

// CumNamespace provides cumulative operations on expressions.
type CumNamespace struct {
	inner Expr
}

func (c *CumNamespace) Sum() Expr {
	r := &cumExpr{inner: c.inner, op: cumSum}
	r.exprBase.self = r
	return r
}

func (c *CumNamespace) Prod() Expr {
	r := &cumExpr{inner: c.inner, op: cumProd}
	r.exprBase.self = r
	return r
}

func (c *CumNamespace) Min() Expr {
	r := &cumExpr{inner: c.inner, op: cumMin}
	r.exprBase.self = r
	return r
}

func (c *CumNamespace) Max() Expr {
	r := &cumExpr{inner: c.inner, op: cumMax}
	r.exprBase.self = r
	return r
}

type cumOp int

const (
	cumSum cumOp = iota
	cumProd
	cumMin
	cumMax
)

type cumExpr struct {
	exprBase
	inner Expr
	op    cumOp
}

func (e *cumExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	switch e.op {
	case cumSum:
		return s.CumSum(), nil
	case cumProd:
		return s.CumProd(), nil
	case cumMin:
		return s.CumMin(), nil
	case cumMax:
		return s.CumMax(), nil
	default:
		return nil, fmt.Errorf("golars: unknown cumulative op %d", e.op)
	}
}

func (e *cumExpr) String() string {
	ops := [...]string{"cum_sum", "cum_prod", "cum_min", "cum_max"}
	return fmt.Sprintf("%s.%s()", e.inner.String(), ops[e.op])
}

// --- RollingNamespace ---

// RollingNamespace provides rolling window operations on expressions.
type RollingNamespace struct {
	inner  Expr
	window int
}

func (r *RollingNamespace) Mean() Expr {
	e := &rollingExpr{inner: r.inner, window: r.window, op: rollMean}
	e.exprBase.self = e
	return e
}

func (r *RollingNamespace) Sum() Expr {
	e := &rollingExpr{inner: r.inner, window: r.window, op: rollSum}
	e.exprBase.self = e
	return e
}

func (r *RollingNamespace) Min() Expr {
	e := &rollingExpr{inner: r.inner, window: r.window, op: rollMin}
	e.exprBase.self = e
	return e
}

func (r *RollingNamespace) Max() Expr {
	e := &rollingExpr{inner: r.inner, window: r.window, op: rollMax}
	e.exprBase.self = e
	return e
}

func (r *RollingNamespace) Std() Expr {
	e := &rollingExpr{inner: r.inner, window: r.window, op: rollStd}
	e.exprBase.self = e
	return e
}

type rollOp int

const (
	rollMean rollOp = iota
	rollSum
	rollMin
	rollMax
	rollStd
)

type rollingExpr struct {
	exprBase
	inner  Expr
	window int
	op     rollOp
}

func (e *rollingExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	switch e.op {
	case rollMean:
		return s.RollingMean(e.window), nil
	case rollSum:
		return s.RollingSum(e.window), nil
	case rollMin:
		return s.RollingMin(e.window), nil
	case rollMax:
		return s.RollingMax(e.window), nil
	case rollStd:
		return s.RollingStd(e.window), nil
	default:
		return nil, fmt.Errorf("golars: unknown rolling op %d", e.op)
	}
}

func (e *rollingExpr) String() string {
	ops := [...]string{"rolling_mean", "rolling_sum", "rolling_min", "rolling_max", "rolling_std"}
	return fmt.Sprintf("%s.%s(%d)", e.inner.String(), ops[e.op], e.window)
}

// --- headExpr ---

type headExpr struct {
	exprBase
	inner Expr
	n     int
}

func (e *headExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Head(e.n), nil
}

func (e *headExpr) String() string { return fmt.Sprintf("%s.head(%d)", e.inner.String(), e.n) }

// --- tailExpr ---

type tailExpr struct {
	exprBase
	inner Expr
	n     int
}

func (e *tailExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Tail(e.n), nil
}

func (e *tailExpr) String() string { return fmt.Sprintf("%s.tail(%d)", e.inner.String(), e.n) }

// --- gatherExpr ---

type gatherExpr struct {
	exprBase
	inner   Expr
	indices []int
}

func (e *gatherExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Take(e.indices), nil
}

func (e *gatherExpr) String() string {
	return fmt.Sprintf("%s.gather(%v)", e.inner.String(), e.indices)
}

// --- uniqueExpr ---

type uniqueExpr struct {
	exprBase
	inner Expr
}

func (e *uniqueExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Unique(), nil
}

func (e *uniqueExpr) String() string { return fmt.Sprintf("%s.unique()", e.inner.String()) }

// --- sortByExpr ---

type sortByExpr struct {
	exprBase
	inner      Expr
	by         Expr
	descending bool
}

func (e *sortByExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	byS, err := e.by.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	// Build a temporary DataFrame to sort
	df, err := dataframe.New(s, byS.Rename("__sort_by__"))
	if err != nil {
		return nil, fmt.Errorf("golars: sort_by: %w", err)
	}
	sorted, err := df.Sort("__sort_by__", e.descending)
	if err != nil {
		return nil, fmt.Errorf("golars: sort_by: %w", err)
	}
	result, _ := sorted.Column(s.Name())
	return result, nil
}

func (e *sortByExpr) String() string {
	return fmt.Sprintf("%s.sort_by(%s, desc=%v)", e.inner.String(), e.by.String(), e.descending)
}
