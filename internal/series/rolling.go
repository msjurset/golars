package series

import (
	"math"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/pool"
)

// RollingMean computes the rolling mean with the given window size.
// Returns a Float64 Series. The first (window-1) values are null.
func (s *Series) RollingMean(window int) *Series {
	return s.rollingAgg(window, rollMean)
}

// RollingSum computes the rolling sum with the given window size.
// Returns a Float64 Series. The first (window-1) values are null.
func (s *Series) RollingSum(window int) *Series {
	return s.rollingAgg(window, rollSum)
}

// RollingMin computes the rolling minimum with the given window size.
// Returns a Float64 Series. The first (window-1) values are null.
func (s *Series) RollingMin(window int) *Series {
	return s.rollingAgg(window, rollMin)
}

// RollingMax computes the rolling maximum with the given window size.
// Returns a Float64 Series. The first (window-1) values are null.
func (s *Series) RollingMax(window int) *Series {
	return s.rollingAgg(window, rollMax)
}

// RollingStd computes the rolling standard deviation with the given window size.
// Returns a Float64 Series. The first (window-1) values are null.
func (s *Series) RollingStd(window int) *Series {
	return s.rollingAgg(window, rollStd)
}

type rollOp int

const (
	rollSum rollOp = iota
	rollMean
	rollMin
	rollMax
	rollStd
)

func (s *Series) rollingAgg(window int, op rollOp) *Series {
	length := s.Len()
	if window <= 0 || length == 0 {
		return NewFloat64(s.name, make([]float64, length))
	}

	values := s.toFloat64Values()
	if values == nil {
		return nil
	}

	data := make([]float64, length)
	valid := make([]bool, length)

	// Build prefix null count for O(1) null-in-window queries.
	// nullPfx[i] = number of null values in values[0..i-1].
	hasNulls := s.HasNulls()
	var nullPfx []int
	if hasNulls {
		nullPfx = make([]int, length+1)
		for i := 0; i < length; i++ {
			nullPfx[i+1] = nullPfx[i]
			if s.IsNull(i) {
				nullPfx[i+1]++
			}
		}
	}

	// windowHasNull returns true if values[start..end] (inclusive) contains a null.
	windowHasNull := func(start, end int) bool {
		if !hasNulls {
			return false
		}
		return nullPfx[end+1]-nullPfx[start] > 0
	}

	switch op {
	case rollSum, rollMean, rollStd:
		s.rollingSumMeanStd(window, op, values, data, valid, windowHasNull)
	case rollMin:
		s.rollingMinMax(window, true, values, data, valid, windowHasNull)
	case rollMax:
		s.rollingMinMax(window, false, values, data, valid, windowHasNull)
	}

	return NewFloat64WithValidity(s.name, data, valid)
}

// rollingSumMeanStd uses prefix sums for O(1) per-position computation and
// supports parallel output via pool.ParallelDo.
func (s *Series) rollingSumMeanStd(window int, op rollOp, values, data []float64, valid []bool, windowHasNull func(int, int) bool) {
	length := len(values)
	w := float64(window)

	// Build prefix sums. pfxSum[i] = sum of values[0..i-1].
	pfxSum := make([]float64, length+1)
	for i := 0; i < length; i++ {
		pfxSum[i+1] = pfxSum[i] + values[i]
	}

	// For Std, also build prefix sum of squares.
	var pfxSumSq []float64
	if op == rollStd {
		pfxSumSq = make([]float64, length+1)
		for i := 0; i < length; i++ {
			pfxSumSq[i+1] = pfxSumSq[i] + values[i]*values[i]
		}
	}

	count := length - window + 1
	if count <= 0 {
		return
	}

	compute := func(start, end int) {
		for idx := start; idx < end; idx++ {
			i := idx + window - 1 // position in output
			lo := i - window + 1
			if windowHasNull(lo, i) {
				continue
			}
			valid[i] = true
			sum := pfxSum[i+1] - pfxSum[lo]
			switch op {
			case rollSum:
				data[i] = sum
			case rollMean:
				data[i] = sum / w
			case rollStd:
				if window > 1 {
					sumSq := pfxSumSq[i+1] - pfxSumSq[lo]
					mean := sum / w
					variance := (sumSq - w*mean*mean) / (w - 1)
					if variance < 0 {
						variance = 0 // guard against floating-point noise
					}
					data[i] = math.Sqrt(variance)
				}
			}
		}
	}

	pool.ParallelDo(count, pool.DefaultThreshold, compute)
}

// rollingMinMax uses a monotonic deque for O(n) total computation.
func (s *Series) rollingMinMax(window int, isMin bool, values, data []float64, valid []bool, windowHasNull func(int, int) bool) {
	length := len(values)
	if window > length {
		return
	}

	// Deque stores indices. For min, front is the index of the current minimum;
	// for max, front is the index of the current maximum.
	deque := make([]int, 0, window)

	less := func(a, b float64) bool { return a < b }
	if !isMin {
		less = func(a, b float64) bool { return a > b }
	}

	for i := 0; i < length; i++ {
		// Remove elements from back that are worse than current.
		for len(deque) > 0 && less(values[i], values[deque[len(deque)-1]]) {
			deque = deque[:len(deque)-1]
		}
		deque = append(deque, i)

		// Remove front if it's outside the window.
		if deque[0] <= i-window {
			deque = deque[1:]
		}

		// Output from position window-1 onward.
		if i >= window-1 {
			lo := i - window + 1
			if windowHasNull(lo, i) {
				continue
			}
			valid[i] = true
			data[i] = values[deque[0]]
		}
	}
}

func (s *Series) toFloat64Values() []float64 {
	length := s.Len()
	values := make([]float64, length)

	switch s.dtype {
	case dtype.Int64:
		for i := 0; i < length; i++ {
			if s.IsValid(i) {
				v, _ := s.GetInt64(i)
				values[i] = float64(v)
			}
		}
	case dtype.Float64:
		for i := 0; i < length; i++ {
			if s.IsValid(i) {
				values[i], _ = s.GetFloat64(i)
			}
		}
	default:
		return nil
	}

	return values
}
