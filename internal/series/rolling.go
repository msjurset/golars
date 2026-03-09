package series

import (
	"math"

	"github.com/msjurseth/golars/internal/dtype"
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

	for i := window - 1; i < length; i++ {
		start := i - window + 1
		windowVals := make([]float64, 0, window)
		allValid := true
		for j := start; j <= i; j++ {
			if s.IsValid(j) {
				windowVals = append(windowVals, values[j])
			} else {
				allValid = false
			}
		}

		if !allValid || len(windowVals) < window {
			continue
		}

		valid[i] = true
		switch op {
		case rollSum:
			sum := 0.0
			for _, v := range windowVals {
				sum += v
			}
			data[i] = sum
		case rollMean:
			sum := 0.0
			for _, v := range windowVals {
				sum += v
			}
			data[i] = sum / float64(len(windowVals))
		case rollMin:
			m := windowVals[0]
			for _, v := range windowVals[1:] {
				if v < m {
					m = v
				}
			}
			data[i] = m
		case rollMax:
			m := windowVals[0]
			for _, v := range windowVals[1:] {
				if v > m {
					m = v
				}
			}
			data[i] = m
		case rollStd:
			mean := 0.0
			for _, v := range windowVals {
				mean += v
			}
			mean /= float64(len(windowVals))
			sumSq := 0.0
			for _, v := range windowVals {
				d := v - mean
				sumSq += d * d
			}
			if len(windowVals) > 1 {
				data[i] = math.Sqrt(sumSq / float64(len(windowVals)-1))
			}
		}
	}

	return NewFloat64WithValidity(s.name, data, valid)
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
