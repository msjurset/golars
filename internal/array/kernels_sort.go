package array

import (
	"bytes"
	"sort"
)

// ArgSort returns the indices that would sort the array. Null values are
// placed at the end of the result regardless of sort direction.
//
// For numeric types, this dispatches to type-specific radix sort
// implementations for better performance on large arrays.
func ArgSort[T Ordered](a *TypedArray[T], descending bool) []int {
	// Try to dispatch to radix sort for numeric types.
	switch arr := any(a).(type) {
	case *TypedArray[int64]:
		return ArgSortInt64(arr, descending)
	case *TypedArray[int32]:
		return ArgSortInt32(arr, descending)
	case *TypedArray[int16]:
		return ArgSortInt16(arr, descending)
	case *TypedArray[int8]:
		return ArgSortInt8(arr, descending)
	case *TypedArray[uint64]:
		return ArgSortUint64(arr, descending)
	case *TypedArray[uint32]:
		return ArgSortUint32(arr, descending)
	case *TypedArray[uint16]:
		return ArgSortUint16(arr, descending)
	case *TypedArray[uint8]:
		return ArgSortUint8(arr, descending)
	case *TypedArray[float64]:
		return ArgSortFloat64(arr, descending)
	case *TypedArray[float32]:
		return ArgSortFloat32(arr, descending)
	}

	// Fallback: comparison-based sort for string and other types.
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
		cmp := bytes.Compare(a.ValueBytes(ii), a.ValueBytes(jj))
		if descending {
			return cmp > 0
		}
		return cmp < 0
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
