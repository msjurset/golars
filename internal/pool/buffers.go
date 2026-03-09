package pool

import "sync"

var (
	int64Pool   sync.Pool
	float64Pool sync.Pool
	intPool     sync.Pool
	bytePool    sync.Pool
	int32Pool   sync.Pool
	uint64Pool  sync.Pool
)

// GetInt64Slice returns an int64 slice with length 0 and at least the
// requested capacity. The slice may be recycled from a pool.
func GetInt64Slice(n int) []int64 {
	if v := int64Pool.Get(); v != nil {
		if s := v.([]int64); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]int64, 0, n)
}

// PutInt64Slice returns an int64 slice to the pool for reuse.
func PutInt64Slice(s []int64) {
	s = s[:0]
	int64Pool.Put(s)
}

// GetFloat64Slice returns a float64 slice with length 0 and at least the
// requested capacity. The slice may be recycled from a pool.
func GetFloat64Slice(n int) []float64 {
	if v := float64Pool.Get(); v != nil {
		if s := v.([]float64); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]float64, 0, n)
}

// PutFloat64Slice returns a float64 slice to the pool for reuse.
func PutFloat64Slice(s []float64) {
	s = s[:0]
	float64Pool.Put(s)
}

// GetIntSlice returns an int slice with length 0 and at least the requested
// capacity. The slice may be recycled from a pool.
func GetIntSlice(n int) []int {
	if v := intPool.Get(); v != nil {
		if s := v.([]int); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]int, 0, n)
}

// PutIntSlice returns an int slice to the pool for reuse.
func PutIntSlice(s []int) {
	s = s[:0]
	intPool.Put(s)
}

// GetByteSlice returns a byte slice with length 0 and at least the requested
// capacity. The slice may be recycled from a pool.
func GetByteSlice(n int) []byte {
	if v := bytePool.Get(); v != nil {
		if s := v.([]byte); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]byte, 0, n)
}

// PutByteSlice returns a byte slice to the pool for reuse.
func PutByteSlice(s []byte) {
	s = s[:0]
	bytePool.Put(s)
}

// GetInt32Slice returns an int32 slice with length 0 and at least the
// requested capacity. The slice may be recycled from a pool.
func GetInt32Slice(n int) []int32 {
	if v := int32Pool.Get(); v != nil {
		if s := v.([]int32); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]int32, 0, n)
}

// PutInt32Slice returns an int32 slice to the pool for reuse.
func PutInt32Slice(s []int32) {
	s = s[:0]
	int32Pool.Put(s)
}

// GetUint64Slice returns a uint64 slice with length 0 and at least the
// requested capacity. The slice may be recycled from a pool.
func GetUint64Slice(n int) []uint64 {
	if v := uint64Pool.Get(); v != nil {
		if s := v.([]uint64); cap(s) >= n {
			return s[:0]
		}
	}
	return make([]uint64, 0, n)
}

// PutUint64Slice returns a uint64 slice to the pool for reuse.
func PutUint64Slice(s []uint64) {
	s = s[:0]
	uint64Pool.Put(s)
}
