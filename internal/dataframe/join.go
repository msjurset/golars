package dataframe

import (
	"fmt"
	"sync"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/pool"
	"github.com/msjurset/golars/internal/series"
)

// JoinType represents the type of join operation.
type JoinType int

// Join type constants.
const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
	SemiJoin
	AntiJoin
	CrossJoin
)

// joinHashEntry stores indices with the same hash in the right table.
// The representative row is used for collision checking.
type joinHashEntry struct {
	indices []int
}

// Join performs a join between this DataFrame and another on the given key columns.
func (df *DataFrame) Join(other *DataFrame, on []string, how JoinType) (*DataFrame, error) {
	return df.JoinOn(other, on, on, how)
}

// JoinOn performs a join with potentially different key column names in left and right.
func (df *DataFrame) JoinOn(other *DataFrame, leftOn, rightOn []string, how JoinType) (*DataFrame, error) {
	if len(leftOn) != len(rightOn) {
		return nil, fmt.Errorf("golars: join: left_on and right_on must have the same length")
	}
	if len(leftOn) == 0 && how != CrossJoin {
		return nil, fmt.Errorf("golars: join: must specify at least one join key")
	}

	// Validate columns exist
	for _, name := range leftOn {
		if !df.schema.Contains(name) {
			return nil, fmt.Errorf("golars: join: left column %q not found", name)
		}
	}
	for _, name := range rightOn {
		if !other.schema.Contains(name) {
			return nil, fmt.Errorf("golars: join: right column %q not found", name)
		}
	}

	if how == CrossJoin {
		return crossJoin(df, other)
	}

	// Build column hashers for right side
	rightKeyCols := make([]*series.Series, len(rightOn))
	rightHashers := make([]colHasher, len(rightOn))
	for i, name := range rightOn {
		c, _ := other.Column(name)
		rightKeyCols[i] = c
		rightHashers[i] = newColHasher(c)
	}

	// Build hash table on right side using uint64 keys.
	// Each bucket stores indices that share a hash. We handle collisions
	// by linear probing, verifying actual value equality.
	estBuckets := other.height
	if estBuckets < 16 {
		estBuckets = 16
	}
	rightIndex := make(map[uint64]*joinHashEntry, estBuckets)

	for i := 0; i < other.height; i++ {
		h := hashRowFast(rightHashers, i)
		entry, exists := rightIndex[h]
		if exists {
			// Verify values match the representative row
			if rowsEqual(rightHashers, i, entry.indices[0]) {
				entry.indices = append(entry.indices, i)
				continue
			}
			// Collision: linear probe
			for probe := uint64(1); ; probe++ {
				ph := h + probe
				pe, pExists := rightIndex[ph]
				if !pExists {
					rightIndex[ph] = &joinHashEntry{indices: []int{i}}
					break
				}
				if rowsEqual(rightHashers, i, pe.indices[0]) {
					pe.indices = append(pe.indices, i)
					break
				}
			}
			continue
		}
		rightIndex[h] = &joinHashEntry{indices: []int{i}}
	}

	// Build column hashers for left side
	leftKeyCols := make([]*series.Series, len(leftOn))
	leftHashers := make([]colHasher, len(leftOn))
	for i, name := range leftOn {
		c, _ := df.Column(name)
		leftKeyCols[i] = c
		leftHashers[i] = newColHasher(c)
	}

	// lookupRight finds matching right-side indices for left row i.
	// It searches the hash table with linear probing and cross-table value comparison.
	lookupRight := func(lHashers []colHasher, rHashers []colHasher, leftRow int) ([]int, bool) {
		h := hashRowFast(lHashers, leftRow)
		for probe := uint64(0); ; probe++ {
			ph := h + probe
			entry, exists := rightIndex[ph]
			if !exists {
				return nil, false
			}
			// Compare left row values against the representative right row
			if crossRowsEqual(lHashers, leftRow, rHashers, entry.indices[0]) {
				return entry.indices, true
			}
			// Only continue probing if we might have collisions
			if probe > 100 {
				// Safety limit to avoid infinite loops
				return nil, false
			}
		}
	}

	// For joins that need to track right-side matches (Right, Full)
	needRightMatched := how == RightJoin || how == FullJoin

	// Determine if we can use parallel probe. We only parallelize Inner and Left
	// joins since they don't need rightMatched tracking.
	canParallel := !needRightMatched && (how == InnerJoin || how == LeftJoin) &&
		df.height >= pool.DefaultThreshold

	var leftIndices, rightIndices []int

	if canParallel {
		leftIndices, rightIndices = parallelJoinProbe(
			df.height, leftHashers, rightHashers, leftKeyCols, how, lookupRight,
		)
	} else {
		// Pre-allocate with estimate
		estResult := df.height
		leftIndices = make([]int, 0, estResult)
		rightIndices = make([]int, 0, estResult)

		var rightMatched map[int]bool
		if needRightMatched {
			rightMatched = make(map[int]bool, other.height/2)
		}

		for i := 0; i < df.height; i++ {
			matches, found := lookupRight(leftHashers, rightHashers, i)

			switch how {
			case InnerJoin:
				if found {
					for _, j := range matches {
						leftIndices = append(leftIndices, i)
						rightIndices = append(rightIndices, j)
					}
				}
			case LeftJoin:
				if found {
					for _, j := range matches {
						leftIndices = append(leftIndices, i)
						rightIndices = append(rightIndices, j)
					}
				} else {
					leftIndices = append(leftIndices, i)
					rightIndices = append(rightIndices, -1)
				}
			case RightJoin:
				if found {
					for _, j := range matches {
						leftIndices = append(leftIndices, i)
						rightIndices = append(rightIndices, j)
						rightMatched[j] = true
					}
				}
			case FullJoin:
				if found {
					for _, j := range matches {
						leftIndices = append(leftIndices, i)
						rightIndices = append(rightIndices, j)
						rightMatched[j] = true
					}
				} else {
					leftIndices = append(leftIndices, i)
					rightIndices = append(rightIndices, -1)
				}
			case SemiJoin:
				if found {
					leftIndices = append(leftIndices, i)
					rightIndices = append(rightIndices, -1)
				}
			case AntiJoin:
				if !found {
					leftIndices = append(leftIndices, i)
					rightIndices = append(rightIndices, -1)
				}
			}
		}

		// For right/full join, add unmatched right rows
		if needRightMatched {
			for j := 0; j < other.height; j++ {
				if !rightMatched[j] {
					leftIndices = append(leftIndices, -1)
					rightIndices = append(rightIndices, j)
				}
			}
		}
	}

	return buildJoinResult(df, other, leftOn, rightOn, leftIndices, rightIndices, how)
}

// crossRowsEqual compares values of left row i against right row j
// across corresponding key columns (which may have different names/types).
func crossRowsEqual(lHashers []colHasher, leftRow int, rHashers []colHasher, rightRow int) bool {
	for k := range lHashers {
		lh := &lHashers[k]
		rh := &rHashers[k]

		lNull := lh.series.IsNull(leftRow)
		rNull := rh.series.IsNull(rightRow)
		if lNull != rNull {
			return false
		}
		if lNull {
			continue
		}

		// Compare by the type of the left column (join keys should be compatible)
		switch lh.dt {
		case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
			var lv, rv int64
			if lh.int64s != nil {
				lv = lh.int64s.Value(leftRow)
			}
			if rh.int64s != nil {
				rv = rh.int64s.Value(rightRow)
			}
			if lv != rv {
				return false
			}
		case dtype.Float64:
			var lv, rv float64
			if lh.float64s != nil {
				lv = lh.float64s.Value(leftRow)
			}
			if rh.float64s != nil {
				rv = rh.float64s.Value(rightRow)
			}
			if lv != rv {
				return false
			}
		case dtype.Int32, dtype.Date:
			var lv, rv int32
			if lh.int32s != nil {
				lv = lh.int32s.Value(leftRow)
			}
			if rh.int32s != nil {
				rv = rh.int32s.Value(rightRow)
			}
			if lv != rv {
				return false
			}
		case dtype.String:
			var lv, rv string
			if lh.strings != nil {
				lv = lh.strings.Value(leftRow)
			}
			if rh.strings != nil {
				rv = rh.strings.Value(rightRow)
			}
			if lv != rv {
				return false
			}
		case dtype.Boolean:
			var lv, rv bool
			if lh.booleans != nil {
				lv = lh.booleans.Value(leftRow)
			}
			if rh.booleans != nil {
				rv = rh.booleans.Value(rightRow)
			}
			if lv != rv {
				return false
			}
		default:
			// For other integer types, fall back to int64 comparison via series
			lv, lok := lh.series.GetInt64(leftRow)
			rv, rok := rh.series.GetInt64(rightRow)
			if lok != rok || lv != rv {
				return false
			}
		}
	}
	return true
}

type joinChunkResult struct {
	leftIdx  []int
	rightIdx []int
}

// parallelJoinProbe probes the right hash table in parallel for Inner/Left joins.
func parallelJoinProbe(
	leftHeight int,
	leftHashers, rightHashers []colHasher,
	leftKeyCols []*series.Series,
	how JoinType,
	lookupRight func([]colHasher, []colHasher, int) ([]int, bool),
) ([]int, []int) {
	chunks := pool.ParallelCollect[joinChunkResult](leftHeight, pool.DefaultThreshold,
		func(start, end int) joinChunkResult {
			// Each chunk builds its own local hashers referencing the same
			// underlying arrays, so there's no data race on the immutable arrays.
			chunkLeft := make([]int, 0, (end-start)*2)
			chunkRight := make([]int, 0, (end-start)*2)

			for i := start; i < end; i++ {
				matches, found := lookupRight(leftHashers, rightHashers, i)

				switch how {
				case InnerJoin:
					if found {
						for _, j := range matches {
							chunkLeft = append(chunkLeft, i)
							chunkRight = append(chunkRight, j)
						}
					}
				case LeftJoin:
					if found {
						for _, j := range matches {
							chunkLeft = append(chunkLeft, i)
							chunkRight = append(chunkRight, j)
						}
					} else {
						chunkLeft = append(chunkLeft, i)
						chunkRight = append(chunkRight, -1)
					}
				}
			}
			return joinChunkResult{leftIdx: chunkLeft, rightIdx: chunkRight}
		},
	)

	// Merge chunk results
	totalLen := 0
	for _, c := range chunks {
		totalLen += len(c.leftIdx)
	}

	leftIndices := make([]int, 0, totalLen)
	rightIndices := make([]int, 0, totalLen)
	for _, c := range chunks {
		leftIndices = append(leftIndices, c.leftIdx...)
		rightIndices = append(rightIndices, c.rightIdx...)
	}
	return leftIndices, rightIndices
}

func buildJoinResult(left, right *DataFrame, leftOn, rightOn []string, leftIdx, rightIdx []int, how JoinType) (*DataFrame, error) {
	n := len(leftIdx)

	// For semi/anti joins, only return left columns
	if how == SemiJoin || how == AntiJoin {
		cols := make([]*series.Series, len(left.columns))
		for i, col := range left.columns {
			cols[i] = gatherColumn(col, leftIdx, n)
		}
		return New(cols...)
	}

	leftKeySet := make(map[string]int)  // left key name -> index in leftOn
	rightKeySet := make(map[string]int) // right key name -> index in rightOn
	for i, name := range leftOn {
		leftKeySet[name] = i
	}
	for i, name := range rightOn {
		rightKeySet[name] = i
	}

	needCoalesce := how == RightJoin || how == FullJoin

	// Build result columns in parallel when there are many rows
	totalCols := len(left.columns) + len(right.columns)
	cols := make([]*series.Series, 0, totalCols)

	if n >= pool.DefaultThreshold {
		// Gather all columns in parallel
		leftGathered := make([]*series.Series, len(left.columns))
		rightGathered := make([]*series.Series, len(right.columns))

		var wg sync.WaitGroup
		for i, col := range left.columns {
			wg.Add(1)
			go func(idx int, c *series.Series) {
				defer wg.Done()
				leftGathered[idx] = gatherColumn(c, leftIdx, n)
			}(i, col)
		}
		for i, col := range right.columns {
			wg.Add(1)
			go func(idx int, c *series.Series) {
				defer wg.Done()
				rightGathered[idx] = gatherColumn(c, rightIdx, n)
			}(i, col)
		}
		wg.Wait()

		// Apply coalescing and build result
		for i, col := range left.columns {
			gathered := leftGathered[i]
			if needCoalesce {
				if keyIdx, ok := leftKeySet[col.Name()]; ok {
					rIdx := 0
					for ri, rc := range right.columns {
						if rc.Name() == rightOn[keyIdx] {
							rIdx = ri
							break
						}
					}
					gathered = coalesceColumns(gathered, rightGathered[rIdx])
				}
			}
			cols = append(cols, gathered)
		}

		for i, col := range right.columns {
			if _, isKey := rightKeySet[col.Name()]; isKey {
				continue
			}
			name := col.Name()
			if left.schema.Contains(name) {
				name = name + "_right"
			}
			cols = append(cols, rightGathered[i].Rename(name))
		}
	} else {
		// Left columns
		for _, col := range left.columns {
			gathered := gatherColumn(col, leftIdx, n)
			if needCoalesce {
				if keyIdx, ok := leftKeySet[col.Name()]; ok {
					rightKeyCol, _ := right.Column(rightOn[keyIdx])
					rightGatheredCol := gatherColumn(rightKeyCol, rightIdx, n)
					gathered = coalesceColumns(gathered, rightGatheredCol)
				}
			}
			cols = append(cols, gathered)
		}

		// Right columns (excluding join keys to avoid duplication)
		for _, col := range right.columns {
			if _, isKey := rightKeySet[col.Name()]; isKey {
				continue
			}
			name := col.Name()
			if left.schema.Contains(name) {
				name = name + "_right"
			}
			gathered := gatherColumn(col, rightIdx, n)
			cols = append(cols, gathered.Rename(name))
		}
	}

	return New(cols...)
}

func gatherColumn(col *series.Series, indices []int, n int) *series.Series {
	switch col.DataType() {
	case dtype.Int64:
		return gatherInt64(col, indices, n)
	case dtype.Float64:
		return gatherFloat64(col, indices, n)
	case dtype.String:
		return gatherString(col, indices, n)
	case dtype.Boolean:
		return gatherBool(col, indices, n)
	default:
		return gatherString(col, indices, n)
	}
}

func gatherInt64(col *series.Series, indices []int, n int) *series.Series {
	data := make([]int64, n)
	valid := make([]bool, n)
	hasNulls := false
	for i, idx := range indices {
		if idx < 0 {
			hasNulls = true
			continue
		}
		if col.IsNull(idx) {
			hasNulls = true
			continue
		}
		v, _ := col.GetInt64(idx)
		data[i] = v
		valid[i] = true
	}
	if hasNulls {
		return series.NewInt64WithValidity(col.Name(), data, valid)
	}
	return series.NewInt64(col.Name(), data)
}

func gatherFloat64(col *series.Series, indices []int, n int) *series.Series {
	data := make([]float64, n)
	valid := make([]bool, n)
	hasNulls := false
	for i, idx := range indices {
		if idx < 0 {
			hasNulls = true
			continue
		}
		if col.IsNull(idx) {
			hasNulls = true
			continue
		}
		v, _ := col.GetFloat64(idx)
		data[i] = v
		valid[i] = true
	}
	if hasNulls {
		return series.NewFloat64WithValidity(col.Name(), data, valid)
	}
	return series.NewFloat64(col.Name(), data)
}

func gatherString(col *series.Series, indices []int, n int) *series.Series {
	data := make([]string, n)
	valid := make([]bool, n)
	hasNulls := false
	for i, idx := range indices {
		if idx < 0 {
			hasNulls = true
			continue
		}
		if col.IsNull(idx) {
			hasNulls = true
			continue
		}
		v, _ := col.GetString(idx)
		data[i] = v
		valid[i] = true
	}
	if hasNulls {
		return series.NewStringWithValidity(col.Name(), data, valid)
	}
	return series.NewString(col.Name(), data)
}

func gatherBool(col *series.Series, indices []int, n int) *series.Series {
	data := make([]bool, n)
	valid := make([]bool, n)
	hasNulls := false
	for i, idx := range indices {
		if idx < 0 {
			hasNulls = true
			continue
		}
		if col.IsNull(idx) {
			hasNulls = true
			continue
		}
		v, _ := col.GetBool(idx)
		data[i] = v
		valid[i] = true
	}
	if hasNulls {
		return series.NewBooleanWithValidity(col.Name(), data, valid)
	}
	return series.NewBoolean(col.Name(), data)
}

// coalesceColumns returns a series where nulls in a are filled from b.
func coalesceColumns(a, b *series.Series) *series.Series {
	n := a.Len()
	switch a.DataType() {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.IsValid(i) {
				data[i], _ = a.GetInt64(i)
				valid[i] = true
			} else if b.IsValid(i) {
				data[i], _ = b.GetInt64(i)
				valid[i] = true
			}
		}
		return series.NewInt64WithValidity(a.Name(), data, valid)
	case dtype.Float64:
		data := make([]float64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.IsValid(i) {
				data[i], _ = a.GetFloat64(i)
				valid[i] = true
			} else if b.IsValid(i) {
				data[i], _ = b.GetFloat64(i)
				valid[i] = true
			}
		}
		return series.NewFloat64WithValidity(a.Name(), data, valid)
	case dtype.String:
		data := make([]string, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.IsValid(i) {
				data[i], _ = a.GetString(i)
				valid[i] = true
			} else if b.IsValid(i) {
				data[i], _ = b.GetString(i)
				valid[i] = true
			}
		}
		return series.NewStringWithValidity(a.Name(), data, valid)
	case dtype.Boolean:
		data := make([]bool, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.IsValid(i) {
				data[i], _ = a.GetBool(i)
				valid[i] = true
			} else if b.IsValid(i) {
				data[i], _ = b.GetBool(i)
				valid[i] = true
			}
		}
		return series.NewBooleanWithValidity(a.Name(), data, valid)
	default:
		return a
	}
}

func crossJoin(left, right *DataFrame) (*DataFrame, error) {
	n := left.height * right.height
	leftIdx := make([]int, 0, n)
	rightIdx := make([]int, 0, n)
	for i := 0; i < left.height; i++ {
		for j := 0; j < right.height; j++ {
			leftIdx = append(leftIdx, i)
			rightIdx = append(rightIdx, j)
		}
	}
	return buildJoinResult(left, right, nil, nil, leftIdx, rightIdx, CrossJoin)
}
