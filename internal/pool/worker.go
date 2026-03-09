package pool

import (
	"runtime"
	"sync"
)

// DefaultThreshold is the minimum number of elements before parallel execution
// is used. Below this threshold, work is performed serially.
const DefaultThreshold = 1024

// ChunkSize computes the optimal chunk size for splitting n items across
// available processors. It returns at least 1.
func ChunkSize(n int) int {
	procs := runtime.GOMAXPROCS(0)
	if procs <= 0 {
		procs = 1
	}
	size := (n + procs - 1) / procs
	if size < 1 {
		size = 1
	}
	return size
}

// ParallelDo splits the range [0, n) into GOMAXPROCS chunks and invokes fn
// for each chunk in a separate goroutine. If n is less than threshold the work
// is performed serially in the calling goroutine.
func ParallelDo(n int, threshold int, fn func(start, end int)) {
	if n <= 0 {
		return
	}
	if n < threshold {
		fn(0, n)
		return
	}

	chunk := ChunkSize(n)
	var wg sync.WaitGroup

	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			fn(s, e)
		}(start, end)
	}
	wg.Wait()
}

// ParallelDoWithError splits the range [0, n) into GOMAXPROCS chunks and
// invokes fn for each chunk in a separate goroutine. If any invocation returns
// a non-nil error, the first error encountered is returned. If n is less than
// threshold the work is performed serially.
func ParallelDoWithError(n int, threshold int, fn func(start, end int) error) error {
	if n <= 0 {
		return nil
	}
	if n < threshold {
		return fn(0, n)
	}

	chunk := ChunkSize(n)
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		firstErr error
	)

	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			if err := fn(s, e); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(start, end)
	}
	wg.Wait()
	return firstErr
}

// ParallelCollect splits the range [0, n) into GOMAXPROCS chunks, invokes fn
// for each chunk in a separate goroutine, and returns the collected results in
// chunk order. If n is less than threshold the work is performed serially.
func ParallelCollect[T any](n int, threshold int, fn func(start, end int) T) []T {
	if n <= 0 {
		return nil
	}
	if n < threshold {
		return []T{fn(0, n)}
	}

	chunk := ChunkSize(n)
	numChunks := (n + chunk - 1) / chunk
	results := make([]T, numChunks)

	var wg sync.WaitGroup
	idx := 0
	for start := 0; start < n; start += chunk {
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(i, s, e int) {
			defer wg.Done()
			results[i] = fn(s, e)
		}(idx, start, end)
		idx++
	}
	wg.Wait()
	return results
}
