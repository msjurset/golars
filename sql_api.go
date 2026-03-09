package golars

import "github.com/msjurset/golars/internal/sql"

// SQLContext holds registered DataFrames that can be queried via SQL.
type SQLContext = sql.Context

// NewSQLContext creates a new SQL context for querying DataFrames with SQL.
func NewSQLContext() *SQLContext {
	return sql.NewContext()
}
