package expr

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unsafe"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// FNV-1a constants for 64-bit hashing.
const (
	wFnvOffset64 = uint64(14695981039346656037)
	wFnvPrime64  = uint64(1099511628211)
)

// windowExpr evaluates an expression per partition and broadcasts the result
// back to the original row order, implementing SQL-style window functions.
type windowExpr struct {
	exprBase
	inner       Expr
	partitionBy []string
}

func (w *windowExpr) Evaluate(ctx *Context) (*series.Series, error) {
	if ctx == nil || ctx.DF == nil {
		return nil, fmt.Errorf("golars: window expression requires a DataFrame context")
	}

	df := ctx.DF
	n := df.Height()

	if len(w.partitionBy) == 0 {
		return nil, fmt.Errorf("golars: window Over() requires at least one partition column")
	}

	// Get partition key columns.
	keyCols := make([]*series.Series, len(w.partitionBy))
	for i, name := range w.partitionBy {
		col, err := df.Column(name)
		if err != nil {
			return nil, fmt.Errorf("golars: window Over: column %q not found", name)
		}
		keyCols[i] = col
	}

	// Build groupIDs using open-addressing hash table with FNV-1a hashing.
	groupIDs, nGroups := windowBuildGroups(keyCols, n)

	// Try fast path: detect aggExpr(colExpr) pattern for Sum/Mean/Min/Max.
	if agg, ok := w.inner.(*aggExpr); ok {
		if col, ok2 := agg.inner.(*colExpr); ok2 {
			switch agg.op {
			case aggSum, aggMean, aggMin, aggMax:
				srcCol, err := df.Column(col.name)
				if err == nil {
					result, err := windowAggFastPath(srcCol, groupIDs, nGroups, n, agg.op)
					if err == nil {
						return result, nil
					}
				}
			}
		}
	}

	// Slow path: build sub-DataFrames per group.
	return windowSlowPath(w, df, groupIDs, nGroups, n)
}

func (w *windowExpr) String() string {
	return fmt.Sprintf("%s.over(%s)", w.inner.String(), strings.Join(w.partitionBy, ", "))
}

// ---------- Open-addressing hash table for window grouping ----------

// whtEmpty is the sentinel for an empty slot.
const whtEmpty = int32(-1)

type whtSlot struct {
	hash     uint64
	firstRow int32
	gid      int32
}

type windowHashTable struct {
	slots   []whtSlot
	mask    uint64
	nGroups int32
}

func newWindowHashTable(est int) *windowHashTable {
	size := uint64(16)
	for size < uint64(est)*2 {
		size <<= 1
	}
	slots := make([]whtSlot, size)
	for i := range slots {
		slots[i].firstRow = whtEmpty
	}
	return &windowHashTable{slots: slots, mask: size - 1}
}

// ---------- Typed column hashers for window (avoids importing dataframe) ----------

type wColHasher struct {
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

func newWColHasher(s *series.Series) wColHasher {
	ch := wColHasher{series: s, dt: s.DataType()}
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

func wHashCombine(hash, value uint64) uint64 {
	b := (*[8]byte)(unsafe.Pointer(&value))
	hash ^= uint64(b[0])
	hash *= wFnvPrime64
	hash ^= uint64(b[1])
	hash *= wFnvPrime64
	hash ^= uint64(b[2])
	hash *= wFnvPrime64
	hash ^= uint64(b[3])
	hash *= wFnvPrime64
	hash ^= uint64(b[4])
	hash *= wFnvPrime64
	hash ^= uint64(b[5])
	hash *= wFnvPrime64
	hash ^= uint64(b[6])
	hash *= wFnvPrime64
	hash ^= uint64(b[7])
	hash *= wFnvPrime64
	return hash
}

func wHashStringBytes(hash uint64, data []byte) uint64 {
	for _, b := range data {
		hash ^= uint64(b)
		hash *= wFnvPrime64
	}
	return hash
}

func (ch *wColHasher) hashValue(hash uint64, i int) uint64 {
	if ch.series.IsNull(i) {
		hash ^= 0
		hash *= wFnvPrime64
		return hash
	}
	hash ^= 1
	hash *= wFnvPrime64

	switch ch.dt {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		if ch.int64s != nil {
			return wHashCombine(hash, uint64(ch.int64s.Value(i)))
		}
	case dtype.Float64:
		if ch.float64s != nil {
			return wHashCombine(hash, math.Float64bits(ch.float64s.Value(i)))
		}
	case dtype.Int32, dtype.Date:
		if ch.int32s != nil {
			return wHashCombine(hash, uint64(ch.int32s.Value(i)))
		}
	case dtype.String:
		if ch.strings != nil {
			return wHashStringBytes(hash, ch.strings.ValueBytes(i))
		}
	case dtype.Boolean:
		if ch.booleans != nil {
			if ch.booleans.Value(i) {
				hash ^= 1
			} else {
				hash ^= 2
			}
			hash *= wFnvPrime64
			return hash
		}
	case dtype.UInt64:
		if ch.uint64s != nil {
			return wHashCombine(hash, ch.uint64s.Value(i))
		}
	case dtype.UInt32:
		if ch.uint32s != nil {
			return wHashCombine(hash, uint64(ch.uint32s.Value(i)))
		}
	case dtype.Int16:
		if ch.int16s != nil {
			return wHashCombine(hash, uint64(ch.int16s.Value(i)))
		}
	case dtype.Int8:
		if ch.int8s != nil {
			return wHashCombine(hash, uint64(ch.int8s.Value(i)))
		}
	case dtype.UInt16:
		if ch.uint16s != nil {
			return wHashCombine(hash, uint64(ch.uint16s.Value(i)))
		}
	case dtype.UInt8:
		if ch.uint8s != nil {
			return wHashCombine(hash, uint64(ch.uint8s.Value(i)))
		}
	case dtype.Float32:
		if ch.float32s != nil {
			return wHashCombine(hash, uint64(math.Float32bits(ch.float32s.Value(i))))
		}
	}
	return hash
}

func wRowsEqual(hashers []wColHasher, i, j int) bool {
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

func wHashRow(hashers []wColHasher, i int) uint64 {
	h := wFnvOffset64
	for k := range hashers {
		h = hashers[k].hashValue(h, i)
	}
	return h
}

// probe returns the group ID for the row, inserting a new group if needed.
func (ht *windowHashTable) probe(hashers []wColHasher, rowIdx int, h uint64) int32 {
	slot := h & ht.mask
	for {
		s := &ht.slots[slot]
		if s.firstRow == whtEmpty {
			gid := ht.nGroups
			ht.nGroups++
			s.hash = h
			s.firstRow = int32(rowIdx)
			s.gid = gid
			if int(ht.nGroups)*2 >= len(ht.slots) {
				ht.grow(hashers)
				return ht.findGID(hashers, rowIdx, h)
			}
			return gid
		}
		if s.hash == h && wRowsEqual(hashers, rowIdx, int(s.firstRow)) {
			return s.gid
		}
		slot = (slot + 1) & ht.mask
	}
}

func (ht *windowHashTable) findGID(hashers []wColHasher, rowIdx int, h uint64) int32 {
	slot := h & ht.mask
	for {
		s := &ht.slots[slot]
		if s.firstRow != whtEmpty && s.hash == h && wRowsEqual(hashers, rowIdx, int(s.firstRow)) {
			return s.gid
		}
		slot = (slot + 1) & ht.mask
	}
}

func (ht *windowHashTable) grow(hashers []wColHasher) {
	newSize := uint64(len(ht.slots)) * 2
	newSlots := make([]whtSlot, newSize)
	for i := range newSlots {
		newSlots[i].firstRow = whtEmpty
	}
	newMask := newSize - 1
	for _, s := range ht.slots {
		if s.firstRow == whtEmpty {
			continue
		}
		slot := s.hash & newMask
		for newSlots[slot].firstRow != whtEmpty {
			slot = (slot + 1) & newMask
		}
		newSlots[slot] = s
	}
	ht.slots = newSlots
	ht.mask = newMask
}

// ---------- Group building ----------

// windowBuildGroups assigns a group ID to each row using FNV-1a hashing
// and an open-addressing hash table. Returns groupIDs slice and nGroups.
func windowBuildGroups(keyCols []*series.Series, n int) ([]int32, int) {
	hashers := make([]wColHasher, len(keyCols))
	for i, col := range keyCols {
		hashers[i] = newWColHasher(col)
	}

	est := n / 4
	if est < 16 {
		est = 16
	}
	ht := newWindowHashTable(est)
	groupIDs := make([]int32, n)

	for i := 0; i < n; i++ {
		h := wHashRow(hashers, i)
		groupIDs[i] = ht.probe(hashers, i, h)
	}

	return groupIDs, int(ht.nGroups)
}

// ---------- Fast-path aggregation (no sub-DataFrames) ----------

// windowAggFastPath computes Sum/Mean/Min/Max per group directly from the
// source column and broadcasts results back to every row via groupIDs.
func windowAggFastPath(srcCol *series.Series, groupIDs []int32, nGroups, n int, op aggOp) (*series.Series, error) {
	switch srcCol.DataType() {
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		return windowAggTyped[int64](srcCol, groupIDs, nGroups, n, op)
	case dtype.Float64:
		return windowAggTyped[float64](srcCol, groupIDs, nGroups, n, op)
	case dtype.Int32, dtype.Date:
		return windowAggTyped[int32](srcCol, groupIDs, nGroups, n, op)
	case dtype.Float32:
		return windowAggTyped[float32](srcCol, groupIDs, nGroups, n, op)
	case dtype.Int16:
		return windowAggTyped[int16](srcCol, groupIDs, nGroups, n, op)
	case dtype.Int8:
		return windowAggTyped[int8](srcCol, groupIDs, nGroups, n, op)
	case dtype.UInt64:
		return windowAggTyped[uint64](srcCol, groupIDs, nGroups, n, op)
	case dtype.UInt32:
		return windowAggTyped[uint32](srcCol, groupIDs, nGroups, n, op)
	case dtype.UInt16:
		return windowAggTyped[uint16](srcCol, groupIDs, nGroups, n, op)
	case dtype.UInt8:
		return windowAggTyped[uint8](srcCol, groupIDs, nGroups, n, op)
	default:
		return nil, fmt.Errorf("golars: window agg fast path: unsupported type %s", srcCol.DataType())
	}
}

func windowAggTyped[T array.Numeric](srcCol *series.Series, groupIDs []int32, nGroups, n int, op aggOp) (*series.Series, error) {
	ta := srcCol.Array().(*array.TypedArray[T])
	vals := ta.Values()
	validity := ta.Validity()
	hasNulls := validity != nil
	name := srcCol.Name()

	// Compute per-group aggregation values.
	groupVals := make([]float64, nGroups)
	groupCounts := make([]int32, nGroups)

	switch op {
	case aggSum, aggMean:
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				groupVals[gid] += float64(v)
				groupCounts[gid]++
			}
		}
		if op == aggMean {
			for g := 0; g < nGroups; g++ {
				if groupCounts[g] > 0 {
					groupVals[g] /= float64(groupCounts[g])
				}
			}
		}
	case aggMin:
		inited := make([]bool, nGroups)
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				fv := float64(v)
				if !inited[gid] || fv < groupVals[gid] {
					groupVals[gid] = fv
					inited[gid] = true
				}
				groupCounts[gid]++
			}
		}
	case aggMax:
		inited := make([]bool, nGroups)
		for i, v := range vals {
			if !hasNulls || validity.IsSet(i) {
				gid := groupIDs[i]
				fv := float64(v)
				if !inited[gid] || fv > groupVals[gid] {
					groupVals[gid] = fv
					inited[gid] = true
				}
				groupCounts[gid]++
			}
		}
	}

	// Broadcast: scatter group result to every row.
	result := make([]float64, n)
	resultValid := make([]bool, n)
	hasNullResult := false

	for i := 0; i < n; i++ {
		gid := groupIDs[i]
		if groupCounts[gid] > 0 {
			result[i] = groupVals[gid]
			resultValid[i] = true
		} else {
			hasNullResult = true
		}
	}

	if hasNullResult {
		return series.NewFloat64WithValidity(name, result, resultValid), nil
	}
	return series.NewFloat64(name, result), nil
}

// ---------- Slow path (sub-DataFrame per group) ----------

func windowSlowPath(w *windowExpr, df *dataframe.DataFrame, groupIDs []int32, nGroups, n int) (*series.Series, error) {
	// Build per-group index lists.
	sizes := make([]int, nGroups)
	for _, gid := range groupIDs {
		sizes[gid]++
	}
	groupIndices := make([][]int, nGroups)
	for g := 0; g < nGroups; g++ {
		groupIndices[g] = make([]int, 0, sizes[g])
	}
	for i, gid := range groupIDs {
		groupIndices[gid] = append(groupIndices[gid], i)
	}

	var resultName string
	var resultDtype dtype.DataType
	resultFloat64 := make([]float64, n)
	resultInt64 := make([]int64, n)
	resultString := make([]string, n)
	resultBool := make([]bool, n)
	resultValid := make([]bool, n)
	for i := range resultValid {
		resultValid[i] = true
	}

	first := true
	for g := 0; g < nGroups; g++ {
		indices := groupIndices[g]

		subDF, err := buildSubDataFrame(df, indices)
		if err != nil {
			return nil, fmt.Errorf("golars: window Over: %w", err)
		}

		subCtx := &Context{DF: subDF}
		result, err := w.inner.Evaluate(subCtx)
		if err != nil {
			return nil, err
		}

		if first {
			resultName = result.Name()
			resultDtype = result.DataType()
			first = false
		}

		isAgg := result.Len() == 1
		groupSize := len(indices)

		if !isAgg && result.Len() != groupSize {
			return nil, fmt.Errorf("golars: window Over: inner expression returned %d rows for group of %d rows", result.Len(), groupSize)
		}

		for j, origIdx := range indices {
			srcIdx := j
			if isAgg {
				srcIdx = 0
			}

			if result.IsNull(srcIdx) {
				resultValid[origIdx] = false
				continue
			}

			switch resultDtype {
			case dtype.Float64:
				v, _ := result.GetFloat64(srcIdx)
				resultFloat64[origIdx] = v
			case dtype.Int64:
				v, _ := result.GetInt64(srcIdx)
				resultInt64[origIdx] = v
			case dtype.String:
				v, _ := result.GetString(srcIdx)
				resultString[origIdx] = v
			case dtype.Boolean:
				v, _ := result.GetBool(srcIdx)
				resultBool[origIdx] = v
			default:
				return nil, fmt.Errorf("golars: window Over: unsupported result type %s", resultDtype)
			}
		}
	}

	if first {
		return series.NewFloat64(resultName, nil), nil
	}

	hasNulls := false
	for _, v := range resultValid {
		if !v {
			hasNulls = true
			break
		}
	}

	switch resultDtype {
	case dtype.Float64:
		if hasNulls {
			return series.NewFloat64WithValidity(resultName, resultFloat64, resultValid), nil
		}
		return series.NewFloat64(resultName, resultFloat64), nil
	case dtype.Int64:
		if hasNulls {
			return series.NewInt64WithValidity(resultName, resultInt64, resultValid), nil
		}
		return series.NewInt64(resultName, resultInt64), nil
	case dtype.String:
		if hasNulls {
			return series.NewStringWithValidity(resultName, resultString, resultValid), nil
		}
		return series.NewString(resultName, resultString), nil
	case dtype.Boolean:
		if hasNulls {
			return series.NewBooleanWithValidity(resultName, resultBool, resultValid), nil
		}
		return series.NewBoolean(resultName, resultBool), nil
	default:
		return nil, fmt.Errorf("golars: window Over: unsupported result type %s", resultDtype)
	}
}

// buildSubDataFrame creates a new DataFrame from selected row indices.
func buildSubDataFrame(df *dataframe.DataFrame, indices []int) (*dataframe.DataFrame, error) {
	cols := df.Columns()
	subCols := make([]*series.Series, len(cols))
	for i, col := range cols {
		subCols[i] = col.Take(indices)
	}
	return dataframe.New(subCols...)
}
