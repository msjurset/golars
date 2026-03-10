package series

import (
	"bytes"
	"math"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

const (
	htEmpty    = int32(-1)
	fibHash64  = uint64(0x9E3779B97F4A7C15)
	fnvOffset  = uint64(14695981039346656037)
	fnvPrime   = uint64(1099511628211)
)

// nextPow2 returns the smallest power of 2 >= n, minimum 16.
func nextPow2(n int) int {
	if n < 16 {
		return 16
	}
	// round up to power of 2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}

// Unique returns a new Series containing only unique values. Order is preserved
// (first occurrence is kept). Null appears at most once in the result.
func (s *Series) Unique() *Series {
	switch s.dtype {
	case dtype.Int64:
		return uniqueInt64(s)
	case dtype.Float64:
		return uniqueFloat64(s)
	case dtype.Int32:
		return uniqueInt32(s)
	case dtype.UInt64:
		return uniqueUInt64(s)
	case dtype.UInt32:
		return uniqueUInt32(s)
	case dtype.Float32:
		return uniqueFloat32(s)
	case dtype.String:
		return uniqueString(s)
	case dtype.Boolean:
		return uniqueBool(s)
	default:
		return uniqueTypedFallback[int64](s)
	}
}

// --- int64 unique ---

func uniqueInt64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[int64])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		// null-free fast path
		for i := 0; i < n; i++ {
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break // duplicate
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- float64 unique ---

func uniqueFloat64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[float64])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			h := math.Float64bits(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := math.Float64bits(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- int32 unique ---

func uniqueInt32(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[int32])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- uint64 unique ---

func uniqueUInt64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[uint64])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			h := vals[i] * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := vals[i] * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- uint32 unique ---

func uniqueUInt32(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[uint32])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := uint64(vals[i]) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- float32 unique ---

func uniqueFloat32(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[float32])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	vals := ta.Values()
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			h := uint64(math.Float32bits(vals[i])) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := ta.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			h := uint64(math.Float32bits(vals[i])) * fibHash64
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && vals[idx] == vals[i] {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// --- string unique (FNV-1a byte-level hashing) ---

func uniqueString(s *Series) *Series {
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return s
	}
	n := sa.Len()
	if n == 0 {
		return s
	}
	mask := bitmap.NewEmpty(n)

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	indices := make([]int32, cap_)
	hashes := make([]uint64, cap_)
	for i := range indices {
		indices[i] = htEmpty
	}

	if !s.HasNulls() {
		for i := 0; i < n; i++ {
			b := sa.ValueBytes(i)
			h := fnvHashBytes(b)
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && bytes.Equal(sa.ValueBytes(int(idx)), b) {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	} else {
		validity := sa.Validity()
		seenNull := false
		for i := 0; i < n; i++ {
			if validity != nil && !validity.IsSet(i) {
				if !seenNull {
					seenNull = true
					mask.Set(i)
				}
				continue
			}
			b := sa.ValueBytes(i)
			h := fnvHashBytes(b)
			pos := h & hmask
			for {
				idx := indices[pos]
				if idx == htEmpty {
					indices[pos] = int32(i)
					hashes[pos] = h
					mask.Set(i)
					break
				}
				if hashes[pos] == h && bytes.Equal(sa.ValueBytes(int(idx)), b) {
					break
				}
				pos = (pos + 1) & hmask
			}
		}
	}
	return New(s.name, array.FilterString(sa, mask))
}

// fnvHashBytes computes FNV-1a hash over a byte slice.
func fnvHashBytes(b []byte) uint64 {
	h := fnvOffset
	for _, c := range b {
		h ^= uint64(c)
		h *= fnvPrime
	}
	return h
}

// --- boolean unique (unchanged, at most 3 values) ---

func uniqueBool(s *Series) *Series {
	ba, ok := s.arr.(*array.BooleanArray)
	if !ok {
		return s
	}
	n := ba.Len()
	if n == 0 {
		return s
	}
	mask := bitmap.NewEmpty(n)
	hasTrue, hasFalse, hasNull := false, false, false

	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			if !hasNull {
				hasNull = true
				mask.Set(i)
			}
			continue
		}
		v := ba.Value(i)
		if v && !hasTrue {
			hasTrue = true
			mask.Set(i)
		} else if !v && !hasFalse {
			hasFalse = true
			mask.Set(i)
		}
		if hasTrue && hasFalse && hasNull {
			break
		}
	}
	return New(s.name, array.FilterBoolean(ba, mask))
}

// uniqueTypedFallback uses Go map for types not covered by specialized functions.
func uniqueTypedFallback[T comparable](s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[T])
	if !ok {
		return s
	}
	n := ta.Len()
	if n == 0 {
		return s
	}
	seen := make(map[T]struct{}, n/2)
	mask := bitmap.NewEmpty(n)
	seenNull := false

	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			if !seenNull {
				seenNull = true
				mask.Set(i)
			}
			continue
		}
		v := ta.Value(i)
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			mask.Set(i)
		}
	}
	return New(s.name, array.FilterTyped(ta, mask))
}

// IsDuplicated returns a Boolean Series where true indicates the value at that
// position appears more than once in the Series.
func (s *Series) IsDuplicated() *Series {
	switch s.dtype {
	case dtype.Int64:
		return isDuplicatedInt64(s)
	case dtype.Float64:
		return isDuplicatedFloat64(s)
	case dtype.Int32:
		return isDuplicatedInt32(s)
	case dtype.UInt64:
		return isDuplicatedUInt64(s)
	case dtype.UInt32:
		return isDuplicatedUInt32(s)
	case dtype.String:
		return isDuplicatedString(s)
	}
	n := s.Len()
	result := make([]bool, n)
	return NewBoolean(s.name, result)
}

// isDuplicated hash table entry: stores the first row index and count.
type dupEntry struct {
	row   int32
	count int32
}

// --- isDuplicated int64 ---

func isDuplicatedInt64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[int64])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := ta.Len()
	vals := ta.Values()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)  // index into entries
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = ta.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

// --- isDuplicated float64 ---

func isDuplicatedFloat64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[float64])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := ta.Len()
	vals := ta.Values()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = ta.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		h := math.Float64bits(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		h := math.Float64bits(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

// --- isDuplicated int32 ---

func isDuplicatedInt32(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[int32])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := ta.Len()
	vals := ta.Values()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = ta.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

// --- isDuplicated uint64 ---

func isDuplicatedUInt64(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[uint64])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := ta.Len()
	vals := ta.Values()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = ta.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		h := vals[i] * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		h := vals[i] * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

// --- isDuplicated uint32 ---

func isDuplicatedUInt32(s *Series) *Series {
	ta, ok := s.arr.(*array.TypedArray[uint32])
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := ta.Len()
	vals := ta.Values()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = ta.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		h := uint64(vals[i]) * fibHash64
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && vals[entries[ei].row] == vals[i] {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

// --- isDuplicated string (FNV-1a) ---

func isDuplicatedString(s *Series) *Series {
	sa, ok := s.arr.(*array.StringArray)
	if !ok {
		return NewBoolean(s.name, make([]bool, s.Len()))
	}
	n := sa.Len()

	cap_ := nextPow2(n * 2)
	hmask := uint64(cap_ - 1)
	htIndices := make([]int32, cap_)
	htHashes := make([]uint64, cap_)
	for i := range htIndices {
		htIndices[i] = htEmpty
	}
	entries := make([]dupEntry, 0, n)

	nullCount := 0
	hasNulls := s.HasNulls()
	var validity *bitmap.Bitmap
	if hasNulls {
		validity = sa.Validity()
	}

	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			nullCount++
			continue
		}
		b := sa.ValueBytes(i)
		h := fnvHashBytes(b)
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if ei == htEmpty {
				htIndices[pos] = int32(len(entries))
				htHashes[pos] = h
				entries = append(entries, dupEntry{row: int32(i), count: 1})
				break
			}
			if htHashes[pos] == h && bytes.Equal(sa.ValueBytes(int(entries[ei].row)), b) {
				entries[ei].count++
				break
			}
			pos = (pos + 1) & hmask
		}
	}

	result := make([]bool, n)
	nullDup := nullCount > 1
	for i := 0; i < n; i++ {
		if hasNulls && validity != nil && !validity.IsSet(i) {
			result[i] = nullDup
			continue
		}
		b := sa.ValueBytes(i)
		h := fnvHashBytes(b)
		pos := h & hmask
		for {
			ei := htIndices[pos]
			if htHashes[pos] == h && bytes.Equal(sa.ValueBytes(int(entries[ei].row)), b) {
				result[i] = entries[ei].count > 1
				break
			}
			pos = (pos + 1) & hmask
		}
	}
	return NewBoolean(s.name, result)
}

