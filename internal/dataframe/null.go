package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// DropNulls returns a new DataFrame with all rows removed where any column
// contains a null value. If no columns have nulls, the original DataFrame is
// returned.
func (df *DataFrame) DropNulls() *DataFrame {
	if df.height == 0 || len(df.columns) == 0 {
		return df
	}

	// Build a combined validity mask: a row is kept only when every column is valid.
	mask := bitmap.New(df.height)
	for _, c := range df.columns {
		for i := 0; i < df.height; i++ {
			if c.IsNull(i) {
				mask.Clear(i)
			}
		}
	}

	if mask.PopCount() == df.height {
		return df
	}
	return df.FilterMask(mask)
}

// FillNull returns a new DataFrame with null values in the specified columns
// replaced by the corresponding values in the map. The map keys are column
// names and the values must match the column type (int64 for Int64, float64
// for Float64, string for String, bool for Boolean). Returns an error if a
// column name is not found or the value type does not match.
func (df *DataFrame) FillNull(values map[string]any) (*DataFrame, error) {
	cols := make([]*series.Series, len(df.columns))
	copy(cols, df.columns)

	for name, fillVal := range values {
		idx := df.columnIndex(name)
		if idx < 0 {
			return nil, fmt.Errorf("golars: column %q not found", name)
		}
		c := cols[idx]
		if !c.HasNulls() {
			continue
		}
		var filled *series.Series
		switch c.DataType() {
		case dtype.Int64:
			v, ok := fillVal.(int64)
			if !ok {
				return nil, fmt.Errorf("golars: fill value for column %q must be int64", name)
			}
			filled = c.FillNullInt64(v)
		case dtype.Float64:
			v, ok := fillVal.(float64)
			if !ok {
				return nil, fmt.Errorf("golars: fill value for column %q must be float64", name)
			}
			filled = c.FillNullFloat64(v)
		case dtype.String:
			v, ok := fillVal.(string)
			if !ok {
				return nil, fmt.Errorf("golars: fill value for column %q must be string", name)
			}
			filled = c.FillNullString(v)
		default:
			return nil, fmt.Errorf("golars: fill null not supported for column %q of type %s", name, c.DataType())
		}
		cols[idx] = filled
	}
	return New(cols...)
}
