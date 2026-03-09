package golars

import (
	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
	"github.com/msjurset/golars/internal/series"
)

// GroupByExpr is an interface for expression-based GroupBy aggregation.
type GroupByExpr = dataframe.GroupByExpr

// exprAdapter wraps an expr.Expr to satisfy dataframe.GroupByExpr.
type exprAdapter struct {
	e expr.Expr
}

func (a *exprAdapter) EvaluateGroupBy(df *dataframe.DataFrame) (*series.Series, error) {
	return a.e.Evaluate(&expr.Context{DF: df})
}

// GroupByAgg performs expression-based groupby aggregation.
func GroupByAgg(g *GroupByResult, exprs ...Expr) (*DataFrame, error) {
	adapters := make([]dataframe.GroupByExpr, len(exprs))
	for i, e := range exprs {
		adapters[i] = &exprAdapter{e: e}
	}
	return g.AggExprs(adapters...)
}
