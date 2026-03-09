package array

import (
	"math"
	"testing"

	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// ---------------------------------------------------------------------------
// TypedArray basics
// ---------------------------------------------------------------------------

func TestTypedArray_Construction(t *testing.T) {
	data := []int64{10, 20, 30, 40, 50}
	arr := NewTypedArray(data, dtype.Int64, nil)

	if arr.DataType() != dtype.Int64 {
		t.Fatalf("DataType: got %v, want %v", arr.DataType(), dtype.Int64)
	}
	if arr.Len() != 5 {
		t.Fatalf("Len: got %d, want 5", arr.Len())
	}
}

func TestTypedArray_Value(t *testing.T) {
	data := []int64{10, 20, 30}
	arr := NewTypedArray(data, dtype.Int64, nil)

	tests := []struct {
		idx  int
		want int64
	}{
		{0, 10},
		{1, 20},
		{2, 30},
	}
	for _, tt := range tests {
		if got := arr.Value(tt.idx); got != tt.want {
			t.Errorf("Value(%d): got %d, want %d", tt.idx, got, tt.want)
		}
	}
}

func TestTypedArray_Values(t *testing.T) {
	data := []float64{1.1, 2.2, 3.3}
	arr := NewFloat64Array(data, nil)
	vals := arr.Values()
	if len(vals) != 3 {
		t.Fatalf("Values len: got %d, want 3", len(vals))
	}
	for i, want := range data {
		if vals[i] != want {
			t.Errorf("Values[%d]: got %f, want %f", i, vals[i], want)
		}
	}
}

func TestTypedArray_NullBitmap(t *testing.T) {
	data := []int64{10, 20, 30, 40}
	v := bitmap.New(4)
	v.Clear(1) // index 1 is null
	v.Clear(3) // index 3 is null
	arr := NewInt64Array(data, v)

	tests := []struct {
		idx     int
		isNull  bool
		isValid bool
	}{
		{0, false, true},
		{1, true, false},
		{2, false, true},
		{3, true, false},
	}
	for _, tt := range tests {
		if got := arr.IsNull(tt.idx); got != tt.isNull {
			t.Errorf("IsNull(%d): got %v, want %v", tt.idx, got, tt.isNull)
		}
		if got := arr.IsValid(tt.idx); got != tt.isValid {
			t.Errorf("IsValid(%d): got %v, want %v", tt.idx, got, tt.isValid)
		}
	}
	if got := arr.NullCount(); got != 2 {
		t.Errorf("NullCount: got %d, want 2", got)
	}
}

func TestTypedArray_NoNulls(t *testing.T) {
	arr := NewInt64Array([]int64{1, 2, 3}, nil)
	for i := 0; i < arr.Len(); i++ {
		if arr.IsNull(i) {
			t.Errorf("IsNull(%d) should be false with nil validity", i)
		}
		if !arr.IsValid(i) {
			t.Errorf("IsValid(%d) should be true with nil validity", i)
		}
	}
	if arr.NullCount() != 0 {
		t.Errorf("NullCount: got %d, want 0", arr.NullCount())
	}
	if arr.Validity() != nil {
		t.Error("Validity should be nil when no nulls")
	}
}

func TestTypedArray_Slice(t *testing.T) {
	data := []int64{10, 20, 30, 40, 50}
	arr := NewInt64Array(data, nil)
	sliced := arr.Slice(1, 4).(*TypedArray[int64])

	if sliced.Len() != 3 {
		t.Fatalf("Slice Len: got %d, want 3", sliced.Len())
	}
	want := []int64{20, 30, 40}
	for i, w := range want {
		if sliced.Value(i) != w {
			t.Errorf("Slice Value(%d): got %d, want %d", i, sliced.Value(i), w)
		}
	}
}

func TestTypedArray_SliceWithValidity(t *testing.T) {
	data := []int64{10, 20, 30, 40, 50}
	v := bitmap.New(5)
	v.Clear(2) // index 2 null
	arr := NewInt64Array(data, v)
	sliced := arr.Slice(1, 4).(*TypedArray[int64])

	if sliced.Len() != 3 {
		t.Fatalf("Slice Len: got %d, want 3", sliced.Len())
	}
	// In the slice, original index 2 maps to slice index 1
	if !sliced.IsNull(1) {
		t.Error("Slice: expected index 1 (orig 2) to be null")
	}
	if sliced.IsNull(0) || sliced.IsNull(2) {
		t.Error("Slice: indices 0 and 2 should be valid")
	}
}

func TestTypedArray_String(t *testing.T) {
	arr := NewInt64Array([]int64{1, 2, 3}, nil)
	s := arr.String()
	if s != "Int64[3]" {
		t.Errorf("String: got %q, want %q", s, "Int64[3]")
	}
}

// ---------------------------------------------------------------------------
// NewInt64Array / NewFloat64Array convenience constructors
// ---------------------------------------------------------------------------

func TestNewInt64Array(t *testing.T) {
	arr := NewInt64Array([]int64{1, 2, 3}, nil)
	if arr.DataType() != dtype.Int64 {
		t.Errorf("DataType: got %v, want %v", arr.DataType(), dtype.Int64)
	}
	if arr.Len() != 3 {
		t.Errorf("Len: got %d, want 3", arr.Len())
	}
}

func TestNewFloat64Array(t *testing.T) {
	arr := NewFloat64Array([]float64{1.5, 2.5}, nil)
	if arr.DataType() != dtype.Float64 {
		t.Errorf("DataType: got %v, want %v", arr.DataType(), dtype.Float64)
	}
	if arr.Value(0) != 1.5 {
		t.Errorf("Value(0): got %f, want 1.5", arr.Value(0))
	}
}

func TestNewFloat64Array_WithValidity(t *testing.T) {
	v := bitmap.New(3)
	v.Clear(1)
	arr := NewFloat64Array([]float64{1.0, 2.0, 3.0}, v)
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.IsNull(0) || arr.IsNull(2) {
		t.Error("expected indices 0 and 2 to be valid")
	}
}

// ---------------------------------------------------------------------------
// BooleanArray
// ---------------------------------------------------------------------------

func TestBooleanArray_Construction(t *testing.T) {
	arr := NewBooleanArray([]bool{true, false, true, true, false}, nil)

	if arr.DataType() != dtype.Boolean {
		t.Fatalf("DataType: got %v, want %v", arr.DataType(), dtype.Boolean)
	}
	if arr.Len() != 5 {
		t.Fatalf("Len: got %d, want 5", arr.Len())
	}
}

func TestBooleanArray_Value(t *testing.T) {
	arr := NewBooleanArray([]bool{true, false, true}, nil)
	tests := []struct {
		idx  int
		want bool
	}{
		{0, true},
		{1, false},
		{2, true},
	}
	for _, tt := range tests {
		if got := arr.Value(tt.idx); got != tt.want {
			t.Errorf("Value(%d): got %v, want %v", tt.idx, got, tt.want)
		}
	}
}

func TestBooleanArray_TrueFalseCount(t *testing.T) {
	arr := NewBooleanArray([]bool{true, false, true, true, false}, nil)
	if got := arr.TrueCount(); got != 3 {
		t.Errorf("TrueCount: got %d, want 3", got)
	}
	if got := arr.FalseCount(); got != 2 {
		t.Errorf("FalseCount: got %d, want 2", got)
	}
}

func TestBooleanArray_TrueFalseCount_WithNulls(t *testing.T) {
	v := bitmap.New(5)
	v.Clear(2) // index 2 is null
	arr := NewBooleanArray([]bool{true, false, true, true, false}, v)

	// TrueCount: only true where both data and validity are set
	// data: [true, false, true, true, false] -> indices 0,2,3 are true
	// validity has index 2 cleared, so true at 0 and 3 = 2
	if got := arr.TrueCount(); got != 2 {
		t.Errorf("TrueCount with nulls: got %d, want 2", got)
	}
	// FalseCount = len - nullCount - trueCount = 5 - 1 - 2 = 2
	if got := arr.FalseCount(); got != 2 {
		t.Errorf("FalseCount with nulls: got %d, want 2", got)
	}
}

func TestBooleanArray_Slice(t *testing.T) {
	arr := NewBooleanArray([]bool{true, false, true, true, false}, nil)
	sliced := arr.Slice(1, 4).(*BooleanArray)

	if sliced.Len() != 3 {
		t.Fatalf("Slice Len: got %d, want 3", sliced.Len())
	}
	want := []bool{false, true, true}
	for i, w := range want {
		if sliced.Value(i) != w {
			t.Errorf("Slice Value(%d): got %v, want %v", i, sliced.Value(i), w)
		}
	}
}

// ---------------------------------------------------------------------------
// StringArray
// ---------------------------------------------------------------------------

func TestStringArray_Construction(t *testing.T) {
	arr := NewStringArray([]string{"hello", "world", "foo"}, nil)
	if arr.DataType() != dtype.String {
		t.Fatalf("DataType: got %v, want %v", arr.DataType(), dtype.String)
	}
	if arr.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", arr.Len())
	}
}

func TestStringArray_Value(t *testing.T) {
	arr := NewStringArray([]string{"hello", "world", "foo"}, nil)
	tests := []struct {
		idx  int
		want string
	}{
		{0, "hello"},
		{1, "world"},
		{2, "foo"},
	}
	for _, tt := range tests {
		if got := arr.Value(tt.idx); got != tt.want {
			t.Errorf("Value(%d): got %q, want %q", tt.idx, got, tt.want)
		}
	}
}

func TestStringArray_ValueBytes(t *testing.T) {
	arr := NewStringArray([]string{"abc", "defgh"}, nil)
	got := arr.ValueBytes(0)
	if string(got) != "abc" {
		t.Errorf("ValueBytes(0): got %q, want %q", got, "abc")
	}
	got = arr.ValueBytes(1)
	if string(got) != "defgh" {
		t.Errorf("ValueBytes(1): got %q, want %q", got, "defgh")
	}
}

func TestStringArray_Slice(t *testing.T) {
	arr := NewStringArray([]string{"a", "bb", "ccc", "dddd"}, nil)
	sliced := arr.Slice(1, 3).(*StringArray)

	if sliced.Len() != 2 {
		t.Fatalf("Slice Len: got %d, want 2", sliced.Len())
	}
	if sliced.Value(0) != "bb" {
		t.Errorf("Slice Value(0): got %q, want %q", sliced.Value(0), "bb")
	}
	if sliced.Value(1) != "ccc" {
		t.Errorf("Slice Value(1): got %q, want %q", sliced.Value(1), "ccc")
	}
}

func TestStringArray_TotalBytes(t *testing.T) {
	arr := NewStringArray([]string{"ab", "cde"}, nil)
	if got := arr.TotalBytes(); got != 5 {
		t.Errorf("TotalBytes: got %d, want 5", got)
	}
}

func TestStringArray_WithNulls(t *testing.T) {
	v := bitmap.New(3)
	v.Clear(1)
	arr := NewStringArray([]string{"a", "", "c"}, v)
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", arr.NullCount())
	}
}

// ---------------------------------------------------------------------------
// TypedBuilder
// ---------------------------------------------------------------------------

func TestTypedBuilder_AppendAndBuild(t *testing.T) {
	b := NewTypedBuilder[int64](dtype.Int64, 4)
	b.Append(10)
	b.Append(20)
	b.Append(30)

	if b.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", b.Len())
	}

	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Build Len: got %d, want 3", arr.Len())
	}
	if arr.Value(0) != 10 || arr.Value(1) != 20 || arr.Value(2) != 30 {
		t.Error("Build values mismatch")
	}
	if arr.NullCount() != 0 {
		t.Errorf("NullCount: got %d, want 0", arr.NullCount())
	}
}

func TestTypedBuilder_AppendNull(t *testing.T) {
	b := NewTypedBuilder[int64](dtype.Int64, 4)
	b.Append(10)
	b.AppendNull()
	b.Append(30)

	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Build Len: got %d, want 3", arr.Len())
	}
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.IsNull(0) || arr.IsNull(2) {
		t.Error("expected indices 0, 2 to be valid")
	}
	if arr.NullCount() != 1 {
		t.Errorf("NullCount: got %d, want 1", arr.NullCount())
	}
}

func TestTypedBuilder_AppendValues(t *testing.T) {
	b := NewTypedBuilder[int64](dtype.Int64, 0)
	b.AppendValues([]int64{1, 2, 3}, nil)
	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", arr.Len())
	}
	if arr.NullCount() != 0 {
		t.Error("expected no nulls with nil valid")
	}
}

func TestTypedBuilder_AppendValues_WithValidity(t *testing.T) {
	b := NewTypedBuilder[int64](dtype.Int64, 0)
	b.AppendValues([]int64{1, 2, 3}, []bool{true, false, true})
	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", arr.Len())
	}
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.IsNull(0) || arr.IsNull(2) {
		t.Error("expected indices 0, 2 to be valid")
	}
}

func TestTypedBuilder_ReuseAfterBuild(t *testing.T) {
	b := NewTypedBuilder[int64](dtype.Int64, 4)
	b.Append(1)
	b.Append(2)
	_ = b.Build()

	// Builder should be empty after Build
	if b.Len() != 0 {
		t.Errorf("Len after Build: got %d, want 0", b.Len())
	}

	b.Append(100)
	b.AppendNull()
	arr := b.Build()
	if arr.Len() != 2 {
		t.Fatalf("Reuse Len: got %d, want 2", arr.Len())
	}
	if arr.Value(0) != 100 {
		t.Errorf("Reuse Value(0): got %d, want 100", arr.Value(0))
	}
	if !arr.IsNull(1) {
		t.Error("Reuse: expected index 1 to be null")
	}
}

// ---------------------------------------------------------------------------
// StringBuilder
// ---------------------------------------------------------------------------

func TestStringBuilder_AppendAndBuild(t *testing.T) {
	b := NewStringBuilder(4)
	b.Append("hello")
	b.Append("world")

	arr := b.Build()
	if arr.Len() != 2 {
		t.Fatalf("Len: got %d, want 2", arr.Len())
	}
	if arr.Value(0) != "hello" {
		t.Errorf("Value(0): got %q, want %q", arr.Value(0), "hello")
	}
	if arr.Value(1) != "world" {
		t.Errorf("Value(1): got %q, want %q", arr.Value(1), "world")
	}
}

func TestStringBuilder_AppendNull(t *testing.T) {
	b := NewStringBuilder(4)
	b.Append("a")
	b.AppendNull()
	b.Append("c")

	arr := b.Build()
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.IsNull(0) || arr.IsNull(2) {
		t.Error("expected indices 0, 2 to be valid")
	}
}

func TestStringBuilder_AppendValues(t *testing.T) {
	b := NewStringBuilder(0)
	b.AppendValues([]string{"x", "y", "z"}, nil)
	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", arr.Len())
	}
	if arr.Value(1) != "y" {
		t.Errorf("Value(1): got %q, want %q", arr.Value(1), "y")
	}
}

func TestStringBuilder_AppendValues_WithValidity(t *testing.T) {
	b := NewStringBuilder(0)
	b.AppendValues([]string{"x", "y", "z"}, []bool{true, false, true})
	arr := b.Build()
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
}

func TestStringBuilder_ReuseAfterBuild(t *testing.T) {
	b := NewStringBuilder(4)
	b.Append("a")
	_ = b.Build()

	if b.Len() != 0 {
		t.Errorf("Len after Build: got %d, want 0", b.Len())
	}
	b.Append("new")
	arr := b.Build()
	if arr.Len() != 1 {
		t.Fatalf("Reuse Len: got %d, want 1", arr.Len())
	}
	if arr.Value(0) != "new" {
		t.Errorf("Reuse Value(0): got %q, want %q", arr.Value(0), "new")
	}
}

// ---------------------------------------------------------------------------
// BooleanBuilder
// ---------------------------------------------------------------------------

func TestBooleanBuilder_AppendAndBuild(t *testing.T) {
	b := NewBooleanBuilder(4)
	b.Append(true)
	b.Append(false)
	b.Append(true)

	arr := b.Build()
	if arr.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", arr.Len())
	}
	if !arr.Value(0) || arr.Value(1) || !arr.Value(2) {
		t.Error("Value mismatch")
	}
}

func TestBooleanBuilder_AppendNull(t *testing.T) {
	b := NewBooleanBuilder(4)
	b.Append(true)
	b.AppendNull()
	b.Append(false)

	arr := b.Build()
	if !arr.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if arr.IsNull(0) || arr.IsNull(2) {
		t.Error("expected indices 0, 2 to be valid")
	}
}

func TestBooleanBuilder_ReuseAfterBuild(t *testing.T) {
	b := NewBooleanBuilder(4)
	b.Append(true)
	_ = b.Build()

	if b.Len() != 0 {
		t.Errorf("Len after Build: got %d, want 0", b.Len())
	}
	b.Append(false)
	b.AppendNull()
	arr := b.Build()
	if arr.Len() != 2 {
		t.Fatalf("Reuse Len: got %d, want 2", arr.Len())
	}
	if arr.Value(0) {
		t.Error("Reuse Value(0): got true, want false")
	}
	if !arr.IsNull(1) {
		t.Error("Reuse: expected index 1 to be null")
	}
}

// ---------------------------------------------------------------------------
// Arithmetic kernels
// ---------------------------------------------------------------------------

func TestAdd(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3}, nil)
	b := NewInt64Array([]int64{10, 20, 30}, nil)
	c := Add(a, b)
	want := []int64{11, 22, 33}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Add[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestSub(t *testing.T) {
	a := NewInt64Array([]int64{10, 20, 30}, nil)
	b := NewInt64Array([]int64{1, 2, 3}, nil)
	c := Sub(a, b)
	want := []int64{9, 18, 27}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Sub[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestMul(t *testing.T) {
	a := NewInt64Array([]int64{2, 3, 4}, nil)
	b := NewInt64Array([]int64{5, 6, 7}, nil)
	c := Mul(a, b)
	want := []int64{10, 18, 28}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Mul[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestDiv(t *testing.T) {
	a := NewInt64Array([]int64{10, 20, 30}, nil)
	b := NewInt64Array([]int64{2, 5, 3}, nil)
	c := Div(a, b)
	want := []int64{5, 4, 10}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Div[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestMod(t *testing.T) {
	a := NewInt64Array([]int64{10, 21, 35}, nil)
	b := NewInt64Array([]int64{3, 5, 8}, nil)
	c := Mod(a, b)
	want := []int64{1, 1, 3}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Mod[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestNeg(t *testing.T) {
	a := NewInt64Array([]int64{1, -2, 3}, nil)
	c := Neg(a)
	want := []int64{-1, 2, -3}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("Neg[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestAddScalar(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3}, nil)
	c := AddScalar(a, int64(10))
	want := []int64{11, 12, 13}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("AddScalar[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestMulScalar(t *testing.T) {
	a := NewInt64Array([]int64{2, 3, 4}, nil)
	c := MulScalar(a, int64(5))
	want := []int64{10, 15, 20}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("MulScalar[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestArithmetic_WithNulls(t *testing.T) {
	va := bitmap.New(3)
	va.Clear(1) // a[1] is null
	a := NewInt64Array([]int64{1, 0, 3}, va)

	vb := bitmap.New(3)
	vb.Clear(2) // b[2] is null
	b := NewInt64Array([]int64{10, 20, 0}, vb)

	c := Add(a, b)
	// Position 0: both valid -> valid
	if c.IsNull(0) {
		t.Error("Add with nulls: index 0 should be valid")
	}
	// Position 1: a is null -> result is null
	if !c.IsNull(1) {
		t.Error("Add with nulls: index 1 should be null")
	}
	// Position 2: b is null -> result is null
	if !c.IsNull(2) {
		t.Error("Add with nulls: index 2 should be null")
	}
	if c.Value(0) != 11 {
		t.Errorf("Add with nulls Value(0): got %d, want 11", c.Value(0))
	}
}

func TestAddScalar_PreservesNulls(t *testing.T) {
	v := bitmap.New(3)
	v.Clear(1)
	a := NewInt64Array([]int64{1, 0, 3}, v)
	c := AddScalar(a, int64(10))
	if !c.IsNull(1) {
		t.Error("AddScalar should preserve null at index 1")
	}
	if c.Value(0) != 11 {
		t.Errorf("AddScalar Value(0): got %d, want 11", c.Value(0))
	}
}

// ---------------------------------------------------------------------------
// Comparison kernels
// ---------------------------------------------------------------------------

func TestEqual(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3}, nil)
	b := NewInt64Array([]int64{1, 99, 3}, nil)
	c := Equal(a, b)
	if !c.Value(0) {
		t.Error("Equal[0]: expected true")
	}
	if c.Value(1) {
		t.Error("Equal[1]: expected false")
	}
	if !c.Value(2) {
		t.Error("Equal[2]: expected true")
	}
}

func TestNotEqual(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3}, nil)
	b := NewInt64Array([]int64{1, 99, 3}, nil)
	c := NotEqual(a, b)
	if c.Value(0) {
		t.Error("NotEqual[0]: expected false")
	}
	if !c.Value(1) {
		t.Error("NotEqual[1]: expected true")
	}
}

func TestLessThan(t *testing.T) {
	a := NewInt64Array([]int64{1, 5, 3}, nil)
	b := NewInt64Array([]int64{2, 3, 3}, nil)
	c := LessThan(a, b)
	if !c.Value(0) {
		t.Error("LessThan[0]: expected true (1<2)")
	}
	if c.Value(1) {
		t.Error("LessThan[1]: expected false (5<3)")
	}
	if c.Value(2) {
		t.Error("LessThan[2]: expected false (3<3)")
	}
}

func TestGreaterThan(t *testing.T) {
	a := NewInt64Array([]int64{1, 5, 3}, nil)
	b := NewInt64Array([]int64{2, 3, 3}, nil)
	c := GreaterThan(a, b)
	if c.Value(0) {
		t.Error("GreaterThan[0]: expected false (1>2)")
	}
	if !c.Value(1) {
		t.Error("GreaterThan[1]: expected true (5>3)")
	}
	if c.Value(2) {
		t.Error("GreaterThan[2]: expected false (3>3)")
	}
}

func TestEqualScalar(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3, 2}, nil)
	c := EqualScalar(a, int64(2))
	want := []bool{false, true, false, true}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("EqualScalar[%d]: got %v, want %v", i, c.Value(i), w)
		}
	}
}

func TestLessThanScalar(t *testing.T) {
	a := NewInt64Array([]int64{1, 5, 3}, nil)
	c := LessThanScalar(a, int64(3))
	if !c.Value(0) {
		t.Error("LessThanScalar[0]: expected true (1<3)")
	}
	if c.Value(1) {
		t.Error("LessThanScalar[1]: expected false (5<3)")
	}
	if c.Value(2) {
		t.Error("LessThanScalar[2]: expected false (3<3)")
	}
}

func TestGreaterThanScalar(t *testing.T) {
	a := NewInt64Array([]int64{1, 5, 3}, nil)
	c := GreaterThanScalar(a, int64(3))
	if c.Value(0) {
		t.Error("GreaterThanScalar[0]: expected false (1>3)")
	}
	if !c.Value(1) {
		t.Error("GreaterThanScalar[1]: expected true (5>3)")
	}
	if c.Value(2) {
		t.Error("GreaterThanScalar[2]: expected false (3>3)")
	}
}

func TestComparison_WithNulls(t *testing.T) {
	va := bitmap.New(3)
	va.Clear(1)
	a := NewInt64Array([]int64{1, 0, 3}, va)
	b := NewInt64Array([]int64{1, 0, 3}, nil)
	c := Equal(a, b)
	// Index 1: a is null -> result should be null
	if !c.IsNull(1) {
		t.Error("Equal with null: index 1 should be null")
	}
}

// ---------------------------------------------------------------------------
// Aggregate kernels
// ---------------------------------------------------------------------------

func TestSum(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3, 4, 5}, nil)
	got, ok := Sum(a)
	if !ok {
		t.Fatal("Sum returned not ok")
	}
	if got != 15 {
		t.Errorf("Sum: got %d, want 15", got)
	}
}

func TestSum_WithNulls(t *testing.T) {
	v := bitmap.New(5)
	v.Clear(2) // index 2 null
	a := NewInt64Array([]int64{1, 2, 0, 4, 5}, v)
	got, ok := Sum(a)
	if !ok {
		t.Fatal("Sum returned not ok")
	}
	if got != 12 {
		t.Errorf("Sum with nulls: got %d, want 12", got)
	}
}

func TestSum_AllNull(t *testing.T) {
	v := bitmap.New(2)
	v.Clear(0)
	v.Clear(1)
	a := NewInt64Array([]int64{0, 0}, v)
	_, ok := Sum(a)
	if ok {
		t.Error("Sum of all nulls should return false")
	}
}

func TestMean(t *testing.T) {
	a := NewFloat64Array([]float64{2.0, 4.0, 6.0}, nil)
	got, ok := Mean(a)
	if !ok {
		t.Fatal("Mean returned not ok")
	}
	if got != 4.0 {
		t.Errorf("Mean: got %f, want 4.0", got)
	}
}

func TestMin(t *testing.T) {
	a := NewInt64Array([]int64{5, 1, 3, 2, 4}, nil)
	got, ok := Min(a)
	if !ok {
		t.Fatal("Min returned not ok")
	}
	if got != 1 {
		t.Errorf("Min: got %d, want 1", got)
	}
}

func TestMax(t *testing.T) {
	a := NewInt64Array([]int64{5, 1, 3, 2, 4}, nil)
	got, ok := Max(a)
	if !ok {
		t.Fatal("Max returned not ok")
	}
	if got != 5 {
		t.Errorf("Max: got %d, want 5", got)
	}
}

func TestVariance(t *testing.T) {
	// Population variance of [2, 4, 4, 4, 5, 5, 7, 9] = 4.0
	a := NewFloat64Array([]float64{2, 4, 4, 4, 5, 5, 7, 9}, nil)
	got, ok := Variance(a, 0)
	if !ok {
		t.Fatal("Variance returned not ok")
	}
	if math.Abs(got-4.0) > 1e-10 {
		t.Errorf("Variance(pop): got %f, want 4.0", got)
	}
}

func TestStd(t *testing.T) {
	a := NewFloat64Array([]float64{2, 4, 4, 4, 5, 5, 7, 9}, nil)
	got, ok := Std(a, 0)
	if !ok {
		t.Fatal("Std returned not ok")
	}
	if math.Abs(got-2.0) > 1e-10 {
		t.Errorf("Std(pop): got %f, want 2.0", got)
	}
}

// ---------------------------------------------------------------------------
// Filter kernels
// ---------------------------------------------------------------------------

func TestFilterTyped(t *testing.T) {
	a := NewInt64Array([]int64{10, 20, 30, 40, 50}, nil)
	mask := bitmap.NewEmpty(5)
	mask.Set(0)
	mask.Set(2)
	mask.Set(4)

	result := FilterTyped(a, mask)
	if result.Len() != 3 {
		t.Fatalf("FilterTyped Len: got %d, want 3", result.Len())
	}
	want := []int64{10, 30, 50}
	for i, w := range want {
		if result.Value(i) != w {
			t.Errorf("FilterTyped[%d]: got %d, want %d", i, result.Value(i), w)
		}
	}
}

func TestFilterBoolean(t *testing.T) {
	a := NewBooleanArray([]bool{true, false, true, false, true}, nil)
	mask := bitmap.NewEmpty(5)
	mask.Set(1)
	mask.Set(3)

	result := FilterBoolean(a, mask)
	if result.Len() != 2 {
		t.Fatalf("FilterBoolean Len: got %d, want 2", result.Len())
	}
	if result.Value(0) || result.Value(1) {
		t.Error("FilterBoolean: expected both false")
	}
}

func TestFilterString(t *testing.T) {
	a := NewStringArray([]string{"a", "b", "c", "d"}, nil)
	mask := bitmap.NewEmpty(4)
	mask.Set(0)
	mask.Set(3)

	result := FilterString(a, mask)
	if result.Len() != 2 {
		t.Fatalf("FilterString Len: got %d, want 2", result.Len())
	}
	if result.Value(0) != "a" {
		t.Errorf("FilterString[0]: got %q, want %q", result.Value(0), "a")
	}
	if result.Value(1) != "d" {
		t.Errorf("FilterString[1]: got %q, want %q", result.Value(1), "d")
	}
}

// ---------------------------------------------------------------------------
// Sort kernels
// ---------------------------------------------------------------------------

func TestArgSort_Ascending(t *testing.T) {
	a := NewInt64Array([]int64{30, 10, 20}, nil)
	indices := ArgSort(a, false)
	want := []int{1, 2, 0}
	for i, w := range want {
		if indices[i] != w {
			t.Errorf("ArgSort asc [%d]: got %d, want %d", i, indices[i], w)
		}
	}
}

func TestArgSort_Descending(t *testing.T) {
	a := NewInt64Array([]int64{30, 10, 20}, nil)
	indices := ArgSort(a, true)
	want := []int{0, 2, 1}
	for i, w := range want {
		if indices[i] != w {
			t.Errorf("ArgSort desc [%d]: got %d, want %d", i, indices[i], w)
		}
	}
}

func TestArgSort_WithNulls(t *testing.T) {
	v := bitmap.New(4)
	v.Clear(1) // index 1 is null
	a := NewInt64Array([]int64{30, 0, 10, 20}, v)
	indices := ArgSort(a, false)
	// Non-null sorted ascending: 10(idx2), 20(idx3), 30(idx0); null(idx1) at end
	want := []int{2, 3, 0, 1}
	for i, w := range want {
		if indices[i] != w {
			t.Errorf("ArgSort with nulls [%d]: got %d, want %d", i, indices[i], w)
		}
	}
}

func TestArgMin(t *testing.T) {
	a := NewInt64Array([]int64{30, 10, 20}, nil)
	idx, ok := ArgMin(a)
	if !ok {
		t.Fatal("ArgMin returned not ok")
	}
	if idx != 1 {
		t.Errorf("ArgMin: got %d, want 1", idx)
	}
}

func TestArgMax(t *testing.T) {
	a := NewInt64Array([]int64{30, 10, 20}, nil)
	idx, ok := ArgMax(a)
	if !ok {
		t.Fatal("ArgMax returned not ok")
	}
	if idx != 0 {
		t.Errorf("ArgMax: got %d, want 0", idx)
	}
}

func TestArgMin_AllNull(t *testing.T) {
	v := bitmap.New(2)
	v.Clear(0)
	v.Clear(1)
	a := NewInt64Array([]int64{0, 0}, v)
	_, ok := ArgMin(a)
	if ok {
		t.Error("ArgMin of all nulls should return false")
	}
}

// ---------------------------------------------------------------------------
// Cumulative kernels
// ---------------------------------------------------------------------------

func TestCumSum(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3, 4, 5}, nil)
	c := CumSum(a)
	want := []int64{1, 3, 6, 10, 15}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("CumSum[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestCumSum_WithNulls(t *testing.T) {
	v := bitmap.New(5)
	v.Clear(2)
	a := NewInt64Array([]int64{1, 2, 0, 4, 5}, v)
	c := CumSum(a)
	// Running: 1, 3, (null), 7, 12
	if c.Value(0) != 1 {
		t.Errorf("CumSum[0]: got %d, want 1", c.Value(0))
	}
	if c.Value(1) != 3 {
		t.Errorf("CumSum[1]: got %d, want 3", c.Value(1))
	}
	if !c.IsNull(2) {
		t.Error("CumSum[2]: expected null")
	}
	if c.Value(3) != 7 {
		t.Errorf("CumSum[3]: got %d, want 7", c.Value(3))
	}
	if c.Value(4) != 12 {
		t.Errorf("CumSum[4]: got %d, want 12", c.Value(4))
	}
}

func TestCumProd(t *testing.T) {
	a := NewInt64Array([]int64{1, 2, 3, 4}, nil)
	c := CumProd(a)
	want := []int64{1, 2, 6, 24}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("CumProd[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestCumMin(t *testing.T) {
	a := NewInt64Array([]int64{5, 3, 7, 1, 4}, nil)
	c := CumMin(a)
	want := []int64{5, 3, 3, 1, 1}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("CumMin[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}

func TestCumMax(t *testing.T) {
	a := NewInt64Array([]int64{1, 5, 3, 7, 2}, nil)
	c := CumMax(a)
	want := []int64{1, 5, 5, 7, 7}
	for i, w := range want {
		if c.Value(i) != w {
			t.Errorf("CumMax[%d]: got %d, want %d", i, c.Value(i), w)
		}
	}
}
