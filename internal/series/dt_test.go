package series

import (
	"testing"
	"time"

	"github.com/msjurset/golars/internal/array"
)

func TestDtAccessorYear(t *testing.T) {
	// 2024-01-15 = days since epoch
	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	days := int32(d.Unix() / 86400)
	s := NewDate("d", []int32{days})

	dt := s.Dt()
	if dt == nil {
		t.Fatal("expected non-nil DtAccessor")
	}

	year := dt.Year()
	v, ok := year.GetInt32(0)
	if !ok || v != 2024 {
		t.Errorf("year: got %d, want 2024", v)
	}
}

func TestDtAccessorComponents(t *testing.T) {
	// 2024-03-15 12:30:45 UTC
	d := time.Date(2024, 3, 15, 12, 30, 45, 0, time.UTC)
	us := d.UnixMicro()
	s := NewDateTime("dt", []int64{us})

	dt := s.Dt()
	if dt == nil {
		t.Fatal("expected non-nil DtAccessor")
	}

	tests := []struct {
		name string
		fn   func() *Series
		want int32
	}{
		{"year", dt.Year, 2024},
		{"month", dt.Month, 3},
		{"day", dt.Day, 15},
		{"hour", dt.Hour, 12},
		{"minute", dt.Minute, 30},
		{"second", dt.Second, 45},
		{"quarter", dt.Quarter, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			v, ok := result.GetInt32(0)
			if !ok || v != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, v, tt.want)
			}
		})
	}
}

func TestDtAccessorTruncate(t *testing.T) {
	// 2024-03-15 14:30:00
	d := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	us := d.UnixMicro()
	s := NewDateTime("dt", []int64{us})

	dt := s.Dt()
	truncated := dt.Truncate("1d")

	// Should be 2024-03-15 00:00:00
	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).UnixMicro()
	if ta, ok := truncated.Array().(*array.TypedArray[int64]); ok {
		got := ta.Value(0)
		if got != expected {
			t.Errorf("truncate 1d: got %d, want %d", got, expected)
		}
	} else {
		t.Fatal("expected int64 array")
	}
}

func TestDtAccessorStrftime(t *testing.T) {
	d := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	days := int32(d.Unix() / 86400)
	s := NewDate("d", []int32{days})

	dt := s.Dt()
	formatted := dt.Strftime("2006-01-02")
	v, ok := formatted.GetString(0)
	if !ok || v != "2024-03-15" {
		t.Errorf("strftime: got %q, want %q", v, "2024-03-15")
	}
}

func TestDtAccessorOffsetBy(t *testing.T) {
	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	days := int32(d.Unix() / 86400)
	s := NewDate("d", []int32{days})

	dt := s.Dt()
	offset := dt.OffsetBy("1mo")

	// Should be 2024-02-15
	expected := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	expectedDays := int32(expected.Unix() / 86400)

	if ta, ok := offset.Array().(*array.TypedArray[int32]); ok {
		got := ta.Value(0)
		if got != expectedDays {
			t.Errorf("offset_by 1mo: got %d days, want %d days", got, expectedDays)
		}
	} else {
		t.Fatal("expected int32 array")
	}
}

func TestDtNonTemporalReturnsNil(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3})
	if s.Dt() != nil {
		t.Error("expected nil DtAccessor for non-temporal series")
	}
}
