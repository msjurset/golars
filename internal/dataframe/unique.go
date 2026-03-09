package dataframe

import (
	"fmt"
	"strings"

	"github.com/msjurseth/golars/internal/series"
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

	seen := make(map[string]struct{}, df.height)
	indices := make([]int, 0, df.height)

	for i := 0; i < df.height; i++ {
		key := rowKey(keyCols, i)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			indices = append(indices, i)
		}
	}

	if len(indices) == df.height {
		return df.Clone(), nil
	}
	return df.take(indices), nil
}

// rowKey builds a string key for the row at index i across the given columns.
func rowKey(cols []*series.Series, i int) string {
	if len(cols) == 1 {
		return singleColKey(cols[0], i)
	}
	parts := make([]string, len(cols))
	for j, c := range cols {
		parts[j] = singleColKey(c, i)
	}
	return strings.Join(parts, "\x00")
}

// singleColKey returns a string representation of a single cell for hashing.
func singleColKey(s *series.Series, i int) string {
	if s.IsNull(i) {
		return "\x01null"
	}
	return formatValue(s, i)
}
