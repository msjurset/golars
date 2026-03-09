// Package sql provides a SQL query interface for golars DataFrames.
// It supports SELECT, FROM, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT,
// and JOIN operations via a recursive-descent parser.
package sql

import (
	"fmt"

	"github.com/msjurset/golars/internal/dataframe"
)

// Context holds registered DataFrames that can be queried via SQL.
type Context struct {
	tables map[string]*dataframe.DataFrame
}

// NewContext creates a new SQL context.
func NewContext() *Context {
	return &Context{
		tables: make(map[string]*dataframe.DataFrame),
	}
}

// Register adds a DataFrame with the given name so it can be queried.
func (c *Context) Register(name string, df *dataframe.DataFrame) {
	c.tables[name] = df
}

// Execute parses and executes a SQL query against registered DataFrames.
func (c *Context) Execute(query string) (*dataframe.DataFrame, error) {
	stmt, err := Parse(query)
	if err != nil {
		return nil, fmt.Errorf("golars: sql: %w", err)
	}

	return c.executeStmt(stmt)
}

func (c *Context) executeStmt(stmt *SelectStmt) (*dataframe.DataFrame, error) {
	// Resolve FROM
	df, ok := c.tables[stmt.From]
	if !ok {
		return nil, fmt.Errorf("golars: sql: table %q not found", stmt.From)
	}

	// JOIN
	for _, j := range stmt.Joins {
		rightDF, ok := c.tables[j.Table]
		if !ok {
			return nil, fmt.Errorf("golars: sql: join table %q not found", j.Table)
		}
		var jt dataframe.JoinType
		switch j.Type {
		case "INNER":
			jt = dataframe.InnerJoin
		case "LEFT":
			jt = dataframe.LeftJoin
		case "RIGHT":
			jt = dataframe.RightJoin
		case "FULL":
			jt = dataframe.FullJoin
		case "CROSS":
			jt = dataframe.CrossJoin
		default:
			jt = dataframe.InnerJoin
		}
		var err error
		df, err = df.Join(rightDF, []string{j.On}, jt)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: join: %w", err)
		}
	}

	// WHERE
	if stmt.Where != nil {
		mask, err := evalSQLExpr(stmt.Where, df)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: where: %w", err)
		}
		df, err = df.Filter(mask)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: filter: %w", err)
		}
	}

	// GROUP BY
	if len(stmt.GroupBy) > 0 {
		grouped, err := df.GroupBy(stmt.GroupBy...)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: group by: %w", err)
		}

		aggs, err := buildAggs(stmt.Columns)
		if err != nil {
			return nil, err
		}

		df, err = grouped.Agg(aggs)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: agg: %w", err)
		}
	}

	// SELECT (projection)
	if !stmt.SelectAll {
		colNames := make([]string, 0, len(stmt.Columns))
		for _, col := range stmt.Columns {
			if col.Alias != "" {
				colNames = append(colNames, col.Alias)
			} else {
				colNames = append(colNames, col.Name)
			}
		}
		// Only select if not doing GROUP BY (which already built the right columns)
		if len(stmt.GroupBy) == 0 {
			var err error
			df, err = df.Select(colNames...)
			if err != nil {
				return nil, fmt.Errorf("golars: sql: select: %w", err)
			}
		}
	}

	// ORDER BY
	if len(stmt.OrderBy) > 0 {
		cols := make([]string, len(stmt.OrderBy))
		descs := make([]bool, len(stmt.OrderBy))
		for i, ob := range stmt.OrderBy {
			cols[i] = ob.Column
			descs[i] = ob.Desc
		}
		var err error
		df, err = df.SortBy(cols, descs)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: order by: %w", err)
		}
	}

	// LIMIT
	if stmt.Limit > 0 {
		df = df.Head(stmt.Limit)
	}

	return df, nil
}

func buildAggs(columns []SelectColumn) (map[string]dataframe.AggFunc, error) {
	aggs := make(map[string]dataframe.AggFunc)
	for _, col := range columns {
		if col.AggFunc == "" {
			continue // group key, not an aggregation
		}
		var fn dataframe.AggFunc
		switch col.AggFunc {
		case "SUM":
			fn = dataframe.AggSum
		case "AVG", "MEAN":
			fn = dataframe.AggMean
		case "MIN":
			fn = dataframe.AggMin
		case "MAX":
			fn = dataframe.AggMax
		case "COUNT":
			fn = dataframe.AggCount
		case "FIRST":
			fn = dataframe.AggFirst
		case "LAST":
			fn = dataframe.AggLast
		default:
			return nil, fmt.Errorf("golars: sql: unknown aggregate function %q", col.AggFunc)
		}
		aggs[col.Name] = fn
	}
	return aggs, nil
}
