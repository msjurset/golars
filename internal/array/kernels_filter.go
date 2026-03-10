package array

import (
	"math/bits"

	"github.com/msjurset/golars/internal/bitmap"
)

// FilterTyped returns a new TypedArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is
// filtered accordingly.
//
// Dense-mask optimization: when an entire 64-bit word is all-ones (or the
// final partial word has all valid bits set), the function uses copy() for
// the contiguous span, which the compiler lowers to an optimized memmove.
// PopCount is O(n/64) and cached-friendly, so callers sharing a mask across
// multiple columns pay negligible repeated cost.
func FilterTyped[T any](arr *TypedArray[T], mask *bitmap.Bitmap) *TypedArray[T] {
	n := mask.PopCount()
	data := make([]T, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil
	srcValidity := arr.Validity()

	if hasNulls {
		validity = bitmap.New(n)
	}

	values := arr.Values()
	words := mask.Words()
	arrLen := arr.Len()
	j := 0

	for wi, w := range words {
		if w == 0 {
			continue
		}
		base := wi * 64
		remaining := arrLen - base
		if remaining > 64 {
			remaining = 64
		}

		// Dense fast path: all bits in this word are set — bulk copy.
		allSet := w == ^uint64(0) || (remaining < 64 && w == (1<<uint(remaining))-1)
		if allSet {
			copy(data[j:], values[base:base+remaining])
			if hasNulls {
				for k := 0; k < remaining; k++ {
					if !srcValidity.IsSet(base + k) {
						validity.Clear(j + k)
					}
				}
			}
			j += remaining
			continue
		}

		// Sparse path: extract set bits one at a time.
		for w != 0 {
			tz := bits.TrailingZeros64(w)
			i := base + tz
			if i >= arrLen {
				break
			}
			data[j] = values[i]
			if hasNulls && !srcValidity.IsSet(i) {
				validity.Clear(j)
			}
			j++
			w &= w - 1 // clear lowest set bit
		}
	}

	return NewTypedArray(data, arr.DataType(), validity)
}

// TakeTyped returns a new TypedArray with elements gathered from arr at the
// given indices. The validity bitmap is carried over for each gathered element.
func TakeTyped[T any](arr *TypedArray[T], indices []int) *TypedArray[T] {
	n := len(indices)
	data := make([]T, n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil

	if hasNulls {
		validity = bitmap.New(n)
	}

	for j, idx := range indices {
		data[j] = arr.Value(idx)
		if hasNulls && arr.IsNull(idx) {
			validity.Clear(j)
		}
	}

	return NewTypedArray(data, arr.DataType(), validity)
}

// FilterBoolean returns a new BooleanArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is filtered
// accordingly.
//
// Dense-mask optimization: when all bits in a word are set and the output
// position is word-aligned, data and validity words are copied directly.
func FilterBoolean(arr *BooleanArray, mask *bitmap.Bitmap) *BooleanArray {
	n := mask.PopCount()
	data := bitmap.NewEmpty(n)
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil
	srcValidity := arr.Validity()
	srcData := arr.DataBitmap()

	if hasNulls {
		validity = bitmap.New(n)
	}

	words := mask.Words()
	srcDataWords := srcData.Words()
	arrLen := arr.Len()
	j := 0

	for wi, w := range words {
		if w == 0 {
			continue
		}
		base := wi * 64
		remaining := arrLen - base
		if remaining > 64 {
			remaining = 64
		}

		allSet := w == ^uint64(0) || (remaining < 64 && w == (1<<uint(remaining))-1)
		if allSet {
			// Dense fast path: copy data bits. When output is word-aligned
			// we can use SetWord directly; otherwise fall through to per-bit.
			if j%64 == 0 {
				outWord := j / 64
				if remaining == 64 {
					data.SetWord(outWord, srcDataWords[wi])
				} else {
					// Partial last word — mask to remaining bits.
					data.SetWord(outWord, srcDataWords[wi]&((1<<uint(remaining))-1))
				}
				if hasNulls {
					srcValidityWords := srcValidity.Words()
					if remaining == 64 {
						validity.SetWord(outWord, srcValidityWords[wi])
					} else {
						validity.SetWord(outWord, srcValidityWords[wi]&((1<<uint(remaining))-1))
					}
				}
				j += remaining
				continue
			}
			// Output not word-aligned — fall through to per-bit below.
		}

		// Sparse path (or unaligned dense): extract set bits one at a time.
		for w != 0 {
			tz := bits.TrailingZeros64(w)
			i := base + tz
			if i >= arrLen {
				break
			}
			if srcData.IsSet(i) {
				data.Set(j)
			}
			if hasNulls && !srcValidity.IsSet(i) {
				validity.Clear(j)
			}
			j++
			w &= w - 1
		}
	}

	return NewBooleanArrayFromBitmap(data, validity)
}

// FilterString returns a new StringArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is filtered
// accordingly. It operates directly on the offset-based byte storage to avoid
// per-element string allocations.
//
// Dense-mask optimization: when all bits in a word are set, the byte data for
// the entire span is contiguous in the source, so a single copy() handles all
// 64 (or fewer) strings at once. Offsets are computed with a simple add loop.
func FilterString(arr *StringArray, mask *bitmap.Bitmap) *StringArray {
	n := mask.PopCount()
	srcOffsets := arr.Offsets()
	srcData := arr.Data()
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil
	srcValidity := arr.Validity()

	if hasNulls {
		validity = bitmap.New(n)
	}

	// First pass: calculate total bytes needed.
	totalBytes := 0
	words := mask.Words()
	arrLen := arr.Len()

	for wi, w := range words {
		if w == 0 {
			continue
		}
		base := wi * 64
		remaining := arrLen - base
		if remaining > 64 {
			remaining = 64
		}

		allSet := w == ^uint64(0) || (remaining < 64 && w == (1<<uint(remaining))-1)
		if allSet {
			// Contiguous span: byte range is [offsets[base], offsets[base+remaining]).
			totalBytes += int(srcOffsets[base+remaining] - srcOffsets[base])
			continue
		}

		for w != 0 {
			tz := bits.TrailingZeros64(w)
			i := base + tz
			if i >= arrLen {
				break
			}
			totalBytes += int(srcOffsets[i+1] - srcOffsets[i])
			w &= w - 1
		}
	}

	// Second pass: copy byte ranges and build offsets.
	data := make([]byte, totalBytes)
	offsets := make([]int32, n+1)
	offsets[0] = 0
	j := 0
	pos := int32(0)

	for wi, w := range words {
		if w == 0 {
			continue
		}
		base := wi * 64
		remaining := arrLen - base
		if remaining > 64 {
			remaining = 64
		}

		allSet := w == ^uint64(0) || (remaining < 64 && w == (1<<uint(remaining))-1)
		if allSet {
			// Dense fast path: single bulk copy of the byte block.
			byteStart := srcOffsets[base]
			byteEnd := srcOffsets[base+remaining]
			copy(data[pos:], srcData[byteStart:byteEnd])
			// Build offsets: each offset is the source offset shifted by
			// (pos - byteStart) to account for the new data position.
			delta := pos - byteStart
			for k := 0; k < remaining; k++ {
				offsets[j+k+1] = srcOffsets[base+k+1] + delta
			}
			if hasNulls {
				for k := 0; k < remaining; k++ {
					if !srcValidity.IsSet(base + k) {
						validity.Clear(j + k)
					}
				}
			}
			pos += byteEnd - byteStart
			j += remaining
			continue
		}

		// Sparse path: copy each string individually.
		for w != 0 {
			tz := bits.TrailingZeros64(w)
			i := base + tz
			if i >= arrLen {
				break
			}
			start := srcOffsets[i]
			end := srcOffsets[i+1]
			copy(data[pos:], srcData[start:end])
			pos += end - start
			offsets[j+1] = pos
			if hasNulls && !srcValidity.IsSet(i) {
				validity.Clear(j)
			}
			j++
			w &= w - 1
		}
	}

	return NewStringArrayFromBytes(data, offsets, validity)
}

// TakeString returns a new StringArray with elements gathered from arr at the
// given indices. It operates directly on the offset-based byte storage to avoid
// per-element string allocations.
func TakeString(arr *StringArray, indices []int) *StringArray {
	n := len(indices)
	srcOffsets := arr.Offsets()
	srcData := arr.Data()
	var validity *bitmap.Bitmap
	hasNulls := arr.Validity() != nil
	srcValidity := arr.Validity()

	if hasNulls {
		validity = bitmap.New(n)
	}

	// Calculate total bytes needed.
	totalBytes := 0
	for _, idx := range indices {
		totalBytes += int(srcOffsets[idx+1] - srcOffsets[idx])
	}

	// Copy byte ranges and build offsets.
	data := make([]byte, totalBytes)
	offsets := make([]int32, n+1)
	offsets[0] = 0
	pos := int32(0)

	for j, idx := range indices {
		start := srcOffsets[idx]
		end := srcOffsets[idx+1]
		copy(data[pos:], srcData[start:end])
		pos += end - start
		offsets[j+1] = pos
		if hasNulls && !srcValidity.IsSet(idx) {
			validity.Clear(j)
		}
	}

	return NewStringArrayFromBytes(data, offsets, validity)
}
