// Package dataframe provides the DataFrame type, an immutable collection of
// named, typed columns (Series) that supports relational-style operations.
package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// DataFrame is an immutable collection of named, typed columns (Series).
// All read operations are safe for concurrent use by multiple goroutines.
type DataFrame struct {
	columns []*series.Series
	schema  *dtype.Schema
	height  int
}

// New creates a new DataFrame from the given columns. All columns must have the
// same length and unique names. Returns an error if validation fails.
func New(columns ...*series.Series) (*DataFrame, error) {
	if len(columns) == 0 {
		return &DataFrame{
			columns: nil,
			schema:  dtype.NewSchema(nil),
			height:  0,
		}, nil
	}

	height := columns[0].Len()
	seen := make(map[string]struct{}, len(columns))
	fields := make([]dtype.Field, len(columns))

	for i, col := range columns {
		if col == nil {
			return nil, fmt.Errorf("golars: column at index %d is nil", i)
		}
		name := col.Name()
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("golars: duplicate column name %q", name)
		}
		seen[name] = struct{}{}
		if col.Len() != height {
			return nil, fmt.Errorf("golars: column %q has length %d, expected %d", name, col.Len(), height)
		}
		fields[i] = dtype.Field{Name: name, Dtype: col.DataType()}
	}

	cols := make([]*series.Series, len(columns))
	copy(cols, columns)

	return &DataFrame{
		columns: cols,
		schema:  dtype.NewSchema(fields),
		height:  height,
	}, nil
}

// FromSchema creates an empty DataFrame with the given schema and height.
// Each column will be a zero-valued series of the specified type and length.
func FromSchema(schema *dtype.Schema, height int) *DataFrame {
	cols := make([]*series.Series, schema.Len())
	for i := 0; i < schema.Len(); i++ {
		f := schema.Field(i)
		cols[i] = emptySeriesForType(f.Name, f.Dtype, height)
	}
	return &DataFrame{
		columns: cols,
		schema:  schema,
		height:  height,
	}
}

// emptySeriesForType creates a zero-valued series with the given name, type,
// and length.
func emptySeriesForType(name string, dt dtype.DataType, n int) *series.Series {
	switch dt {
	case dtype.Int64:
		return series.NewInt64(name, make([]int64, n))
	case dtype.Int32:
		return series.NewInt32(name, make([]int32, n))
	case dtype.Int16:
		return series.NewInt16(name, make([]int16, n))
	case dtype.Int8:
		return series.NewInt8(name, make([]int8, n))
	case dtype.UInt64:
		return series.NewUInt64(name, make([]uint64, n))
	case dtype.UInt32:
		return series.NewUInt32(name, make([]uint32, n))
	case dtype.UInt16:
		return series.NewUInt16(name, make([]uint16, n))
	case dtype.UInt8:
		return series.NewUInt8(name, make([]uint8, n))
	case dtype.Float64:
		return series.NewFloat64(name, make([]float64, n))
	case dtype.Float32:
		return series.NewFloat32(name, make([]float32, n))
	case dtype.Boolean:
		return series.NewBoolean(name, make([]bool, n))
	case dtype.String:
		return series.NewString(name, make([]string, n))
	default:
		return series.NewString(name, make([]string, n))
	}
}

// Height returns the number of rows in the DataFrame.
func (df *DataFrame) Height() int { return df.height }

// Width returns the number of columns in the DataFrame.
func (df *DataFrame) Width() int { return len(df.columns) }

// Shape returns the (height, width) dimensions of the DataFrame.
func (df *DataFrame) Shape() (int, int) { return df.height, len(df.columns) }

// Schema returns the schema describing column names and types.
func (df *DataFrame) Schema() *dtype.Schema { return df.schema }

// Columns returns a copy of the column slice.
func (df *DataFrame) Columns() []*series.Series {
	out := make([]*series.Series, len(df.columns))
	copy(out, df.columns)
	return out
}

// Column returns the column with the given name.
// Returns an error if no column with that name exists.
func (df *DataFrame) Column(name string) (*series.Series, error) {
	idx := df.schema.Index(name)
	if idx < 0 {
		return nil, fmt.Errorf("golars: column %q not found", name)
	}
	return df.columns[idx], nil
}

// ColumnByIndex returns the column at position i.
// Panics if i is out of range.
func (df *DataFrame) ColumnByIndex(i int) *series.Series {
	return df.columns[i]
}

// Head returns a new DataFrame with the first n rows.
// If n exceeds the height, the full DataFrame is returned.
func (df *DataFrame) Head(n int) *DataFrame {
	if n > df.height {
		n = df.height
	}
	if n < 0 {
		n = 0
	}
	return df.Slice(0, n)
}

// Tail returns a new DataFrame with the last n rows.
// If n exceeds the height, the full DataFrame is returned.
func (df *DataFrame) Tail(n int) *DataFrame {
	if n > df.height {
		n = df.height
	}
	if n < 0 {
		n = 0
	}
	return df.Slice(df.height-n, df.height)
}

// Slice returns a new DataFrame for the row range [start, end).
func (df *DataFrame) Slice(start, end int) *DataFrame {
	if start < 0 {
		start = 0
	}
	if end > df.height {
		end = df.height
	}
	if start > end {
		start = end
	}
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.Slice(start, end)
	}
	fields := make([]dtype.Field, len(df.columns))
	for i, c := range cols {
		fields[i] = dtype.Field{Name: c.Name(), Dtype: c.DataType()}
	}
	return &DataFrame{
		columns: cols,
		schema:  dtype.NewSchema(fields),
		height:  end - start,
	}
}

// Len returns the number of rows. It is an alias for Height.
func (df *DataFrame) Len() int { return df.height }

// IsEmpty returns true if the DataFrame has zero rows.
func (df *DataFrame) IsEmpty() bool { return df.height == 0 }

// Clone returns a shallow clone of the DataFrame. The underlying array data
// is shared; only the column and schema metadata are copied.
func (df *DataFrame) Clone() *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	copy(cols, df.columns)
	fields := make([]dtype.Field, df.schema.Len())
	for i := 0; i < df.schema.Len(); i++ {
		fields[i] = df.schema.Field(i)
	}
	return &DataFrame{
		columns: cols,
		schema:  dtype.NewSchema(fields),
		height:  df.height,
	}
}

// String returns a formatted ASCII table representation of the DataFrame.
func (df *DataFrame) String() string {
	return formatTable(df)
}

// columnIndex returns the position of the named column, or -1 if not found.
func (df *DataFrame) columnIndex(name string) int {
	return df.schema.Index(name)
}
