package series

import (
	"math"
	"strings"
	"testing"

	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNewInt64(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3})
	if s.Name() != "x" {
		t.Errorf("Name: got %q, want %q", s.Name(), "x")
	}
	if s.DataType() != dtype.Int64 {
		t.Errorf("DataType: got %v, want %v", s.DataType(), dtype.Int64)
	}
	if s.Len() != 3 {
		t.Errorf("Len: got %d, want 3", s.Len())
	}
}

func TestNewFloat64(t *testing.T) {
	s := NewFloat64("y", []float64{1.5, 2.5})
	if s.DataType() != dtype.Float64 {
		t.Errorf("DataType: got %v, want %v", s.DataType(), dtype.Float64)
	}
	if s.Len() != 2 {
		t.Errorf("Len: got %d, want 2", s.Len())
	}
}

func TestNewString(t *testing.T) {
	s := NewString("s", []string{"a", "b", "c"})
	if s.DataType() != dtype.String {
		t.Errorf("DataType: got %v, want %v", s.DataType(), dtype.String)
	}
	if s.Len() != 3 {
		t.Errorf("Len: got %d, want 3", s.Len())
	}
}

func TestNewBoolean(t *testing.T) {
	s := NewBoolean("b", []bool{true, false, true})
	if s.DataType() != dtype.Boolean {
		t.Errorf("DataType: got %v, want %v", s.DataType(), dtype.Boolean)
	}
	if s.Len() != 3 {
		t.Errorf("Len: got %d, want 3", s.Len())
	}
}

// ---------------------------------------------------------------------------
// Construction with validity
// ---------------------------------------------------------------------------

func TestNewInt64WithValidity(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true})
	if !s.HasNulls() {
		t.Error("expected HasNulls to be true")
	}
	if s.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", s.NullCount())
	}
	if !s.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if !s.IsValid(0) || !s.IsValid(2) {
		t.Error("expected indices 0, 2 to be valid")
	}
}

func TestNewFloat64WithValidity(t *testing.T) {
	s := NewFloat64WithValidity("y", []float64{1.0, 0, 3.0}, []bool{true, false, true})
	if s.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", s.NullCount())
	}
}

func TestNewStringWithValidity(t *testing.T) {
	s := NewStringWithValidity("s", []string{"a", "", "c"}, []bool{true, false, true})
	if s.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", s.NullCount())
	}
}

func TestNewBooleanWithValidity(t *testing.T) {
	s := NewBooleanWithValidity("b", []bool{true, false, true}, []bool{true, false, true})
	if s.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", s.NullCount())
	}
}

// ---------------------------------------------------------------------------
// Accessors: Name, DataType, Len, IsNull, IsValid, HasNulls, NullCount
// ---------------------------------------------------------------------------

func TestAccessors_NoNulls(t *testing.T) {
	s := NewInt64("col", []int64{10, 20, 30})
	if s.Name() != "col" {
		t.Errorf("Name: got %q", s.Name())
	}
	if s.Len() != 3 {
		t.Errorf("Len: got %d", s.Len())
	}
	if s.HasNulls() {
		t.Error("HasNulls should be false")
	}
	if s.NullCount() != 0 {
		t.Errorf("NullCount: got %d", s.NullCount())
	}
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			t.Errorf("IsNull(%d) should be false", i)
		}
		if !s.IsValid(i) {
			t.Errorf("IsValid(%d) should be true", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Get methods
// ---------------------------------------------------------------------------

func TestGetInt64(t *testing.T) {
	s := NewInt64("x", []int64{10, 20, 30})
	tests := []struct {
		idx  int
		want int64
	}{
		{0, 10},
		{1, 20},
		{2, 30},
	}
	for _, tt := range tests {
		got, ok := s.GetInt64(tt.idx)
		if !ok {
			t.Errorf("GetInt64(%d): not ok", tt.idx)
		}
		if got != tt.want {
			t.Errorf("GetInt64(%d): got %d, want %d", tt.idx, got, tt.want)
		}
	}
}

func TestGetInt64_Null(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{10, 0, 30}, []bool{true, false, true})
	_, ok := s.GetInt64(1)
	if ok {
		t.Error("GetInt64 of null should return false")
	}
}

func TestGetFloat64(t *testing.T) {
	s := NewFloat64("y", []float64{1.5, 2.5})
	got, ok := s.GetFloat64(0)
	if !ok || got != 1.5 {
		t.Errorf("GetFloat64(0): got %f, ok=%v", got, ok)
	}
}

func TestGetString(t *testing.T) {
	s := NewString("s", []string{"hello", "world"})
	got, ok := s.GetString(1)
	if !ok || got != "world" {
		t.Errorf("GetString(1): got %q, ok=%v", got, ok)
	}
}

func TestGetBool(t *testing.T) {
	s := NewBoolean("b", []bool{true, false})
	got, ok := s.GetBool(0)
	if !ok || !got {
		t.Errorf("GetBool(0): got %v, ok=%v", got, ok)
	}
	got, ok = s.GetBool(1)
	if !ok || got {
		t.Errorf("GetBool(1): got %v, ok=%v", got, ok)
	}
}

// ---------------------------------------------------------------------------
// Slice, Head, Tail
// ---------------------------------------------------------------------------

func TestSlice(t *testing.T) {
	s := NewInt64("x", []int64{10, 20, 30, 40, 50})
	sliced := s.Slice(1, 4)
	if sliced.Len() != 3 {
		t.Fatalf("Slice Len: got %d, want 3", sliced.Len())
	}
	if sliced.Name() != "x" {
		t.Errorf("Slice Name: got %q, want %q", sliced.Name(), "x")
	}
	v, ok := sliced.GetInt64(0)
	if !ok || v != 20 {
		t.Errorf("Slice GetInt64(0): got %d, ok=%v", v, ok)
	}
}

func TestHead(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3, 4, 5})
	h := s.Head(3)
	if h.Len() != 3 {
		t.Fatalf("Head Len: got %d, want 3", h.Len())
	}
	v, _ := h.GetInt64(2)
	if v != 3 {
		t.Errorf("Head GetInt64(2): got %d, want 3", v)
	}
}

func TestHead_LargerThanLen(t *testing.T) {
	s := NewInt64("x", []int64{1, 2})
	h := s.Head(10)
	if h.Len() != 2 {
		t.Errorf("Head with n>Len: got %d, want 2", h.Len())
	}
}

func TestTail(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3, 4, 5})
	tl := s.Tail(3)
	if tl.Len() != 3 {
		t.Fatalf("Tail Len: got %d, want 3", tl.Len())
	}
	v, _ := tl.GetInt64(0)
	if v != 3 {
		t.Errorf("Tail GetInt64(0): got %d, want 3", v)
	}
}

func TestTail_LargerThanLen(t *testing.T) {
	s := NewInt64("x", []int64{1, 2})
	tl := s.Tail(10)
	if tl.Len() != 2 {
		t.Errorf("Tail with n>Len: got %d, want 2", tl.Len())
	}
}

// ---------------------------------------------------------------------------
// Rename, Equal
// ---------------------------------------------------------------------------

func TestRename(t *testing.T) {
	s := NewInt64("old", []int64{1, 2, 3})
	r := s.Rename("new")
	if r.Name() != "new" {
		t.Errorf("Rename Name: got %q, want %q", r.Name(), "new")
	}
	// Original unchanged
	if s.Name() != "old" {
		t.Errorf("Original Name: got %q, want %q", s.Name(), "old")
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b *Series
		want bool
	}{
		{
			name: "same int64",
			a:    NewInt64("x", []int64{1, 2, 3}),
			b:    NewInt64("x", []int64{1, 2, 3}),
			want: true,
		},
		{
			name: "different values",
			a:    NewInt64("x", []int64{1, 2, 3}),
			b:    NewInt64("x", []int64{1, 2, 4}),
			want: false,
		},
		{
			name: "different names",
			a:    NewInt64("x", []int64{1, 2}),
			b:    NewInt64("y", []int64{1, 2}),
			want: false,
		},
		{
			name: "different lengths",
			a:    NewInt64("x", []int64{1, 2}),
			b:    NewInt64("x", []int64{1, 2, 3}),
			want: false,
		},
		{
			name: "same float64",
			a:    NewFloat64("f", []float64{1.0, 2.0}),
			b:    NewFloat64("f", []float64{1.0, 2.0}),
			want: true,
		},
		{
			name: "same string",
			a:    NewString("s", []string{"a", "b"}),
			b:    NewString("s", []string{"a", "b"}),
			want: true,
		},
		{
			name: "same boolean",
			a:    NewBoolean("b", []bool{true, false}),
			b:    NewBoolean("b", []bool{true, false}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqual_WithNulls(t *testing.T) {
	a := NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true})
	b := NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true})
	if !a.Equal(b) {
		t.Error("Equal: same data with same nulls should be equal")
	}

	c := NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, true, true})
	if a.Equal(c) {
		t.Error("Equal: different null patterns should not be equal")
	}
}

// ---------------------------------------------------------------------------
// String() output
// ---------------------------------------------------------------------------

func TestString_Output(t *testing.T) {
	s := NewInt64("col", []int64{10, 20, 30})
	out := s.String()
	if !strings.Contains(out, "'col'") {
		t.Error("String: missing series name")
	}
	if !strings.Contains(out, "Int64") {
		t.Error("String: missing data type")
	}
	if !strings.Contains(out, "10") {
		t.Error("String: missing value 10")
	}
}

func TestString_WithNulls(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true})
	out := s.String()
	if !strings.Contains(out, "null") {
		t.Error("String: missing null representation")
	}
}

// ---------------------------------------------------------------------------
// Aggregations
// ---------------------------------------------------------------------------

func TestSeries_Sum(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3, 4, 5})
	got, ok := s.Sum()
	if !ok {
		t.Fatal("Sum returned not ok")
	}
	if got != 15.0 {
		t.Errorf("Sum: got %f, want 15.0", got)
	}
}

func TestSeries_Sum_Float64(t *testing.T) {
	s := NewFloat64("x", []float64{1.5, 2.5, 3.0})
	got, ok := s.Sum()
	if !ok {
		t.Fatal("Sum returned not ok")
	}
	if got != 7.0 {
		t.Errorf("Sum: got %f, want 7.0", got)
	}
}

func TestSeries_Mean(t *testing.T) {
	s := NewFloat64("x", []float64{2.0, 4.0, 6.0})
	got, ok := s.Mean()
	if !ok {
		t.Fatal("Mean returned not ok")
	}
	if got != 4.0 {
		t.Errorf("Mean: got %f, want 4.0", got)
	}
}

func TestSeries_Min(t *testing.T) {
	s := NewInt64("x", []int64{5, 1, 3})
	got, ok := s.Min()
	if !ok {
		t.Fatal("Min returned not ok")
	}
	if got != 1.0 {
		t.Errorf("Min: got %f, want 1.0", got)
	}
}

func TestSeries_Max(t *testing.T) {
	s := NewInt64("x", []int64{5, 1, 3})
	got, ok := s.Max()
	if !ok {
		t.Fatal("Max returned not ok")
	}
	if got != 5.0 {
		t.Errorf("Max: got %f, want 5.0", got)
	}
}

func TestSeries_Std(t *testing.T) {
	s := NewFloat64("x", []float64{2, 4, 4, 4, 5, 5, 7, 9})
	got, ok := s.Std()
	if !ok {
		t.Fatal("Std returned not ok")
	}
	// Sample std with ddof=1
	// pop var = 4.0, sample var = 4.0 * 8/7 = 32/7 ~= 4.5714
	// sample std = sqrt(32/7) ~= 2.1381
	if math.Abs(got-math.Sqrt(32.0/7.0)) > 1e-10 {
		t.Errorf("Std: got %f, want %f", got, math.Sqrt(32.0/7.0))
	}
}

func TestSeries_Var(t *testing.T) {
	s := NewFloat64("x", []float64{2, 4, 4, 4, 5, 5, 7, 9})
	got, ok := s.Var()
	if !ok {
		t.Fatal("Var returned not ok")
	}
	if math.Abs(got-32.0/7.0) > 1e-10 {
		t.Errorf("Var: got %f, want %f", got, 32.0/7.0)
	}
}

func TestSeries_Count(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 3, 0, 5}, []bool{true, false, true, false, true})
	if got := s.Count(); got != 3 {
		t.Errorf("Count: got %d, want 3", got)
	}
}

func TestSeries_NUnique(t *testing.T) {
	tests := []struct {
		name string
		s    *Series
		want int
	}{
		{
			name: "int64 distinct",
			s:    NewInt64("x", []int64{1, 2, 2, 3, 3, 3}),
			want: 3,
		},
		{
			name: "float64 distinct",
			s:    NewFloat64("y", []float64{1.0, 1.0, 2.0}),
			want: 2,
		},
		{
			name: "string distinct",
			s:    NewString("s", []string{"a", "b", "a", "c"}),
			want: 3,
		},
		{
			name: "boolean distinct",
			s:    NewBoolean("b", []bool{true, false, true}),
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.NUnique(); got != tt.want {
				t.Errorf("NUnique: got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSeries_NUnique_WithNulls(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 1, 0}, []bool{true, false, true, false})
	// Unique non-null values: {1}
	if got := s.NUnique(); got != 1 {
		t.Errorf("NUnique with nulls: got %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// Sort, ArgSort, ArgMin, ArgMax
// ---------------------------------------------------------------------------

func TestSeries_Sort(t *testing.T) {
	s := NewInt64("x", []int64{30, 10, 20})
	sorted := s.Sort(false)
	want := []int64{10, 20, 30}
	for i, w := range want {
		v, ok := sorted.GetInt64(i)
		if !ok || v != w {
			t.Errorf("Sort asc [%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_Sort_Descending(t *testing.T) {
	s := NewInt64("x", []int64{30, 10, 20})
	sorted := s.Sort(true)
	want := []int64{30, 20, 10}
	for i, w := range want {
		v, ok := sorted.GetInt64(i)
		if !ok || v != w {
			t.Errorf("Sort desc [%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_ArgSort(t *testing.T) {
	s := NewInt64("x", []int64{30, 10, 20})
	indices := s.ArgSort(false)
	want := []int{1, 2, 0}
	for i, w := range want {
		if indices[i] != w {
			t.Errorf("ArgSort[%d]: got %d, want %d", i, indices[i], w)
		}
	}
}

func TestSeries_ArgMin(t *testing.T) {
	s := NewInt64("x", []int64{30, 10, 20})
	idx, ok := s.ArgMin()
	if !ok {
		t.Fatal("ArgMin returned not ok")
	}
	if idx != 1 {
		t.Errorf("ArgMin: got %d, want 1", idx)
	}
}

func TestSeries_ArgMax(t *testing.T) {
	s := NewInt64("x", []int64{30, 10, 20})
	idx, ok := s.ArgMax()
	if !ok {
		t.Fatal("ArgMax returned not ok")
	}
	if idx != 0 {
		t.Errorf("ArgMax: got %d, want 0", idx)
	}
}

// ---------------------------------------------------------------------------
// Unique, IsDuplicated
// ---------------------------------------------------------------------------

func TestSeries_Unique(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 2, 3, 3, 3})
	u := s.Unique()
	if u.Len() != 3 {
		t.Fatalf("Unique Len: got %d, want 3", u.Len())
	}
	// First occurrences: 1, 2, 3
	want := []int64{1, 2, 3}
	for i, w := range want {
		v, ok := u.GetInt64(i)
		if !ok || v != w {
			t.Errorf("Unique[%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_Unique_String(t *testing.T) {
	s := NewString("s", []string{"a", "b", "a", "c", "b"})
	u := s.Unique()
	if u.Len() != 3 {
		t.Fatalf("Unique String Len: got %d, want 3", u.Len())
	}
}

func TestSeries_Unique_WithNulls(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 1, 0}, []bool{true, false, true, false})
	u := s.Unique()
	// Should have: 1 (first occurrence), null (first occurrence)
	if u.Len() != 2 {
		t.Fatalf("Unique with nulls Len: got %d, want 2", u.Len())
	}
}

func TestSeries_IsDuplicated(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 2, 3})
	dup := s.IsDuplicated()
	if dup.Len() != 4 {
		t.Fatalf("IsDuplicated Len: got %d, want 4", dup.Len())
	}
	want := []bool{false, true, true, false}
	for i, w := range want {
		v, ok := dup.GetBool(i)
		if !ok {
			t.Fatalf("IsDuplicated GetBool(%d): not ok", i)
		}
		if v != w {
			t.Errorf("IsDuplicated[%d]: got %v, want %v", i, v, w)
		}
	}
}

func TestSeries_IsDuplicated_String(t *testing.T) {
	s := NewString("s", []string{"a", "b", "a"})
	dup := s.IsDuplicated()
	v0, _ := dup.GetBool(0)
	v1, _ := dup.GetBool(1)
	v2, _ := dup.GetBool(2)
	if !v0 || v1 || !v2 {
		t.Errorf("IsDuplicated string: got [%v,%v,%v], want [true,false,true]", v0, v1, v2)
	}
}

// ---------------------------------------------------------------------------
// DropNulls, FillNull
// ---------------------------------------------------------------------------

func TestSeries_DropNulls(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 3, 0, 5}, []bool{true, false, true, false, true})
	dropped := s.DropNulls()
	if dropped.Len() != 3 {
		t.Fatalf("DropNulls Len: got %d, want 3", dropped.Len())
	}
	want := []int64{1, 3, 5}
	for i, w := range want {
		v, ok := dropped.GetInt64(i)
		if !ok || v != w {
			t.Errorf("DropNulls[%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_DropNulls_NoNulls(t *testing.T) {
	s := NewInt64("x", []int64{1, 2, 3})
	dropped := s.DropNulls()
	// Should return the same series when no nulls
	if dropped.Len() != 3 {
		t.Errorf("DropNulls no nulls Len: got %d, want 3", dropped.Len())
	}
}

func TestSeries_FillNullInt64(t *testing.T) {
	s := NewInt64WithValidity("x", []int64{1, 0, 3, 0, 5}, []bool{true, false, true, false, true})
	filled := s.FillNullInt64(99)
	if filled.HasNulls() {
		t.Error("FillNullInt64: should have no nulls after fill")
	}
	want := []int64{1, 99, 3, 99, 5}
	for i, w := range want {
		v, ok := filled.GetInt64(i)
		if !ok || v != w {
			t.Errorf("FillNullInt64[%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_FillNullFloat64(t *testing.T) {
	s := NewFloat64WithValidity("y", []float64{1.0, 0, 3.0}, []bool{true, false, true})
	filled := s.FillNullFloat64(0.0)
	if filled.HasNulls() {
		t.Error("FillNullFloat64: should have no nulls after fill")
	}
	v, ok := filled.GetFloat64(1)
	if !ok || v != 0.0 {
		t.Errorf("FillNullFloat64[1]: got %f, want 0.0", v)
	}
}

func TestSeries_FillNullString(t *testing.T) {
	s := NewStringWithValidity("s", []string{"a", "", "c"}, []bool{true, false, true})
	filled := s.FillNullString("N/A")
	if filled.HasNulls() {
		t.Error("FillNullString: should have no nulls after fill")
	}
	v, ok := filled.GetString(1)
	if !ok || v != "N/A" {
		t.Errorf("FillNullString[1]: got %q, want %q", v, "N/A")
	}
}

// ---------------------------------------------------------------------------
// Filter, Take
// ---------------------------------------------------------------------------

func TestSeries_Filter(t *testing.T) {
	s := NewInt64("x", []int64{10, 20, 30, 40, 50})
	mask := bitmap.NewEmpty(5)
	mask.Set(0)
	mask.Set(2)
	mask.Set(4)
	filtered := s.Filter(mask)
	if filtered.Len() != 3 {
		t.Fatalf("Filter Len: got %d, want 3", filtered.Len())
	}
	want := []int64{10, 30, 50}
	for i, w := range want {
		v, ok := filtered.GetInt64(i)
		if !ok || v != w {
			t.Errorf("Filter[%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_Filter_Float64(t *testing.T) {
	s := NewFloat64("y", []float64{1.0, 2.0, 3.0})
	mask := bitmap.NewEmpty(3)
	mask.Set(1)
	filtered := s.Filter(mask)
	if filtered.Len() != 1 {
		t.Fatalf("Filter Float64 Len: got %d, want 1", filtered.Len())
	}
	v, _ := filtered.GetFloat64(0)
	if v != 2.0 {
		t.Errorf("Filter Float64[0]: got %f, want 2.0", v)
	}
}

func TestSeries_Filter_String(t *testing.T) {
	s := NewString("s", []string{"a", "b", "c"})
	mask := bitmap.NewEmpty(3)
	mask.Set(0)
	mask.Set(2)
	filtered := s.Filter(mask)
	if filtered.Len() != 2 {
		t.Fatalf("Filter String Len: got %d, want 2", filtered.Len())
	}
}

func TestSeries_Filter_Boolean(t *testing.T) {
	s := NewBoolean("b", []bool{true, false, true})
	mask := bitmap.NewEmpty(3)
	mask.Set(0)
	mask.Set(2)
	filtered := s.Filter(mask)
	if filtered.Len() != 2 {
		t.Fatalf("Filter Boolean Len: got %d, want 2", filtered.Len())
	}
}

func TestSeries_Take(t *testing.T) {
	s := NewInt64("x", []int64{10, 20, 30, 40, 50})
	taken := s.Take([]int{4, 2, 0})
	if taken.Len() != 3 {
		t.Fatalf("Take Len: got %d, want 3", taken.Len())
	}
	want := []int64{50, 30, 10}
	for i, w := range want {
		v, ok := taken.GetInt64(i)
		if !ok || v != w {
			t.Errorf("Take[%d]: got %d, want %d", i, v, w)
		}
	}
}

func TestSeries_Take_String(t *testing.T) {
	s := NewString("s", []string{"a", "b", "c", "d"})
	taken := s.Take([]int{3, 1})
	if taken.Len() != 2 {
		t.Fatalf("Take String Len: got %d, want 2", taken.Len())
	}
	v, ok := taken.GetString(0)
	if !ok || v != "d" {
		t.Errorf("Take String[0]: got %q, want %q", v, "d")
	}
	v, ok = taken.GetString(1)
	if !ok || v != "b" {
		t.Errorf("Take String[1]: got %q, want %q", v, "b")
	}
}

// Ensure unused import is used
var _ = bitmap.New
