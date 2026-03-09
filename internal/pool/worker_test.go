package pool

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestParallelDoZero(t *testing.T) {
	called := false
	ParallelDo(0, DefaultThreshold, func(start, end int) {
		called = true
	})
	if called {
		t.Fatal("fn should not be called when n is 0")
	}
}

func TestParallelDoOne(t *testing.T) {
	var count int64
	ParallelDo(1, DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			atomic.AddInt64(&count, 1)
		}
	})
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
}

func TestParallelDoSmall(t *testing.T) {
	n := 100
	visited := make([]int32, n)
	ParallelDo(n, DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			visited[i] = 1
		}
	})
	for i, v := range visited {
		if v != 1 {
			t.Fatalf("element %d not visited", i)
		}
	}
}

func TestParallelDoLarge(t *testing.T) {
	n := 10000
	visited := make([]int32, n)
	ParallelDo(n, DefaultThreshold, func(start, end int) {
		for i := start; i < end; i++ {
			atomic.AddInt32(&visited[i], 1)
		}
	})
	for i, v := range visited {
		if v != 1 {
			t.Fatalf("element %d visited %d times, want exactly 1", i, v)
		}
	}
}

func TestParallelDoBelowThresholdRunsSerially(t *testing.T) {
	// With a very high threshold, work should run in the calling goroutine.
	n := 500
	visited := make([]int32, n)
	ParallelDo(n, n+1, func(start, end int) {
		for i := start; i < end; i++ {
			visited[i] = 1
		}
	})
	for i, v := range visited {
		if v != 1 {
			t.Fatalf("element %d not visited in serial path", i)
		}
	}
}

func TestParallelDoAllElementsVisitedOnce(t *testing.T) {
	n := 5000
	visited := make([]int64, n)
	ParallelDo(n, 1, func(start, end int) {
		for i := start; i < end; i++ {
			atomic.AddInt64(&visited[i], 1)
		}
	})
	for i, v := range visited {
		if v != 1 {
			t.Fatalf("element %d visited %d times", i, v)
		}
	}
}

func TestParallelDoWithErrorNil(t *testing.T) {
	err := ParallelDoWithError(100, DefaultThreshold, func(start, end int) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestParallelDoWithErrorReturnsError(t *testing.T) {
	sentinel := errors.New("test error")
	err := ParallelDoWithError(5000, 1, func(start, end int) error {
		return sentinel
	})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestParallelDoWithErrorZero(t *testing.T) {
	err := ParallelDoWithError(0, DefaultThreshold, func(start, end int) error {
		return errors.New("should not be called")
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestParallelDoWithErrorSerial(t *testing.T) {
	sentinel := errors.New("serial error")
	err := ParallelDoWithError(10, 100, func(start, end int) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error in serial path, got %v", err)
	}
}

func TestParallelCollectSums(t *testing.T) {
	n := 10000
	results := ParallelCollect(n, 1, func(start, end int) int {
		sum := 0
		for i := start; i < end; i++ {
			sum += i
		}
		return sum
	})
	total := 0
	for _, v := range results {
		total += v
	}
	expected := n * (n - 1) / 2
	if total != expected {
		t.Fatalf("expected sum %d, got %d", expected, total)
	}
}

func TestParallelCollectSerial(t *testing.T) {
	results := ParallelCollect(10, 100, func(start, end int) int {
		return end - start
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result in serial path, got %d", len(results))
	}
	if results[0] != 10 {
		t.Fatalf("expected 10, got %d", results[0])
	}
}

func TestParallelCollectZero(t *testing.T) {
	results := ParallelCollect(0, 1, func(start, end int) int {
		return 0
	})
	if results != nil {
		t.Fatalf("expected nil for n=0, got %v", results)
	}
}

func TestChunkSize(t *testing.T) {
	cs := ChunkSize(100)
	if cs < 1 {
		t.Fatalf("chunk size must be >= 1, got %d", cs)
	}
	if cs > 100 {
		t.Fatalf("chunk size %d exceeds n=100", cs)
	}
}
