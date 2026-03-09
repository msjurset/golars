package bitmap

import (
	"testing"
)

func TestNewAllSet(t *testing.T) {
	tests := []struct {
		name string
		len  int
	}{
		{"zero length", 0},
		{"single bit", 1},
		{"63 bits", 63},
		{"64 bits", 64},
		{"65 bits", 65},
		{"128 bits", 128},
		{"129 bits", 129},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.len)
			if b.Len() != tt.len {
				t.Errorf("Len() = %d, want %d", b.Len(), tt.len)
			}
			if b.PopCount() != tt.len {
				t.Errorf("PopCount() = %d, want %d", b.PopCount(), tt.len)
			}
			if tt.len > 0 && !b.AllSet() {
				t.Error("AllSet() = false, want true")
			}
			if b.NullCount() != 0 {
				t.Errorf("NullCount() = %d, want 0", b.NullCount())
			}
		})
	}
}

func TestNewEmpty(t *testing.T) {
	tests := []struct {
		name string
		len  int
	}{
		{"zero length", 0},
		{"single bit", 1},
		{"64 bits", 64},
		{"65 bits", 65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewEmpty(tt.len)
			if b.Len() != tt.len {
				t.Errorf("Len() = %d, want %d", b.Len(), tt.len)
			}
			if b.PopCount() != 0 {
				t.Errorf("PopCount() = %d, want 0", b.PopCount())
			}
			if tt.len > 0 && !b.NoneSet() {
				t.Error("NoneSet() = false, want true")
			}
			if b.NullCount() != tt.len {
				t.Errorf("NullCount() = %d, want %d", b.NullCount(), tt.len)
			}
		})
	}
}

func TestSetClearIsSet(t *testing.T) {
	positions := []int{0, 1, 62, 63, 64, 65, 126, 127, 128}
	bitmapLen := 200

	t.Run("set on empty bitmap", func(t *testing.T) {
		for _, pos := range positions {
			b := NewEmpty(bitmapLen)
			b.Set(pos)
			if !b.IsSet(pos) {
				t.Errorf("bit %d not set after Set()", pos)
			}
			if b.PopCount() != 1 {
				t.Errorf("PopCount() = %d after setting bit %d, want 1", b.PopCount(), pos)
			}
		}
	})

	t.Run("clear on full bitmap", func(t *testing.T) {
		for _, pos := range positions {
			b := New(bitmapLen)
			b.Clear(pos)
			if b.IsSet(pos) {
				t.Errorf("bit %d still set after Clear()", pos)
			}
			if b.PopCount() != bitmapLen-1 {
				t.Errorf("PopCount() = %d after clearing bit %d, want %d", b.PopCount(), pos, bitmapLen-1)
			}
		}
	})

	t.Run("set then clear", func(t *testing.T) {
		b := NewEmpty(bitmapLen)
		for _, pos := range positions {
			b.Set(pos)
		}
		if b.PopCount() != len(positions) {
			t.Errorf("PopCount() = %d, want %d", b.PopCount(), len(positions))
		}
		for _, pos := range positions {
			b.Clear(pos)
		}
		if b.PopCount() != 0 {
			t.Errorf("PopCount() = %d after clearing all, want 0", b.PopCount())
		}
	})
}

func TestSetClearPanic(t *testing.T) {
	b := New(10)

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}

	assertPanics("Set(-1)", func() { b.Set(-1) })
	assertPanics("Set(10)", func() { b.Set(10) })
	assertPanics("Clear(-1)", func() { b.Clear(-1) })
	assertPanics("Clear(10)", func() { b.Clear(10) })
	assertPanics("IsSet(-1)", func() { b.IsSet(-1) })
	assertPanics("IsSet(10)", func() { b.IsSet(10) })
}

func TestAndOrNot(t *testing.T) {
	t.Run("AND", func(t *testing.T) {
		tests := []struct {
			name     string
			len      int
			aSet     []int
			bSet     []int
			wantSet  []int
		}{
			{"both empty", 8, nil, nil, nil},
			{"disjoint", 8, []int{0, 2, 4}, []int{1, 3, 5}, nil},
			{"overlap", 8, []int{0, 1, 2}, []int{1, 2, 3}, []int{1, 2}},
			{"all set", 4, []int{0, 1, 2, 3}, []int{0, 1, 2, 3}, []int{0, 1, 2, 3}},
			{"cross word boundary", 128, []int{63, 64}, []int{64, 65}, []int{64}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := NewEmpty(tt.len)
				b := NewEmpty(tt.len)
				for _, i := range tt.aSet {
					a.Set(i)
				}
				for _, i := range tt.bSet {
					b.Set(i)
				}
				result := a.And(b)
				if result.Len() != tt.len {
					t.Errorf("Len() = %d, want %d", result.Len(), tt.len)
				}
				if result.PopCount() != len(tt.wantSet) {
					t.Errorf("PopCount() = %d, want %d", result.PopCount(), len(tt.wantSet))
				}
				for _, i := range tt.wantSet {
					if !result.IsSet(i) {
						t.Errorf("bit %d not set in AND result", i)
					}
				}
			})
		}
	})

	t.Run("OR", func(t *testing.T) {
		a := NewEmpty(128)
		b := NewEmpty(128)
		a.Set(0)
		a.Set(63)
		b.Set(64)
		b.Set(127)
		result := a.Or(b)
		if result.PopCount() != 4 {
			t.Errorf("PopCount() = %d, want 4", result.PopCount())
		}
		for _, i := range []int{0, 63, 64, 127} {
			if !result.IsSet(i) {
				t.Errorf("bit %d not set in OR result", i)
			}
		}
	})

	t.Run("NOT", func(t *testing.T) {
		tests := []struct {
			name string
			len  int
		}{
			{"single bit", 1},
			{"63 bits", 63},
			{"64 bits", 64},
			{"65 bits", 65},
			{"128 bits", 128},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b := New(tt.len)
				notB := b.Not()
				if notB.PopCount() != 0 {
					t.Errorf("NOT of all-set: PopCount() = %d, want 0", notB.PopCount())
				}
				if notB.Len() != tt.len {
					t.Errorf("NOT Len() = %d, want %d", notB.Len(), tt.len)
				}

				e := NewEmpty(tt.len)
				notE := e.Not()
				if notE.PopCount() != tt.len {
					t.Errorf("NOT of all-clear: PopCount() = %d, want %d", notE.PopCount(), tt.len)
				}
			})
		}
	})
}

func TestPopCountAccuracy(t *testing.T) {
	tests := []struct {
		name    string
		len     int
		setBits []int
		want    int
	}{
		{"none set", 64, nil, 0},
		{"all set", 64, nil, 64}, // will use New()
		{"one set", 100, []int{50}, 1},
		{"every other", 8, []int{0, 2, 4, 6}, 4},
		{"across words", 130, []int{0, 63, 64, 127, 128, 129}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b *Bitmap
			if tt.name == "all set" {
				b = New(tt.len)
			} else {
				b = NewEmpty(tt.len)
				for _, i := range tt.setBits {
					b.Set(i)
				}
			}
			if b.PopCount() != tt.want {
				t.Errorf("PopCount() = %d, want %d", b.PopCount(), tt.want)
			}
		})
	}
}

func TestNullCount(t *testing.T) {
	b := New(100)
	b.Clear(10)
	b.Clear(50)
	b.Clear(99)
	if b.NullCount() != 3 {
		t.Errorf("NullCount() = %d, want 3", b.NullCount())
	}
}

func TestSlice(t *testing.T) {
	tests := []struct {
		name      string
		srcLen    int
		srcSet    []int
		start     int
		end       int
		wantLen   int
		wantPop   int
		wantSet   []int
	}{
		{"empty slice", 10, nil, 0, 0, 0, 0, nil},
		{"full slice", 8, []int{0, 3, 7}, 0, 8, 8, 3, []int{0, 3, 7}},
		{"prefix", 8, []int{0, 1, 2, 3, 4, 5, 6, 7}, 0, 4, 4, 4, []int{0, 1, 2, 3}},
		{"suffix", 8, []int{0, 1, 2, 3, 4, 5, 6, 7}, 4, 8, 4, 4, []int{0, 1, 2, 3}},
		{"middle", 10, []int{3, 4, 5, 6}, 3, 7, 4, 4, []int{0, 1, 2, 3}},
		{"cross word boundary", 128, []int{62, 63, 64, 65}, 62, 66, 4, 4, []int{0, 1, 2, 3}},
		{"single bit set", 100, []int{50}, 50, 51, 1, 1, []int{0}},
		{"single bit unset", 100, nil, 50, 51, 1, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewEmpty(tt.srcLen)
			for _, i := range tt.srcSet {
				src.Set(i)
			}
			result := src.Slice(tt.start, tt.end)
			if result.Len() != tt.wantLen {
				t.Errorf("Len() = %d, want %d", result.Len(), tt.wantLen)
			}
			if result.PopCount() != tt.wantPop {
				t.Errorf("PopCount() = %d, want %d", result.PopCount(), tt.wantPop)
			}
			for _, i := range tt.wantSet {
				if !result.IsSet(i) {
					t.Errorf("bit %d not set in slice result", i)
				}
			}
		})
	}
}

func TestSlicePanic(t *testing.T) {
	b := New(10)
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}
	assertPanics("start > end", func() { b.Slice(5, 3) })
	assertPanics("end > len", func() { b.Slice(0, 11) })
	assertPanics("negative start", func() { b.Slice(-1, 5) })
}

func TestCloneIndependence(t *testing.T) {
	original := New(128)
	original.Clear(10)
	original.Clear(64)

	clone := original.Clone()

	// Verify they match initially.
	if original.PopCount() != clone.PopCount() {
		t.Errorf("clone PopCount() = %d, original = %d", clone.PopCount(), original.PopCount())
	}

	// Modify clone, check original is unaffected.
	clone.Clear(0)
	clone.Clear(1)
	if original.IsSet(0) != true {
		t.Error("modifying clone affected original bit 0")
	}
	if original.IsSet(1) != true {
		t.Error("modifying clone affected original bit 1")
	}
	if original.PopCount() != 126 {
		t.Errorf("original PopCount() = %d, want 126", original.PopCount())
	}
	if clone.PopCount() != 124 {
		t.Errorf("clone PopCount() = %d, want 124", clone.PopCount())
	}

	// Modify original, check clone is unaffected.
	original.Set(10)
	if clone.IsSet(10) {
		t.Error("modifying original affected clone bit 10")
	}
}

func TestAllSetNoneSet(t *testing.T) {
	tests := []struct {
		name    string
		len     int
		allSet  bool
		noneSet bool
	}{
		{"zero length all set", 0, true, true},
		{"all set 1", 1, true, true},
		{"all set 64", 64, true, true},
		{"all set 65", 65, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.len)
			if b.AllSet() != tt.allSet {
				t.Errorf("AllSet() = %v, want %v", b.AllSet(), tt.allSet)
			}
			if tt.len > 0 && b.NoneSet() {
				t.Error("NoneSet() = true on all-set bitmap, want false")
			}
			e := NewEmpty(tt.len)
			if e.NoneSet() != tt.noneSet {
				t.Errorf("NoneSet() = %v, want %v", e.NoneSet(), tt.noneSet)
			}
			if tt.len > 0 && e.AllSet() {
				t.Error("AllSet() = true on empty bitmap, want false")
			}
		})
	}
}

func TestSetRange(t *testing.T) {
	tests := []struct {
		name    string
		len     int
		start   int
		end     int
		wantPop int
	}{
		{"empty range", 64, 5, 5, 0},
		{"single bit", 64, 0, 1, 1},
		{"within word", 64, 10, 20, 10},
		{"full word", 64, 0, 64, 64},
		{"cross boundary", 128, 60, 68, 8},
		{"multi word", 200, 0, 200, 200},
		{"partial last word", 65, 0, 65, 65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewEmpty(tt.len)
			b.SetRange(tt.start, tt.end)
			if b.PopCount() != tt.wantPop {
				t.Errorf("PopCount() = %d, want %d", b.PopCount(), tt.wantPop)
			}
			// Verify each bit in range is set.
			for i := tt.start; i < tt.end; i++ {
				if !b.IsSet(i) {
					t.Errorf("bit %d not set after SetRange(%d, %d)", i, tt.start, tt.end)
					break
				}
			}
			// Verify bits outside range are still clear.
			for i := 0; i < tt.start; i++ {
				if b.IsSet(i) {
					t.Errorf("bit %d set outside SetRange(%d, %d)", i, tt.start, tt.end)
					break
				}
			}
			for i := tt.end; i < tt.len; i++ {
				if b.IsSet(i) {
					t.Errorf("bit %d set outside SetRange(%d, %d)", i, tt.start, tt.end)
					break
				}
			}
		})
	}
}

func TestClearRange(t *testing.T) {
	tests := []struct {
		name      string
		len       int
		start     int
		end       int
		wantClear int
	}{
		{"empty range", 64, 5, 5, 0},
		{"single bit", 64, 0, 1, 1},
		{"within word", 64, 10, 20, 10},
		{"cross boundary", 128, 60, 68, 8},
		{"full range", 100, 0, 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.len)
			b.ClearRange(tt.start, tt.end)
			if b.NullCount() != tt.wantClear {
				t.Errorf("NullCount() = %d, want %d", b.NullCount(), tt.wantClear)
			}
			for i := tt.start; i < tt.end; i++ {
				if b.IsSet(i) {
					t.Errorf("bit %d still set after ClearRange(%d, %d)", i, tt.start, tt.end)
					break
				}
			}
		})
	}
}

func TestGrow(t *testing.T) {
	t.Run("grow within same word", func(t *testing.T) {
		b := New(10)
		b.Grow(20)
		if b.Len() != 20 {
			t.Errorf("Len() = %d, want 20", b.Len())
		}
		// Original bits should still be set.
		for i := 0; i < 10; i++ {
			if !b.IsSet(i) {
				t.Errorf("original bit %d cleared after Grow", i)
			}
		}
		// New bits should be clear.
		for i := 10; i < 20; i++ {
			if b.IsSet(i) {
				t.Errorf("new bit %d set after Grow", i)
			}
		}
	})

	t.Run("grow across words", func(t *testing.T) {
		b := New(60)
		b.Grow(130)
		if b.Len() != 130 {
			t.Errorf("Len() = %d, want 130", b.Len())
		}
		if b.PopCount() != 60 {
			t.Errorf("PopCount() = %d, want 60", b.PopCount())
		}
	})

	t.Run("grow no-op", func(t *testing.T) {
		b := New(100)
		b.Grow(50)
		if b.Len() != 100 {
			t.Errorf("Len() = %d, want 100 (no-op)", b.Len())
		}
	})

	t.Run("grow from zero", func(t *testing.T) {
		b := NewEmpty(0)
		b.Grow(10)
		if b.Len() != 10 {
			t.Errorf("Len() = %d, want 10", b.Len())
		}
		if b.PopCount() != 0 {
			t.Errorf("PopCount() = %d, want 0", b.PopCount())
		}
	})
}

func TestEdgeCaseEmptyBitmap(t *testing.T) {
	b := New(0)
	if b.Len() != 0 {
		t.Errorf("Len() = %d, want 0", b.Len())
	}
	if b.PopCount() != 0 {
		t.Errorf("PopCount() = %d, want 0", b.PopCount())
	}
	if !b.AllSet() {
		t.Error("AllSet() = false for empty bitmap, want true")
	}
	if !b.NoneSet() {
		t.Error("NoneSet() = false for empty bitmap, want true")
	}

	clone := b.Clone()
	if clone.Len() != 0 {
		t.Errorf("clone Len() = %d, want 0", clone.Len())
	}

	s := b.Slice(0, 0)
	if s.Len() != 0 {
		t.Errorf("slice Len() = %d, want 0", s.Len())
	}
}

func TestExactly64Bits(t *testing.T) {
	b := New(64)
	if b.PopCount() != 64 {
		t.Errorf("PopCount() = %d, want 64", b.PopCount())
	}
	b.Clear(0)
	b.Clear(63)
	if b.PopCount() != 62 {
		t.Errorf("PopCount() = %d, want 62", b.PopCount())
	}
	notB := b.Not()
	if notB.PopCount() != 2 {
		t.Errorf("NOT PopCount() = %d, want 2", notB.PopCount())
	}
	if !notB.IsSet(0) || !notB.IsSet(63) {
		t.Error("NOT did not flip bits 0 and 63")
	}
}

func TestAndOrMismatchedLengthPanics(t *testing.T) {
	a := New(10)
	b := New(20)
	defer func() {
		if r := recover(); r == nil {
			t.Error("And with mismatched lengths: expected panic")
		}
	}()
	a.And(b)
}

func TestOrMismatchedLengthPanics(t *testing.T) {
	a := New(10)
	b := New(20)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Or with mismatched lengths: expected panic")
		}
	}()
	a.Or(b)
}
