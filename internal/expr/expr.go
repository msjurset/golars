// Package expr provides a composable expression DSL for evaluating
// column-level operations against a DataFrame context.
package expr

import (
	"fmt"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// Expr represents a composable expression that can be evaluated against a DataFrame.
type Expr interface {
	// Evaluate evaluates this expression against the given context.
	Evaluate(ctx *Context) (*series.Series, error)
	// Alias returns a new expression with the given output name.
	Alias(name string) Expr
	// String returns a human-readable representation of the expression.
	String() string

	// Arithmetic
	Add(other Expr) Expr
	Sub(other Expr) Expr
	Mul(other Expr) Expr
	Div(other Expr) Expr

	// Comparison
	Eq(other Expr) Expr
	Neq(other Expr) Expr
	Lt(other Expr) Expr
	Gt(other Expr) Expr
	Lte(other Expr) Expr
	Gte(other Expr) Expr

	// Logical
	And(other Expr) Expr
	Or(other Expr) Expr
	Not() Expr

	// Aggregation
	Sum() Expr
	Mean() Expr
	Min() Expr
	Max() Expr
	Count() Expr
	Std() Expr

	// Null handling
	IsNull() Expr
	IsNotNull() Expr
	FillNull(value Expr) Expr

	// Sort
	Sort(descending bool) Expr
	ArgSort(descending bool) Expr

	// Cast
	Cast(dt dtype.DataType) Expr

	// Window
	Over(partitionBy ...string) Expr

	// String namespace
	Str() *StrNamespace
}

// Context holds the evaluation context for expressions.
type Context struct {
	DF *dataframe.DataFrame
}

// baseExpr provides default implementations for all Expr methods,
// embedding types can override Evaluate, Alias, and String.
type baseExpr struct{}

func (b baseExpr) Add(other Expr) Expr               { return nil }
func (b baseExpr) Sub(other Expr) Expr               { return nil }
func (b baseExpr) Mul(other Expr) Expr               { return nil }
func (b baseExpr) Div(other Expr) Expr               { return nil }
func (b baseExpr) Eq(other Expr) Expr                { return nil }
func (b baseExpr) Neq(other Expr) Expr               { return nil }
func (b baseExpr) Lt(other Expr) Expr                { return nil }
func (b baseExpr) Gt(other Expr) Expr                { return nil }
func (b baseExpr) Lte(other Expr) Expr               { return nil }
func (b baseExpr) Gte(other Expr) Expr               { return nil }
func (b baseExpr) And(other Expr) Expr               { return nil }
func (b baseExpr) Or(other Expr) Expr                { return nil }
func (b baseExpr) Not() Expr                         { return nil }
func (b baseExpr) Sum() Expr                         { return nil }
func (b baseExpr) Mean() Expr                        { return nil }
func (b baseExpr) Min() Expr                         { return nil }
func (b baseExpr) Max() Expr                         { return nil }
func (b baseExpr) Count() Expr                       { return nil }
func (b baseExpr) Std() Expr                         { return nil }
func (b baseExpr) IsNull() Expr                      { return nil }
func (b baseExpr) IsNotNull() Expr                   { return nil }
func (b baseExpr) FillNull(value Expr) Expr          { return nil }
func (b baseExpr) Sort(descending bool) Expr         { return nil }
func (b baseExpr) ArgSort(descending bool) Expr      { return nil }
func (b baseExpr) Cast(dt dtype.DataType) Expr       { return nil }
func (b baseExpr) Str() *StrNamespace                { return nil }
func (b baseExpr) Over(partitionBy ...string) Expr   { return nil }
func (b baseExpr) Alias(name string) Expr            { return nil }
func (b baseExpr) Evaluate(*Context) (*series.Series, error) {
	return nil, fmt.Errorf("golars: expression not implemented")
}
func (b baseExpr) String() string { return "<expr>" }

// exprBase provides the shared method implementations for all expression types.
// Every concrete expression embeds this via a self-referential pointer.
type exprBase struct {
	self Expr
}

func (e *exprBase) Add(other Expr) Expr      { r := &binaryExpr{left: e.self, right: other, op: opAdd}; r.exprBase.self = r; return r }
func (e *exprBase) Sub(other Expr) Expr      { r := &binaryExpr{left: e.self, right: other, op: opSub}; r.exprBase.self = r; return r }
func (e *exprBase) Mul(other Expr) Expr      { r := &binaryExpr{left: e.self, right: other, op: opMul}; r.exprBase.self = r; return r }
func (e *exprBase) Div(other Expr) Expr      { r := &binaryExpr{left: e.self, right: other, op: opDiv}; r.exprBase.self = r; return r }
func (e *exprBase) Eq(other Expr) Expr       { r := &comparisonExpr{left: e.self, right: other, op: cmpEq}; r.exprBase.self = r; return r }
func (e *exprBase) Neq(other Expr) Expr      { r := &comparisonExpr{left: e.self, right: other, op: cmpNeq}; r.exprBase.self = r; return r }
func (e *exprBase) Lt(other Expr) Expr       { r := &comparisonExpr{left: e.self, right: other, op: cmpLt}; r.exprBase.self = r; return r }
func (e *exprBase) Gt(other Expr) Expr       { r := &comparisonExpr{left: e.self, right: other, op: cmpGt}; r.exprBase.self = r; return r }
func (e *exprBase) Lte(other Expr) Expr      { r := &comparisonExpr{left: e.self, right: other, op: cmpLte}; r.exprBase.self = r; return r }
func (e *exprBase) Gte(other Expr) Expr      { r := &comparisonExpr{left: e.self, right: other, op: cmpGte}; r.exprBase.self = r; return r }
func (e *exprBase) And(other Expr) Expr      { r := &logicalExpr{left: e.self, right: other, op: logAnd}; r.exprBase.self = r; return r }
func (e *exprBase) Or(other Expr) Expr       { r := &logicalExpr{left: e.self, right: other, op: logOr}; r.exprBase.self = r; return r }
func (e *exprBase) Not() Expr                { r := &notExpr{inner: e.self}; r.exprBase.self = r; return r }
func (e *exprBase) Sum() Expr                { r := &aggExpr{inner: e.self, op: aggSum}; r.exprBase.self = r; return r }
func (e *exprBase) Mean() Expr               { r := &aggExpr{inner: e.self, op: aggMean}; r.exprBase.self = r; return r }
func (e *exprBase) Min() Expr                { r := &aggExpr{inner: e.self, op: aggMin}; r.exprBase.self = r; return r }
func (e *exprBase) Max() Expr                { r := &aggExpr{inner: e.self, op: aggMax}; r.exprBase.self = r; return r }
func (e *exprBase) Count() Expr              { r := &aggExpr{inner: e.self, op: aggCount}; r.exprBase.self = r; return r }
func (e *exprBase) Std() Expr                { r := &aggExpr{inner: e.self, op: aggStd}; r.exprBase.self = r; return r }
func (e *exprBase) IsNull() Expr             { r := &isNullExpr{inner: e.self, negate: false}; r.exprBase.self = r; return r }
func (e *exprBase) IsNotNull() Expr          { r := &isNullExpr{inner: e.self, negate: true}; r.exprBase.self = r; return r }
func (e *exprBase) FillNull(value Expr) Expr { r := &fillNullExpr{inner: e.self, fill: value}; r.exprBase.self = r; return r }
func (e *exprBase) Sort(descending bool) Expr    { r := &sortExpr{inner: e.self, descending: descending, argSort: false}; r.exprBase.self = r; return r }
func (e *exprBase) ArgSort(descending bool) Expr { r := &sortExpr{inner: e.self, descending: descending, argSort: true}; r.exprBase.self = r; return r }
func (e *exprBase) Cast(dt dtype.DataType) Expr  { r := &castExpr{inner: e.self, target: dt}; r.exprBase.self = r; return r }
func (e *exprBase) Str() *StrNamespace           { return &StrNamespace{inner: e.self} }
func (e *exprBase) Over(partitionBy ...string) Expr { r := &windowExpr{inner: e.self, partitionBy: partitionBy}; r.exprBase.self = r; return r }
func (e *exprBase) Alias(name string) Expr       { r := &aliasExpr{inner: e.self, name: name}; r.exprBase.self = r; return r }

// colExpr references a column by name.
type colExpr struct {
	exprBase
	name string
}

func (c *colExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := ctx.DF.Column(c.name)
	if err != nil {
		return nil, fmt.Errorf("golars: column %q not found", c.name)
	}
	return s, nil
}

func (c *colExpr) String() string { return fmt.Sprintf("col(%q)", c.name) }

// Col creates a column reference expression.
func Col(name string) Expr {
	e := &colExpr{name: name}
	e.exprBase.self = e
	return e
}

// litExpr holds a literal scalar value.
type litExpr struct {
	exprBase
	value any
}

func (l *litExpr) Evaluate(ctx *Context) (*series.Series, error) {
	n := 1
	if ctx != nil && ctx.DF != nil {
		n = ctx.DF.Height()
	}
	return broadcastLiteral(l.value, n)
}

func (l *litExpr) String() string { return fmt.Sprintf("lit(%v)", l.value) }

// Lit creates a literal value expression. Supports int, int64, float64, string, bool.
func Lit(value any) Expr {
	e := &litExpr{value: value}
	e.exprBase.self = e
	return e
}

// broadcastLiteral creates a series of length n filled with the given scalar.
func broadcastLiteral(value any, n int) (*series.Series, error) {
	switch v := value.(type) {
	case int:
		return broadcastInt64(int64(v), n), nil
	case int64:
		return broadcastInt64(v, n), nil
	case int32:
		return broadcastInt64(int64(v), n), nil
	case float64:
		return broadcastFloat64(v, n), nil
	case float32:
		return broadcastFloat64(float64(v), n), nil
	case string:
		data := make([]string, n)
		for i := range data {
			data[i] = v
		}
		return series.NewString("literal", data), nil
	case bool:
		data := make([]bool, n)
		for i := range data {
			data[i] = v
		}
		return series.NewBoolean("literal", data), nil
	default:
		return nil, fmt.Errorf("golars: unsupported literal type %T", value)
	}
}

func broadcastInt64(v int64, n int) *series.Series {
	data := make([]int64, n)
	for i := range data {
		data[i] = v
	}
	return series.NewInt64("literal", data)
}

func broadcastFloat64(v float64, n int) *series.Series {
	data := make([]float64, n)
	for i := range data {
		data[i] = v
	}
	return series.NewFloat64("literal", data)
}

// Cols creates multiple column reference expressions.
func Cols(names ...string) []Expr {
	exprs := make([]Expr, len(names))
	for i, name := range names {
		exprs[i] = Col(name)
	}
	return exprs
}

// allColsExpr represents a wildcard selecting all columns.
type allColsExpr struct {
	exprBase
}

func (a *allColsExpr) Evaluate(ctx *Context) (*series.Series, error) {
	return nil, fmt.Errorf("golars: AllCols() cannot be evaluated directly; use in Select context")
}

func (a *allColsExpr) String() string { return "all()" }

// AllCols returns an expression representing all columns.
func AllCols() Expr {
	e := &allColsExpr{}
	e.exprBase.self = e
	return e
}

// aliasExpr wraps an expression to rename its output.
type aliasExpr struct {
	exprBase
	inner Expr
	name  string
}

func (a *aliasExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := a.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Rename(a.name), nil
}

func (a *aliasExpr) String() string { return fmt.Sprintf("%s.alias(%q)", a.inner.String(), a.name) }

// Helper to convert int64 series to float64 for mixed-type operations.
func int64ToFloat64(s *series.Series) *series.Series {
	ta, ok := s.Array().(*array.TypedArray[int64])
	if !ok {
		return s
	}
	vals := ta.Values()
	fvals := make([]float64, len(vals))
	for i, v := range vals {
		fvals[i] = float64(v)
	}
	var result *series.Series
	if ta.Validity() != nil {
		result = series.New(s.Name(), array.NewTypedArray(fvals, dtype.Float64, ta.Validity().Clone()))
	} else {
		result = series.NewFloat64(s.Name(), fvals)
	}
	return result
}

// promoteToFloat64 ensures a series is Float64, converting Int64 if needed.
func promoteToFloat64(s *series.Series) (*series.Series, error) {
	switch s.DataType() {
	case dtype.Float64:
		return s, nil
	case dtype.Int64:
		return int64ToFloat64(s), nil
	default:
		return nil, fmt.Errorf("golars: cannot promote %s to Float64", s.DataType())
	}
}
