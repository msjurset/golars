package array

import (
	"cmp"
	"math"
	"slices"
	"unsafe"

	"github.com/msjurset/golars/internal/bitmap"
)

// isMostlySorted samples values and returns true if the data appears to be
// nearly sorted (ascending or descending). Samples up to 256 evenly-spaced
// elements and checks if >90% are in order.
func isMostlySorted[T interface {
	~int64 | ~float64 | ~int32 | ~float32 | ~uint64 | ~uint32
}](vals []T, descending bool) bool {
	n := len(vals)
	if n < 256 {
		inOrder := 0
		for i := 1; i < n; i++ {
			if descending {
				if vals[i] <= vals[i-1] {
					inOrder++
				}
			} else {
				if vals[i] >= vals[i-1] {
					inOrder++
				}
			}
		}
		return inOrder >= (n-1)*9/10
	}

	step := n / 256
	inOrder := 0
	total := 0
	for i := step; i < n; i += step {
		total++
		if descending {
			if vals[i] <= vals[i-step] {
				inOrder++
			}
		} else {
			if vals[i] >= vals[i-step] {
				inOrder++
			}
		}
	}
	return inOrder >= total*9/10
}

// radixSortThreshold is the minimum number of elements for radix sort to be
// worthwhile. Below this threshold we fall back to comparison-based sort.
const radixSortThreshold = 256

// partitionNulls performs a stable partition of indices so that non-null
// indices come first and null indices come last. Returns the count of
// non-null elements.
func partitionNulls(indices []int, validity *bitmap.Bitmap) int {
	n := len(indices)
	nonNull := make([]int, 0, n)
	nulls := make([]int, 0, n/8)
	for _, idx := range indices {
		if validity.IsSet(idx) {
			nonNull = append(nonNull, idx)
		} else {
			nulls = append(nulls, idx)
		}
	}
	copy(indices, nonNull)
	copy(indices[len(nonNull):], nulls)
	return len(nonNull)
}

// radixArgSort64 sorts indices by uint64 keys using LSD radix sort.
// The keys slice is parallel to indices and contains the sort keys after
// appropriate bit manipulation. Only the first n elements are sorted.
func radixArgSort64(keys []uint64, indices []int, descending bool) {
	n := len(indices)
	if n <= 1 {
		return
	}

	if n < radixSortThreshold {
		if descending {
			slices.SortStableFunc(indices, func(a, b int) int {
				ka, kb := keys[a], keys[b]
				if ka > kb {
					return -1
				}
				if ka < kb {
					return 1
				}
				return 0
			})
		} else {
			slices.SortStableFunc(indices, func(a, b int) int {
				ka, kb := keys[a], keys[b]
				if ka < kb {
					return -1
				}
				if ka > kb {
					return 1
				}
				return 0
			})
		}
		return
	}

	buf := make([]int, n)
	src, dst := indices, buf

	for pass := 0; pass < 8; pass++ {
		shift := uint(pass * 8)

		// Count occurrences of each byte value.
		var count [256]int
		for _, idx := range src {
			b := byte(keys[idx] >> shift)
			count[b]++
		}

		// Check if this pass is a no-op (all same byte).
		allSame := false
		for _, c := range count {
			if c == n {
				allSame = true
				break
			}
		}
		if allSame {
			continue
		}

		// Prefix sum.
		if descending {
			total := 0
			for i := 255; i >= 0; i-- {
				count[i], total = total, total+count[i]
			}
		} else {
			total := 0
			for i := 0; i < 256; i++ {
				count[i], total = total, total+count[i]
			}
		}

		// Scatter.
		for _, idx := range src {
			b := byte(keys[idx] >> shift)
			dst[count[b]] = idx
			count[b]++
		}

		src, dst = dst, src
	}

	// If result ended up in buf, copy back.
	if unsafe.Pointer(&src[0]) != unsafe.Pointer(&indices[0]) {
		copy(indices, src)
	}
}

// radixArgSort32 sorts indices by uint32 keys using LSD radix sort (4 passes).
func radixArgSort32(keys []uint32, indices []int, descending bool) {
	n := len(indices)
	if n <= 1 {
		return
	}

	if n < radixSortThreshold {
		if descending {
			slices.SortStableFunc(indices, func(a, b int) int {
				ka, kb := keys[a], keys[b]
				if ka > kb {
					return -1
				}
				if ka < kb {
					return 1
				}
				return 0
			})
		} else {
			slices.SortStableFunc(indices, func(a, b int) int {
				ka, kb := keys[a], keys[b]
				if ka < kb {
					return -1
				}
				if ka > kb {
					return 1
				}
				return 0
			})
		}
		return
	}

	buf := make([]int, n)
	src, dst := indices, buf

	for pass := 0; pass < 4; pass++ {
		shift := uint(pass * 8)

		var count [256]int
		for _, idx := range src {
			b := byte(keys[idx] >> shift)
			count[b]++
		}

		allSame := false
		for _, c := range count {
			if c == n {
				allSame = true
				break
			}
		}
		if allSame {
			continue
		}

		if descending {
			total := 0
			for i := 255; i >= 0; i-- {
				count[i], total = total, total+count[i]
			}
		} else {
			total := 0
			for i := 0; i < 256; i++ {
				count[i], total = total, total+count[i]
			}
		}

		for _, idx := range src {
			b := byte(keys[idx] >> shift)
			dst[count[b]] = idx
			count[b]++
		}

		src, dst = dst, src
	}

	if unsafe.Pointer(&src[0]) != unsafe.Pointer(&indices[0]) {
		copy(indices, src)
	}
}

// int64ToSortableUint64 converts an int64 to a uint64 that sorts in the same
// order by flipping the sign bit.
func int64ToSortableUint64(v int64) uint64 {
	return uint64(v) ^ (1 << 63)
}

// int32ToSortableUint32 converts an int32 to a uint32 that sorts in the same
// order by flipping the sign bit.
func int32ToSortableUint32(v int32) uint32 {
	return uint32(v) ^ (1 << 31)
}

// int16ToSortableUint32 converts an int16 to a uint32 for sorting.
func int16ToSortableUint32(v int16) uint32 {
	return uint32(uint16(v) ^ (1 << 15))
}

// int8ToSortableUint32 converts an int8 to a uint32 for sorting.
func int8ToSortableUint32(v int8) uint32 {
	return uint32(uint8(v) ^ (1 << 7))
}

// float64ToSortableUint64 converts a float64 to a uint64 that sorts in the
// correct order using the IEEE 754 bit-flip trick.
func float64ToSortableUint64(v float64) uint64 {
	bits := math.Float64bits(v)
	if bits>>63 != 0 {
		bits = ^bits
	} else {
		bits ^= 1 << 63
	}
	return bits
}

// float32ToSortableUint32 converts a float32 to a uint32 that sorts in the
// correct order using the IEEE 754 bit-flip trick.
func float32ToSortableUint32(v float32) uint32 {
	bits := math.Float32bits(v)
	if bits>>31 != 0 {
		bits = ^bits
	} else {
		bits ^= 1 << 31
	}
	return bits
}

// ArgSortInt64 sorts a TypedArray[int64] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortInt64(a *TypedArray[int64], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint64, n)
	for _, idx := range validIndices {
		keys[idx] = int64ToSortableUint64(vals[idx])
	}

	radixArgSort64(keys, validIndices, descending)
	return indices
}

// ArgSortInt32 sorts a TypedArray[int32] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortInt32(a *TypedArray[int32], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint32, n)
	for _, idx := range validIndices {
		keys[idx] = int32ToSortableUint32(vals[idx])
	}

	radixArgSort32(keys, validIndices, descending)
	return indices
}

// ArgSortInt16 sorts a TypedArray[int16] using comparison sort.
// Small-range types don't benefit from radix sort.
func ArgSortInt16(a *TypedArray[int16], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]
	slices.SortStableFunc(validIndices, func(i, j int) int {
		if descending {
			return cmp.Compare(vals[j], vals[i])
		}
		return cmp.Compare(vals[i], vals[j])
	})
	return indices
}

// ArgSortInt8 sorts a TypedArray[int8] using comparison sort.
// Small-range types don't benefit from radix sort.
func ArgSortInt8(a *TypedArray[int8], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]
	slices.SortStableFunc(validIndices, func(i, j int) int {
		if descending {
			return cmp.Compare(vals[j], vals[i])
		}
		return cmp.Compare(vals[i], vals[j])
	})
	return indices
}

// ArgSortUint64 sorts a TypedArray[uint64] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortUint64(a *TypedArray[uint64], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint64, n)
	for _, idx := range validIndices {
		keys[idx] = vals[idx]
	}

	radixArgSort64(keys, validIndices, descending)
	return indices
}

// ArgSortUint32 sorts a TypedArray[uint32] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortUint32(a *TypedArray[uint32], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint32, n)
	for _, idx := range validIndices {
		keys[idx] = vals[idx]
	}

	radixArgSort32(keys, validIndices, descending)
	return indices
}

// ArgSortUint16 sorts a TypedArray[uint16] using comparison sort.
// Small-range types don't benefit from radix sort.
func ArgSortUint16(a *TypedArray[uint16], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]
	slices.SortStableFunc(validIndices, func(i, j int) int {
		if descending {
			return cmp.Compare(vals[j], vals[i])
		}
		return cmp.Compare(vals[i], vals[j])
	})
	return indices
}

// ArgSortUint8 sorts a TypedArray[uint8] using comparison sort.
// Small-range types don't benefit from radix sort.
func ArgSortUint8(a *TypedArray[uint8], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]
	slices.SortStableFunc(validIndices, func(i, j int) int {
		if descending {
			return cmp.Compare(vals[j], vals[i])
		}
		return cmp.Compare(vals[i], vals[j])
	})
	return indices
}

// ArgSortFloat64 sorts a TypedArray[float64] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortFloat64(a *TypedArray[float64], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint64, n)
	for _, idx := range validIndices {
		keys[idx] = float64ToSortableUint64(vals[idx])
	}

	radixArgSort64(keys, validIndices, descending)
	return indices
}

// ArgSortFloat32 sorts a TypedArray[float32] using radix sort when beneficial.
// Falls back to comparison sort for nearly-sorted data or small arrays.
func ArgSortFloat32(a *TypedArray[float32], descending bool) []int {
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

	vals := a.Values()
	validIndices := indices[:validCount]

	if isMostlySorted(vals, descending) {
		slices.SortStableFunc(validIndices, func(i, j int) int {
			if descending {
				return cmp.Compare(vals[j], vals[i])
			}
			return cmp.Compare(vals[i], vals[j])
		})
		return indices
	}

	keys := make([]uint32, n)
	for _, idx := range validIndices {
		keys[idx] = float32ToSortableUint32(vals[idx])
	}

	radixArgSort32(keys, validIndices, descending)
	return indices
}
