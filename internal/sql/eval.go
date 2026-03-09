package sql

import (
	"fmt"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/expr"
	"github.com/msjurseth/golars/internal/series"
)

// evalSQLExpr converts a SQL expression into a boolean Series mask.
func evalSQLExpr(sqlExpr SQLExpr, df *dataframe.DataFrame) (*series.Series, error) {
	e, err := sqlToExpr(sqlExpr)
	if err != nil {
		return nil, err
	}

	ctx := &expr.Context{DF: df}
	return e.Evaluate(ctx)
}

// sqlToExpr converts a SQLExpr into a golars Expr.
func sqlToExpr(s SQLExpr) (expr.Expr, error) {
	switch e := s.(type) {
	case ColumnRef:
		return expr.Col(e.Name), nil
	case LiteralInt:
		return expr.Lit(e.Value), nil
	case LiteralFloat:
		return expr.Lit(e.Value), nil
	case LiteralString:
		return expr.Lit(e.Value), nil
	case BinaryOp:
		left, err := sqlToExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := sqlToExpr(e.Right)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case "=":
			return left.Eq(right), nil
		case "!=":
			return left.Neq(right), nil
		case "<":
			return left.Lt(right), nil
		case ">":
			return left.Gt(right), nil
		case "<=":
			return left.Lte(right), nil
		case ">=":
			return left.Gte(right), nil
		case "AND":
			return left.And(right), nil
		case "OR":
			return left.Or(right), nil
		default:
			return nil, fmt.Errorf("golars: sql: unsupported operator %q", e.Op)
		}
	default:
		return nil, fmt.Errorf("golars: sql: unsupported expression type %T", s)
	}
}
