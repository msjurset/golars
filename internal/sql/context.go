// Package sql provides a SQL query interface for golars DataFrames.
// It supports SELECT, FROM, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT,
// and JOIN operations via a recursive-descent parser.
package sql

import (
	"fmt"
	"strings"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/series"
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

func prefixColumns(df *dataframe.DataFrame, prefix string) *dataframe.DataFrame {
	if prefix == "" {
		return df
	}
	cols := make([]*series.Series, df.Schema().Len())
	for i := 0; i < df.Schema().Len(); i++ {
		f := df.Schema().Field(i)
		col, _ := df.Column(f.Name)
		cols[i] = col.Rename(prefix + "." + f.Name)
	}
	res, _ := dataframe.New(cols...)
	return res
}

func (c *Context) executeStmt(stmt *SelectStmt) (*dataframe.DataFrame, error) {
	// Resolve FROM
	df, ok := c.tables[stmt.From]
	if !ok {
		return nil, fmt.Errorf("golars: sql: table %q not found", stmt.From)
	}
	
	fromAlias := stmt.FromAlias
	if fromAlias == "" {
		fromAlias = stmt.From
	}
	df = prefixColumns(df, fromAlias)

	// Map of alias -> table name for resolving bare columns
	aliases := make(map[string]string)
	aliases[fromAlias] = stmt.From

	// JOIN
	for _, j := range stmt.Joins {
		rightDF, ok := c.tables[j.Table]
		if !ok {
			return nil, fmt.Errorf("golars: sql: join table %q not found", j.Table)
		}
		
		jAlias := j.Alias
		if jAlias == "" {
			jAlias = j.Table
		}
		aliases[jAlias] = j.Table
		rightDF = prefixColumns(rightDF, jAlias)

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
		
		leftOn := j.LeftOn
		rightOn := j.RightOn
		// if leftOn does not have a dot, try to prepend fromAlias
		if leftOn != "" && !strings.Contains(leftOn, ".") {
			leftOn = fromAlias + "." + leftOn
		}
		if rightOn != "" && !strings.Contains(rightOn, ".") {
			rightOn = jAlias + "." + rightOn
		}
		
		df, err = df.JoinOn(rightDF, []string{leftOn}, []string{rightOn}, jt)
		if err != nil {
			return nil, fmt.Errorf("golars: sql: join: %w", err)
		}
	}
	
	// Resolve bare columns
	resolveName := func(name string) string {
		if strings.Contains(name, ".") || name == "*" {
			return name
		}
		// find matching column in df
		for i := 0; i < df.Schema().Len(); i++ {
			f := df.Schema().Field(i)
			if strings.HasSuffix(f.Name, "."+name) {
				return f.Name
			}
		}
		return name
	}
	
	for i, col := range stmt.Columns {
		stmt.Columns[i].Name = resolveName(col.Name)
	}
	var resolveExpr func(e SQLExpr) SQLExpr
	resolveExpr = func(e SQLExpr) SQLExpr {
		switch v := e.(type) {
		case ColumnRef:
			return ColumnRef{Name: resolveName(v.Name)}
		case BinaryOp:
			return BinaryOp{Left: resolveExpr(v.Left), Op: v.Op, Right: resolveExpr(v.Right)}
		default:
			return e
		}
	}
	if stmt.Where != nil {
		stmt.Where = resolveExpr(stmt.Where)
	}
	for i, g := range stmt.GroupBy {
		stmt.GroupBy[i] = resolveName(g)
	}
	for i, o := range stmt.OrderBy {
		colName := o.Column
		for _, sc := range stmt.Columns {
			if sc.Alias == colName {
				colName = sc.Name
				break
			}
		}
		stmt.OrderBy[i].Column = resolveName(colName)
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

	// SELECT (projection)
	if stmt.SelectAll {
		for i := 0; i < df.Schema().Len(); i++ {
			f := df.Schema().Field(i)
			stmt.Columns = append(stmt.Columns, SelectColumn{Name: f.Name})
		}
		stmt.SelectAll = false
	}

	if !stmt.SelectAll {
		colNames := make([]string, 0, len(stmt.Columns))
		for _, col := range stmt.Columns {
			colNames = append(colNames, col.Name)
		}
		if len(stmt.GroupBy) == 0 {
			var err error
			df, err = df.Select(colNames...)
			if err != nil {
				return nil, fmt.Errorf("golars: sql: select: %w", err)
			}
		}
		// Rename columns to strip table prefix or apply alias
		finalCols := make([]*series.Series, df.Schema().Len())
		for i := 0; i < df.Schema().Len(); i++ {
			f := df.Schema().Field(i)
			c, _ := df.Column(f.Name)
			newName := f.Name
			
			// Find corresponding select column
			for _, sc := range stmt.Columns {
				if sc.Name == f.Name {
					if sc.Alias != "" {
						newName = sc.Alias
					} else {
						parts := strings.Split(f.Name, ".")
						if len(parts) > 1 {
							newName = parts[1]
						}
					}
					break
				}
			}
			
			// Handle collisions? Just rename. If collision occurs, golars DataFrame New will error out.
			// Let's ensure uniqueness to avoid golars panic.
			for j := 0; j < i; j++ {
				if finalCols[j].Name() == newName {
					newName = newName + "_" + strings.Split(f.Name, ".")[0]
				}
			}
			finalCols[i] = c.Rename(newName)
		}
		df, _ = dataframe.New(finalCols...)
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
