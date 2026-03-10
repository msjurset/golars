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

// hashStringBytes hashes raw bytes into the running hash using FNV-1a.
func hashStringBytes(hash uint64, data []byte) uint64 {
	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime64
	}
	return hash
}

// colHasher provides type-specific hashing for a single column.
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

func (ch *colHasher) hashValue(hash uint64, i int) uint64 {
	if ch.series.IsNull(i) {
		hash ^= 0
		hash *= fnvPrime64
		return hash
	}
	hash ^= 1
	hash *= fnvPrime64

	switch ch.dt {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		if ch.int64s != nil {
			return hashCombine(hash, uint64(ch.int64s.Value(i)))
		}
	case dtype.Float64:
		if ch.float64s != nil {
			return hashCombine(hash, math.Float64bits(ch.float64s.Value(i)))
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
			if ch.int64s != nil && ch.int64s.Value(i) != ch.int64s.Value(j) {
				return false
			}
		case dtype.Float64:
			if ch.float64s != nil && ch.float64s.Value(i) != ch.float64s.Value(j) {
				return false
			}
		case dtype.Int32, dtype.Date:
			if ch.int32s != nil && ch.int32s.Value(i) != ch.int32s.Value(j) {
				return false
			}
		case dtype.String:
			if ch.strings != nil && !bytes.Equal(ch.strings.ValueBytes(i), ch.strings.ValueBytes(j)) {
				return false
			}
		case dtype.Boolean:
			if ch.booleans != nil && ch.booleans.Value(i) != ch.booleans.Value(j) {
				return false
			}
		case dtype.UInt64:
			if ch.uint64s != nil && ch.uint64s.Value(i) != ch.uint64s.Value(j) {
				return false
			}
		case dtype.UInt32:
			if ch.uint32s != nil && ch.uint32s.Value(i) != ch.uint32s.Value(j) {
				return false
			}
		case dtype.Int16:
			if ch.int16s != nil && ch.int16s.Value(i) != ch.int16s.Value(j) {
				return false
			}
		case dtype.Int8:
			if ch.int8s != nil && ch.int8s.Value(i) != ch.int8s.Value(j) {
				return false
			}
		case dtype.UInt16:
			if ch.uint16s != nil && ch.uint16s.Value(i) != ch.uint16s.Value(j) {
				return false
			}
		case dtype.UInt8:
			if ch.uint8s != nil && ch.uint8s.Value(i) != ch.uint8s.Value(j) {
				return false
			}
		case dtype.Float32:
			if ch.float32s != nil && ch.float32s.Value(i) != ch.float32s.Value(j) {
				return false
			}
		}
	}
	return true
}

func hashRowFast(hashers []colHasher, i int) uint64 {
	h := fnvOffset64
	for k := range hashers {
		h = hashers[k].hashValue(h, i)
	}
	return h
}

// htEmpty is the sentinel for an empty slot in the hash table.
const htEmpty = int32(-1)

// htSlot is one slot in the open-addressing hash table.
type htSlot struct {
	hash     uint64
	firstRow int32 // row index of first member of this group; htEmpty = empty slot
	gid      int32 // group ID (0-based, insertion order)
}

// groupHashTable is a custom open-addressing hash table that maps row keys to
// group IDs using linear probing. Its capacity is always a power of two.
type groupHashTable struct {
	slots   []htSlot
	mask    uint64
	nGroups int32
}

// newGroupHashTable allocates a table sized for at least estGroups groups with
// load factor < 0.5.
func newGroupHashTable(estGroups int) *groupHashTable {
	size := uint64(16)
	for size < uint64(estGroups)*2 {
		size <<= 1
	}
	slots := make([]htSlot, size)
	for i := range slots {
		slots[i].firstRow = htEmpty
	}
	return &groupHashTable{slots: slots, mask: size - 1}
}

// probe returns the group ID for the row at rowIdx with hash h, inserting a
// new group if none exists.
func (ht *groupHashTable) probe(hashers []colHasher, rowIdx int, h uint64) int32 {
	slot := h & ht.mask
	for {
		s := &ht.slots[slot]
		if s.firstRow == htEmpty {
			gid := ht.nGroups
			ht.nGroups++
			s.hash = h
			s.firstRow = int32(rowIdx)
			s.gid = gid
			// Grow table if load factor reaches 0.5
			if int(ht.nGroups)*2 >= len(ht.slots) {
				ht.grow(hashers)
				// Re-probe after grow (slot may have moved)
				return ht.findGID(hashers, rowIdx, h)
			}
			return gid
		}
		if s.hash == h && rowsEqual(hashers, rowIdx, int(s.firstRow)) {
			return s.gid
		}
		slot = (slot + 1) & ht.mask
	}
}

// findGID finds the group ID for an already-inserted row after a grow.
func (ht *groupHashTable) findGID(hashers []colHasher, rowIdx int, h uint64) int32 {
	slot := h & ht.mask
	for {
		s := &ht.slots[slot]
		if s.firstRow != htEmpty && s.hash == h && rowsEqual(hashers, rowIdx, int(s.firstRow)) {
			return s.gid
		}
		slot = (slot + 1) & ht.mask
	}
}

// grow doubles the table capacity and rehashes all entries.
func (ht *groupHashTable) grow(hashers []colHasher) {
	newSize := uint64(len(ht.slots)) * 2
	newSlots := make([]htSlot, newSize)
	for i := range newSlots {
		newSlots[i].firstRow = htEmpty
	}
	newMask := newSize - 1
	for _, s := range ht.slots {
		if s.firstRow == htEmpty {
			continue
		}
		slot := s.hash & newMask
		for newSlots[slot].firstRow != htEmpty {
			slot = (slot + 1) & newMask
		}
		newSlots[slot] = s
	}
	ht.slots = newSlots
	ht.mask = newMask
}

// GroupByResult holds the result of a GroupBy operation.
type GroupByResult struct {
	df         *DataFrame
	keys       []string
	nGroups    int
	groupIDs   []int32  // per-row group ID (gid 0 = first group encountered)
	groupKeys  [][]any  // key values per group, indexed by gid
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

	estGroups := df.height / 4
	if estGroups < 16 {
		estGroups = 16
	}
	ht := newGroupHashTable(estGroups)
	groupIDs := make([]int32, df.height)

	for i := 0; i < df.height; i++ {
		h := hashRowFast(hashers, i)
		groupIDs[i] = ht.probe(hashers, i, h)
	}

	nGroups := int(ht.nGroups)

	// Build group key values from each group's representative row (firstRow).
	groupKeyValues := make([][]any, nGroups)
	for i := range ht.slots {
		s := &ht.slots[i]
		if s.firstRow != htEmpty {
			vals := make([]any, len(keys))
			for j, col := range keyCols {
				vals[j] = getAny(col, int(s.firstRow))
			}
			groupKeyValues[s.gid] = vals
		}
	}

	return &GroupByResult{
		df:         df,
		keys:       keys,
		nGroups:    nGroups,
		groupIDs:   groupIDs,
		groupKeys:  groupKeyValues,
		keyHashers: hashers,
	}, nil
}

// Agg applies aggregation functions to each group and returns a new DataFrame.
func (g *GroupByResult) Agg(aggs map[string]AggFunc) (*DataFrame, error) {
	nGroups := g.nGroups

	keyCols := make([]*series.Series, len(g.keys))
	for i, key := range g.keys {
		keyCol, _ := g.df.Column(key)
		keyCols[i] = buildGroupKeyColumn(key, keyCol.DataType(), g.groupKeys, i, nGroups)
	}

	var aggCols []*series.Series
	for colName, aggFn := range aggs {
		srcCol, err := g.df.Column(colName)
		if err != nil {
			return nil, fmt.Errorf("golars: groupby agg: column %q not found", colName)
		}
		resultCol, err := applyGroupAgg(srcCol, g.groupIDs, aggFn, nGroups)
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

func applyGroupAgg(col *series.Series, groupIDs []int32, fn AggFunc, nGroups int) (*series.Series, error) {
	name := col.Name()

	switch fn {
	case AggCount:
		data := make([]int64, nGroups)
		for i := 0; i < col.Len(); i++ {
			if col.IsValid(i) {
				data[groupIDs[i]]++
			}
		}
		return series.NewInt64(name, data), nil

	case AggSum, AggMean, AggMin, AggMax:
		return applyNumericGroupAgg(col, groupIDs, fn, nGroups, name)

	case AggFirst:
		return applyFirstLast(col, groupIDs, nGroups, name, true)

	case AggLast:
		return applyFirstLast(col, groupIDs, nGroups, name, false)

	default:
		return nil, fmt.Errorf("golars: groupby: unknown aggregation function")
	}
}

func applyNumericGroupAgg(col *series.Series, groupIDs []int32, fn AggFunc, n int, name string) (*series.Series, error) {
	switch col.DataType() {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		return applyGroupAggTyped[int64](col, groupIDs, fn, n, name)
	case dtype.Float64:
		return applyGroupAggTyped[float64](col, groupIDs, fn, n, name)
	case dtype.Int32, dtype.Date:
		return applyGroupAggTyped[int32](col, groupIDs, fn, n, name)
	case dtype.Float32:
		return applyGroupAggTyped[float32](col, groupIDs, fn, n, name)
	case dtype.Int16:
		return applyGroupAggTyped[int16](col, groupIDs, fn, n, name)
	case dtype.Int8:
		return applyGroupAggTyped[int8](col, groupIDs, fn, n, name)
	case dtype.UInt64:
		return applyGroupAggTyped[uint64](col, groupIDs, fn, n, name)
	case dtype.UInt32:
		return applyGroupAggTyped[uint32](col, groupIDs, fn, n, name)
	case dtype.UInt16:
		return applyGroupAggTyped[uint16](col, groupIDs, fn, n, name)
	case dtype.UInt8:
		return applyGroupAggTyped[uint8](col, groupIDs, fn, n, name)
	default:
		return nil, fmt.Errorf("golars: groupby: numeric aggregation not supported for %s", col.DataType())
	}
}

// applyGroupAggTyped performs numeric group aggregation using a single
// sequential pass over the data array, accumulating directly into per-group
// result slots indexed by groupIDs. This avoids per-group index list
// traversal and is cache-friendly for small group counts.
func applyGroupAggTyped[T array.Numeric](col *series.Series, groupIDs []int32, fn AggFunc, n int, name string) (*series.Series, error) {
	ta := col.Array().(*array.TypedArray[T])
	vals := ta.Values()
	validity := ta.Validity()
	hasNulls := validity != nil

	data := make([]float64, n)
	valid := make([]bool, n)

	switch fn {
	case AggSum, AggMean:
		counts := make([]int32, n)
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				data[gid] += float64(v)
				counts[gid]++
			}
		}
		if fn == AggMean {
			for i, c := range counts {
				if c > 0 {
					data[i] /= float64(c)
					valid[i] = true
				}
			}
		} else {
			for i, c := range counts {
				if c > 0 {
					valid[i] = true
				}
			}
		}
	case AggMin:
		inited := make([]bool, n)
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				fv := float64(v)
				if !inited[gid] || fv < data[gid] {
					data[gid] = fv
					inited[gid] = true
				}
			}
		}
		copy(valid, inited)
	case AggMax:
		inited := make([]bool, n)
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				fv := float64(v)
				if !inited[gid] || fv > data[gid] {
					data[gid] = fv
					inited[gid] = true
				}
			}
		}
		copy(valid, inited)
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

func applyFirstLast(col *series.Series, groupIDs []int32, n int, name string, first bool) (*series.Series, error) {
	nRows := col.Len()
	switch col.DataType() {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		if first {
			seen := make([]bool, n)
			for i := 0; i < nRows; i++ {
				gid := int(groupIDs[i])
				if !seen[gid] && col.IsValid(i) {
					v, _ := col.GetInt64(i)
					data[gid] = v
					valid[gid] = true
					seen[gid] = true
				}
			}
		} else {
			for i := 0; i < nRows; i++ {
				gid := int(groupIDs[i])
				if col.IsValid(i) {
					v, _ := col.GetInt64(i)
					data[gid] = v
					valid[gid] = true
				}
			}
		}
		return series.NewInt64WithValidity(name, data, valid), nil

	case dtype.String:
		data := make([]string, n)
		valid := make([]bool, n)
		if first {
			seen := make([]bool, n)
			for i := 0; i < nRows; i++ {
				gid := int(groupIDs[i])
				if !seen[gid] && col.IsValid(i) {
					v, _ := col.GetString(i)
					data[gid] = v
					valid[gid] = true
					seen[gid] = true
				}
			}
		} else {
			for i := 0; i < nRows; i++ {
				gid := int(groupIDs[i])
				if col.IsValid(i) {
					v, _ := col.GetString(i)
					data[gid] = v
					valid[gid] = true
				}
			}
		}
		return series.NewStringWithValidity(name, data, valid), nil

	default:
		return series.NewFloat64(name, make([]float64, n)), nil
	}
}

// GroupByExpr is an interface for expression-based GroupBy aggregation.
type GroupByExpr interface {
	EvaluateGroupBy(df *DataFrame) (*series.Series, error)
}

// buildIndexLists reconstructs per-group row index lists from the flat groupIDs
// array. Used by AggExprs which needs sub-DataFrames per group.
func (g *GroupByResult) buildIndexLists() [][]int {
	sizes := make([]int, g.nGroups)
	for _, gid := range g.groupIDs {
		sizes[gid]++
	}
	lists := make([][]int, g.nGroups)
	for i, sz := range sizes {
		lists[i] = make([]int, 0, sz)
	}
	for i, gid := range g.groupIDs {
		lists[gid] = append(lists[gid], i)
	}
	return lists
}

// AggExprs applies expression-based aggregations to each group.
func (g *GroupByResult) AggExprs(exprs ...GroupByExpr) (*DataFrame, error) {
	nGroups := g.nGroups
	indexLists := g.buildIndexLists()

	keyCols := make([]*series.Series, len(g.keys))
	for i, key := range g.keys {
		keyCol, _ := g.df.Column(key)
		keyCols[i] = buildGroupKeyColumn(key, keyCol.DataType(), g.groupKeys, i, nGroups)
	}

	aggCols := make([]*series.Series, len(exprs))
	for ei, gbe := range exprs {
		results := make([]*series.Series, nGroups)
		for gi := 0; gi < nGroups; gi++ {
			indices := indexLists[gi]
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

		colName := ""
		if nGroups > 0 && results[0] != nil {
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
