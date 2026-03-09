package expr

import (
	"fmt"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// sortExpr sorts or argSorts a series.
type sortExpr struct {
	exprBase
	inner      Expr
	descending bool
	argSort    bool
}

func (e *sortExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if e.argSort {
		indices := s.ArgSort(e.descending)
		data := make([]int64, len(indices))
		for i, idx := range indices {
			data[i] = int64(idx)
		}
		return series.NewInt64(s.Name(), data), nil
	}
	return s.Sort(e.descending), nil
}

func (e *sortExpr) String() string {
	if e.argSort {
		return fmt.Sprintf("%s.arg_sort(desc=%v)", e.inner.String(), e.descending)
	}
	return fmt.Sprintf("%s.sort(desc=%v)", e.inner.String(), e.descending)
}

// Ensure array and dtype are used.
var _ = array.NewInt64Array
var _ = dtype.Int64
