package golars

import (
	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/lazy"
	"github.com/msjurset/golars/internal/series"
)

// DataType represents the logical data type of a column.
type DataType = dtype.DataType

// Type constants for all supported data types.
const (
	Null     = dtype.Null
	Boolean  = dtype.Boolean
	Int8     = dtype.Int8
	Int16    = dtype.Int16
	Int32    = dtype.Int32
	Int64    = dtype.Int64
	UInt8    = dtype.UInt8
	UInt16   = dtype.UInt16
	UInt32   = dtype.UInt32
	UInt64   = dtype.UInt64
	Float32  = dtype.Float32
	Float64  = dtype.Float64
	Decimal  = dtype.Decimal
	String   = dtype.String
	Binary   = dtype.Binary
	Date     = dtype.Date
	DateTime = dtype.DateTime
	Time     = dtype.Time
	Duration = dtype.Duration

	// Sort direction constants for Sort operations.
	Ascending  SortDirection = false
	Descending SortDirection = true
)

// SortDirection specifies ascending or descending sort order.
type SortDirection bool

// Series is a named, typed column of data.
type Series = series.Series

// Schema represents an ordered collection of named, typed fields.
type Schema = dtype.Schema

// Field represents a named column with its data type.
type Field = dtype.Field

// TimeUnit represents the resolution of temporal types.
type TimeUnit = dtype.TimeUnit

// TimeUnit constants.
const (
	Nanoseconds  = dtype.Nanoseconds
	Microseconds = dtype.Microseconds
	Milliseconds = dtype.Milliseconds
)

// NewSchema creates a new Schema from the given fields.
func NewSchema(fields ...Field) *Schema {
	return dtype.NewSchema(fields)
}

// DataFrame is an immutable collection of named, typed columns (Series).
type DataFrame = dataframe.DataFrame

// NewDataFrame creates a new DataFrame from the given Series columns.
// All columns must have the same length and unique names.
func NewDataFrame(columns ...*Series) (*DataFrame, error) {
	return dataframe.New(columns...)
}

// DataFrameFromSchema creates an empty DataFrame with the given schema and row count.
func DataFrameFromSchema(schema *Schema, height int) *DataFrame {
	return dataframe.FromSchema(schema, height)
}

// ConcatDataFrames vertically concatenates DataFrames that share the same schema.
func ConcatDataFrames(dfs ...*DataFrame) (*DataFrame, error) {
	return dataframe.Concat(dfs...)
}

// ConcatDataFramesHorizontal concatenates DataFrames side by side.
func ConcatDataFramesHorizontal(dfs ...*DataFrame) (*DataFrame, error) {
	return dataframe.ConcatHorizontal(dfs...)
}

// GroupByResult holds the result of a GroupBy operation.
type GroupByResult = dataframe.GroupByResult

// AggFunc represents an aggregation function for use with GroupBy.
type AggFunc = dataframe.AggFunc

// Aggregation function constants.
const (
	AggSum   = dataframe.AggSum
	AggMean  = dataframe.AggMean
	AggMin   = dataframe.AggMin
	AggMax   = dataframe.AggMax
	AggCount = dataframe.AggCount
	AggFirst = dataframe.AggFirst
	AggLast  = dataframe.AggLast
)

// JoinType represents the type of join operation.
type JoinType = dataframe.JoinType

// Join type constants.
const (
	InnerJoin = dataframe.InnerJoin
	LeftJoin  = dataframe.LeftJoin
	RightJoin = dataframe.RightJoin
	FullJoin  = dataframe.FullJoin
	SemiJoin  = dataframe.SemiJoin
	AntiJoin  = dataframe.AntiJoin
	CrossJoin = dataframe.CrossJoin
)

// RowAccessor provides typed access to a single row of a DataFrame.
// Use with the DataFrame.Rows() iterator.
type RowAccessor = dataframe.RowAccessor

// LazyFrame represents a lazy computation over a DataFrame. Operations are
// recorded and only executed when Collect is called.
type LazyFrame = lazy.LazyFrame

// LazyGroupBy holds a deferred groupby operation on a LazyFrame.
type LazyGroupBy = lazy.LazyGroupBy

// Lazy creates a LazyFrame from an eager DataFrame.
func Lazy(df *DataFrame) *LazyFrame {
	return lazy.FromDataFrame(df)
}
