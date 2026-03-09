package pool

import "testing"

func TestGetPutInt64Slice(t *testing.T) {
	s := GetInt64Slice(128)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 128 {
		t.Fatalf("expected cap >= 128, got %d", cap(s))
	}
	s = append(s, 1, 2, 3)
	PutInt64Slice(s)

	// After put, a subsequent get should return a pooled slice.
	s2 := GetInt64Slice(64)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetPutFloat64Slice(t *testing.T) {
	s := GetFloat64Slice(256)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 256 {
		t.Fatalf("expected cap >= 256, got %d", cap(s))
	}
	s = append(s, 3.14)
	PutFloat64Slice(s)

	s2 := GetFloat64Slice(128)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetPutIntSlice(t *testing.T) {
	s := GetIntSlice(64)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 64 {
		t.Fatalf("expected cap >= 64, got %d", cap(s))
	}
	s = append(s, 42)
	PutIntSlice(s)

	s2 := GetIntSlice(32)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetPutByteSlice(t *testing.T) {
	s := GetByteSlice(512)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 512 {
		t.Fatalf("expected cap >= 512, got %d", cap(s))
	}
	s = append(s, 0xFF)
	PutByteSlice(s)

	s2 := GetByteSlice(256)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetPutInt32Slice(t *testing.T) {
	s := GetInt32Slice(100)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 100 {
		t.Fatalf("expected cap >= 100, got %d", cap(s))
	}
	s = append(s, 7)
	PutInt32Slice(s)

	s2 := GetInt32Slice(50)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetPutUint64Slice(t *testing.T) {
	s := GetUint64Slice(200)
	if len(s) != 0 {
		t.Fatalf("expected len 0, got %d", len(s))
	}
	if cap(s) < 200 {
		t.Fatalf("expected cap >= 200, got %d", cap(s))
	}
	s = append(s, 99)
	PutUint64Slice(s)

	s2 := GetUint64Slice(100)
	if len(s2) != 0 {
		t.Fatalf("expected len 0 after round-trip, got %d", len(s2))
	}
}

func TestGetSliceLargerThanPooled(t *testing.T) {
	// Put a small slice, then request a larger capacity.
	small := make([]int64, 0, 16)
	PutInt64Slice(small)

	large := GetInt64Slice(1024)
	if len(large) != 0 {
		t.Fatalf("expected len 0, got %d", len(large))
	}
	if cap(large) < 1024 {
		t.Fatalf("expected cap >= 1024, got %d", cap(large))
	}
}
