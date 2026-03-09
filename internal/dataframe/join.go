package dataframe

import (
	"fmt"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
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

	// Build hash table on right side
	rightKeyCols := make([]*series.Series, len(rightOn))
	for i, name := range rightOn {
		c, _ := other.Column(name)
		rightKeyCols[i] = c
	}

	rightIndex := make(map[string][]int)
	for i := 0; i < other.height; i++ {
		h := hashRow(rightKeyCols, i)
		rightIndex[h] = append(rightIndex[h], i)
	}

	leftKeyCols := make([]*series.Series, len(leftOn))
	for i, name := range leftOn {
		c, _ := df.Column(name)
		leftKeyCols[i] = c
	}

	// Compute matching pairs
	var leftIndices, rightIndices []int
	rightMatched := make(map[int]bool)

	for i := 0; i < df.height; i++ {
		h := hashRow(leftKeyCols, i)
		matches, found := rightIndex[h]

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
				rightIndices = append(rightIndices, -1) // null
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
				rightIndices = append(rightIndices, -1) // not used
			}
		case AntiJoin:
			if !found {
				leftIndices = append(leftIndices, i)
				rightIndices = append(rightIndices, -1)
			}
		}
	}

	// For right/full join, add unmatched right rows
	if how == RightJoin || how == FullJoin {
		for j := 0; j < other.height; j++ {
			if !rightMatched[j] {
				leftIndices = append(leftIndices, -1) // null
				rightIndices = append(rightIndices, j)
			}
		}
	}

	// Build result columns
	return buildJoinResult(df, other, leftOn, rightOn, leftIndices, rightIndices, how)
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

	var cols []*series.Series

	// Left columns
	for _, col := range left.columns {
		gathered := gatherColumn(col, leftIdx, n)
		// For join key columns in right/full join, coalesce with right side
		if needCoalesce {
			if keyIdx, ok := leftKeySet[col.Name()]; ok {
				rightKeyCol, _ := right.Column(rightOn[keyIdx])
				rightGathered := gatherColumn(rightKeyCol, rightIdx, n)
				gathered = coalesceColumns(gathered, rightGathered)
			}
		}
		cols = append(cols, gathered)
	}

	// Right columns (excluding join keys to avoid duplication)
	for _, col := range right.columns {
		if _, isKey := rightKeySet[col.Name()]; isKey {
			continue
		}
		// Suffix right columns that conflict with left names
		name := col.Name()
		if left.schema.Contains(name) {
			name = name + "_right"
		}
		gathered := gatherColumn(col, rightIdx, n)
		cols = append(cols, gathered.Rename(name))
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
	var leftIdx, rightIdx []int
	for i := 0; i < left.height; i++ {
		for j := 0; j < right.height; j++ {
			leftIdx = append(leftIdx, i)
			rightIdx = append(rightIdx, j)
		}
	}
	return buildJoinResult(left, right, nil, nil, leftIdx, rightIdx, CrossJoin)
}
