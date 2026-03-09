package series

import (
	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
)

// Interpolate fills null values using the specified method.
// Supported methods: "linear", "pad"/"ffill" (forward fill), "bfill" (backward fill).
// Only works for numeric types. The "linear" method always returns Float64.
func (s *Series) Interpolate(method string) *Series {
	if !s.HasNulls() {
		if method == "linear" {
			return toFloat64Series(s)
		}
		return s
	}

	switch method {
	case "linear":
		return interpolateLinear(s)
	case "pad", "ffill":
		return interpolatePad(s, true)
	case "bfill":
		return interpolatePad(s, false)
	default:
		return s
	}
}

func getNumericFloat64(s *Series, i int) float64 {
	switch s.DataType() {
	case dtype.Int64:
		if ta, ok := s.arr.(*array.TypedArray[int64]); ok {
			return float64(ta.Value(i))
		}
	case dtype.Float64:
		if ta, ok := s.arr.(*array.TypedArray[float64]); ok {
			return ta.Value(i)
		}
	case dtype.Int32:
		if ta, ok := s.arr.(*array.TypedArray[int32]); ok {
			return float64(ta.Value(i))
		}
	case dtype.Float32:
		if ta, ok := s.arr.(*array.TypedArray[float32]); ok {
			return float64(ta.Value(i))
		}
	case dtype.Int8:
		if ta, ok := s.arr.(*array.TypedArray[int8]); ok {
			return float64(ta.Value(i))
		}
	case dtype.Int16:
		if ta, ok := s.arr.(*array.TypedArray[int16]); ok {
			return float64(ta.Value(i))
		}
	case dtype.UInt8:
		if ta, ok := s.arr.(*array.TypedArray[uint8]); ok {
			return float64(ta.Value(i))
		}
	case dtype.UInt16:
		if ta, ok := s.arr.(*array.TypedArray[uint16]); ok {
			return float64(ta.Value(i))
		}
	case dtype.UInt32:
		if ta, ok := s.arr.(*array.TypedArray[uint32]); ok {
			return float64(ta.Value(i))
		}
	case dtype.UInt64:
		if ta, ok := s.arr.(*array.TypedArray[uint64]); ok {
			return float64(ta.Value(i))
		}
	}
	return 0
}

func toFloat64Series(s *Series) *Series {
	if s.DataType() == dtype.Float64 {
		return s
	}
	n := s.Len()
	data := make([]float64, n)
	for i := 0; i < n; i++ {
		data[i] = getNumericFloat64(s, i)
	}
	return NewFloat64(s.name, data)
}

func interpolateLinear(s *Series) *Series {
	if !dtype.IsNumeric(s.DataType()) {
		return s
	}

	n := s.Len()
	data := make([]float64, n)
	valid := make([]bool, n)

	// First pass: copy non-null values
	for i := 0; i < n; i++ {
		if s.IsValid(i) {
			data[i] = getNumericFloat64(s, i)
			valid[i] = true
		}
	}

	// Second pass: linear interpolation between known values
	i := 0
	for i < n {
		if valid[i] {
			i++
			continue
		}

		// Find the start of a null gap
		gapStart := i

		// Find the end of the null gap
		for i < n && !valid[i] {
			i++
		}
		gapEnd := i // exclusive

		// Find surrounding known values
		prevIdx := gapStart - 1
		nextIdx := gapEnd

		if prevIdx < 0 || nextIdx >= n {
			// Leading or trailing nulls - leave as null
			continue
		}

		// Linearly interpolate
		prevVal := data[prevIdx]
		nextVal := data[nextIdx]
		gapLen := float64(nextIdx - prevIdx)

		for j := gapStart; j < gapEnd; j++ {
			t := float64(j-prevIdx) / gapLen
			data[j] = prevVal + t*(nextVal-prevVal)
			valid[j] = true
		}
	}

	// Check if there are still nulls (leading/trailing)
	hasNulls := false
	for _, v := range valid {
		if !v {
			hasNulls = true
			break
		}
	}

	if hasNulls {
		return NewFloat64WithValidity(s.name, data, valid)
	}
	return NewFloat64(s.name, data)
}

func interpolatePad(s *Series, forward bool) *Series {
	if !dtype.IsNumeric(s.DataType()) {
		return s
	}

	n := s.Len()

	switch s.DataType() {
	case dtype.Int64:
		return padInt64(s, n, forward)
	case dtype.Float64:
		return padFloat64(s, n, forward)
	default:
		// For all other numeric types, promote to float64
		return padFloat64Generic(s, n, forward)
	}
}

func padFloat64(s *Series, n int, forward bool) *Series {
	data := make([]float64, n)
	valid := make([]bool, n)

	if forward {
		var lastValid float64
		hasLast := false
		for i := 0; i < n; i++ {
			if s.IsValid(i) {
				v, _ := s.GetFloat64(i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	} else {
		var lastValid float64
		hasLast := false
		for i := n - 1; i >= 0; i-- {
			if s.IsValid(i) {
				v, _ := s.GetFloat64(i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	}

	hasNulls := false
	for _, v := range valid {
		if !v {
			hasNulls = true
			break
		}
	}
	if hasNulls {
		return NewFloat64WithValidity(s.name, data, valid)
	}
	return NewFloat64(s.name, data)
}

func padInt64(s *Series, n int, forward bool) *Series {
	data := make([]int64, n)
	valid := make([]bool, n)

	if forward {
		var lastValid int64
		hasLast := false
		for i := 0; i < n; i++ {
			if s.IsValid(i) {
				v, _ := s.GetInt64(i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	} else {
		var lastValid int64
		hasLast := false
		for i := n - 1; i >= 0; i-- {
			if s.IsValid(i) {
				v, _ := s.GetInt64(i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	}

	hasNulls := false
	for _, v := range valid {
		if !v {
			hasNulls = true
			break
		}
	}
	if hasNulls {
		return NewInt64WithValidity(s.name, data, valid)
	}
	return NewInt64(s.name, data)
}

func padFloat64Generic(s *Series, n int, forward bool) *Series {
	data := make([]float64, n)
	valid := make([]bool, n)

	if forward {
		var lastValid float64
		hasLast := false
		for i := 0; i < n; i++ {
			if s.IsValid(i) {
				v := getNumericFloat64(s, i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	} else {
		var lastValid float64
		hasLast := false
		for i := n - 1; i >= 0; i-- {
			if s.IsValid(i) {
				v := getNumericFloat64(s, i)
				data[i] = v
				valid[i] = true
				lastValid = v
				hasLast = true
			} else if hasLast {
				data[i] = lastValid
				valid[i] = true
			}
		}
	}

	hasNulls := false
	for _, v := range valid {
		if !v {
			hasNulls = true
			break
		}
	}
	if hasNulls {
		return NewFloat64WithValidity(s.name, data, valid)
	}
	return NewFloat64(s.name, data)
}
