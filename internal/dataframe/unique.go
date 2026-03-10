package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/series"
)

// Unique returns a new DataFrame containing only unique rows based on the
// given subset of column names. If no subset is provided, all columns are used.
// The first occurrence of each unique combination is kept.
func (df *DataFrame) Unique(subset ...string) (*DataFrame, error) {
	if df.height == 0 {
		return df.Clone(), nil
	}

	var keyCols []*series.Series
	if len(subset) == 0 {
		keyCols = df.columns
	} else {
		keyCols = make([]*series.Series, len(subset))
		for i, name := range subset {
			idx := df.columnIndex(name)
			if idx < 0 {
				return nil, fmt.Errorf("golars: column %q not found", name)
			}
			keyCols[i] = df.columns[idx]
		}
	}

	hashers := make([]colHasher, len(keyCols))
	for i, c := range keyCols {
		hashers[i] = newColHasher(c)
	}

	ht := newGroupHashTable(df.height / 2) // estimate ~50% unique
	indices := make([]int, 0, df.height)

	for i := 0; i < df.height; i++ {
		h := hashRowFast(hashers, i)
		gid := ht.probe(hashers, i, h)
		if int(gid) == len(indices) {
			indices = append(indices, i)
		}
	}

	if len(indices) == df.height {
		return df.Clone(), nil
	}
	return df.take(indices), nil
}
