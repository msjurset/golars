package array

import "math"

// Sum computes the sum of all valid (non-null) elements in a. The second
// return value is false when every element is null, indicating the result
// is not meaningful.
func Sum[T Numeric](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	var total T
	found := false
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			total += vals[i]
			found = true
		}
	}
	return total, found
}

// Mean computes the arithmetic mean of all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Mean[T Numeric](a *TypedArray[T]) (float64, bool) {
	n := a.Len()
	var total float64
	count := 0
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			total += float64(vals[i])
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

// Min returns the minimum value among all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Min[T Ordered](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	var min T
	found := false
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if !found || v < min {
				min = v
				found = true
			}
		}
	}
	return min, found
}

// Max returns the maximum value among all valid (non-null) elements in a.
// The second return value is false when every element is null.
func Max[T Ordered](a *TypedArray[T]) (T, bool) {
	n := a.Len()
	var max T
	found := false
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if !found || v > max {
				max = v
				found = true
			}
		}
	}
	return max, found
}

// Variance computes the variance of all valid (non-null) elements in a.
// The ddof parameter controls the degrees-of-freedom correction: use 0 for
// population variance and 1 for sample variance. The second return value is
// false when there are insufficient valid elements for the given ddof.
func Variance[T Numeric](a *TypedArray[T], ddof int) (float64, bool) {
	n := a.Len()
	var sum, sumSq float64
	count := 0
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := float64(vals[i])
			sum += v
			sumSq += v * v
			count++
		}
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
