package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/series"
)

// rankExpr assigns ranks to values.
type rankExpr struct {
	exprBase
	inner  Expr
	method string
}

func (e *rankExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return s.Rank(e.method), nil
}

func (e *rankExpr) String() string {
	return fmt.Sprintf("%s.rank(%q)", e.inner.String(), e.method)
}
