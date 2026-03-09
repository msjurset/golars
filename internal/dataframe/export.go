package dataframe

import (
	"fmt"
	"iter"

	"github.com/msjurset/golars/internal/dtype"
)

// RowAccessor provides typed access to a single row of a DataFrame.
// It is designed for use with the Rows iterator.
type RowAccessor struct {
	df  *DataFrame
	idx int
}

// Index returns the row index within the DataFrame.
func (r RowAccessor) Index() int { return r.idx }

// Int64 returns the int64 value for the named column. Returns an error if the
// column does not exist or is null.
func (r RowAccessor) Int64(name string) (int64, error) {
	col, err := r.df.Column(name)
	if err != nil {
		return 0, err
	}
	v, ok := col.GetInt64(r.idx)
	if !ok {
		return 0, fmt.Errorf("golars: row %d column %q is null", r.idx, name)
	}
	return v, nil
}

// Float64 returns the float64 value for the named column.
func (r RowAccessor) Float64(name string) (float64, error) {
	col, err := r.df.Column(name)
	if err != nil {
		return 0, err
	}
	v, ok := col.GetFloat64(r.idx)
	if !ok {
		return 0, fmt.Errorf("golars: row %d column %q is null", r.idx, name)
	}
	return v, nil
}

// String returns the string value for the named column.
func (r RowAccessor) String(name string) (string, error) {
	col, err := r.df.Column(name)
	if err != nil {
		return "", err
	}
	v, ok := col.GetString(r.idx)
	if !ok {
		return "", fmt.Errorf("golars: row %d column %q is null", r.idx, name)
	}
	return v, nil
}

// Bool returns the boolean value for the named column.
func (r RowAccessor) Bool(name string) (bool, error) {
	col, err := r.df.Column(name)
	if err != nil {
		return false, err
	}
	v, ok := col.GetBool(r.idx)
	if !ok {
		return false, fmt.Errorf("golars: row %d column %q is null", r.idx, name)
	}
	return v, nil
}

// IsNull returns true if the named column is null at this row.
func (r RowAccessor) IsNull(name string) bool {
	col, err := r.df.Column(name)
	if err != nil {
		return true
	}
	return col.IsNull(r.idx)
}

// Get returns the value for the named column as any, or nil if null.
func (r RowAccessor) Get(name string) any {
	col, err := r.df.Column(name)
	if err != nil {
		return nil
	}
	if col.IsNull(r.idx) {
		return nil
	}
	switch col.DataType() {
	case dtype.Int64:
		v, _ := col.GetInt64(r.idx)
		return v
	case dtype.Float64:
		v, _ := col.GetFloat64(r.idx)
		return v
	case dtype.String:
		v, _ := col.GetString(r.idx)
		return v
	case dtype.Boolean:
		v, _ := col.GetBool(r.idx)
		return v
	default:
		return formatValue(col, r.idx)
	}
}

// Rows returns an iterator over the rows of the DataFrame. Each iteration
// yields a RowAccessor that provides typed access to the row's values.
//
// Usage:
//
//	for row := range df.Rows() {
//	    name, _ := row.String("name")
//	    age, _ := row.Int64("age")
//	}
func (df *DataFrame) Rows() iter.Seq[RowAccessor] {
	return func(yield func(RowAccessor) bool) {
		for i := 0; i < df.height; i++ {
			if !yield(RowAccessor{df: df, idx: i}) {
				return
			}
		}
	}
}

// ToMap returns a map where each key is a column name and the value is a typed
// slice of that column's data. Supported types produce []int64, []float64,
// []string, or []bool slices; unsupported types produce []string with formatted
// values.
func (df *DataFrame) ToMap() map[string]any {
	result := make(map[string]any, len(df.columns))
	for _, c := range df.columns {
		switch c.DataType() {
		case dtype.Int64:
			vals := make([]int64, df.height)
			for i := 0; i < df.height; i++ {
				if v, ok := c.GetInt64(i); ok {
					vals[i] = v
				}
			}
			result[c.Name()] = vals
		case dtype.Float64:
			vals := make([]float64, df.height)
			for i := 0; i < df.height; i++ {
				if v, ok := c.GetFloat64(i); ok {
					vals[i] = v
				}
			}
			result[c.Name()] = vals
		case dtype.String:
			vals := make([]string, df.height)
			for i := 0; i < df.height; i++ {
				if v, ok := c.GetString(i); ok {
					vals[i] = v
				}
			}
			result[c.Name()] = vals
		case dtype.Boolean:
			vals := make([]bool, df.height)
			for i := 0; i < df.height; i++ {
				if v, ok := c.GetBool(i); ok {
					vals[i] = v
				}
			}
			result[c.Name()] = vals
		default:
			vals := make([]string, df.height)
			for i := 0; i < df.height; i++ {
				vals[i] = formatValue(c, i)
			}
			result[c.Name()] = vals
		}
	}
	return result
}

// ToMaps returns a slice of maps, one per row. Each map key is a column name
// and the value is the typed cell value (int64, float64, string, bool, or nil
// for null).
func (df *DataFrame) ToMaps() []map[string]any {
	rows := make([]map[string]any, df.height)
	for i := 0; i < df.height; i++ {
		rows[i] = df.Row(i)
	}
	return rows
}

// Row returns a single row as a map from column name to typed value. Null
// values are represented as nil.
func (df *DataFrame) Row(i int) map[string]any {
	row := make(map[string]any, len(df.columns))
	for _, c := range df.columns {
		if c.IsNull(i) {
			row[c.Name()] = nil
			continue
		}
		switch c.DataType() {
		case dtype.Int64:
			v, _ := c.GetInt64(i)
			row[c.Name()] = v
		case dtype.Float64:
			v, _ := c.GetFloat64(i)
			row[c.Name()] = v
		case dtype.String:
			v, _ := c.GetString(i)
			row[c.Name()] = v
		case dtype.Boolean:
			v, _ := c.GetBool(i)
			row[c.Name()] = v
		default:
			row[c.Name()] = formatValue(c, i)
		}
	}
	return row
}
