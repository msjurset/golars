package dataframe

import (
	"strings"
	"testing"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// helper builds a simple 3-row DataFrame with columns "a" (Int64) and "b" (String).
func helperDF(t *testing.T) *DataFrame {
	t.Helper()
	df, err := New(
		series.NewInt64("a", []int64{1, 2, 3}),
		series.NewString("b", []string{"x", "y", "z"}),
	)
	if err != nil {
		t.Fatalf("helperDF: %v", err)
	}
	return df
}

// helperNumericDF builds a DataFrame with numeric columns for Describe tests.
func helperNumericDF(t *testing.T) *DataFrame {
	t.Helper()
	df, err := New(
		series.NewInt64("ints", []int64{10, 20, 30, 40, 50}),
		series.NewFloat64("floats", []float64{1.0, 2.0, 3.0, 4.0, 5.0}),
		series.NewString("labels", []string{"a", "b", "c", "d", "e"}),
	)
	if err != nil {
		t.Fatalf("helperNumericDF: %v", err)
	}
	return df
}

func TestNew(t *testing.T) {
	t.Run("valid columns", func(t *testing.T) {
		df, err := New(
			series.NewInt64("x", []int64{1, 2}),
			series.NewString("y", []string{"a", "b"}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if df.Height() != 2 {
			t.Errorf("Height() = %d, want 2", df.Height())
		}
		if df.Width() != 2 {
			t.Errorf("Width() = %d, want 2", df.Width())
		}
	})

	t.Run("no columns", func(t *testing.T) {
		df, err := New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if df.Height() != 0 || df.Width() != 0 {
			t.Errorf("empty DF: Height=%d, Width=%d, want 0,0", df.Height(), df.Width())
		}
	})

	t.Run("nil column", func(t *testing.T) {
		// nil at index > 0 is caught by the validation loop.
		_, err := New(
			series.NewInt64("x", []int64{1}),
			nil,
		)
		if err == nil {
			t.Fatal("expected error for nil column")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Errorf("error = %q, want mention of nil", err)
		}
	})

	t.Run("duplicate names", func(t *testing.T) {
		_, err := New(
			series.NewInt64("x", []int64{1}),
			series.NewInt64("x", []int64{2}),
		)
		if err == nil {
			t.Fatal("expected error for duplicate column names")
		}
		if !strings.Contains(err.Error(), "duplicate") {
			t.Errorf("error = %q, want mention of duplicate", err)
		}
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		_, err := New(
			series.NewInt64("x", []int64{1, 2}),
			series.NewInt64("y", []int64{1, 2, 3}),
		)
		if err == nil {
			t.Fatal("expected error for mismatched lengths")
		}
		if !strings.Contains(err.Error(), "length") {
			t.Errorf("error = %q, want mention of length", err)
		}
	})
}

func TestAccessors(t *testing.T) {
	df := helperDF(t)

	t.Run("Height", func(t *testing.T) {
		if df.Height() != 3 {
			t.Errorf("Height() = %d, want 3", df.Height())
		}
	})

	t.Run("Width", func(t *testing.T) {
		if df.Width() != 2 {
			t.Errorf("Width() = %d, want 2", df.Width())
		}
	})

	t.Run("Shape", func(t *testing.T) {
		h, w := df.Shape()
		if h != 3 || w != 2 {
			t.Errorf("Shape() = (%d, %d), want (3, 2)", h, w)
		}
	})

	t.Run("Schema", func(t *testing.T) {
		s := df.Schema()
		if s.Len() != 2 {
			t.Fatalf("Schema.Len() = %d, want 2", s.Len())
		}
		if s.Field(0).Name != "a" || s.Field(0).Dtype != dtype.Int64 {
			t.Errorf("Field(0) = %v, want {a, Int64}", s.Field(0))
		}
		if s.Field(1).Name != "b" || s.Field(1).Dtype != dtype.String {
			t.Errorf("Field(1) = %v, want {b, String}", s.Field(1))
		}
	})

	t.Run("Columns", func(t *testing.T) {
		cols := df.Columns()
		if len(cols) != 2 {
			t.Fatalf("len(Columns()) = %d, want 2", len(cols))
		}
		if cols[0].Name() != "a" {
			t.Errorf("Columns()[0].Name() = %q, want %q", cols[0].Name(), "a")
		}
	})

	t.Run("Column found", func(t *testing.T) {
		col, err := df.Column("a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if col.Name() != "a" {
			t.Errorf("Column('a').Name() = %q", col.Name())
		}
	})

	t.Run("Column not found", func(t *testing.T) {
		_, err := df.Column("missing")
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})

	t.Run("ColumnByIndex", func(t *testing.T) {
		col := df.ColumnByIndex(1)
		if col.Name() != "b" {
			t.Errorf("ColumnByIndex(1).Name() = %q, want %q", col.Name(), "b")
		}
	})

	t.Run("Len", func(t *testing.T) {
		if df.Len() != 3 {
			t.Errorf("Len() = %d, want 3", df.Len())
		}
	})

	t.Run("IsEmpty false", func(t *testing.T) {
		if df.IsEmpty() {
			t.Error("IsEmpty() = true, want false")
		}
	})

	t.Run("IsEmpty true", func(t *testing.T) {
		empty, _ := New()
		if !empty.IsEmpty() {
			t.Error("IsEmpty() = false for empty DF")
		}
	})
}

func TestSlicing(t *testing.T) {
	df, err := New(
		series.NewInt64("v", []int64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name   string
		fn     func() *DataFrame
		wantH  int
		wantV0 int64
	}{
		{"Head(3)", func() *DataFrame { return df.Head(3) }, 3, 10},
		{"Head(100) clamps", func() *DataFrame { return df.Head(100) }, 5, 10},
		{"Head(0)", func() *DataFrame { return df.Head(0) }, 0, 0},
		{"Head(-1)", func() *DataFrame { return df.Head(-1) }, 0, 0},
		{"Tail(2)", func() *DataFrame { return df.Tail(2) }, 2, 40},
		{"Tail(100) clamps", func() *DataFrame { return df.Tail(100) }, 5, 10},
		{"Tail(0)", func() *DataFrame { return df.Tail(0) }, 0, 0},
		{"Slice(1,4)", func() *DataFrame { return df.Slice(1, 4) }, 3, 20},
		{"Slice clamped start<0", func() *DataFrame { return df.Slice(-1, 3) }, 3, 10},
		{"Slice clamped end>height", func() *DataFrame { return df.Slice(3, 100) }, 2, 40},
		{"Slice start>end", func() *DataFrame { return df.Slice(4, 2) }, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn()
			if result.Height() != tc.wantH {
				t.Errorf("Height() = %d, want %d", result.Height(), tc.wantH)
			}
			if tc.wantH > 0 {
				col, _ := result.Column("v")
				v, ok := col.GetInt64(0)
				if !ok {
					t.Fatal("GetInt64(0) not valid")
				}
				if v != tc.wantV0 {
					t.Errorf("first value = %d, want %d", v, tc.wantV0)
				}
			}
		})
	}
}

func TestSelect(t *testing.T) {
	df := helperDF(t)

	t.Run("select single column", func(t *testing.T) {
		result, err := df.Select("b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 1 {
			t.Errorf("Width() = %d, want 1", result.Width())
		}
		if result.ColumnByIndex(0).Name() != "b" {
			t.Error("selected column not 'b'")
		}
	})

	t.Run("select reorders", func(t *testing.T) {
		result, err := df.Select("b", "a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ColumnByIndex(0).Name() != "b" || result.ColumnByIndex(1).Name() != "a" {
			t.Error("columns not reordered correctly")
		}
	})

	t.Run("select missing column", func(t *testing.T) {
		_, err := df.Select("missing")
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestDrop(t *testing.T) {
	df := helperDF(t)

	t.Run("drop single column", func(t *testing.T) {
		result, err := df.Drop("a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 1 {
			t.Errorf("Width() = %d, want 1", result.Width())
		}
		if result.ColumnByIndex(0).Name() != "b" {
			t.Error("remaining column should be 'b'")
		}
	})

	t.Run("drop missing column", func(t *testing.T) {
		_, err := df.Drop("missing")
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestRename(t *testing.T) {
	df := helperDF(t)

	t.Run("rename success", func(t *testing.T) {
		result, err := df.Rename("a", "alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Schema().Contains("alpha") {
			t.Error("renamed column not found")
		}
		if result.Schema().Contains("a") {
			t.Error("old column name still present")
		}
	})

	t.Run("rename to same name", func(t *testing.T) {
		result, err := df.Rename("a", "a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Schema().Contains("a") {
			t.Error("column 'a' should still exist")
		}
	})

	t.Run("rename old not found", func(t *testing.T) {
		_, err := df.Rename("missing", "new")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rename new already exists", func(t *testing.T) {
		_, err := df.Rename("a", "b")
		if err == nil {
			t.Fatal("expected error for name collision")
		}
	})
}

func TestWithColumn(t *testing.T) {
	df := helperDF(t)

	t.Run("add new column", func(t *testing.T) {
		result, err := df.WithColumn(series.NewFloat64("c", []float64{1.1, 2.2, 3.3}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 3 {
			t.Errorf("Width() = %d, want 3", result.Width())
		}
	})

	t.Run("replace existing column", func(t *testing.T) {
		result, err := df.WithColumn(series.NewInt64("a", []int64{10, 20, 30}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 2 {
			t.Errorf("Width() = %d, want 2 (replaced, not added)", result.Width())
		}
		col, _ := result.Column("a")
		v, _ := col.GetInt64(0)
		if v != 10 {
			t.Errorf("replaced value = %d, want 10", v)
		}
	})

	t.Run("nil column", func(t *testing.T) {
		_, err := df.WithColumn(nil)
		if err == nil {
			t.Fatal("expected error for nil column")
		}
	})

	t.Run("mismatched length", func(t *testing.T) {
		_, err := df.WithColumn(series.NewInt64("c", []int64{1}))
		if err == nil {
			t.Fatal("expected error for length mismatch")
		}
	})
}

func TestWithColumns(t *testing.T) {
	df := helperDF(t)

	t.Run("add multiple columns", func(t *testing.T) {
		result, err := df.WithColumns(
			series.NewFloat64("c", []float64{1.1, 2.2, 3.3}),
			series.NewBoolean("d", []bool{true, false, true}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 4 {
			t.Errorf("Width() = %d, want 4", result.Width())
		}
	})

	t.Run("error propagates", func(t *testing.T) {
		_, err := df.WithColumns(
			series.NewFloat64("c", []float64{1.1, 2.2, 3.3}),
			series.NewFloat64("bad", []float64{1.0}), // wrong length
		)
		if err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}

func TestFilter(t *testing.T) {
	df := helperDF(t)

	t.Run("filter keeps matching rows", func(t *testing.T) {
		mask := series.NewBoolean("mask", []bool{true, false, true})
		result, err := df.Filter(mask)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 2 {
			t.Errorf("Height() = %d, want 2", result.Height())
		}
		col, _ := result.Column("a")
		v0, _ := col.GetInt64(0)
		v1, _ := col.GetInt64(1)
		if v0 != 1 || v1 != 3 {
			t.Errorf("values = [%d, %d], want [1, 3]", v0, v1)
		}
	})

	t.Run("filter all false", func(t *testing.T) {
		mask := series.NewBoolean("mask", []bool{false, false, false})
		result, err := df.Filter(mask)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 0 {
			t.Errorf("Height() = %d, want 0", result.Height())
		}
	})

	t.Run("filter nil mask", func(t *testing.T) {
		_, err := df.Filter(nil)
		if err == nil {
			t.Fatal("expected error for nil mask")
		}
	})

	t.Run("filter non-boolean mask", func(t *testing.T) {
		mask := series.NewInt64("mask", []int64{1, 0, 1})
		_, err := df.Filter(mask)
		if err == nil {
			t.Fatal("expected error for non-boolean mask")
		}
	})

	t.Run("filter length mismatch", func(t *testing.T) {
		mask := series.NewBoolean("mask", []bool{true, false})
		_, err := df.Filter(mask)
		if err == nil {
			t.Fatal("expected error for mask length mismatch")
		}
	})
}

func TestSort(t *testing.T) {
	df, err := New(
		series.NewInt64("v", []int64{3, 1, 2}),
		series.NewString("s", []string{"c", "a", "b"}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("ascending", func(t *testing.T) {
		result, err := df.Sort("v", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("v")
		for i, want := range []int64{1, 2, 3} {
			v, _ := col.GetInt64(i)
			if v != want {
				t.Errorf("row %d: got %d, want %d", i, v, want)
			}
		}
	})

	t.Run("descending", func(t *testing.T) {
		result, err := df.Sort("v", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("v")
		for i, want := range []int64{3, 2, 1} {
			v, _ := col.GetInt64(i)
			if v != want {
				t.Errorf("row %d: got %d, want %d", i, v, want)
			}
		}
	})

	t.Run("sort by string column", func(t *testing.T) {
		result, err := df.Sort("s", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("s")
		for i, want := range []string{"a", "b", "c"} {
			v, _ := col.GetString(i)
			if v != want {
				t.Errorf("row %d: got %q, want %q", i, v, want)
			}
		}
	})

	t.Run("sort column not found", func(t *testing.T) {
		_, err := df.Sort("missing", false)
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestSortBy(t *testing.T) {
	df, err := New(
		series.NewString("group", []string{"a", "b", "a", "b"}),
		series.NewInt64("val", []int64{2, 1, 1, 2}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("multi-column sort", func(t *testing.T) {
		result, err := df.SortBy([]string{"group", "val"}, []bool{false, false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		gcol, _ := result.Column("group")
		vcol, _ := result.Column("val")
		wantG := []string{"a", "a", "b", "b"}
		wantV := []int64{1, 2, 1, 2}
		for i := range wantG {
			g, _ := gcol.GetString(i)
			v, _ := vcol.GetInt64(i)
			if g != wantG[i] || v != wantV[i] {
				t.Errorf("row %d: got (%q, %d), want (%q, %d)", i, g, v, wantG[i], wantV[i])
			}
		}
	})

	t.Run("empty sort returns clone", func(t *testing.T) {
		result, err := df.SortBy(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != df.Height() {
			t.Errorf("Height() = %d, want %d", result.Height(), df.Height())
		}
	})

	t.Run("mismatched slice lengths", func(t *testing.T) {
		_, err := df.SortBy([]string{"group"}, []bool{true, false})
		if err == nil {
			t.Fatal("expected error for mismatched lengths")
		}
	})

	t.Run("missing column", func(t *testing.T) {
		_, err := df.SortBy([]string{"missing"}, []bool{false})
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestDescribe(t *testing.T) {
	df := helperNumericDF(t)

	t.Run("has statistic column", func(t *testing.T) {
		desc := df.Describe()
		col, err := desc.Column("statistic")
		if err != nil {
			t.Fatalf("statistic column missing: %v", err)
		}
		if col.Len() != 5 {
			t.Errorf("statistic column len = %d, want 5", col.Len())
		}
		labels := []string{"count", "mean", "std", "min", "max"}
		for i, want := range labels {
			v, _ := col.GetString(i)
			if v != want {
				t.Errorf("row %d: label = %q, want %q", i, v, want)
			}
		}
	})

	t.Run("includes numeric columns only", func(t *testing.T) {
		desc := df.Describe()
		// Should have statistic + ints + floats = 3 columns (no labels)
		if desc.Width() != 3 {
			t.Errorf("Width() = %d, want 3 (statistic + 2 numeric)", desc.Width())
		}
		_, err := desc.Column("labels")
		if err == nil {
			t.Error("non-numeric column 'labels' should not appear in Describe")
		}
	})

	t.Run("count row is correct", func(t *testing.T) {
		desc := df.Describe()
		intsCol, _ := desc.Column("ints")
		v, _ := intsCol.GetFloat64(0) // count row
		if v != 5.0 {
			t.Errorf("ints count = %f, want 5.0", v)
		}
	})
}

func TestGlimpse(t *testing.T) {
	df := helperDF(t)

	t.Run("contains basic info", func(t *testing.T) {
		g := df.Glimpse()
		if !strings.Contains(g, "Rows: 3") {
			t.Error("Glimpse missing 'Rows: 3'")
		}
		if !strings.Contains(g, "Columns: 2") {
			t.Error("Glimpse missing 'Columns: 2'")
		}
		if !strings.Contains(g, "a") || !strings.Contains(g, "b") {
			t.Error("Glimpse missing column names")
		}
	})
}

func TestString(t *testing.T) {
	t.Run("non-empty DataFrame", func(t *testing.T) {
		df := helperDF(t)
		s := df.String()
		if !strings.Contains(s, "shape: (3, 2)") {
			t.Errorf("String() missing shape header, got:\n%s", s)
		}
		if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
			t.Error("String() missing column names")
		}
		// Verify data values appear
		if !strings.Contains(s, "1") || !strings.Contains(s, "x") {
			t.Error("String() missing data values")
		}
	})

	t.Run("empty width DataFrame", func(t *testing.T) {
		df, _ := New()
		s := df.String()
		if !strings.Contains(s, "shape: (0, 0)") {
			t.Errorf("String() = %q, want shape (0, 0)", s)
		}
	})
}

func TestDropNulls(t *testing.T) {
	t.Run("drops rows with nulls", func(t *testing.T) {
		df, err := New(
			series.NewInt64WithValidity("a", []int64{1, 0, 3}, []bool{true, false, true}),
			series.NewString("b", []string{"x", "y", "z"}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result := df.DropNulls()
		if result.Height() != 2 {
			t.Errorf("Height() = %d, want 2", result.Height())
		}
	})

	t.Run("no nulls returns same", func(t *testing.T) {
		df := helperDF(t)
		result := df.DropNulls()
		if result != df {
			t.Error("DropNulls should return same DF when no nulls present")
		}
	})

	t.Run("empty DataFrame", func(t *testing.T) {
		df, _ := New()
		result := df.DropNulls()
		if result != df {
			t.Error("DropNulls on empty DF should return same DF")
		}
	})
}

func TestFillNull(t *testing.T) {
	t.Run("fill int64 nulls", func(t *testing.T) {
		df, err := New(
			series.NewInt64WithValidity("a", []int64{1, 0, 3}, []bool{true, false, true}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result, err := df.FillNull(map[string]any{"a": int64(99)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("a")
		v, ok := col.GetInt64(1)
		if !ok || v != 99 {
			t.Errorf("filled value = %d (ok=%v), want 99", v, ok)
		}
	})

	t.Run("fill float64 nulls", func(t *testing.T) {
		df, err := New(
			series.NewFloat64WithValidity("f", []float64{1.0, 0, 3.0}, []bool{true, false, true}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result, err := df.FillNull(map[string]any{"f": float64(2.5)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("f")
		v, ok := col.GetFloat64(1)
		if !ok || v != 2.5 {
			t.Errorf("filled value = %f (ok=%v), want 2.5", v, ok)
		}
	})

	t.Run("fill string nulls", func(t *testing.T) {
		df, err := New(
			series.NewStringWithValidity("s", []string{"a", "", "c"}, []bool{true, false, true}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result, err := df.FillNull(map[string]any{"s": "filled"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		col, _ := result.Column("s")
		v, ok := col.GetString(1)
		if !ok || v != "filled" {
			t.Errorf("filled value = %q (ok=%v), want 'filled'", v, ok)
		}
	})

	t.Run("column not found", func(t *testing.T) {
		df := helperDF(t)
		_, err := df.FillNull(map[string]any{"missing": int64(0)})
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})

	t.Run("wrong fill value type", func(t *testing.T) {
		df, err := New(
			series.NewInt64WithValidity("a", []int64{1, 0, 3}, []bool{true, false, true}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		_, err = df.FillNull(map[string]any{"a": "not an int"})
		if err == nil {
			t.Fatal("expected error for wrong fill value type")
		}
	})
}

func TestUnique(t *testing.T) {
	t.Run("removes duplicates", func(t *testing.T) {
		df, err := New(
			series.NewInt64("a", []int64{1, 2, 1, 2, 3}),
			series.NewString("b", []string{"x", "y", "x", "y", "z"}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result, err := df.Unique()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 3 {
			t.Errorf("Height() = %d, want 3", result.Height())
		}
	})

	t.Run("unique by subset", func(t *testing.T) {
		df, err := New(
			series.NewInt64("a", []int64{1, 1, 2}),
			series.NewString("b", []string{"x", "y", "x"}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		result, err := df.Unique("a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 2 {
			t.Errorf("Height() = %d, want 2", result.Height())
		}
	})

	t.Run("all unique returns clone", func(t *testing.T) {
		df := helperDF(t)
		result, err := df.Unique()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != df.Height() {
			t.Errorf("Height() = %d, want %d", result.Height(), df.Height())
		}
	})

	t.Run("empty DataFrame", func(t *testing.T) {
		df, _ := New()
		result, err := df.Unique()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 0 {
			t.Errorf("Height() = %d, want 0", result.Height())
		}
	})

	t.Run("missing subset column", func(t *testing.T) {
		df := helperDF(t)
		_, err := df.Unique("missing")
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestSample(t *testing.T) {
	df := helperNumericDF(t)

	t.Run("sample n rows", func(t *testing.T) {
		result, err := df.Sample(3, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 3 {
			t.Errorf("Height() = %d, want 3", result.Height())
		}
	})

	t.Run("sample 0 rows", func(t *testing.T) {
		result, err := df.Sample(0, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 0 {
			t.Errorf("Height() = %d, want 0", result.Height())
		}
	})

	t.Run("sample all rows", func(t *testing.T) {
		result, err := df.Sample(5, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 5 {
			t.Errorf("Height() = %d, want 5", result.Height())
		}
	})

	t.Run("sample negative", func(t *testing.T) {
		_, err := df.Sample(-1, 42)
		if err == nil {
			t.Fatal("expected error for negative sample size")
		}
	})

	t.Run("sample exceeds height", func(t *testing.T) {
		_, err := df.Sample(100, 42)
		if err == nil {
			t.Fatal("expected error when sample exceeds height")
		}
	})

	t.Run("deterministic with same seed", func(t *testing.T) {
		r1, _ := df.Sample(3, 123)
		r2, _ := df.Sample(3, 123)
		for i := 0; i < 3; i++ {
			row1 := r1.Row(i)
			row2 := r2.Row(i)
			if row1["ints"] != row2["ints"] {
				t.Errorf("row %d differs between same-seed samples", i)
			}
		}
	})
}

func TestSampleFraction(t *testing.T) {
	df := helperNumericDF(t)

	t.Run("half", func(t *testing.T) {
		result, err := df.SampleFraction(0.5, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 0.5 * 5 = 2.5, rounded to 3
		if result.Height() < 2 || result.Height() > 3 {
			t.Errorf("Height() = %d, expected 2 or 3", result.Height())
		}
	})

	t.Run("zero fraction", func(t *testing.T) {
		result, err := df.SampleFraction(0.0, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 0 {
			t.Errorf("Height() = %d, want 0", result.Height())
		}
	})

	t.Run("full fraction", func(t *testing.T) {
		result, err := df.SampleFraction(1.0, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 5 {
			t.Errorf("Height() = %d, want 5", result.Height())
		}
	})

	t.Run("negative fraction", func(t *testing.T) {
		_, err := df.SampleFraction(-0.1, 42)
		if err == nil {
			t.Fatal("expected error for negative fraction")
		}
	})

	t.Run("fraction > 1", func(t *testing.T) {
		_, err := df.SampleFraction(1.5, 42)
		if err == nil {
			t.Fatal("expected error for fraction > 1")
		}
	})
}

func TestToMap(t *testing.T) {
	df := helperDF(t)

	t.Run("produces correct map", func(t *testing.T) {
		m := df.ToMap()
		ints, ok := m["a"].([]int64)
		if !ok {
			t.Fatal("column 'a' not []int64")
		}
		if len(ints) != 3 || ints[0] != 1 || ints[1] != 2 || ints[2] != 3 {
			t.Errorf("a = %v, want [1, 2, 3]", ints)
		}
		strs, ok := m["b"].([]string)
		if !ok {
			t.Fatal("column 'b' not []string")
		}
		if len(strs) != 3 || strs[0] != "x" {
			t.Errorf("b = %v, want [x, y, z]", strs)
		}
	})
}

func TestToMaps(t *testing.T) {
	df := helperDF(t)

	t.Run("produces correct row maps", func(t *testing.T) {
		rows := df.ToMaps()
		if len(rows) != 3 {
			t.Fatalf("len(ToMaps()) = %d, want 3", len(rows))
		}
		if rows[0]["a"] != int64(1) {
			t.Errorf("row 0 'a' = %v, want 1", rows[0]["a"])
		}
		if rows[0]["b"] != "x" {
			t.Errorf("row 0 'b' = %v, want 'x'", rows[0]["b"])
		}
	})
}

func TestRow(t *testing.T) {
	t.Run("valid row", func(t *testing.T) {
		df := helperDF(t)
		row := df.Row(1)
		if row["a"] != int64(2) {
			t.Errorf("Row(1)['a'] = %v, want 2", row["a"])
		}
		if row["b"] != "y" {
			t.Errorf("Row(1)['b'] = %v, want 'y'", row["b"])
		}
	})

	t.Run("null values", func(t *testing.T) {
		df, err := New(
			series.NewInt64WithValidity("a", []int64{1, 0}, []bool{true, false}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		row := df.Row(1)
		if row["a"] != nil {
			t.Errorf("Row(1)['a'] = %v, want nil", row["a"])
		}
	})

	t.Run("boolean and float columns", func(t *testing.T) {
		df, err := New(
			series.NewBoolean("flag", []bool{true, false}),
			series.NewFloat64("val", []float64{3.14, 2.72}),
		)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		row := df.Row(0)
		if row["flag"] != true {
			t.Errorf("Row(0)['flag'] = %v, want true", row["flag"])
		}
		if row["val"] != 3.14 {
			t.Errorf("Row(0)['val'] = %v, want 3.14", row["val"])
		}
	})
}

func TestConcat(t *testing.T) {
	t.Run("vertical concat", func(t *testing.T) {
		df1, _ := New(
			series.NewInt64("a", []int64{1, 2}),
			series.NewString("b", []string{"x", "y"}),
		)
		df2, _ := New(
			series.NewInt64("a", []int64{3}),
			series.NewString("b", []string{"z"}),
		)
		result, err := Concat(df1, df2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 3 {
			t.Errorf("Height() = %d, want 3", result.Height())
		}
		col, _ := result.Column("a")
		v, _ := col.GetInt64(2)
		if v != 3 {
			t.Errorf("last value = %d, want 3", v)
		}
	})

	t.Run("single DataFrame", func(t *testing.T) {
		df := helperDF(t)
		result, err := Concat(df)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != df.Height() {
			t.Errorf("Height() = %d, want %d", result.Height(), df.Height())
		}
	})

	t.Run("no DataFrames", func(t *testing.T) {
		_, err := Concat()
		if err == nil {
			t.Fatal("expected error for no DataFrames")
		}
	})

	t.Run("schema mismatch", func(t *testing.T) {
		df1, _ := New(series.NewInt64("a", []int64{1}))
		df2, _ := New(series.NewString("a", []string{"x"}))
		_, err := Concat(df1, df2)
		if err == nil {
			t.Fatal("expected error for schema mismatch")
		}
	})

	t.Run("column name mismatch", func(t *testing.T) {
		df1, _ := New(series.NewInt64("a", []int64{1}))
		df2, _ := New(series.NewInt64("b", []int64{2}))
		_, err := Concat(df1, df2)
		if err == nil {
			t.Fatal("expected error for column name mismatch")
		}
	})
}

func TestConcatHorizontal(t *testing.T) {
	t.Run("horizontal concat", func(t *testing.T) {
		df1, _ := New(series.NewInt64("a", []int64{1, 2}))
		df2, _ := New(series.NewString("b", []string{"x", "y"}))
		result, err := ConcatHorizontal(df1, df2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != 2 {
			t.Errorf("Width() = %d, want 2", result.Width())
		}
		if result.Height() != 2 {
			t.Errorf("Height() = %d, want 2", result.Height())
		}
	})

	t.Run("single DataFrame", func(t *testing.T) {
		df := helperDF(t)
		result, err := ConcatHorizontal(df)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Width() != df.Width() {
			t.Errorf("Width() = %d, want %d", result.Width(), df.Width())
		}
	})

	t.Run("no DataFrames", func(t *testing.T) {
		_, err := ConcatHorizontal()
		if err == nil {
			t.Fatal("expected error for no DataFrames")
		}
	})

	t.Run("height mismatch", func(t *testing.T) {
		df1, _ := New(series.NewInt64("a", []int64{1, 2}))
		df2, _ := New(series.NewInt64("b", []int64{1}))
		_, err := ConcatHorizontal(df1, df2)
		if err == nil {
			t.Fatal("expected error for height mismatch")
		}
	})

	t.Run("duplicate column names", func(t *testing.T) {
		df1, _ := New(series.NewInt64("a", []int64{1}))
		df2, _ := New(series.NewInt64("a", []int64{2}))
		_, err := ConcatHorizontal(df1, df2)
		if err == nil {
			t.Fatal("expected error for duplicate column names")
		}
	})
}

func TestClone(t *testing.T) {
	df := helperDF(t)
	clone := df.Clone()

	t.Run("same dimensions", func(t *testing.T) {
		if clone.Height() != df.Height() || clone.Width() != df.Width() {
			t.Errorf("clone dimensions (%d, %d) != original (%d, %d)",
				clone.Height(), clone.Width(), df.Height(), df.Width())
		}
	})

	t.Run("same schema", func(t *testing.T) {
		if !clone.Schema().Equal(df.Schema()) {
			t.Error("clone schema differs from original")
		}
	})

	t.Run("same data", func(t *testing.T) {
		col, _ := clone.Column("a")
		v, _ := col.GetInt64(0)
		if v != 1 {
			t.Errorf("clone value = %d, want 1", v)
		}
	})
}

func TestFromSchema(t *testing.T) {
	schema := dtype.NewSchema([]dtype.Field{
		{Name: "x", Dtype: dtype.Int64},
		{Name: "y", Dtype: dtype.Float64},
		{Name: "z", Dtype: dtype.String},
		{Name: "b", Dtype: dtype.Boolean},
	})

	df := FromSchema(schema, 3)
	if df.Height() != 3 {
		t.Errorf("Height() = %d, want 3", df.Height())
	}
	if df.Width() != 4 {
		t.Errorf("Width() = %d, want 4", df.Width())
	}

	// Values should be zero-initialized.
	col, _ := df.Column("x")
	v, ok := col.GetInt64(0)
	if !ok || v != 0 {
		t.Errorf("zero-init int64 = %d (ok=%v), want 0", v, ok)
	}
}

func TestConcatWithNulls(t *testing.T) {
	t.Run("vertical concat preserves nulls", func(t *testing.T) {
		df1, _ := New(
			series.NewInt64WithValidity("a", []int64{1, 0}, []bool{true, false}),
		)
		df2, _ := New(
			series.NewInt64("a", []int64{3}),
		)
		result, err := Concat(df1, df2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Height() != 3 {
			t.Errorf("Height() = %d, want 3", result.Height())
		}
		col, _ := result.Column("a")
		if !col.IsNull(1) {
			t.Error("expected null at index 1 after concat")
		}
		if col.IsNull(2) {
			t.Error("expected non-null at index 2 after concat")
		}
	})
}

func TestFilterPreservesColumns(t *testing.T) {
	df, err := New(
		series.NewInt64("a", []int64{10, 20, 30}),
		series.NewFloat64("b", []float64{1.1, 2.2, 3.3}),
		series.NewString("c", []string{"x", "y", "z"}),
		series.NewBoolean("d", []bool{true, false, true}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	mask := series.NewBoolean("mask", []bool{false, true, true})
	result, err := df.Filter(mask)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Width() != 4 {
		t.Errorf("Width() = %d, want 4", result.Width())
	}
	if result.Height() != 2 {
		t.Errorf("Height() = %d, want 2", result.Height())
	}

	// Verify each column type is preserved.
	aCol, _ := result.Column("a")
	v, _ := aCol.GetInt64(0)
	if v != 20 {
		t.Errorf("a[0] = %d, want 20", v)
	}

	bCol, _ := result.Column("b")
	fv, _ := bCol.GetFloat64(0)
	if fv != 2.2 {
		t.Errorf("b[0] = %f, want 2.2", fv)
	}

	cCol, _ := result.Column("c")
	sv, _ := cCol.GetString(0)
	if sv != "y" {
		t.Errorf("c[0] = %q, want 'y'", sv)
	}

	dCol, _ := result.Column("d")
	bv, _ := dCol.GetBool(0)
	if bv != false {
		t.Errorf("d[0] = %v, want false", bv)
	}
}

func TestSortStability(t *testing.T) {
	// Test that sort is stable: equal elements preserve original order.
	df, err := New(
		series.NewInt64("key", []int64{1, 1, 1}),
		series.NewString("val", []string{"first", "second", "third"}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := df.Sort("key", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	col, _ := result.Column("val")
	expected := []string{"first", "second", "third"}
	for i, want := range expected {
		v, _ := col.GetString(i)
		if v != want {
			t.Errorf("row %d: got %q, want %q (sort not stable)", i, v, want)
		}
	}
}

func TestUniquePreservesFirstOccurrence(t *testing.T) {
	df, err := New(
		series.NewInt64("a", []int64{1, 2, 1}),
		series.NewString("b", []string{"first", "only", "duplicate"}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := df.Unique("a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Height() != 2 {
		t.Fatalf("Height() = %d, want 2", result.Height())
	}

	col, _ := result.Column("b")
	v, _ := col.GetString(0)
	if v != "first" {
		t.Errorf("first occurrence b = %q, want 'first'", v)
	}
}

func TestToMapWithBooleanAndFloat(t *testing.T) {
	df, err := New(
		series.NewBoolean("flag", []bool{true, false}),
		series.NewFloat64("val", []float64{1.5, 2.5}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	m := df.ToMap()
	flags, ok := m["flag"].([]bool)
	if !ok {
		t.Fatal("column 'flag' not []bool")
	}
	if flags[0] != true || flags[1] != false {
		t.Errorf("flags = %v, want [true, false]", flags)
	}

	vals, ok := m["val"].([]float64)
	if !ok {
		t.Fatal("column 'val' not []float64")
	}
	if vals[0] != 1.5 || vals[1] != 2.5 {
		t.Errorf("vals = %v, want [1.5, 2.5]", vals)
	}
}

func TestStringOutputContainsTableBorders(t *testing.T) {
	df := helperDF(t)
	s := df.String()

	// Verify table border characters are present.
	borders := []string{"\u250c", "\u2510", "\u2514", "\u2518", "\u2502"}
	for _, b := range borders {
		if !strings.Contains(s, b) {
			t.Errorf("String() missing border character %q", b)
		}
	}
}

func TestStringTruncation(t *testing.T) {
	// Build a DataFrame with > 20 rows to trigger truncation.
	n := 25
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = int64(i)
	}
	df, err := New(series.NewInt64("v", vals))
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	s := df.String()
	if !strings.Contains(s, "...") {
		t.Error("String() should contain '...' for truncated display")
	}
	if !strings.Contains(s, "shape: (25, 1)") {
		t.Errorf("String() missing correct shape, got:\n%s", s)
	}
}

func TestEmptyDataFrameOperations(t *testing.T) {
	df, _ := New()

	t.Run("IsEmpty", func(t *testing.T) {
		if !df.IsEmpty() {
			t.Error("empty DF should be empty")
		}
	})

	t.Run("Len", func(t *testing.T) {
		if df.Len() != 0 {
			t.Errorf("Len() = %d, want 0", df.Len())
		}
	})

	t.Run("Clone", func(t *testing.T) {
		clone := df.Clone()
		if clone.Height() != 0 || clone.Width() != 0 {
			t.Error("clone of empty DF should be empty")
		}
	})

	t.Run("ToMap", func(t *testing.T) {
		m := df.ToMap()
		if len(m) != 0 {
			t.Errorf("ToMap() len = %d, want 0", len(m))
		}
	})

	t.Run("ToMaps", func(t *testing.T) {
		rows := df.ToMaps()
		if len(rows) != 0 {
			t.Errorf("ToMaps() len = %d, want 0", len(rows))
		}
	})
}

func TestDescribeEmpty(t *testing.T) {
	df, _ := New(series.NewInt64("a", []int64{}))
	desc := df.Describe()
	// Should still produce a DataFrame with statistic labels.
	if desc.Height() != 5 {
		t.Errorf("Describe().Height() = %d, want 5", desc.Height())
	}
}

func TestGlimpseWithManyRows(t *testing.T) {
	n := 100
	vals := make([]int64, n)
	for i := range vals {
		vals[i] = int64(i)
	}
	df, err := New(series.NewInt64("big", vals))
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	g := df.Glimpse()
	if !strings.Contains(g, "Rows: 100") {
		t.Error("Glimpse missing 'Rows: 100'")
	}
	if !strings.Contains(g, "...") {
		t.Error("Glimpse should truncate preview with '...'")
	}
}

func TestConcatMultipleDataFrames(t *testing.T) {
	df1, _ := New(series.NewInt64("a", []int64{1}))
	df2, _ := New(series.NewInt64("a", []int64{2}))
	df3, _ := New(series.NewInt64("a", []int64{3}))

	result, err := Concat(df1, df2, df3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Height() != 3 {
		t.Errorf("Height() = %d, want 3", result.Height())
	}
	col, _ := result.Column("a")
	for i, want := range []int64{1, 2, 3} {
		v, _ := col.GetInt64(i)
		if v != want {
			t.Errorf("row %d = %d, want %d", i, v, want)
		}
	}
}

func TestConcatHorizontalMultiple(t *testing.T) {
	df1, _ := New(series.NewInt64("a", []int64{1, 2}))
	df2, _ := New(series.NewString("b", []string{"x", "y"}))
	df3, _ := New(series.NewFloat64("c", []float64{1.1, 2.2}))

	result, err := ConcatHorizontal(df1, df2, df3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Width() != 3 {
		t.Errorf("Width() = %d, want 3", result.Width())
	}
	if result.Height() != 2 {
		t.Errorf("Height() = %d, want 2", result.Height())
	}
}

func TestWithColumnOnEmptyDF(t *testing.T) {
	df, _ := New()
	result, err := df.WithColumn(series.NewInt64("a", []int64{1, 2, 3}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Width() != 1 {
		t.Errorf("Width() = %d, want 1", result.Width())
	}
	if result.Height() != 3 {
		t.Errorf("Height() = %d, want 3", result.Height())
	}
}

func TestSelectPreservesOrder(t *testing.T) {
	df, err := New(
		series.NewInt64("a", []int64{1}),
		series.NewInt64("b", []int64{2}),
		series.NewInt64("c", []int64{3}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := df.Select("c", "a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := result.Schema().Names()
	want := []string{"c", "a", "b"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("column %d: got %q, want %q", i, names[i], want[i])
		}
	}
}

func TestDropAllColumns(t *testing.T) {
	df := helperDF(t)
	result, err := df.Drop("a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Width() != 0 {
		t.Errorf("Width() = %d, want 0 after dropping all columns", result.Width())
	}
}

func TestConcatBooleanColumns(t *testing.T) {
	df1, _ := New(series.NewBoolean("flag", []bool{true, false}))
	df2, _ := New(series.NewBoolean("flag", []bool{false, true}))

	result, err := Concat(df1, df2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Height() != 4 {
		t.Errorf("Height() = %d, want 4", result.Height())
	}
	col, _ := result.Column("flag")
	expected := []bool{true, false, false, true}
	for i, want := range expected {
		v, ok := col.GetBool(i)
		if !ok || v != want {
			t.Errorf("row %d: got %v (ok=%v), want %v", i, v, ok, want)
		}
	}
}

func TestConcatFloat64Columns(t *testing.T) {
	df1, _ := New(series.NewFloat64("val", []float64{1.1, 2.2}))
	df2, _ := New(series.NewFloat64("val", []float64{3.3}))

	result, err := Concat(df1, df2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Height() != 3 {
		t.Errorf("Height() = %d, want 3", result.Height())
	}
	col, _ := result.Column("val")
	v, _ := col.GetFloat64(2)
	if v != 3.3 {
		t.Errorf("val[2] = %f, want 3.3", v)
	}
}

func TestConcatStringColumns(t *testing.T) {
	df1, _ := New(series.NewString("s", []string{"a", "b"}))
	df2, _ := New(series.NewString("s", []string{"c"}))

	result, err := Concat(df1, df2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	col, _ := result.Column("s")
	v, _ := col.GetString(2)
	if v != "c" {
		t.Errorf("s[2] = %q, want 'c'", v)
	}
}

// ---------------------------------------------------------------------------
// GroupBy tests
// ---------------------------------------------------------------------------

func TestGroupByBasicAggregations(t *testing.T) {
	// Data:
	//   group | value
	//   "a"   | 10
	//   "b"   | 20
	//   "a"   | 30
	//   "b"   | 40
	//   "a"   | 50
	df, err := New(
		series.NewString("group", []string{"a", "b", "a", "b", "a"}),
		series.NewInt64("value", []int64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("AggSum", func(t *testing.T) {
		gb, err := df.GroupBy("group")
		if err != nil {
			t.Fatalf("GroupBy: %v", err)
		}
		result, err := gb.Agg(map[string]AggFunc{"value": AggSum})
		if err != nil {
			t.Fatalf("Agg: %v", err)
		}
		if result.Height() != 2 {
			t.Fatalf("Height() = %d, want 2", result.Height())
		}
		groupCol, _ := result.Column("group")
		valCol, _ := result.Column("value")

		// Groups should be "a" and "b" (insertion order)
		g0, _ := groupCol.GetString(0)
		g1, _ := groupCol.GetString(1)
		v0, _ := valCol.GetFloat64(0)
		v1, _ := valCol.GetFloat64(1)

		if g0 != "a" || v0 != 90.0 {
			t.Errorf("group a: got sum=%f, want 90", v0)
		}
		if g1 != "b" || v1 != 60.0 {
			t.Errorf("group b: got sum=%f, want 60", v1)
		}
	})

	t.Run("AggMean", func(t *testing.T) {
		gb, err := df.GroupBy("group")
		if err != nil {
			t.Fatalf("GroupBy: %v", err)
		}
		result, err := gb.Agg(map[string]AggFunc{"value": AggMean})
		if err != nil {
			t.Fatalf("Agg: %v", err)
		}
		groupCol, _ := result.Column("group")
		valCol, _ := result.Column("value")

		g0, _ := groupCol.GetString(0)
		v0, _ := valCol.GetFloat64(0)
		g1, _ := groupCol.GetString(1)
		v1, _ := valCol.GetFloat64(1)

		if g0 != "a" || v0 != 30.0 {
			t.Errorf("group a: got mean=%f, want 30", v0)
		}
		if g1 != "b" || v1 != 30.0 {
			t.Errorf("group b: got mean=%f, want 30", v1)
		}
	})

	t.Run("AggCount", func(t *testing.T) {
		gb, err := df.GroupBy("group")
		if err != nil {
			t.Fatalf("GroupBy: %v", err)
		}
		result, err := gb.Agg(map[string]AggFunc{"value": AggCount})
		if err != nil {
			t.Fatalf("Agg: %v", err)
		}
		groupCol, _ := result.Column("group")
		valCol, _ := result.Column("value")

		g0, _ := groupCol.GetString(0)
		c0, _ := valCol.GetInt64(0)
		g1, _ := groupCol.GetString(1)
		c1, _ := valCol.GetInt64(1)

		if g0 != "a" || c0 != 3 {
			t.Errorf("group a: got count=%d, want 3", c0)
		}
		if g1 != "b" || c1 != 2 {
			t.Errorf("group b: got count=%d, want 2", c1)
		}
	})
}

func TestGroupByMultipleKeys(t *testing.T) {
	df, err := New(
		series.NewString("dept", []string{"eng", "eng", "sales", "sales", "eng"}),
		series.NewString("level", []string{"sr", "jr", "sr", "sr", "sr"}),
		series.NewInt64("salary", []int64{100, 50, 80, 90, 110}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	gb, err := df.GroupBy("dept", "level")
	if err != nil {
		t.Fatalf("GroupBy: %v", err)
	}
	result, err := gb.Agg(map[string]AggFunc{"salary": AggSum})
	if err != nil {
		t.Fatalf("Agg: %v", err)
	}

	// Expected groups (insertion order): (eng, sr), (eng, jr), (sales, sr)
	if result.Height() != 3 {
		t.Fatalf("Height() = %d, want 3", result.Height())
	}

	deptCol, _ := result.Column("dept")
	levelCol, _ := result.Column("level")
	salaryCol, _ := result.Column("salary")

	type row struct {
		dept, level string
		salary      float64
	}
	var rows []row
	for i := 0; i < result.Height(); i++ {
		d, _ := deptCol.GetString(i)
		l, _ := levelCol.GetString(i)
		s, _ := salaryCol.GetFloat64(i)
		rows = append(rows, row{d, l, s})
	}

	expected := []row{
		{"eng", "sr", 210},
		{"eng", "jr", 50},
		{"sales", "sr", 170},
	}
	for i, want := range expected {
		got := rows[i]
		if got.dept != want.dept || got.level != want.level || got.salary != want.salary {
			t.Errorf("row %d: got %+v, want %+v", i, got, want)
		}
	}
}

func TestGroupByMinMaxFirstLast(t *testing.T) {
	df, err := New(
		series.NewString("grp", []string{"x", "x", "x", "y", "y"}),
		series.NewInt64("val", []int64{30, 10, 20, 5, 15}),
	)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name    string
		aggFunc AggFunc
		wantX   float64
		wantY   float64
	}{
		{"AggMin", AggMin, 10, 5},
		{"AggMax", AggMax, 30, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gb, _ := df.GroupBy("grp")
			result, err := gb.Agg(map[string]AggFunc{"val": tt.aggFunc})
			if err != nil {
				t.Fatalf("Agg: %v", err)
			}
			valCol, _ := result.Column("val")
			v0, _ := valCol.GetFloat64(0)
			v1, _ := valCol.GetFloat64(1)
			if v0 != tt.wantX {
				t.Errorf("group x: got %f, want %f", v0, tt.wantX)
			}
			if v1 != tt.wantY {
				t.Errorf("group y: got %f, want %f", v1, tt.wantY)
			}
		})
	}

	t.Run("AggFirst", func(t *testing.T) {
		gb, _ := df.GroupBy("grp")
		result, err := gb.Agg(map[string]AggFunc{"val": AggFirst})
		if err != nil {
			t.Fatalf("Agg: %v", err)
		}
		valCol, _ := result.Column("val")
		// First valid value for group x (rows 0,1,2) is 30; for y (rows 3,4) is 5
		v0, _ := valCol.GetInt64(0)
		v1, _ := valCol.GetInt64(1)
		if v0 != 30 {
			t.Errorf("group x first: got %d, want 30", v0)
		}
		if v1 != 5 {
			t.Errorf("group y first: got %d, want 5", v1)
		}
	})

	t.Run("AggLast", func(t *testing.T) {
		gb, _ := df.GroupBy("grp")
		result, err := gb.Agg(map[string]AggFunc{"val": AggLast})
		if err != nil {
			t.Fatalf("Agg: %v", err)
		}
		valCol, _ := result.Column("val")
		// Last valid value for group x (rows 0,1,2) is 20; for y (rows 3,4) is 15
		v0, _ := valCol.GetInt64(0)
		v1, _ := valCol.GetInt64(1)
		if v0 != 20 {
			t.Errorf("group x last: got %d, want 20", v0)
		}
		if v1 != 15 {
			t.Errorf("group y last: got %d, want 15", v1)
		}
	})
}

func TestGroupByColumnNotFound(t *testing.T) {
	df := helperDF(t)
	_, err := df.GroupBy("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing groupby column")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestGroupByAggColumnNotFound(t *testing.T) {
	df := helperDF(t)
	gb, err := df.GroupBy("a")
	if err != nil {
		t.Fatalf("GroupBy: %v", err)
	}
	_, err = gb.Agg(map[string]AggFunc{"nonexistent": AggSum})
	if err == nil {
		t.Fatal("expected error for missing agg column")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Join tests
// ---------------------------------------------------------------------------

// buildLeftDF and buildRightDF create standard test DataFrames for join tests.
//
//	left:                    right:
//	  id | lval               id | rval
//	   1 | "a"                 1 | "x"
//	   2 | "b"                 3 | "y"
//	   3 | "c"                 4 | "z"
func buildLeftDF(t *testing.T) *DataFrame {
	t.Helper()
	df, err := New(
		series.NewInt64("id", []int64{1, 2, 3}),
		series.NewString("lval", []string{"a", "b", "c"}),
	)
	if err != nil {
		t.Fatalf("buildLeftDF: %v", err)
	}
	return df
}

func buildRightDF(t *testing.T) *DataFrame {
	t.Helper()
	df, err := New(
		series.NewInt64("id", []int64{1, 3, 4}),
		series.NewString("rval", []string{"x", "y", "z"}),
	)
	if err != nil {
		t.Fatalf("buildRightDF: %v", err)
	}
	return df
}

func TestJoinInner(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, InnerJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Inner join: only rows with id in both: id=1, id=3
	if result.Height() != 2 {
		t.Fatalf("Height() = %d, want 2", result.Height())
	}

	idCol, _ := result.Column("id")
	lvalCol, _ := result.Column("lval")
	rvalCol, _ := result.Column("rval")

	id0, _ := idCol.GetInt64(0)
	id1, _ := idCol.GetInt64(1)
	lv0, _ := lvalCol.GetString(0)
	lv1, _ := lvalCol.GetString(1)
	rv0, _ := rvalCol.GetString(0)
	rv1, _ := rvalCol.GetString(1)

	if id0 != 1 || lv0 != "a" || rv0 != "x" {
		t.Errorf("row 0: got id=%d lval=%q rval=%q, want 1/a/x", id0, lv0, rv0)
	}
	if id1 != 3 || lv1 != "c" || rv1 != "y" {
		t.Errorf("row 1: got id=%d lval=%q rval=%q, want 3/c/y", id1, lv1, rv1)
	}
}

func TestJoinLeft(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, LeftJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Left join: all left rows (1,2,3). id=2 has no match -> rval is null.
	if result.Height() != 3 {
		t.Fatalf("Height() = %d, want 3", result.Height())
	}

	idCol, _ := result.Column("id")
	rvalCol, _ := result.Column("rval")

	// Row with id=2 should have null rval
	for i := 0; i < result.Height(); i++ {
		id, _ := idCol.GetInt64(i)
		if id == 2 {
			if !rvalCol.IsNull(i) {
				t.Errorf("row with id=2: rval should be null")
			}
		} else {
			if rvalCol.IsNull(i) {
				t.Errorf("row with id=%d: rval should not be null", id)
			}
		}
	}

	// Verify matched values
	id0, _ := idCol.GetInt64(0)
	rv0, _ := rvalCol.GetString(0)
	if id0 != 1 || rv0 != "x" {
		t.Errorf("row 0: got id=%d rval=%q, want 1/x", id0, rv0)
	}
}

func TestJoinRight(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, RightJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Right join: all right rows (1,3,4). id=4 has no match -> lval is null.
	if result.Height() != 3 {
		t.Fatalf("Height() = %d, want 3", result.Height())
	}

	idCol, _ := result.Column("id")
	lvalCol, _ := result.Column("lval")

	// Find the row with id=4 (unmatched right row)
	foundUnmatched := false
	for i := 0; i < result.Height(); i++ {
		id, ok := idCol.GetInt64(i)
		if ok && id == 4 {
			foundUnmatched = true
			if !lvalCol.IsNull(i) {
				t.Errorf("row with id=4: lval should be null")
			}
		}
	}
	if !foundUnmatched {
		t.Error("expected a row with id=4 from unmatched right side")
	}
}

func TestJoinFull(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, FullJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Full join: ids 1,2,3,4 -> 4 rows.
	// id=2: rval null, id=4: lval null
	if result.Height() != 4 {
		t.Fatalf("Height() = %d, want 4", result.Height())
	}

	idCol, _ := result.Column("id")
	lvalCol, _ := result.Column("lval")
	rvalCol, _ := result.Column("rval")

	for i := 0; i < result.Height(); i++ {
		id, _ := idCol.GetInt64(i)
		switch id {
		case 1:
			lv, _ := lvalCol.GetString(i)
			rv, _ := rvalCol.GetString(i)
			if lv != "a" || rv != "x" {
				t.Errorf("id=1: got lval=%q rval=%q, want a/x", lv, rv)
			}
		case 2:
			if !rvalCol.IsNull(i) {
				t.Errorf("id=2: rval should be null")
			}
			lv, _ := lvalCol.GetString(i)
			if lv != "b" {
				t.Errorf("id=2: lval=%q, want b", lv)
			}
		case 3:
			lv, _ := lvalCol.GetString(i)
			rv, _ := rvalCol.GetString(i)
			if lv != "c" || rv != "y" {
				t.Errorf("id=3: got lval=%q rval=%q, want c/y", lv, rv)
			}
		case 4:
			if !lvalCol.IsNull(i) {
				t.Errorf("id=4: lval should be null")
			}
			rv, _ := rvalCol.GetString(i)
			if rv != "z" {
				t.Errorf("id=4: rval=%q, want z", rv)
			}
		}
	}
}

func TestJoinSemi(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, SemiJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Semi join: left rows that have matches in right -> ids 1,3
	if result.Height() != 2 {
		t.Fatalf("Height() = %d, want 2", result.Height())
	}

	// Semi join should only have left columns
	if result.Width() != 2 {
		t.Errorf("Width() = %d, want 2 (only left columns)", result.Width())
	}

	idCol, _ := result.Column("id")
	lvalCol, _ := result.Column("lval")

	id0, _ := idCol.GetInt64(0)
	id1, _ := idCol.GetInt64(1)
	lv0, _ := lvalCol.GetString(0)
	lv1, _ := lvalCol.GetString(1)

	if id0 != 1 || lv0 != "a" {
		t.Errorf("row 0: got id=%d lval=%q, want 1/a", id0, lv0)
	}
	if id1 != 3 || lv1 != "c" {
		t.Errorf("row 1: got id=%d lval=%q, want 3/c", id1, lv1)
	}

	// Verify right columns are NOT present
	_, err = result.Column("rval")
	if err == nil {
		t.Error("semi join should not include right columns")
	}
}

func TestJoinAnti(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	result, err := left.Join(right, []string{"id"}, AntiJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Anti join: left rows that do NOT have matches in right -> id=2
	if result.Height() != 1 {
		t.Fatalf("Height() = %d, want 1", result.Height())
	}

	// Anti join should only have left columns
	if result.Width() != 2 {
		t.Errorf("Width() = %d, want 2 (only left columns)", result.Width())
	}

	idCol, _ := result.Column("id")
	lvalCol, _ := result.Column("lval")

	id0, _ := idCol.GetInt64(0)
	lv0, _ := lvalCol.GetString(0)
	if id0 != 2 || lv0 != "b" {
		t.Errorf("row 0: got id=%d lval=%q, want 2/b", id0, lv0)
	}
}

func TestJoinCross(t *testing.T) {
	left, _ := New(
		series.NewInt64("lid", []int64{1, 2}),
	)
	right, _ := New(
		series.NewString("rval", []string{"a", "b", "c"}),
	)

	result, err := left.Join(right, nil, CrossJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Cross join: 2 * 3 = 6 rows
	if result.Height() != 6 {
		t.Fatalf("Height() = %d, want 6", result.Height())
	}

	lidCol, _ := result.Column("lid")
	rvalCol, _ := result.Column("rval")

	// Expected: (1,a), (1,b), (1,c), (2,a), (2,b), (2,c)
	expectedIDs := []int64{1, 1, 1, 2, 2, 2}
	expectedVals := []string{"a", "b", "c", "a", "b", "c"}

	for i := 0; i < 6; i++ {
		id, _ := lidCol.GetInt64(i)
		rv, _ := rvalCol.GetString(i)
		if id != expectedIDs[i] || rv != expectedVals[i] {
			t.Errorf("row %d: got id=%d rval=%q, want %d/%q",
				i, id, rv, expectedIDs[i], expectedVals[i])
		}
	}
}

func TestJoinColumnNameConflict(t *testing.T) {
	// Both DataFrames have a "val" column (non-key)
	left, _ := New(
		series.NewInt64("id", []int64{1, 2}),
		series.NewString("val", []string{"left1", "left2"}),
	)
	right, _ := New(
		series.NewInt64("id", []int64{1, 2}),
		series.NewString("val", []string{"right1", "right2"}),
	)

	result, err := left.Join(right, []string{"id"}, InnerJoin)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	// Should have columns: id, val, val_right
	if result.Width() != 3 {
		t.Fatalf("Width() = %d, want 3", result.Width())
	}

	valCol, err := result.Column("val")
	if err != nil {
		t.Fatalf("missing column 'val': %v", err)
	}
	valRightCol, err := result.Column("val_right")
	if err != nil {
		t.Fatalf("missing column 'val_right': %v", err)
	}

	v0, _ := valCol.GetString(0)
	vr0, _ := valRightCol.GetString(0)
	if v0 != "left1" {
		t.Errorf("val[0] = %q, want 'left1'", v0)
	}
	if vr0 != "right1" {
		t.Errorf("val_right[0] = %q, want 'right1'", vr0)
	}
}

func TestJoinOnDifferentColumnNames(t *testing.T) {
	left, _ := New(
		series.NewInt64("left_key", []int64{1, 2, 3}),
		series.NewString("lval", []string{"a", "b", "c"}),
	)
	right, _ := New(
		series.NewInt64("right_key", []int64{2, 3, 4}),
		series.NewString("rval", []string{"x", "y", "z"}),
	)

	result, err := left.JoinOn(right, []string{"left_key"}, []string{"right_key"}, InnerJoin)
	if err != nil {
		t.Fatalf("JoinOn: %v", err)
	}

	// Inner join on left_key == right_key: matches at 2 and 3
	if result.Height() != 2 {
		t.Fatalf("Height() = %d, want 2", result.Height())
	}

	lkCol, _ := result.Column("left_key")
	lvalCol, _ := result.Column("lval")
	rvalCol, _ := result.Column("rval")

	lk0, _ := lkCol.GetInt64(0)
	lv0, _ := lvalCol.GetString(0)
	rv0, _ := rvalCol.GetString(0)
	if lk0 != 2 || lv0 != "b" || rv0 != "x" {
		t.Errorf("row 0: got left_key=%d lval=%q rval=%q, want 2/b/x", lk0, lv0, rv0)
	}

	lk1, _ := lkCol.GetInt64(1)
	lv1, _ := lvalCol.GetString(1)
	rv1, _ := rvalCol.GetString(1)
	if lk1 != 3 || lv1 != "c" || rv1 != "y" {
		t.Errorf("row 1: got left_key=%d lval=%q rval=%q, want 3/c/y", lk1, lv1, rv1)
	}
}

func TestJoinValidationErrors(t *testing.T) {
	left := buildLeftDF(t)
	right := buildRightDF(t)

	t.Run("left column not found", func(t *testing.T) {
		_, err := left.Join(right, []string{"nonexistent"}, InnerJoin)
		if err == nil {
			t.Fatal("expected error for missing left column")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to mention 'not found'", err.Error())
		}
	})

	t.Run("right column not found", func(t *testing.T) {
		_, err := left.JoinOn(right, []string{"id"}, []string{"nonexistent"}, InnerJoin)
		if err == nil {
			t.Fatal("expected error for missing right column")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want it to mention 'not found'", err.Error())
		}
	})

	t.Run("empty keys for non-cross join", func(t *testing.T) {
		_, err := left.Join(right, []string{}, InnerJoin)
		if err == nil {
			t.Fatal("expected error for empty join keys")
		}
	})

	t.Run("mismatched left_on right_on length", func(t *testing.T) {
		_, err := left.JoinOn(right, []string{"id"}, []string{"id", "rval"}, InnerJoin)
		if err == nil {
			t.Fatal("expected error for mismatched key lengths")
		}
	})
}
