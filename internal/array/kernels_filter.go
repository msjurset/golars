package array

import (
	"math/bits"

	"github.com/msjurset/golars/internal/bitmap"
)

// FilterTyped returns a new TypedArray containing only the elements of arr
// where the corresponding bit in mask is set. The validity bitmap is
// filtered accordingly.
func FilterTyped[T any](arr *TypedArray[T], mask *bitmap.Bitmap) *TypedArray[T] {
	n := mask.PopCount()
	data := make([]T, 0, n)
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
		for w != 0 {
			tz := bits.TrailingZeros64(w)
			i := base + tz
			if i >= arrLen {
				break
			}
			data = append(data, values[i])
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
	arrLen := arr.Len()
	j := 0

	for wi, w := range words {
		if w == 0 {
			continue
		}
		base := wi * 64
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
