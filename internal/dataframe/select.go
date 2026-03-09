package dataframe

import (
	"fmt"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// Select returns a new DataFrame containing only the named columns, in the
// order specified. Returns an error if any name is not found.
func (df *DataFrame) Select(names ...string) (*DataFrame, error) {
	cols := make([]*series.Series, len(names))
	for i, name := range names {
		idx := df.columnIndex(name)
		if idx < 0 {
			return nil, fmt.Errorf("golars: column %q not found", name)
		}
		cols[i] = df.columns[idx]
	}
	return New(cols...)
}

// Drop returns a new DataFrame with the named columns removed.
// Returns an error if any name is not found.
func (df *DataFrame) Drop(names ...string) (*DataFrame, error) {
	drop := make(map[string]struct{}, len(names))
	for _, name := range names {
		if !df.schema.Contains(name) {
			return nil, fmt.Errorf("golars: column %q not found", name)
		}
		drop[name] = struct{}{}
	}
	cols := make([]*series.Series, 0, len(df.columns)-len(drop))
	for _, c := range df.columns {
		if _, ok := drop[c.Name()]; !ok {
			cols = append(cols, c)
		}
	}
	return New(cols...)
}

// Rename returns a new DataFrame with the specified column renamed.
// Returns an error if oldName is not found or newName already exists.
func (df *DataFrame) Rename(oldName, newName string) (*DataFrame, error) {
	idx := df.columnIndex(oldName)
	if idx < 0 {
		return nil, fmt.Errorf("golars: column %q not found", oldName)
	}
	if oldName != newName && df.schema.Contains(newName) {
		return nil, fmt.Errorf("golars: column %q already exists", newName)
	}
	cols := make([]*series.Series, len(df.columns))
	copy(cols, df.columns)
	cols[idx] = cols[idx].Rename(newName)

	fields := make([]dtype.Field, len(cols))
	for i, c := range cols {
		fields[i] = dtype.Field{Name: c.Name(), Dtype: c.DataType()}
	}
	return &DataFrame{
		columns: cols,
		schema:  dtype.NewSchema(fields),
		height:  df.height,
	}, nil
}

// WithColumn returns a new DataFrame with the column added or replaced. If a
// column with the same name already exists it is replaced in place; otherwise
// the new column is appended. Returns an error if the column length does not
// match the DataFrame height.
func (df *DataFrame) WithColumn(col *series.Series) (*DataFrame, error) {
	if col == nil {
		return nil, fmt.Errorf("golars: column is nil")
	}
	if df.height > 0 && col.Len() != df.height {
		return nil, fmt.Errorf("golars: column %q has length %d, expected %d", col.Name(), col.Len(), df.height)
	}

	cols := make([]*series.Series, len(df.columns))
	copy(cols, df.columns)

	idx := df.columnIndex(col.Name())
	if idx >= 0 {
		cols[idx] = col
	} else {
		cols = append(cols, col)
	}
	return New(cols...)
}

// WithColumns returns a new DataFrame with the columns added or replaced.
// Each column is processed in order using the same logic as WithColumn.
func (df *DataFrame) WithColumns(cols ...*series.Series) (*DataFrame, error) {
	result := df
	for _, col := range cols {
		var err error
		result, err = result.WithColumn(col)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
