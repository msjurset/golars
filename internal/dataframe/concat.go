package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// Concat vertically concatenates the given DataFrames. All DataFrames must
// share the same schema (column names and types in the same order). Returns
// an error if schemas differ or no DataFrames are provided.
func Concat(dfs ...*DataFrame) (*DataFrame, error) {
	if len(dfs) == 0 {
		return nil, fmt.Errorf("golars: no DataFrames to concatenate")
	}
	if len(dfs) == 1 {
		return dfs[0].Clone(), nil
	}

	base := dfs[0].Schema()
	for i := 1; i < len(dfs); i++ {
		if !base.Equal(dfs[i].Schema()) {
			return nil, fmt.Errorf("golars: schema mismatch at DataFrame index %d", i)
		}
	}

	totalHeight := 0
	for _, df := range dfs {
		totalHeight += df.Height()
	}

	w := base.Len()
	cols := make([]*series.Series, w)

	for j := 0; j < w; j++ {
		f := base.Field(j)
		cols[j] = concatColumn(f.Name, f.Dtype, dfs, j, totalHeight)
	}

	return New(cols...)
}

// concatColumn concatenates a single column across all DataFrames.
func concatColumn(name string, dt dtype.DataType, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	switch dt {
	case dtype.Int64:
		return concatTypedColumn[int64](name, dfs, colIdx, totalHeight,
			func(s *series.Series, i int) int64 { v, _ := s.GetInt64(i); return v },
			series.NewInt64, series.NewInt64WithValidity)
	case dtype.Float64:
		return concatTypedColumn[float64](name, dfs, colIdx, totalHeight,
			func(s *series.Series, i int) float64 { v, _ := s.GetFloat64(i); return v },
			series.NewFloat64, series.NewFloat64WithValidity)
	case dtype.String:
		return concatStringColumn(name, dfs, colIdx, totalHeight)
	case dtype.Boolean:
		return concatBoolColumn(name, dfs, colIdx, totalHeight)
	default:
		// Fallback: treat as string.
		return concatStringColumn(name, dfs, colIdx, totalHeight)
	}
}

// concatTypedColumn concatenates a numeric column across DataFrames.
func concatTypedColumn[T any](
	name string,
	dfs []*DataFrame,
	colIdx int,
	totalHeight int,
	getter func(*series.Series, int) T,
	newFn func(string, []T) *series.Series,
	newWithValidityFn func(string, []T, []bool) *series.Series,
) *series.Series {
	data := make([]T, 0, totalHeight)
	hasNulls := false
	for _, df := range dfs {
		c := df.columns[colIdx]
		if c.HasNulls() {
			hasNulls = true
		}
		for i := 0; i < c.Len(); i++ {
			data = append(data, getter(c, i))
		}
	}
	if !hasNulls {
		return newFn(name, data)
	}
	valid := make([]bool, totalHeight)
	offset := 0
	for _, df := range dfs {
		c := df.columns[colIdx]
		for i := 0; i < c.Len(); i++ {
			valid[offset+i] = c.IsValid(i)
		}
		offset += c.Len()
	}
	return newWithValidityFn(name, data, valid)
}

// concatStringColumn concatenates string columns across DataFrames.
func concatStringColumn(name string, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	data := make([]string, 0, totalHeight)
	hasNulls := false
	for _, df := range dfs {
		c := df.columns[colIdx]
		if c.HasNulls() {
			hasNulls = true
		}
		for i := 0; i < c.Len(); i++ {
			if c.IsNull(i) {
				data = append(data, "")
			} else {
				v, _ := c.GetString(i)
				data = append(data, v)
			}
		}
	}
	if !hasNulls {
		return series.NewString(name, data)
	}
	valid := make([]bool, totalHeight)
	offset := 0
	for _, df := range dfs {
		c := df.columns[colIdx]
		for i := 0; i < c.Len(); i++ {
			valid[offset+i] = c.IsValid(i)
		}
		offset += c.Len()
	}
	return series.NewStringWithValidity(name, data, valid)
}

// concatBoolColumn concatenates boolean columns across DataFrames.
func concatBoolColumn(name string, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	data := make([]bool, 0, totalHeight)
	hasNulls := false
	for _, df := range dfs {
		c := df.columns[colIdx]
		if c.HasNulls() {
			hasNulls = true
		}
		ba := c.BooleanArray()
		for i := 0; i < c.Len(); i++ {
			if ba != nil && c.IsValid(i) {
				data = append(data, ba.Value(i))
			} else {
				data = append(data, false)
			}
		}
	}
	if !hasNulls {
		return series.NewBoolean(name, data)
	}
	valid := make([]bool, totalHeight)
	offset := 0
	for _, df := range dfs {
		c := df.columns[colIdx]
		for i := 0; i < c.Len(); i++ {
			valid[offset+i] = c.IsValid(i)
		}
		offset += c.Len()
	}
	return series.NewBooleanWithValidity(name, data, valid)
}

// ConcatHorizontal concatenates DataFrames side by side. All DataFrames must
// have the same height and no duplicate column names across them. Returns an
// error if heights differ or column names collide.
func ConcatHorizontal(dfs ...*DataFrame) (*DataFrame, error) {
	if len(dfs) == 0 {
		return nil, fmt.Errorf("golars: no DataFrames to concatenate")
	}
	if len(dfs) == 1 {
		return dfs[0].Clone(), nil
	}

	height := dfs[0].Height()
	seen := make(map[string]struct{})
	var allCols []*series.Series

	for i, df := range dfs {
		if df.Height() != height {
			return nil, fmt.Errorf("golars: height mismatch at DataFrame index %d: got %d, expected %d", i, df.Height(), height)
		}
		for _, c := range df.columns {
			if _, exists := seen[c.Name()]; exists {
				return nil, fmt.Errorf("golars: duplicate column name %q during horizontal concat", c.Name())
			}
			seen[c.Name()] = struct{}{}
			allCols = append(allCols, c)
		}
	}

	return New(allCols...)
}
