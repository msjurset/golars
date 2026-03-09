package dataframe

import (
	"bytes"
	"fmt"
	"slices"
	"sync"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
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

	// Fast path: single numeric column sort using radix sort.
	if len(sortCols) == 1 {
		col := sortCols[0]
		desc := descending[0]
		var indices []int
		switch col.DataType() {
		case dtype.Int8:
			indices = array.ArgSortInt8(col.Array().(*array.TypedArray[int8]), desc)
		case dtype.Int16:
			indices = array.ArgSortInt16(col.Array().(*array.TypedArray[int16]), desc)
		case dtype.Int32, dtype.Date:
			indices = array.ArgSortInt32(col.Array().(*array.TypedArray[int32]), desc)
		case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
			indices = array.ArgSortInt64(col.Array().(*array.TypedArray[int64]), desc)
		case dtype.UInt8:
			indices = array.ArgSortUint8(col.Array().(*array.TypedArray[uint8]), desc)
		case dtype.UInt16:
			indices = array.ArgSortUint16(col.Array().(*array.TypedArray[uint16]), desc)
		case dtype.UInt32:
			indices = array.ArgSortUint32(col.Array().(*array.TypedArray[uint32]), desc)
		case dtype.UInt64:
			indices = array.ArgSortUint64(col.Array().(*array.TypedArray[uint64]), desc)
		case dtype.Float32:
			indices = array.ArgSortFloat32(col.Array().(*array.TypedArray[float32]), desc)
		case dtype.Float64:
			indices = array.ArgSortFloat64(col.Array().(*array.TypedArray[float64]), desc)
		}
		if indices != nil {
			return df.take(indices), nil
		}
	}

	// Pre-build comparators for each sort column.
	cmps := make([]func(i, j int) int, len(sortCols))
	for k, col := range sortCols {
		cmps[k] = buildComparator(col, descending[k])
	}

	// Build index permutation.
	indices := make([]int, df.height)
	for i := range indices {
		indices[i] = i
	}

	slices.SortStableFunc(indices, func(a, b int) int {
		for _, cmp := range cmps {
			if c := cmp(a, b); c != 0 {
				return c
			}
		}
		return 0
	})

	return df.take(indices), nil
}

// buildComparator creates a pre-bound comparison function for a single sort
// column. The returned closure compares two row indices and returns -1, 0, or 1.
// Null values are sorted to the end (greater than any non-null value).
// The descending flag is baked into the closure.
func buildComparator(s *series.Series, descending bool) func(i, j int) int {
	hasNulls := s.NullCount() > 0
	validity := s.Validity()

	// Helper: check null via pre-extracted bitmap.
	isNull := func(i int) bool {
		return !validity.IsSet(i)
	}

	// nullCmp handles the null-vs-null and null-vs-value cases.
	// Returns (result, handled). If handled is true, the caller should return result.
	nullCmp := func(i, j int) (int, bool) {
		iN := isNull(i)
		jN := isNull(j)
		if iN && jN {
			return 0, true
		}
		if iN {
			return 1, true // nulls last, even when descending
		}
		if jN {
			return -1, true
		}
		return 0, false
	}

	dt := s.DataType()

	switch dt {
	case dtype.Int8:
		ta := s.Array().(*array.TypedArray[int8])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.Int16:
		ta := s.Array().(*array.TypedArray[int16])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.Int32, dtype.Date:
		ta := s.Array().(*array.TypedArray[int32])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		ta := s.Array().(*array.TypedArray[int64])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.UInt8:
		ta := s.Array().(*array.TypedArray[uint8])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.UInt16:
		ta := s.Array().(*array.TypedArray[uint16])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.UInt32:
		ta := s.Array().(*array.TypedArray[uint32])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.UInt64:
		ta := s.Array().(*array.TypedArray[uint64])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.Float32:
		ta := s.Array().(*array.TypedArray[float32])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.Float64:
		ta := s.Array().(*array.TypedArray[float64])
		vals := ta.Values()
		return buildOrderedCmp(vals, hasNulls, nullCmp, descending)
	case dtype.String, dtype.Binary:
		sa := s.Array().(*array.StringArray)
		if hasNulls {
			return func(i, j int) int {
				if c, done := nullCmp(i, j); done {
					return c
				}
				c := bytes.Compare(sa.ValueBytes(i), sa.ValueBytes(j))
				if descending {
					return -c
				}
				return c
			}
		}
		return func(i, j int) int {
			c := bytes.Compare(sa.ValueBytes(i), sa.ValueBytes(j))
			if descending {
				return -c
			}
			return c
		}
	case dtype.Boolean:
		ba := s.Array().(*array.BooleanArray)
		dataBm := ba.DataBitmap()
		if hasNulls {
			return func(i, j int) int {
				if c, done := nullCmp(i, j); done {
					return c
				}
				vi := dataBm.IsSet(i)
				vj := dataBm.IsSet(j)
				c := cmpBool(vi, vj)
				if descending {
					return -c
				}
				return c
			}
		}
		return func(i, j int) int {
			vi := dataBm.IsSet(i)
			vj := dataBm.IsSet(j)
			c := cmpBool(vi, vj)
			if descending {
				return -c
			}
			return c
		}
	default:
		// Fallback: use the interface-level comparison for unsupported types.
		return func(i, j int) int {
			return 0
		}
	}
}

// buildOrderedCmp returns a comparator closure for any ordered numeric type.
// Splitting hasNulls into two paths avoids a branch in the hot loop.
func buildOrderedCmp[T interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}](vals []T, hasNulls bool, nullCmp func(int, int) (int, bool), descending bool) func(int, int) int {
	if hasNulls {
		return func(i, j int) int {
			if c, done := nullCmp(i, j); done {
				return c
			}
			c := cmpOrdered(vals[i], vals[j])
			if descending {
				return -c
			}
			return c
		}
	}
	return func(i, j int) int {
		c := cmpOrdered(vals[i], vals[j])
		if descending {
			return -c
		}
		return c
	}
}

// cmpOrdered compares two ordered values.
func cmpOrdered[T interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string
}](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// cmpBool compares two booleans (false < true).
func cmpBool(a, b bool) int {
	if a == b {
		return 0
	}
	if !a {
		return -1
	}
	return 1
}

// take returns a new DataFrame with rows reordered according to indices.
// When there are multiple columns, the take operations run in parallel.
func (df *DataFrame) take(indices []int) *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	if len(df.columns) > 1 {
		var wg sync.WaitGroup
		for i, c := range df.columns {
			wg.Add(1)
			go func(idx int, s *series.Series) {
				defer wg.Done()
				cols[idx] = s.Take(indices)
			}(i, c)
		}
		wg.Wait()
	} else {
		for i, c := range df.columns {
			cols[i] = c.Take(indices)
		}
	}
	result, _ := New(cols...)
	return result
}
