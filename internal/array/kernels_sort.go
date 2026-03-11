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
// Null values are placed at the end. Uses MSD radix sort on the raw byte
// representation for large arrays, falling back to insertion sort for
// small buckets.
func ArgSortString(a *StringArray, descending bool) []int {
	n := a.Len()
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	validCount := n
	if a.Validity() != nil {
		validCount = partitionNulls(indices, a.Validity())
	}

	if validCount <= 1 {
		return indices
	}

	validIndices := indices[:validCount]
	offsets := a.offsets
	data := a.data

	// Check if already sorted (ascending).
	sorted := true
	for i := 1; i < len(validIndices); i++ {
		ai, bi := validIndices[i-1], validIndices[i]
		sa := data[offsets[ai]:offsets[ai+1]]
		sb := data[offsets[bi]:offsets[bi+1]]
		if bytes.Compare(sa, sb) > 0 {
			sorted = false
			break
		}
	}
	if sorted {
		if descending {
			reverseIndices(validIndices)
		}
		return indices
	}

	// Check if reverse-sorted.
	sorted = true
	for i := 1; i < len(validIndices); i++ {
		ai, bi := validIndices[i-1], validIndices[i]
		sa := data[offsets[ai]:offsets[ai+1]]
		sb := data[offsets[bi]:offsets[bi+1]]
		if bytes.Compare(sa, sb) < 0 {
			sorted = false
			break
		}
	}
	if sorted {
		if !descending {
			reverseIndices(validIndices)
		}
		return indices
	}

	// MSD radix sort on raw bytes.
	buf := make([]int, len(validIndices))
	msdRadixSortString(validIndices, buf, data, offsets, 0, descending)
	return indices
}

// msdStringInsertionThreshold is the bucket size below which we fall back
// to insertion sort. Kept small to minimise comparison-sort overhead while
// avoiding excessive recursion for tiny buckets.
const msdStringInsertionThreshold = 32

// msdRadixSortString performs a stable MSD radix sort on indices into a
// StringArray. At each recursion level it examines the byte at position
// `depth` in each string. Bucket 0 holds strings that have ended (length
// <= depth); buckets 1-256 hold strings whose byte at `depth` is b (placed
// into bucket b+1). A counting-sort scatter into buf keeps the sort stable,
// then each non-trivial bucket is recursed on depth+1.
func msdRadixSortString(indices, buf []int, data []byte, offsets []int32, depth int, descending bool) {
	n := len(indices)
	if n <= 1 {
		return
	}

	// Small bucket: insertion sort with bytes.Compare.
	if n <= msdStringInsertionThreshold {
		msdStringInsertionSort(indices, data, offsets, descending)
		return
	}

	// 257 buckets: 0 = string ended, 1..256 = byte value + 1.
	var count [257]int
	for _, idx := range indices {
		slen := int(offsets[idx+1] - offsets[idx])
		if slen <= depth {
			count[0]++
		} else {
			b := data[offsets[idx]+int32(depth)]
			count[int(b)+1]++
		}
	}

	// Compute prefix sums. For ascending, bucket order is 0,1,..,256.
	// For descending, we reverse: 256,255,..,0 — but "ended" strings
	// (bucket 0) should still come first for ascending (empty < any) and
	// last for descending.
	var starts [257]int
	if !descending {
		total := 0
		for i := 0; i < 257; i++ {
			starts[i] = total
			total += count[i]
		}
	} else {
		total := 0
		// Bucket 256 down to 1 first (byte values in reverse), then bucket 0.
		for i := 256; i >= 1; i-- {
			starts[i] = total
			total += count[i]
		}
		starts[0] = total
	}

	// Scatter (stable: iterate indices in order).
	pos := starts // copy for scatter
	for _, idx := range indices {
		slen := int(offsets[idx+1] - offsets[idx])
		var bucket int
		if slen <= depth {
			bucket = 0
		} else {
			bucket = int(data[offsets[idx]+int32(depth)]) + 1
		}
		buf[pos[bucket]] = idx
		pos[bucket]++
	}

	// Copy back.
	copy(indices, buf[:n])

	// Recurse into each bucket that has >1 element. Bucket 0 (ended strings)
	// needs no further sorting — all strings there are equal up to this depth
	// and have the same length.
	if !descending {
		offset := count[0] // skip bucket 0
		for b := 1; b < 257; b++ {
			c := count[b]
			if c > 1 {
				msdRadixSortString(indices[offset:offset+c], buf[offset:offset+c], data, offsets, depth+1, descending)
			}
			offset += c
		}
	} else {
		// For descending, buckets are laid out 256,255,..,1,0.
		offset := 0
		for b := 256; b >= 1; b-- {
			c := count[b]
			if c > 1 {
				msdRadixSortString(indices[offset:offset+c], buf[offset:offset+c], data, offsets, depth+1, descending)
			}
			offset += c
		}
		// Bucket 0 at end, no recursion needed.
	}
}

// msdStringInsertionSort performs a stable insertion sort on a small slice
// of string indices, comparing via the underlying byte data.
func msdStringInsertionSort(indices []int, data []byte, offsets []int32, descending bool) {
	n := len(indices)
	for i := 1; i < n; i++ {
		key := indices[i]
		keyStart := offsets[key]
		keyEnd := offsets[key+1]
		keyBytes := data[keyStart:keyEnd]
		j := i - 1
		for j >= 0 {
			oj := indices[j]
			ojBytes := data[offsets[oj]:offsets[oj+1]]
			cmp := bytes.Compare(ojBytes, keyBytes)
			if descending {
				if cmp >= 0 {
					break
				}
			} else {
				if cmp <= 0 {
					break
				}
			}
			indices[j+1] = indices[j]
			j--
		}
		indices[j+1] = key
	}
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
