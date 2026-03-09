package series

import (
	"github.com/msjurset/golars/internal/dtype"
)

// Rank assigns ranks to the values in the series.
// Supported methods: "average", "min", "max", "dense", "ordinal".
// Null values receive null in the output.
func (s *Series) Rank(method string) *Series {
	n := s.Len()
	if n == 0 {
		return NewFloat64(s.name, nil)
	}

	// Get sort indices (ascending, nulls at end)
	indices := s.ArgSort(false)

	// Find where nulls start
	nullStart := n
	for i, idx := range indices {
		if s.IsNull(idx) {
			nullStart = i
			break
		}
	}

	// Assign ranks
	ranks := make([]float64, n)
	valid := make([]bool, n)

	if nullStart == 0 {
		// All nulls
		return NewFloat64WithValidity(s.name, ranks, valid)
	}

	// Group equal values
	i := 0
	denseRank := 0
	for i < nullStart {
		j := i + 1
		for j < nullStart && valuesEqual(s, indices[i], indices[j]) {
			j++
		}
		// Positions i..j-1 have equal values
		denseRank++

		switch method {
		case "average":
			avg := float64(i+1+j) / 2.0
			for k := i; k < j; k++ {
				ranks[indices[k]] = avg
				valid[indices[k]] = true
			}
		case "min":
			for k := i; k < j; k++ {
				ranks[indices[k]] = float64(i + 1)
				valid[indices[k]] = true
			}
		case "max":
			for k := i; k < j; k++ {
				ranks[indices[k]] = float64(j)
				valid[indices[k]] = true
			}
		case "dense":
			for k := i; k < j; k++ {
				ranks[indices[k]] = float64(denseRank)
				valid[indices[k]] = true
			}
		case "ordinal":
			for k := i; k < j; k++ {
				ranks[indices[k]] = float64(k + 1)
				valid[indices[k]] = true
			}
		default:
			return NewFloat64(s.name, ranks) // fallback
		}

		i = j
	}

	if s.HasNulls() {
		return NewFloat64WithValidity(s.name, ranks, valid)
	}
	return NewFloat64(s.name, ranks)
}

// valuesEqual checks if two values in a series are equal.
func valuesEqual(s *Series, i, j int) bool {
	switch s.DataType() {
	case dtype.Int64:
		a, _ := s.GetInt64(i)
		b, _ := s.GetInt64(j)
		return a == b
	case dtype.Float64:
		a, _ := s.GetFloat64(i)
		b, _ := s.GetFloat64(j)
		return a == b
	case dtype.String:
		a, _ := s.GetString(i)
		b, _ := s.GetString(j)
		return a == b
	default:
		return false
	}
}
