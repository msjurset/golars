package dataframe

import (
	"fmt"
	"sort"

	"github.com/msjurseth/golars/internal/series"
)

// Sort returns a new DataFrame sorted by the named column. If descending is
// true the sort order is reversed. Returns an error if the column is not found.
func (df *DataFrame) Sort(colName string, descending bool) (*DataFrame, error) {
	return df.SortBy([]string{colName}, []bool{descending})
}

// SortBy returns a new DataFrame sorted by multiple columns in priority order.
// The first column name has the highest sort priority. The descending slice
// must be the same length as colNames. Returns an error if any column is not
// found or the slice lengths differ.
func (df *DataFrame) SortBy(colNames []string, descending []bool) (*DataFrame, error) {
	if len(colNames) != len(descending) {
		return nil, fmt.Errorf("golars: colNames length %d does not match descending length %d", len(colNames), len(descending))
	}
	if len(colNames) == 0 {
		return df.Clone(), nil
	}

	// Validate columns exist.
	sortCols := make([]*series.Series, len(colNames))
	for i, name := range colNames {
		idx := df.columnIndex(name)
		if idx < 0 {
			return nil, fmt.Errorf("golars: column %q not found", name)
		}
		sortCols[i] = df.columns[idx]
	}

	// Build index permutation.
	indices := make([]int, df.height)
	for i := range indices {
		indices[i] = i
	}

	sort.SliceStable(indices, func(a, b int) bool {
		for k, col := range sortCols {
			cmp := compareSeriesValues(col, indices[a], indices[b])
			if cmp == 0 {
				continue
			}
			if descending[k] {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})

	return df.take(indices), nil
}

// compareSeriesValues compares two values within a series. Returns -1, 0, or 1.
// Null values are sorted to the end (considered greater than any non-null value).
func compareSeriesValues(s *series.Series, i, j int) int {
	iNull := s.IsNull(i)
	jNull := s.IsNull(j)
	if iNull && jNull {
		return 0
	}
	if iNull {
		return 1
	}
	if jNull {
		return -1
	}

	switch s.DataType().String() {
	case "Int64":
		a, _ := s.GetInt64(i)
		b, _ := s.GetInt64(j)
		return cmpOrdered(a, b)
	case "Float64":
		a, _ := s.GetFloat64(i)
		b, _ := s.GetFloat64(j)
		return cmpOrdered(a, b)
	case "String":
		a, _ := s.GetString(i)
		b, _ := s.GetString(j)
		return cmpOrdered(a, b)
	case "Boolean":
		a, _ := s.GetBool(i)
		b, _ := s.GetBool(j)
		if a == b {
			return 0
		}
		if !a {
			return -1
		}
		return 1
	}
	return 0
}

// cmpOrdered compares two ordered values.
func cmpOrdered[T ~int64 | ~float64 | ~string](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// take returns a new DataFrame with rows reordered according to indices.
func (df *DataFrame) take(indices []int) *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.Take(indices)
	}
	result, _ := New(cols...)
	return result
}
