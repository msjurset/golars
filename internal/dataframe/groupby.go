package dataframe

import (
	"fmt"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// GroupByResult holds the result of a GroupBy operation, which can then be
// aggregated with Agg.
type GroupByResult struct {
	df       *DataFrame
	keys     []string
	groups   map[string][]int // hash -> row indices
	groupKeys [][]any         // ordered group key values
}

// GroupBy groups the DataFrame by the given column names.
func (df *DataFrame) GroupBy(keys ...string) (*GroupByResult, error) {
	for _, key := range keys {
		if !df.schema.Contains(key) {
			return nil, fmt.Errorf("golars: groupby: column %q not found", key)
		}
	}

	keyCols := make([]*series.Series, len(keys))
	for i, k := range keys {
		c, _ := df.Column(k)
		keyCols[i] = c
	}

	groups := make(map[string][]int)
	var orderedKeys []string
	var groupKeyValues [][]any

	for i := 0; i < df.height; i++ {
		hash := hashRow(keyCols, i)
		if _, exists := groups[hash]; !exists {
			orderedKeys = append(orderedKeys, hash)
			vals := make([]any, len(keys))
			for j, col := range keyCols {
				vals[j] = getAny(col, i)
			}
			groupKeyValues = append(groupKeyValues, vals)
		}
		groups[hash] = append(groups[hash], i)
	}

	// Reorder groups map to preserve insertion order
	orderedGroups := make(map[string][]int, len(groups))
	for _, k := range orderedKeys {
		orderedGroups[k] = groups[k]
	}

	return &GroupByResult{
		df:        df,
		keys:      keys,
		groups:    orderedGroups,
		groupKeys: groupKeyValues,
	}, nil
}

// Agg applies aggregation functions to each group and returns a new DataFrame.
// Each aggFunc maps column names to aggregation operations.
func (g *GroupByResult) Agg(aggs map[string]AggFunc) (*DataFrame, error) {
	nGroups := len(g.groupKeys)

	// Build key columns
	keyCols := make([]*series.Series, len(g.keys))
	for i, key := range g.keys {
		keyCol, _ := g.df.Column(key)
		keyCols[i] = buildGroupKeyColumn(key, keyCol.DataType(), g.groupKeys, i, nGroups)
	}

	// Build aggregation columns
	var aggCols []*series.Series
	for colName, aggFn := range aggs {
		srcCol, err := g.df.Column(colName)
		if err != nil {
			return nil, fmt.Errorf("golars: groupby agg: column %q not found", colName)
		}
		resultCol, err := applyGroupAgg(srcCol, g.groupKeys, g.groups, aggFn, nGroups)
		if err != nil {
			return nil, err
		}
		aggCols = append(aggCols, resultCol)
	}

	allCols := make([]*series.Series, 0, len(keyCols)+len(aggCols))
	allCols = append(allCols, keyCols...)
	allCols = append(allCols, aggCols...)

	return New(allCols...)
}

// AggFunc represents an aggregation function.
type AggFunc int

// Aggregation function constants.
const (
	AggSum AggFunc = iota
	AggMean
	AggMin
	AggMax
	AggCount
	AggFirst
	AggLast
)

func hashRow(cols []*series.Series, i int) string {
	if len(cols) == 1 {
		return fmt.Sprintf("%v", getAny(cols[0], i))
	}
	key := ""
	for j, col := range cols {
		if j > 0 {
			key += "\x00"
		}
		key += fmt.Sprintf("%v", getAny(col, i))
	}
	return key
}

func getAny(s *series.Series, i int) any {
	if s.IsNull(i) {
		return nil
	}
	switch s.DataType() {
	case dtype.Int64:
		v, _ := s.GetInt64(i)
		return v
	case dtype.Float64:
		v, _ := s.GetFloat64(i)
		return v
	case dtype.String:
		v, _ := s.GetString(i)
		return v
	case dtype.Boolean:
		v, _ := s.GetBool(i)
		return v
	default:
		return nil
	}
}

func buildGroupKeyColumn(name string, dt dtype.DataType, groupKeys [][]any, keyIdx int, nGroups int) *series.Series {
	switch dt {
	case dtype.Int64:
		data := make([]int64, nGroups)
		valid := make([]bool, nGroups)
		for i, gk := range groupKeys {
			if gk[keyIdx] != nil {
				data[i] = gk[keyIdx].(int64)
				valid[i] = true
			}
		}
		hasNulls := false
		for _, v := range valid {
			if !v {
				hasNulls = true
				break
			}
		}
		if hasNulls {
			return series.NewInt64WithValidity(name, data, valid)
		}
		return series.NewInt64(name, data)
	case dtype.Float64:
		data := make([]float64, nGroups)
		for i, gk := range groupKeys {
			if gk[keyIdx] != nil {
				data[i] = gk[keyIdx].(float64)
			}
		}
		return series.NewFloat64(name, data)
	case dtype.String:
		data := make([]string, nGroups)
		for i, gk := range groupKeys {
			if gk[keyIdx] != nil {
				data[i] = gk[keyIdx].(string)
			}
		}
		return series.NewString(name, data)
	case dtype.Boolean:
		data := make([]bool, nGroups)
		for i, gk := range groupKeys {
			if gk[keyIdx] != nil {
				data[i] = gk[keyIdx].(bool)
			}
		}
		return series.NewBoolean(name, data)
	default:
		return series.NewString(name, make([]string, nGroups))
	}
}

func applyGroupAgg(col *series.Series, groupKeys [][]any, groups map[string][]int, fn AggFunc, nGroups int) (*series.Series, error) {
	name := col.Name()

	// Get ordered group keys to match groupKeys order
	orderedHashes := make([]string, nGroups)
	for i, gk := range groupKeys {
		if len(gk) == 1 {
			orderedHashes[i] = fmt.Sprintf("%v", gk[0])
		} else {
			h := ""
			for j, v := range gk {
				if j > 0 {
					h += "\x00"
				}
				h += fmt.Sprintf("%v", v)
			}
			orderedHashes[i] = h
		}
	}

	switch fn {
	case AggCount:
		data := make([]int64, nGroups)
		for i, hash := range orderedHashes {
			indices := groups[hash]
			count := int64(0)
			for _, idx := range indices {
				if col.IsValid(idx) {
					count++
				}
			}
			data[i] = count
		}
		return series.NewInt64(name, data), nil

	case AggSum, AggMean, AggMin, AggMax:
		return applyNumericGroupAgg(col, orderedHashes, groups, fn, nGroups, name)

	case AggFirst:
		return applyFirstLast(col, orderedHashes, groups, nGroups, name, true)

	case AggLast:
		return applyFirstLast(col, orderedHashes, groups, nGroups, name, false)

	default:
		return nil, fmt.Errorf("golars: groupby: unknown aggregation function")
	}
}

func applyNumericGroupAgg(col *series.Series, hashes []string, groups map[string][]int, fn AggFunc, n int, name string) (*series.Series, error) {
	data := make([]float64, n)
	valid := make([]bool, n)

	for i, hash := range hashes {
		indices := groups[hash]
		vals := make([]float64, 0, len(indices))
		for _, idx := range indices {
			if col.IsValid(idx) {
				var v float64
				switch col.DataType() {
				case dtype.Int64:
					iv, _ := col.GetInt64(idx)
					v = float64(iv)
				case dtype.Float64:
					v, _ = col.GetFloat64(idx)
				default:
					continue
				}
				vals = append(vals, v)
			}
		}

		if len(vals) == 0 {
			continue
		}

		valid[i] = true
		switch fn {
		case AggSum:
			s := 0.0
			for _, v := range vals {
				s += v
			}
			data[i] = s
		case AggMean:
			s := 0.0
			for _, v := range vals {
				s += v
			}
			data[i] = s / float64(len(vals))
		case AggMin:
			m := vals[0]
			for _, v := range vals[1:] {
				if v < m {
					m = v
				}
			}
			data[i] = m
		case AggMax:
			m := vals[0]
			for _, v := range vals[1:] {
				if v > m {
					m = v
				}
			}
			data[i] = m
		}
	}

	hasNulls := false
	for _, v := range valid {
		if !v {
			hasNulls = true
			break
		}
	}
	if hasNulls {
		return series.NewFloat64WithValidity(name, data, valid), nil
	}
	return series.NewFloat64(name, data), nil
}

func applyFirstLast(col *series.Series, hashes []string, groups map[string][]int, n int, name string, first bool) (*series.Series, error) {
	switch col.DataType() {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		for i, hash := range hashes {
			indices := groups[hash]
			if first {
				for _, idx := range indices {
					if col.IsValid(idx) {
						v, _ := col.GetInt64(idx)
						data[i] = v
						valid[i] = true
						break
					}
				}
			} else {
				for j := len(indices) - 1; j >= 0; j-- {
					idx := indices[j]
					if col.IsValid(idx) {
						v, _ := col.GetInt64(idx)
						data[i] = v
						valid[i] = true
						break
					}
				}
			}
		}
		return series.NewInt64WithValidity(name, data, valid), nil

	case dtype.String:
		data := make([]string, n)
		valid := make([]bool, n)
		for i, hash := range hashes {
			indices := groups[hash]
			if first {
				for _, idx := range indices {
					if col.IsValid(idx) {
						v, _ := col.GetString(idx)
						data[i] = v
						valid[i] = true
						break
					}
				}
			} else {
				for j := len(indices) - 1; j >= 0; j-- {
					idx := indices[j]
					if col.IsValid(idx) {
						v, _ := col.GetString(idx)
						data[i] = v
						valid[i] = true
						break
					}
				}
			}
		}
		return series.NewStringWithValidity(name, data, valid), nil

	default:
		return series.NewFloat64(name, make([]float64, n)), nil
	}
}

// Ensure imports are used.
var _ = array.NewInt64Array
var _ = bitmap.New
