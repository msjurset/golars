package array

import (
	"math"

	"github.com/msjurset/golars/internal/pool"
)

// sumPartial holds the partial result of a sum computation over a chunk.
type sumPartial[T Numeric] struct {
	total T
	found bool
}

// Sum computes the sum of all valid (non-null) elements in a. The second
// return value is false when every element is null, indicating the result
// is not meaningful.
func Sum[T Numeric](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	vals := a.Values()
	bm := a.Validity()

	parts := pool.ParallelCollect(n, pool.DefaultThreshold, func(start, end int) sumPartial[T] {
		var p sumPartial[T]
		if bm == nil {
			for i := start; i < end; i++ {
				p.total += vals[i]
			}
			p.found = start < end
		} else {
			for i := start; i < end; i++ {
				if bm.IsSet(i) {
					p.total += vals[i]
					p.found = true
				}
			}
		}
		return p
	})

	var total T
	found := false
	for _, p := range parts {
		if p.found {
			total += p.total
			found = true
		}
	}
	return total, found
}

// meanPartial holds the partial result of a mean computation over a chunk.
type meanPartial struct {
	sum   float64
	count int
}

// Mean computes the arithmetic mean of all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Mean[T Numeric](a *TypedArray[T]) (float64, bool) {
	n := a.Len()
	vals := a.Values()
	bm := a.Validity()

	parts := pool.ParallelCollect(n, pool.DefaultThreshold, func(start, end int) meanPartial {
		var p meanPartial
		if bm == nil {
			for i := start; i < end; i++ {
				p.sum += float64(vals[i])
			}
			p.count = end - start
		} else {
			for i := start; i < end; i++ {
				if bm.IsSet(i) {
					p.sum += float64(vals[i])
					p.count++
				}
			}
		}
		return p
	})

	var total float64
	count := 0
	for _, p := range parts {
		total += p.sum
		count += p.count
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

// minPartial holds the partial result of a min computation over a chunk.
type minPartial[T Ordered] struct {
	min   T
	found bool
}

// Min returns the minimum value among all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Min[T Ordered](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	vals := a.Values()
	bm := a.Validity()

	parts := pool.ParallelCollect(n, pool.DefaultThreshold, func(start, end int) minPartial[T] {
		var p minPartial[T]
		if bm == nil {
			if start < end {
				p.min = vals[start]
				p.found = true
				for i := start + 1; i < end; i++ {
					if vals[i] < p.min {
						p.min = vals[i]
					}
				}
			}
		} else {
			for i := start; i < end; i++ {
				if bm.IsSet(i) {
					v := vals[i]
					if !p.found || v < p.min {
						p.min = v
						p.found = true
					}
				}
			}
		}
		return p
	})

	var min T
	found := false
	for _, p := range parts {
		if p.found {
			if !found || p.min < min {
				min = p.min
				found = true
			}
		}
	}
	return min, found
}

// maxPartial holds the partial result of a max computation over a chunk.
type maxPartial[T Ordered] struct {
	max   T
	found bool
}

// Max returns the maximum value among all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Max[T Ordered](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	vals := a.Values()
	bm := a.Validity()

	parts := pool.ParallelCollect(n, pool.DefaultThreshold, func(start, end int) maxPartial[T] {
		var p maxPartial[T]
		if bm == nil {
			if start < end {
				p.max = vals[start]
				p.found = true
				for i := start + 1; i < end; i++ {
					if vals[i] > p.max {
						p.max = vals[i]
					}
				}
			}
		} else {
			for i := start; i < end; i++ {
				if bm.IsSet(i) {
					v := vals[i]
					if !p.found || v > p.max {
						p.max = v
						p.found = true
					}
				}
			}
		}
		return p
	})

	var max T
	found := false
	for _, p := range parts {
		if p.found {
			if !found || p.max > max {
				max = p.max
				found = true
			}
		}
	}
	return max, found
}

// varPartial holds the partial result of a variance computation over a chunk.
type varPartial struct {
	sum   float64
	sumSq float64
	count int
}

// Variance computes the variance of all valid (non-null) elements in a.
// The ddof parameter controls the degrees-of-freedom correction: use 0 for
// population variance and 1 for sample variance. The second return value is
// false when there are insufficient valid elements for the given ddof.
func Variance[T Numeric](a *TypedArray[T], ddof int) (float64, bool) {
	n := a.Len()
	vals := a.Values()
	bm := a.Validity()

	parts := pool.ParallelCollect(n, pool.DefaultThreshold, func(start, end int) varPartial {
		var p varPartial
		if bm == nil {
			for i := start; i < end; i++ {
				v := float64(vals[i])
				p.sum += v
				p.sumSq += v * v
			}
			p.count = end - start
		} else {
			for i := start; i < end; i++ {
				if bm.IsSet(i) {
					v := float64(vals[i])
					p.sum += v
					p.sumSq += v * v
					p.count++
				}
			}
		}
		return p
	})

	var sum, sumSq float64
	count := 0
	for _, p := range parts {
		sum += p.sum
		sumSq += p.sumSq
		count += p.count
	}
	if count <= ddof {
		return 0, false
	}
	mean := sum / float64(count)
	variance := (sumSq/float64(count) - mean*mean) * float64(count) / float64(count-ddof)
	return variance, true
}

// Std computes the standard deviation of all valid (non-null) elements in a.
// The ddof parameter controls the degrees-of-freedom correction: use 0 for
// population standard deviation and 1 for sample standard deviation. The
// second return value is false when there are insufficient valid elements.
func Std[T Numeric](a *TypedArray[T], ddof int) (float64, bool) {
	v, ok := Variance(a, ddof)
	if !ok {
		return 0, false
	}
	return math.Sqrt(v), true
}
