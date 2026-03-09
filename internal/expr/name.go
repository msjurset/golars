package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/series"
)

// NameNamespace provides operations on expression output names.
type NameNamespace struct {
	inner Expr
}

// Prefix prepends a string to the output column name.
func (n *NameNamespace) Prefix(prefix string) Expr {
	e := &nameTransformExpr{inner: n.inner, transform: func(name string) string { return prefix + name }}
	e.exprBase.self = e
	e.desc = fmt.Sprintf("name.prefix(%q)", prefix)
	return e
}

// Suffix appends a string to the output column name.
func (n *NameNamespace) Suffix(suffix string) Expr {
	e := &nameTransformExpr{inner: n.inner, transform: func(name string) string { return name + suffix }}
	e.exprBase.self = e
	e.desc = fmt.Sprintf("name.suffix(%q)", suffix)
	return e
}

// Map applies a function to transform the output column name.
func (n *NameNamespace) Map(fn func(string) string) Expr {
	e := &nameTransformExpr{inner: n.inner, transform: fn}
	e.exprBase.self = e
	e.desc = "name.map(<fn>)"
	return e
}

// nameTransformExpr transforms the output name of an expression.
type nameTransformExpr struct {
	exprBase
	inner     Expr
	transform func(string) string
	desc      string
}

func (e *nameTransformExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	newName := e.transform(s.Name())
	return s.Rename(newName), nil
}

func (e *nameTransformExpr) String() string {
	return fmt.Sprintf("%s.%s", e.inner.String(), e.desc)
}
