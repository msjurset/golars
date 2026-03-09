package dataframe

import (
	"bytes"
	"fmt"
	"math"
	"unsafe"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// FNV-1a constants for 64-bit hashing.
const (
	fnvOffset64 = uint64(14695981039346656037)
	fnvPrime64  = uint64(1099511628211)
)

// hashCombine mixes an additional uint64 value into the running hash.
func hashCombine(hash, value uint64) uint64 {
	// Mix bytes of value into hash using FNV-1a byte-at-a-time
	b := (*[8]byte)(unsafe.Pointer(&value))
	hash ^= uint64(b[0])
	hash *= fnvPrime64
	hash ^= uint64(b[1])
	hash *= fnvPrime64
	hash ^= uint64(b[2])
	hash *= fnvPrime64
	hash ^= uint64(b[3])
	hash *= fnvPrime64
	hash ^= uint64(b[4])
	hash *= fnvPrime64
	hash ^= uint64(b[5])
	hash *= fnvPrime64
	hash ^= uint64(b[6])
	hash *= fnvPrime64
	hash ^= uint64(b[7])
	hash *= fnvPrime64
	return hash
}

// hashString hashes a string into the running hash using FNV-1a.
func hashString(hash uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= fnvPrime64
	}
	return hash
}

// hashStringBytes hashes raw bytes into the running hash using FNV-1a,
// avoiding the string allocation of hashString.
func hashStringBytes(hash uint64, data []byte) uint64 {
	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime64
	}
	return hash
}

// colHasher provides type-specific hashing for a single column. It avoids
// interface boxing by caching the underlying typed array pointer.
type colHasher struct {
	series   *series.Series
	dt       dtype.DataType
	int64s   *array.TypedArray[int64]
	float64s *array.TypedArray[float64]
	int32s   *array.TypedArray[int32]
	strings  *array.StringArray
	booleans *array.BooleanArray
	uint64s  *array.TypedArray[uint64]
	uint32s  *array.TypedArray[uint32]
	int16s   *array.TypedArray[int16]
	int8s    *array.TypedArray[int8]
	uint16s  *array.TypedArray[uint16]
	uint8s   *array.TypedArray[uint8]
	float32s *array.TypedArray[float32]
}

func newColHasher(s *series.Series) colHasher {
	ch := colHasher{series: s, dt: s.DataType()}
	arr := s.Array()
	switch ch.dt {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		ch.int64s, _ = arr.(*array.TypedArray[int64])
	case dtype.Float64:
		ch.float64s, _ = arr.(*array.TypedArray[float64])
	case dtype.Int32, dtype.Date:
		ch.int32s, _ = arr.(*array.TypedArray[int32])
	case dtype.String:
		ch.strings, _ = arr.(*array.StringArray)
	case dtype.Boolean:
		ch.booleans, _ = arr.(*array.BooleanArray)
	case dtype.UInt64:
		ch.uint64s, _ = arr.(*array.TypedArray[uint64])
	case dtype.UInt32:
		ch.uint32s, _ = arr.(*array.TypedArray[uint32])
	case dtype.Int16:
		ch.int16s, _ = arr.(*array.TypedArray[int16])
	case dtype.Int8:
		ch.int8s, _ = arr.(*array.TypedArray[int8])
	case dtype.UInt16:
		ch.uint16s, _ = arr.(*array.TypedArray[uint16])
	case dtype.UInt8:
		ch.uint8s, _ = arr.(*array.TypedArray[uint8])
	case dtype.Float32:
		ch.float32s, _ = arr.(*array.TypedArray[float32])
	}
	return ch
}

// hashValue hashes the value at index i into the running hash.
// Null values are hashed as a sentinel byte 0x00; non-null values start with 0x01.
func (ch *colHasher) hashValue(hash uint64, i int) uint64 {
	if ch.series.IsNull(i) {
		hash ^= 0
		hash *= fnvPrime64
		return hash
	}
	// Non-null sentinel
	hash ^= 1
	hash *= fnvPrime64

	switch ch.dt {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		if ch.int64s != nil {
			return hashCombine(hash, uint64(ch.int64s.Value(i)))
		}
	case dtype.Float64:
		if ch.float64s != nil {
			v := ch.float64s.Value(i)
			return hashCombine(hash, math.Float64bits(v))
		}
	case dtype.Int32, dtype.Date:
		if ch.int32s != nil {
			return hashCombine(hash, uint64(ch.int32s.Value(i)))
		}
	case dtype.String:
		if ch.strings != nil {
			return hashStringBytes(hash, ch.strings.ValueBytes(i))
		}
	case dtype.Boolean:
		if ch.booleans != nil {
			if ch.booleans.Value(i) {
				hash ^= 1
			} else {
				hash ^= 2
			}
			hash *= fnvPrime64
			return hash
		}
	case dtype.UInt64:
		if ch.uint64s != nil {
			return hashCombine(hash, ch.uint64s.Value(i))
		}
	case dtype.UInt32:
		if ch.uint32s != nil {
			return hashCombine(hash, uint64(ch.uint32s.Value(i)))
		}
	case dtype.Int16:
		if ch.int16s != nil {
			return hashCombine(hash, uint64(ch.int16s.Value(i)))
		}
	case dtype.Int8:
		if ch.int8s != nil {
			return hashCombine(hash, uint64(ch.int8s.Value(i)))
		}
	case dtype.UInt16:
		if ch.uint16s != nil {
			return hashCombine(hash, uint64(ch.uint16s.Value(i)))
		}
	case dtype.UInt8:
		if ch.uint8s != nil {
			return hashCombine(hash, uint64(ch.uint8s.Value(i)))
		}
	case dtype.Float32:
		if ch.float32s != nil {
			return hashCombine(hash, uint64(math.Float32bits(ch.float32s.Value(i))))
		}
	}
	return hash
}

// rowsEqual returns true if two rows have equal values across all key columns.
func rowsEqual(hashers []colHasher, i, j int) bool {
	for _, ch := range hashers {
		ni := ch.series.IsNull(i)
		nj := ch.series.IsNull(j)
		if ni != nj {
			return false
		}
		if ni {
			continue
		}
		switch ch.dt {
		case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
			if ch.int64s != nil {
				if ch.int64s.Value(i) != ch.int64s.Value(j) {
					return false
				}
			}
		case dtype.Float64:
			if ch.float64s != nil {
				if ch.float64s.Value(i) != ch.float64s.Value(j) {
					return false
				}
			}
		case dtype.Int32, dtype.Date:
			if ch.int32s != nil {
				if ch.int32s.Value(i) != ch.int32s.Value(j) {
					return false
				}
			}
		case dtype.String:
			if ch.strings != nil {
				if !bytes.Equal(ch.strings.ValueBytes(i), ch.strings.ValueBytes(j)) {
					return false
				}
			}
		case dtype.Boolean:
			if ch.booleans != nil {
				if ch.booleans.Value(i) != ch.booleans.Value(j) {
					return false
				}
			}
		case dtype.UInt64:
			if ch.uint64s != nil {
				if ch.uint64s.Value(i) != ch.uint64s.Value(j) {
					return false
				}
			}
		case dtype.UInt32:
			if ch.uint32s != nil {
				if ch.uint32s.Value(i) != ch.uint32s.Value(j) {
					return false
				}
			}
		case dtype.Int16:
			if ch.int16s != nil {
				if ch.int16s.Value(i) != ch.int16s.Value(j) {
					return false
				}
			}
		case dtype.Int8:
			if ch.int8s != nil {
				if ch.int8s.Value(i) != ch.int8s.Value(j) {
					return false
				}
			}
		case dtype.UInt16:
			if ch.uint16s != nil {
				if ch.uint16s.Value(i) != ch.uint16s.Value(j) {
					return false
				}
			}
		case dtype.UInt8:
			if ch.uint8s != nil {
				if ch.uint8s.Value(i) != ch.uint8s.Value(j) {
					return false
				}
			}
		case dtype.Float32:
			if ch.float32s != nil {
				if ch.float32s.Value(i) != ch.float32s.Value(j) {
					return false
				}
			}
		}
	}
	return true
}

// hashRowFast computes an FNV-1a hash for a row across multiple columns.
func hashRowFast(hashers []colHasher, i int) uint64 {
	h := fnvOffset64
	for k := range hashers {
		h = hashers[k].hashValue(h, i)
	}
	return h
}

// GroupByResult holds the result of a GroupBy operation, which can then be
// aggregated with Agg.
type GroupByResult struct {
	df        *DataFrame
	keys      []string
	groups    map[uint64][]int // hash -> row indices
	groupKeys [][]any          // ordered group key values
	// groupOrder preserves insertion order of groups; each entry is a hash key.
	groupOrder []uint64
	// keyHashers are cached for collision checks during aggregation.
	keyHashers []colHasher
}

// GroupBy groups the DataFrame by the given column names.
func (df *DataFrame) GroupBy(keys ...string) (*GroupByResult, error) {
	for _, key := range keys {
		if !df.schema.Contains(key) {
			return nil, fmt.Errorf("golars: groupby: column %q not found", key)
		}
	}

	keyCols := make([]*series.Series, len(keys))
	hashers := make([]colHasher, len(keys))
	for i, k := range keys {
		c, _ := df.Column(k)
		keyCols[i] = c
		hashers[i] = newColHasher(c)
	}

	// Pre-allocate with reasonable capacity
	estGroups := df.height / 4
	if estGroups < 16 {
		estGroups = 16
	}
	groups := make(map[uint64][]int, estGroups)
	groupOrder := make([]uint64, 0, estGroups)
	var groupKeyValues [][]any

	for i := 0; i < df.height; i++ {
		h := hashRowFast(hashers, i)

		bucket, exists := groups[h]
		if exists {
			// Check for hash collision: does this row actually match the group?
			// Compare against the first row in the bucket.
			if rowsEqual(hashers, i, bucket[0]) {
				groups[h] = append(bucket, i)
				continue
			}
			// Hash collision with different values. Use a secondary probe:
			// linear probe on hash space until we find a matching bucket or empty slot.
			for probe := uint64(1); ; probe++ {
				ph := h + probe
				pb, pExists := groups[ph]
				if !pExists {
					// New group at probed slot
					groupOrder = append(groupOrder, ph)
					vals := make([]any, len(keys))
					for j, col := range keyCols {
						vals[j] = getAny(col, i)
					}
					groupKeyValues = append(groupKeyValues, vals)
					groups[ph] = append(make([]int, 0, 4), i)
					break
				}
				if rowsEqual(hashers, i, pb[0]) {
					groups[ph] = append(pb, i)
					break
				}
			}
			continue
		}

		// New group
		groupOrder = append(groupOrder, h)
		vals := make([]any, len(keys))
		for j, col := range keyCols {
			vals[j] = getAny(col, i)
		}
		groupKeyValues = append(groupKeyValues, vals)
		groups[h] = append(make([]int, 0, 4), i)
	}

	return &GroupByResult{
		df:         df,
		keys:       keys,
		groups:     groups,
		groupKeys:  groupKeyValues,
		groupOrder: groupOrder,
		keyHashers: hashers,
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
		resultCol, err := applyGroupAgg(srcCol, g.groupOrder, g.groups, aggFn, nGroups)
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

func applyGroupAgg(col *series.Series, groupOrder []uint64, groups map[uint64][]int, fn AggFunc, nGroups int) (*series.Series, error) {
	name := col.Name()

	switch fn {
	case AggCount:
		data := make([]int64, nGroups)
		for i, hash := range groupOrder {
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
		return applyNumericGroupAgg(col, groupOrder, groups, fn, nGroups, name)

	case AggFirst:
		return applyFirstLast(col, groupOrder, groups, nGroups, name, true)

	case AggLast:
		return applyFirstLast(col, groupOrder, groups, nGroups, name, false)

	default:
		return nil, fmt.Errorf("golars: groupby: unknown aggregation function")
	}
}

func applyNumericGroupAgg(col *series.Series, groupOrder []uint64, groups map[uint64][]int, fn AggFunc, n int, name string) (*series.Series, error) {
	switch col.DataType() {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		return applyGroupAggTyped[int64](col, groupOrder, groups, fn, n, name)
	case dtype.Float64:
		return applyGroupAggTyped[float64](col, groupOrder, groups, fn, n, name)
	case dtype.Int32, dtype.Date:
		return applyGroupAggTyped[int32](col, groupOrder, groups, fn, n, name)
	case dtype.Float32:
		return applyGroupAggTyped[float32](col, groupOrder, groups, fn, n, name)
	case dtype.Int16:
		return applyGroupAggTyped[int16](col, groupOrder, groups, fn, n, name)
	case dtype.Int8:
		return applyGroupAggTyped[int8](col, groupOrder, groups, fn, n, name)
	case dtype.UInt64:
		return applyGroupAggTyped[uint64](col, groupOrder, groups, fn, n, name)
	case dtype.UInt32:
		return applyGroupAggTyped[uint32](col, groupOrder, groups, fn, n, name)
	case dtype.UInt16:
		return applyGroupAggTyped[uint16](col, groupOrder, groups, fn, n, name)
	case dtype.UInt8:
		return applyGroupAggTyped[uint8](col, groupOrder, groups, fn, n, name)
	default:
		return nil, fmt.Errorf("golars: groupby: numeric aggregation not supported for %s", col.DataType())
	}
}

// applyGroupAggTyped performs numeric group aggregation directly on the typed
// array's backing slice, avoiding per-element type switches, interface
// assertions, and intermediate slice allocations.
func applyGroupAggTyped[T array.Numeric](col *series.Series, groupOrder []uint64, groups map[uint64][]int, fn AggFunc, n int, name string) (*series.Series, error) {
	ta := col.Array().(*array.TypedArray[T])
	vals := ta.Values()
	validity := ta.Validity()
	hasNulls := validity != nil

	data := make([]float64, n)
	valid := make([]bool, n)

	for i, hash := range groupOrder {
		indices := groups[hash]

		switch fn {
		case AggSum:
			sum := 0.0
			count := 0
			for _, idx := range indices {
				if !hasNulls || validity.IsSet(idx) {
					sum += float64(vals[idx])
					count++
				}
			}
			if count > 0 {
				data[i] = sum
				valid[i] = true
			}
		case AggMean:
			sum := 0.0
			count := 0
			for _, idx := range indices {
				if !hasNulls || validity.IsSet(idx) {
					sum += float64(vals[idx])
					count++
				}
			}
			if count > 0 {
				data[i] = sum / float64(count)
				valid[i] = true
			}
		case AggMin:
			first := true
			min := 0.0
			for _, idx := range indices {
				if !hasNulls || validity.IsSet(idx) {
					v := float64(vals[idx])
					if first || v < min {
						min = v
						first = false
					}
				}
			}
			if !first {
				data[i] = min
				valid[i] = true
			}
		case AggMax:
			first := true
			max := 0.0
			for _, idx := range indices {
				if !hasNulls || validity.IsSet(idx) {
					v := float64(vals[idx])
					if first || v > max {
						max = v
						first = false
					}
				}
			}
			if !first {
				data[i] = max
				valid[i] = true
			}
		}
	}

	hasNullResult := false
	for _, v := range valid {
		if !v {
			hasNullResult = true
			break
		}
	}
	if hasNullResult {
		return series.NewFloat64WithValidity(name, data, valid), nil
	}
	return series.NewFloat64(name, data), nil
}

func applyFirstLast(col *series.Series, groupOrder []uint64, groups map[uint64][]int, n int, name string, first bool) (*series.Series, error) {
	switch col.DataType() {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		for i, hash := range groupOrder {
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
		for i, hash := range groupOrder {
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

// GroupByExpr is an interface for expression-based GroupBy aggregation.
// This avoids circular imports between dataframe and expr packages.
type GroupByExpr interface {
	EvaluateGroupBy(df *DataFrame) (*series.Series, error)
}

// AggExprs applies expression-based aggregations to each group.
func (g *GroupByResult) AggExprs(exprs ...GroupByExpr) (*DataFrame, error) {
	nGroups := len(g.groupKeys)

	// Build key columns (same as in Agg)
	keyCols := make([]*series.Series, len(g.keys))
	for i, key := range g.keys {
		keyCol, _ := g.df.Column(key)
		keyCols[i] = buildGroupKeyColumn(key, keyCol.DataType(), g.groupKeys, i, nGroups)
	}

	// Evaluate each expression per group
	aggCols := make([]*series.Series, len(exprs))
	for ei, gbe := range exprs {
		// For each group, build a sub-DataFrame, evaluate the expression, collect scalar results
		results := make([]*series.Series, nGroups)
		for gi, hash := range g.groupOrder {
			indices := g.groups[hash]
			subCols := make([]*series.Series, len(g.df.Columns()))
			for ci, col := range g.df.Columns() {
				subCols[ci] = col.Take(indices)
			}
			subDF, err := New(subCols...)
			if err != nil {
				return nil, fmt.Errorf("golars: groupby expr: %w", err)
			}
			result, err := gbe.EvaluateGroupBy(subDF)
			if err != nil {
				return nil, fmt.Errorf("golars: groupby expr: %w", err)
			}
			results[gi] = result
		}

		// Collect results into a single column using the first result's name
		colName := ""
		if nGroups > 0 {
			colName = results[0].Name()
		}
		aggCol, err := collectGroupResults(colName, results, nGroups)
		if err != nil {
			return nil, err
		}
		aggCols[ei] = aggCol
	}

	allCols := make([]*series.Series, 0, len(keyCols)+len(aggCols))
	allCols = append(allCols, keyCols...)
	allCols = append(allCols, aggCols...)

	return New(allCols...)
}

func collectGroupResults(name string, results []*series.Series, nGroups int) (*series.Series, error) {
	if nGroups == 0 {
		return series.NewFloat64(name, nil), nil
	}

	// Determine result type from first result
	dt := results[0].DataType()

	switch dt {
	case dtype.Float64:
		data := make([]float64, nGroups)
		valid := make([]bool, nGroups)
		for i, r := range results {
			if r.Len() > 0 && r.IsValid(0) {
				v, _ := r.GetFloat64(0)
				data[i] = v
				valid[i] = true
			}
		}
		if hasAnyFalse(valid) {
			return series.NewFloat64WithValidity(name, data, valid), nil
		}
		return series.NewFloat64(name, data), nil
	case dtype.Int64:
		data := make([]int64, nGroups)
		valid := make([]bool, nGroups)
		for i, r := range results {
			if r.Len() > 0 && r.IsValid(0) {
				v, _ := r.GetInt64(0)
				data[i] = v
				valid[i] = true
			}
		}
		if hasAnyFalse(valid) {
			return series.NewInt64WithValidity(name, data, valid), nil
		}
		return series.NewInt64(name, data), nil
	case dtype.String:
		data := make([]string, nGroups)
		valid := make([]bool, nGroups)
		for i, r := range results {
			if r.Len() > 0 && r.IsValid(0) {
				v, _ := r.GetString(0)
				data[i] = v
				valid[i] = true
			}
		}
		if hasAnyFalse(valid) {
			return series.NewStringWithValidity(name, data, valid), nil
		}
		return series.NewString(name, data), nil
	case dtype.Boolean:
		data := make([]bool, nGroups)
		valid := make([]bool, nGroups)
		for i, r := range results {
			if r.Len() > 0 && r.IsValid(0) {
				v, _ := r.GetBool(0)
				data[i] = v
				valid[i] = true
			}
		}
		if hasAnyFalse(valid) {
			return series.NewBooleanWithValidity(name, data, valid), nil
		}
		return series.NewBoolean(name, data), nil
	default:
		return nil, fmt.Errorf("golars: groupby expr: unsupported result type %s", dt)
	}
}

func hasAnyFalse(valid []bool) bool {
	for _, v := range valid {
		if !v {
			return true
		}
	}
	return false
}

// Ensure imports are used.
var _ = array.NewInt64Array
var _ = bitmap.New
