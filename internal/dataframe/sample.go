package dataframe

import (
	"fmt"
	"math"
	"math/rand"
)

// Sample returns a new DataFrame with n randomly selected rows. The seed
// parameter controls the random number generator for reproducibility.
// Returns an error if n is negative or exceeds the DataFrame height.
func (df *DataFrame) Sample(n int, seed int64) (*DataFrame, error) {
	if n < 0 {
		return nil, fmt.Errorf("golars: sample size must be non-negative, got %d", n)
	}
	if n > df.height {
		return nil, fmt.Errorf("golars: sample size %d exceeds DataFrame height %d", n, df.height)
	}
	if n == 0 {
		return df.Head(0), nil
	}
	if n == df.height {
		return df.Clone(), nil
	}

	rng := rand.New(rand.NewSource(seed))
	// Fisher-Yates partial shuffle to pick n indices.
	indices := make([]int, df.height)
	for i := range indices {
		indices[i] = i
	}
	for i := 0; i < n; i++ {
		j := i + rng.Intn(df.height-i)
		indices[i], indices[j] = indices[j], indices[i]
	}
	return df.take(indices[:n]), nil
}

// SampleFraction returns a new DataFrame with a random fraction of rows.
// The fraction must be in the range [0, 1]. The seed controls the RNG.
func (df *DataFrame) SampleFraction(fraction float64, seed int64) (*DataFrame, error) {
	if fraction < 0 || fraction > 1 {
		return nil, fmt.Errorf("golars: sample fraction must be in [0, 1], got %f", fraction)
	}
	n := int(math.Round(fraction * float64(df.height)))
	return df.Sample(n, seed)
}
