package array

import "sort"

// ArgSort returns the indices that would sort the array. Null values are
// placed at the end of the result regardless of sort direction.
func ArgSort[T Ordered](a *TypedArray[T], descending bool) []int {
	n := a.Len()
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	vals := a.Values()
	hasNulls := a.Validity() != nil

	sort.SliceStable(indices, func(i, j int) bool {
		ii, jj := indices[i], indices[j]
		iNull := hasNulls && a.IsNull(ii)
		jNull := hasNulls && a.IsNull(jj)

		if iNull && jNull {
			return false
		}
		if iNull {
			return false // nulls to end
		}
		if jNull {
			return true // nulls to end
		}
		if descending {
			return vals[ii] > vals[jj]
		}
		return vals[ii] < vals[jj]
	})
	return indices
}

// ArgSortString returns the indices that would sort a StringArray.
// Null values are placed at the end.
func ArgSortString(a *StringArray, descending bool) []int {
	n := a.Len()
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	hasNulls := a.Validity() != nil

	sort.SliceStable(indices, func(i, j int) bool {
		ii, jj := indices[i], indices[j]
		iNull := hasNulls && a.IsNull(ii)
		jNull := hasNulls && a.IsNull(jj)

		if iNull && jNull {
			return false
		}
		if iNull {
			return false
		}
		if jNull {
			return true
		}
		vi, vj := a.Value(ii), a.Value(jj)
		if descending {
			return vi > vj
		}
		return vi < vj
	})
	return indices
}

// ArgMin returns the index of the minimum non-null value. The second return
// value is false if the array is empty or all null.
func ArgMin[T Ordered](a *TypedArray[T]) (int, bool) {
	n := a.Len()
	bestIdx := -1
	var bestVal T
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if bestIdx == -1 || v < bestVal {
				bestIdx = i
				bestVal = v
			}
		}
	}
	if bestIdx == -1 {
		return 0, false
	}
	return bestIdx, true
}

// ArgMax returns the index of the maximum non-null value. The second return
// value is false if the array is empty or all null.
func ArgMax[T Ordered](a *TypedArray[T]) (int, bool) {
	n := a.Len()
	bestIdx := -1
	var bestVal T
	vals := a.Values()
	for i := 0; i < n; i++ {
		if a.IsValid(i) {
			v := vals[i]
			if bestIdx == -1 || v > bestVal {
				bestIdx = i
				bestVal = v
			}
		}
	}
	if bestIdx == -1 {
		return 0, false
	}
	return bestIdx, true
}
