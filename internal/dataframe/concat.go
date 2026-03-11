package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// Concat vertically concatenates the given DataFrames. All DataFrames must
// share the same schema (column names and types in the same order). Returns
// an error if schemas differ or no DataFrames are provided.
func Concat(dfs ...*DataFrame) (*DataFrame, error) {
	if len(dfs) == 0 {
		return nil, fmt.Errorf("golars: no DataFrames to concatenate")
	}
	if len(dfs) == 1 {
		return dfs[0].Clone(), nil
	}

	base := dfs[0].Schema()
	for i := 1; i < len(dfs); i++ {
		if !base.Equal(dfs[i].Schema()) {
			return nil, fmt.Errorf("golars: schema mismatch at DataFrame index %d", i)
		}
	}

	totalHeight := 0
	for _, df := range dfs {
		totalHeight += df.Height()
	}

	w := base.Len()
	cols := make([]*series.Series, w)

	for j := 0; j < w; j++ {
		f := base.Field(j)
		cols[j] = concatColumn(f.Name, f.Dtype, dfs, j, totalHeight)
	}

	return New(cols...)
}

// concatColumn concatenates a single column across all DataFrames.
func concatColumn(name string, dt dtype.DataType, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	switch dt {
	case dtype.Int8:
		return concatTypedColumn[int8](name, dt, dfs, colIdx, totalHeight)
	case dtype.Int16:
		return concatTypedColumn[int16](name, dt, dfs, colIdx, totalHeight)
	case dtype.Int32, dtype.Date:
		return concatTypedColumn[int32](name, dt, dfs, colIdx, totalHeight)
	case dtype.Int64, dtype.DateTime, dtype.Time, dtype.Duration:
		return concatTypedColumn[int64](name, dt, dfs, colIdx, totalHeight)
	case dtype.UInt8:
		return concatTypedColumn[uint8](name, dt, dfs, colIdx, totalHeight)
	case dtype.UInt16:
		return concatTypedColumn[uint16](name, dt, dfs, colIdx, totalHeight)
	case dtype.UInt32:
		return concatTypedColumn[uint32](name, dt, dfs, colIdx, totalHeight)
	case dtype.UInt64:
		return concatTypedColumn[uint64](name, dt, dfs, colIdx, totalHeight)
	case dtype.Float32:
		return concatTypedColumn[float32](name, dt, dfs, colIdx, totalHeight)
	case dtype.Float64:
		return concatTypedColumn[float64](name, dt, dfs, colIdx, totalHeight)
	case dtype.String:
		return concatStringColumn(name, dfs, colIdx, totalHeight)
	case dtype.Boolean:
		return concatBoolColumn(name, dfs, colIdx, totalHeight)
	default:
		return concatStringColumn(name, dfs, colIdx, totalHeight)
	}
}

// concatTypedColumn concatenates a numeric/typed column using bulk copy.
func concatTypedColumn[T any](
	name string,
	dt dtype.DataType,
	dfs []*DataFrame,
	colIdx int,
	totalHeight int,
) *series.Series {
	data := make([]T, totalHeight)
	hasNulls := false
	offset := 0

	for _, df := range dfs {
		c := df.columns[colIdx]
		ta := c.Array().(*array.TypedArray[T])
		vals := ta.Values()
		copy(data[offset:], vals)
		if ta.Validity() != nil {
			hasNulls = true
		}
		offset += len(vals)
	}

	var validity *bitmap.Bitmap
	if hasNulls {
		validity = concatBitmaps(dfs, colIdx, totalHeight)
	}

	return series.New(name, array.NewTypedArray(data, dt, validity))
}

// concatStringColumn concatenates string columns using bulk byte/offset copy.
func concatStringColumn(name string, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	// Calculate total bytes needed
	totalBytes := 0
	hasNulls := false
	for _, df := range dfs {
		sa := df.columns[colIdx].StringArray()
		totalBytes += len(sa.Data())
		if sa.Validity() != nil {
			hasNulls = true
		}
	}

	// Bulk copy bytes and build offsets
	allData := make([]byte, totalBytes)
	allOffsets := make([]int32, totalHeight+1)
	allOffsets[0] = 0
	byteOff := 0
	elemOff := 1 // skip the leading 0

	for _, df := range dfs {
		sa := df.columns[colIdx].StringArray()
		srcData := sa.Data()
		srcOffsets := sa.Offsets()
		baseOffset := int32(byteOff)

		copy(allData[byteOff:], srcData)
		byteOff += len(srcData)

		// Copy offsets (skip first 0), adjusting by base offset
		n := len(srcOffsets) - 1
		for i := 0; i < n; i++ {
			allOffsets[elemOff+i] = srcOffsets[i+1] + baseOffset
		}
		elemOff += n
	}

	var validity *bitmap.Bitmap
	if hasNulls {
		validity = concatBitmaps(dfs, colIdx, totalHeight)
	}

	return series.New(name, array.NewStringArrayFromBytes(allData, allOffsets, validity))
}

// concatBoolColumn concatenates boolean columns using bulk bitmap copy.
func concatBoolColumn(name string, dfs []*DataFrame, colIdx int, totalHeight int) *series.Series {
	hasNulls := false
	for _, df := range dfs {
		if df.columns[colIdx].BooleanArray().Validity() != nil {
			hasNulls = true
			break
		}
	}

	// Build data bitmap
	dataBm := bitmap.NewEmpty(totalHeight)
	dataWords := dataBm.Words()
	bitOffset := 0

	for _, df := range dfs {
		ba := df.columns[colIdx].BooleanArray()
		srcBm := ba.DataBitmap()
		srcWords := srcBm.Words()
		n := ba.Len()
		copyBitmapBits(dataWords, bitOffset, srcWords, n)
		bitOffset += n
	}

	var validity *bitmap.Bitmap
	if hasNulls {
		validity = concatBitmaps(dfs, colIdx, totalHeight)
	}

	return series.New(name, array.NewBooleanArrayFromBitmap(dataBm, validity))
}

// concatBitmaps merges validity bitmaps from multiple DataFrames for a column.
// Columns without nulls are treated as all-valid.
func concatBitmaps(dfs []*DataFrame, colIdx int, totalHeight int) *bitmap.Bitmap {
	result := bitmap.New(totalHeight) // all bits set
	words := result.Words()
	bitOffset := 0

	for _, df := range dfs {
		c := df.columns[colIdx]
		n := c.Len()
		v := c.Array().Validity()

		if v != nil {
			// Has nulls: copy source bitmap bits
			copyBitmapBits(words, bitOffset, v.Words(), n)
		}
		// If v == nil, all valid — bits already set from bitmap.New
		bitOffset += n
	}

	return result
}

// copyBitmapBits copies n bits from src words into dst words starting at dstBitOffset.
func copyBitmapBits(dst []uint64, dstBitOffset int, src []uint64, n int) {
	if n == 0 {
		return
	}

	dstWord := dstBitOffset / 64
	dstBit := uint(dstBitOffset % 64)

	if dstBit == 0 {
		// Aligned: bulk copy full words, then handle remainder
		fullWords := n / 64
		for i := 0; i < fullWords; i++ {
			dst[dstWord+i] = src[i]
		}
		rem := n % 64
		if rem > 0 {
			mask := (uint64(1) << rem) - 1
			dst[dstWord+fullWords] = (dst[dstWord+fullWords] &^ mask) | (src[fullWords] & mask)
		}
		return
	}

	// Unaligned: shift and merge
	remaining := n
	srcIdx := 0
	for remaining > 0 {
		bits := uint64(0)
		if srcIdx < len(src) {
			bits = src[srcIdx]
		}
		chunk := remaining
		if chunk > 64 {
			chunk = 64
		}

		// Mask to only the bits we need from this source word
		srcMask := uint64(0)
		if chunk >= 64 {
			srcMask = ^uint64(0)
		} else {
			srcMask = (uint64(1) << chunk) - 1
		}
		bits &= srcMask

		// Place lower portion into current dst word
		avail := 64 - dstBit
		if uint(chunk) <= avail {
			// All bits fit in current word
			dstMask := srcMask << dstBit
			dst[dstWord] = (dst[dstWord] &^ dstMask) | (bits << dstBit)
		} else {
			// Split across two dst words
			dst[dstWord] = (dst[dstWord] & ((uint64(1) << dstBit) - 1)) | (bits << dstBit)
			dstWord++
			overflow := bits >> avail
			overflowBits := uint(chunk) - avail
			overflowMask := (uint64(1) << overflowBits) - 1
			dst[dstWord] = (dst[dstWord] &^ overflowMask) | (overflow & overflowMask)
		}

		dstBit += uint(chunk)
		if dstBit >= 64 {
			if uint(chunk) <= avail {
				dstWord++
			}
			dstBit %= 64
		}

		remaining -= chunk
		srcIdx++
	}
}

// ConcatHorizontal concatenates DataFrames side by side. All DataFrames must
// have the same height and no duplicate column names across them. Returns an
// error if heights differ or column names collide.
func ConcatHorizontal(dfs ...*DataFrame) (*DataFrame, error) {
	if len(dfs) == 0 {
		return nil, fmt.Errorf("golars: no DataFrames to concatenate")
	}
	if len(dfs) == 1 {
		return dfs[0].Clone(), nil
	}

	height := dfs[0].Height()
	seen := make(map[string]struct{})
	var allCols []*series.Series

	for i, df := range dfs {
		if df.Height() != height {
			return nil, fmt.Errorf("golars: height mismatch at DataFrame index %d: got %d, expected %d", i, df.Height(), height)
		}
		for _, c := range df.columns {
			if _, exists := seen[c.Name()]; exists {
				return nil, fmt.Errorf("golars: duplicate column name %q during horizontal concat", c.Name())
			}
			seen[c.Name()] = struct{}{}
			allCols = append(allCols, c)
		}
	}

	return New(allCols...)
}
