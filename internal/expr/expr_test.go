package expr

import (
	"math"
	"testing"
	"time"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

func testDF(t *testing.T) (*dataframe.DataFrame, *Context) {
	t.Helper()
	df, err := dataframe.New(
		series.NewInt64("age", []int64{25, 30, 35, 40}),
		series.NewFloat64("score", []float64{88.5, 92.3, 76.1, 95.0}),
		series.NewString("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		series.NewBoolean("active", []bool{true, false, true, true}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return df, &Context{DF: df}
}

func TestCol(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("existing column", func(t *testing.T) {
		s, err := Col("age").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Name() != "age" {
			t.Errorf("expected name 'age', got %q", s.Name())
		}
		if s.Len() != 4 {
			t.Errorf("expected len 4, got %d", s.Len())
		}
	})

	t.Run("missing column", func(t *testing.T) {
		_, err := Col("missing").Evaluate(ctx)
		if err == nil {
			t.Fatal("expected error for missing column")
		}
	})
}

func TestLit(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("int literal", func(t *testing.T) {
		s, err := Lit(42).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 4 {
			t.Errorf("expected len 4, got %d", s.Len())
		}
		v, ok := s.GetInt64(0)
		if !ok || v != 42 {
			t.Errorf("expected 42, got %d", v)
		}
	})

	t.Run("float literal", func(t *testing.T) {
		s, err := Lit(3.14).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetFloat64(2)
		if !ok || v != 3.14 {
			t.Errorf("expected 3.14, got %g", v)
		}
	})

	t.Run("string literal", func(t *testing.T) {
		s, err := Lit("hello").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetString(0)
		if !ok || v != "hello" {
			t.Errorf("expected 'hello', got %q", v)
		}
	})

	t.Run("bool literal", func(t *testing.T) {
		s, err := Lit(true).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetBool(0)
		if !ok || !v {
			t.Error("expected true")
		}
	})
}

func TestArithmetic(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("add int64", func(t *testing.T) {
		s, err := Col("age").Add(Lit(10)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetInt64(0)
		if !ok || v != 35 {
			t.Errorf("expected 35, got %d", v)
		}
	})

	t.Run("mul float64", func(t *testing.T) {
		s, err := Col("score").Mul(Lit(2.0)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetFloat64(0)
		if !ok || v != 177.0 {
			t.Errorf("expected 177.0, got %g", v)
		}
	})

	t.Run("mixed int + float", func(t *testing.T) {
		s, err := Col("age").Add(Lit(0.5)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.DataType() != dtype.Float64 {
			t.Errorf("expected Float64, got %s", s.DataType())
		}
		v, ok := s.GetFloat64(0)
		if !ok || v != 25.5 {
			t.Errorf("expected 25.5, got %g", v)
		}
	})

	t.Run("sub", func(t *testing.T) {
		s, err := Col("age").Sub(Lit(5)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(0)
		if v != 20 {
			t.Errorf("expected 20, got %d", v)
		}
	})

	t.Run("div", func(t *testing.T) {
		s, err := Col("score").Div(Lit(2.0)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 44.25 {
			t.Errorf("expected 44.25, got %g", v)
		}
	})
}

func TestComparison(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("gt", func(t *testing.T) {
		s, err := Col("age").Gt(Lit(30)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// [25>30=false, 30>30=false, 35>30=true, 40>30=true]
		v, _ := s.GetBool(0)
		if v {
			t.Error("expected false for 25 > 30")
		}
		v, _ = s.GetBool(2)
		if !v {
			t.Error("expected true for 35 > 30")
		}
	})

	t.Run("eq", func(t *testing.T) {
		s, err := Col("age").Eq(Lit(30)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetBool(1)
		if !v {
			t.Error("expected true for 30 == 30")
		}
	})

	t.Run("lt", func(t *testing.T) {
		s, err := Col("age").Lt(Lit(30)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetBool(0)
		if !v {
			t.Error("expected true for 25 < 30")
		}
	})

	t.Run("lte", func(t *testing.T) {
		s, err := Col("age").Lte(Lit(30)).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v0, _ := s.GetBool(0) // 25 <= 30 = true
		v1, _ := s.GetBool(1) // 30 <= 30 = true
		v2, _ := s.GetBool(2) // 35 <= 30 = false
		if !v0 || !v1 || v2 {
			t.Errorf("lte: got [%v,%v,%v], want [true,true,false]", v0, v1, v2)
		}
	})
}

func TestLogical(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("and", func(t *testing.T) {
		s, err := Col("age").Gt(Lit(25)).And(Col("score").Gt(Lit(80.0))).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// age>25: [F,T,T,T], score>80: [T,T,F,T] => and: [F,T,F,T]
		v0, _ := s.GetBool(0)
		v1, _ := s.GetBool(1)
		v2, _ := s.GetBool(2)
		v3, _ := s.GetBool(3)
		if v0 || !v1 || v2 || !v3 {
			t.Errorf("and: got [%v,%v,%v,%v], want [false,true,false,true]", v0, v1, v2, v3)
		}
	})

	t.Run("not", func(t *testing.T) {
		s, err := Col("active").Not().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// active: [T,F,T,T] => not: [F,T,F,F]
		v0, _ := s.GetBool(0)
		v1, _ := s.GetBool(1)
		if v0 || !v1 {
			t.Errorf("not: got [%v,%v,...], want [false,true,...]", v0, v1)
		}
	})
}

func TestAgg(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("sum", func(t *testing.T) {
		s, err := Col("age").Sum().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 130.0 {
			t.Errorf("expected 130, got %g", v)
		}
	})

	t.Run("mean", func(t *testing.T) {
		s, err := Col("age").Mean().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 32.5 {
			t.Errorf("expected 32.5, got %g", v)
		}
	})

	t.Run("count", func(t *testing.T) {
		s, err := Col("age").Count().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(0)
		if v != 4 {
			t.Errorf("expected 4, got %d", v)
		}
	})

	t.Run("min", func(t *testing.T) {
		s, err := Col("score").Min().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 76.1 {
			t.Errorf("expected 76.1, got %g", v)
		}
	})

	t.Run("max", func(t *testing.T) {
		s, err := Col("score").Max().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 95.0 {
			t.Errorf("expected 95.0, got %g", v)
		}
	})

	t.Run("std", func(t *testing.T) {
		s, err := Col("score").Std().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if math.IsNaN(v) || v <= 0 {
			t.Errorf("expected positive std, got %g", v)
		}
	})
}

func TestWhenThenOtherwise(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("basic", func(t *testing.T) {
		e := When(Col("age").Gt(Lit(30))).
			Then(Lit("senior")).
			Otherwise(Lit("junior"))

		s, err := e.Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v0, _ := s.GetString(0) // 25 => junior
		v2, _ := s.GetString(2) // 35 => senior
		if v0 != "junior" {
			t.Errorf("expected 'junior', got %q", v0)
		}
		if v2 != "senior" {
			t.Errorf("expected 'senior', got %q", v2)
		}
	})

	t.Run("numeric", func(t *testing.T) {
		e := When(Col("active")).
			Then(Lit(100)).
			Otherwise(Lit(0))

		s, err := e.Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v0, _ := s.GetInt64(0) // active=true => 100
		v1, _ := s.GetInt64(1) // active=false => 0
		if v0 != 100 || v1 != 0 {
			t.Errorf("expected [100,0], got [%d,%d]", v0, v1)
		}
	})
}

func TestAlias(t *testing.T) {
	_, ctx := testDF(t)

	s, err := Col("age").Add(Lit(1)).Alias("age_plus_1").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name() != "age_plus_1" {
		t.Errorf("expected name 'age_plus_1', got %q", s.Name())
	}
}

func TestIsNull(t *testing.T) {
	df, err := dataframe.New(
		series.NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	t.Run("is_null", func(t *testing.T) {
		s, err := Col("x").IsNull().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v0, _ := s.GetBool(0)
		v1, _ := s.GetBool(1)
		v2, _ := s.GetBool(2)
		if v0 || !v1 || v2 {
			t.Errorf("is_null: got [%v,%v,%v], want [false,true,false]", v0, v1, v2)
		}
	})

	t.Run("is_not_null", func(t *testing.T) {
		s, err := Col("x").IsNotNull().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v1, _ := s.GetBool(1)
		if v1 {
			t.Error("expected false for null element")
		}
	})
}

func TestFillNull(t *testing.T) {
	df, err := dataframe.New(
		series.NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	s, err := Col("x").FillNull(Lit(99)).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v1, ok := s.GetInt64(1)
	if !ok {
		t.Error("expected non-null after fill")
	}
	// FillNull currently goes through FillNullInt64 which takes int64
	// But Lit(99) produces int64 series - this should work through the fill mechanism
	_ = v1
}

func TestCast(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("int to float", func(t *testing.T) {
		s, err := Col("age").Cast(dtype.Float64).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.DataType() != dtype.Float64 {
			t.Errorf("expected Float64, got %s", s.DataType())
		}
		v, _ := s.GetFloat64(0)
		if v != 25.0 {
			t.Errorf("expected 25.0, got %g", v)
		}
	})

	t.Run("int to string", func(t *testing.T) {
		s, err := Col("age").Cast(dtype.String).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetString(0)
		if v != "25" {
			t.Errorf("expected '25', got %q", v)
		}
	})
}

func TestSort(t *testing.T) {
	_, ctx := testDF(t)

	s, err := Col("score").Sort(false).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Ascending: 76.1, 88.5, 92.3, 95.0
	v0, _ := s.GetFloat64(0)
	if v0 != 76.1 {
		t.Errorf("expected 76.1 first, got %g", v0)
	}
}

func TestStrNamespace(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("contains", func(t *testing.T) {
		s, err := Col("name").Str().Contains("li").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// Alice=true, Bob=false, Charlie=true (has "li"), Diana=false
		// Actually "Ali" contains "li" -> true. "Charlie" contains "li" -> true
		v0, _ := s.GetBool(0)
		v1, _ := s.GetBool(1)
		v2, _ := s.GetBool(2)
		if !v0 || v1 || !v2 {
			t.Errorf("contains 'li': got [%v,%v,%v], want [true,false,true]", v0, v1, v2)
		}
	})

	t.Run("to_upper", func(t *testing.T) {
		s, err := Col("name").Str().ToUpper().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetString(0)
		if v != "ALICE" {
			t.Errorf("expected ALICE, got %q", v)
		}
	})

	t.Run("lengths", func(t *testing.T) {
		s, err := Col("name").Str().Lengths().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(0) // "Alice" = 5
		if v != 5 {
			t.Errorf("expected 5, got %d", v)
		}
	})
}

func TestExprString(t *testing.T) {
	e := Col("age").Add(Lit(1)).Alias("age_plus_1")
	s := e.String()
	if s == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestWindowOver(t *testing.T) {
	df, err := dataframe.New(
		series.NewString("group", []string{"a", "a", "b", "b", "b"}),
		series.NewFloat64("score", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	t.Run("mean over group", func(t *testing.T) {
		s, err := Col("score").Mean().Over("group").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 5 {
			t.Fatalf("expected 5 rows, got %d", s.Len())
		}
		// group "a": mean(10,20)=15, group "b": mean(30,40,50)=40
		expected := []float64{15, 15, 40, 40, 40}
		for i, want := range expected {
			got, ok := s.GetFloat64(i)
			if !ok {
				t.Errorf("row %d: expected valid value", i)
				continue
			}
			if got != want {
				t.Errorf("row %d: expected %g, got %g", i, want, got)
			}
		}
	})

	t.Run("sum over group", func(t *testing.T) {
		s, err := Col("score").Sum().Over("group").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// group "a": sum(10,20)=30, group "b": sum(30,40,50)=120
		expected := []float64{30, 30, 120, 120, 120}
		for i, want := range expected {
			got, ok := s.GetFloat64(i)
			if !ok {
				t.Errorf("row %d: expected valid value", i)
				continue
			}
			if got != want {
				t.Errorf("row %d: expected %g, got %g", i, want, got)
			}
		}
	})

	t.Run("count over group", func(t *testing.T) {
		s, err := Col("score").Count().Over("group").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// group "a": count=2, group "b": count=3
		expectedInt := []int64{2, 2, 3, 3, 3}
		for i, want := range expectedInt {
			got, ok := s.GetInt64(i)
			if !ok {
				t.Errorf("row %d: expected valid value", i)
				continue
			}
			if got != want {
				t.Errorf("row %d: expected %d, got %d", i, want, got)
			}
		}
	})

	t.Run("min over group", func(t *testing.T) {
		s, err := Col("score").Min().Over("group").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// group "a": min(10,20)=10, group "b": min(30,40,50)=30
		expected := []float64{10, 10, 30, 30, 30}
		for i, want := range expected {
			got, ok := s.GetFloat64(i)
			if !ok {
				t.Errorf("row %d: expected valid value", i)
				continue
			}
			if got != want {
				t.Errorf("row %d: expected %g, got %g", i, want, got)
			}
		}
	})

	t.Run("max over group", func(t *testing.T) {
		s, err := Col("score").Max().Over("group").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// group "a": max(10,20)=20, group "b": max(30,40,50)=50
		expected := []float64{20, 20, 50, 50, 50}
		for i, want := range expected {
			got, ok := s.GetFloat64(i)
			if !ok {
				t.Errorf("row %d: expected valid value", i)
				continue
			}
			if got != want {
				t.Errorf("row %d: expected %g, got %g", i, want, got)
			}
		}
	})

	t.Run("with alias", func(t *testing.T) {
		s, err := Col("score").Mean().Over("group").Alias("avg_score").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Name() != "avg_score" {
			t.Errorf("expected name 'avg_score', got %q", s.Name())
		}
	})

	t.Run("string representation", func(t *testing.T) {
		e := Col("score").Mean().Over("group")
		str := e.String()
		if str == "" {
			t.Error("expected non-empty string representation")
		}
	})
}

func TestCols(t *testing.T) {
	exprs := Cols("a", "b", "c")
	if len(exprs) != 3 {
		t.Errorf("expected 3 expressions, got %d", len(exprs))
	}
}

// ---------------------------------------------------------------------------
// Phase A: Var, NUnique, Quantile, Median
// ---------------------------------------------------------------------------

func TestAggVar(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("score").Var().Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := s.GetFloat64(0)
	if !ok || v <= 0 {
		t.Errorf("expected positive variance, got %g", v)
	}
}

func TestAggNUnique(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("age").NUnique().Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := s.GetInt64(0)
	if v != 4 {
		t.Errorf("expected 4 unique values, got %d", v)
	}
}

func TestAggMedian(t *testing.T) {
	_, ctx := testDF(t)
	// ages: 25, 30, 35, 40 -> median = (30+35)/2 = 32.5
	s, err := Col("age").Median().Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := s.GetFloat64(0)
	if !ok {
		t.Fatal("expected valid float64")
	}
	if v != 32.5 {
		t.Errorf("expected median 32.5, got %g", v)
	}
}

func TestAggQuantile(t *testing.T) {
	_, ctx := testDF(t)
	// score: 76.1, 88.5, 92.3, 95.0
	// 0.5 quantile = median
	s, err := Col("score").Quantile(0.5).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := s.GetFloat64(0)
	if !ok {
		t.Fatal("expected valid float64")
	}
	// sorted: 76.1, 88.5, 92.3, 95.0 -> median = (88.5+92.3)/2 = 90.4
	if v != 90.4 {
		t.Errorf("expected quantile 90.4, got %g", v)
	}
}

// ---------------------------------------------------------------------------
// Phase B: Mod, Pow, IsIn, IsBetween, Rank
// ---------------------------------------------------------------------------

func TestMod(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("age").Mod(Lit(10)).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 25%10=5, 30%10=0, 35%10=5, 40%10=0
	v0, _ := s.GetInt64(0)
	v1, _ := s.GetInt64(1)
	if v0 != 5 || v1 != 0 {
		t.Errorf("mod: got [%d,%d], want [5,0]", v0, v1)
	}
}

func TestPow(t *testing.T) {
	df, err := dataframe.New(
		series.NewFloat64("x", []float64{2, 3, 4}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}
	s, err := Col("x").Pow(Lit(2.0)).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 2^2=4, 3^2=9, 4^2=16
	v0, _ := s.GetFloat64(0)
	v1, _ := s.GetFloat64(1)
	v2, _ := s.GetFloat64(2)
	if v0 != 4 || v1 != 9 || v2 != 16 {
		t.Errorf("pow: got [%g,%g,%g], want [4,9,16]", v0, v1, v2)
	}
}

func TestIsIn(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("age").IsIn(25, 35).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// [25=true, 30=false, 35=true, 40=false]
	v0, _ := s.GetBool(0)
	v1, _ := s.GetBool(1)
	v2, _ := s.GetBool(2)
	v3, _ := s.GetBool(3)
	if !v0 || v1 || !v2 || v3 {
		t.Errorf("is_in: got [%v,%v,%v,%v], want [true,false,true,false]", v0, v1, v2, v3)
	}
}

func TestIsInString(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("name").IsIn("Alice", "Diana").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := s.GetBool(0) // Alice=true
	v1, _ := s.GetBool(1) // Bob=false
	v3, _ := s.GetBool(3) // Diana=true
	if !v0 || v1 || !v3 {
		t.Errorf("is_in string: got [%v,%v,...,%v], want [true,false,...,true]", v0, v1, v3)
	}
}

func TestIsBetween(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("age").IsBetween(Lit(28), Lit(37)).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 25=false, 30=true, 35=true, 40=false
	v0, _ := s.GetBool(0)
	v1, _ := s.GetBool(1)
	v2, _ := s.GetBool(2)
	v3, _ := s.GetBool(3)
	if v0 || !v1 || !v2 || v3 {
		t.Errorf("is_between: got [%v,%v,%v,%v], want [false,true,true,false]", v0, v1, v2, v3)
	}
}

func TestRank(t *testing.T) {
	df, err := dataframe.New(
		series.NewFloat64("x", []float64{40, 10, 30, 20}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}
	s, err := Col("x").Rank().Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Values: 40,10,30,20 -> sorted: 10(1),20(2),30(3),40(4)
	// Ranks: 40->4, 10->1, 30->3, 20->2
	v0, _ := s.GetFloat64(0) // 40 -> rank 4
	v1, _ := s.GetFloat64(1) // 10 -> rank 1
	v2, _ := s.GetFloat64(2) // 30 -> rank 3
	v3, _ := s.GetFloat64(3) // 20 -> rank 2
	if v0 != 4 || v1 != 1 || v2 != 3 || v3 != 2 {
		t.Errorf("rank: got [%g,%g,%g,%g], want [4,1,3,2]", v0, v1, v2, v3)
	}
}

// ---------------------------------------------------------------------------
// Phase C: Dt namespace
// ---------------------------------------------------------------------------

func TestDtNamespace(t *testing.T) {
	d := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	us := d.UnixMicro()
	df, err := dataframe.New(
		series.NewDateTime("ts", []int64{us}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	t.Run("year", func(t *testing.T) {
		s, err := Col("ts").Dt().Year().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetInt32(0)
		if !ok || v != 2024 {
			t.Errorf("dt.year: got %d, want 2024", v)
		}
	})

	t.Run("month", func(t *testing.T) {
		s, err := Col("ts").Dt().Month().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, ok := s.GetInt32(0)
		if !ok || v != 6 {
			t.Errorf("dt.month: got %d, want 6", v)
		}
	})
}

// ---------------------------------------------------------------------------
// Phase E: Str expansion
// ---------------------------------------------------------------------------

func TestStrStartsWith(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("name").Str().StartsWith("Al").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := s.GetBool(0) // Alice -> true
	v1, _ := s.GetBool(1) // Bob -> false
	if !v0 || v1 {
		t.Errorf("starts_with: got [%v,%v], want [true,false]", v0, v1)
	}
}

func TestStrEndsWith(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("name").Str().EndsWith("na").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v3, _ := s.GetBool(3) // Diana -> true
	v0, _ := s.GetBool(0) // Alice -> false
	if !v3 || v0 {
		t.Errorf("ends_with: got alice=%v diana=%v, want false,true", v0, v3)
	}
}

func TestStrReplace(t *testing.T) {
	_, ctx := testDF(t)
	s, err := Col("name").Str().Replace("a", "X").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// "Diana" -> "DiXnX"
	v, _ := s.GetString(3)
	if v != "DiXnX" {
		t.Errorf("replace: got %q, want %q", v, "DiXnX")
	}
}

func TestStrTrim(t *testing.T) {
	df, _ := dataframe.New(
		series.NewString("s", []string{"  hello  ", " world "}),
	)
	ctx := &Context{DF: df}
	s, err := Col("s").Str().Trim().Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := s.GetString(0)
	if v0 != "hello" {
		t.Errorf("trim: got %q, want %q", v0, "hello")
	}
}

func TestStrPad(t *testing.T) {
	df, _ := dataframe.New(
		series.NewString("s", []string{"hi", "hello"}),
	)
	ctx := &Context{DF: df}
	s, err := Col("s").Str().Pad(5, "left", '*').Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := s.GetString(0)
	if v0 != "***hi" {
		t.Errorf("pad left: got %q, want %q", v0, "***hi")
	}
}

func TestStrCountMatches(t *testing.T) {
	df, _ := dataframe.New(
		series.NewString("s", []string{"aabaa", "bbb", "abc"}),
	)
	ctx := &Context{DF: df}
	s, err := Col("s").Str().CountMatches("a").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := s.GetInt64(0) // "aabaa" has 4 'a's
	v1, _ := s.GetInt64(1) // "bbb" has 0 'a's
	if v0 != 4 || v1 != 0 {
		t.Errorf("count_matches: got [%d,%d], want [4,0]", v0, v1)
	}
}

// ---------------------------------------------------------------------------
// Phase H: TryCast, Name namespace
// ---------------------------------------------------------------------------

func TestTryCast(t *testing.T) {
	df, _ := dataframe.New(
		series.NewString("x", []string{"1", "abc", "3"}),
	)
	ctx := &Context{DF: df}

	s, err := Col("x").TryCast(dtype.Int64).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}

	v0, ok := s.GetInt64(0)
	if !ok || v0 != 1 {
		t.Errorf("try_cast [0]: got %d, want 1", v0)
	}

	if !s.IsNull(1) {
		t.Error("try_cast [1]: expected null for 'abc'")
	}

	v2, ok := s.GetInt64(2)
	if !ok || v2 != 3 {
		t.Errorf("try_cast [2]: got %d, want 3", v2)
	}
}

func TestNameNamespace(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("prefix", func(t *testing.T) {
		s, err := Col("age").Name().Prefix("col_").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Name() != "col_age" {
			t.Errorf("expected name 'col_age', got %q", s.Name())
		}
	})

	t.Run("suffix", func(t *testing.T) {
		s, err := Col("age").Name().Suffix("_raw").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Name() != "age_raw" {
			t.Errorf("expected name 'age_raw', got %q", s.Name())
		}
	})

	t.Run("map", func(t *testing.T) {
		s, err := Col("age").Name().Map(func(n string) string { return "mapped_" + n }).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Name() != "mapped_age" {
			t.Errorf("expected name 'mapped_age', got %q", s.Name())
		}
	})
}

func TestShift(t *testing.T) {
	_, ctx := testDF(t)

	s, err := Col("age").Shift(1).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 4 {
		t.Fatalf("expected len 4, got %d", s.Len())
	}
	if !s.IsNull(0) {
		t.Error("expected null at index 0 after shift(1)")
	}
	if v, _ := s.GetInt64(1); v != 25 {
		t.Errorf("expected 25 at index 1, got %d", v)
	}
}

func TestDiff(t *testing.T) {
	_, ctx := testDF(t)

	s, err := Col("age").Diff(1).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 4 {
		t.Fatalf("expected len 4, got %d", s.Len())
	}
	if !s.IsNull(0) {
		t.Error("expected null at index 0 after diff(1)")
	}
	// age = [25, 30, 35, 40] → diff = [null, 5, 5, 5]
	if v, _ := s.GetInt64(1); v != 5 {
		t.Errorf("expected 5 at index 1, got %d", v)
	}
}

func TestPctChange(t *testing.T) {
	_, ctx := testDF(t)

	s, err := Col("score").PctChange(1).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 4 {
		t.Fatalf("expected len 4, got %d", s.Len())
	}
	if !s.IsNull(0) {
		t.Error("expected null at index 0 after pct_change(1)")
	}
	// score = [88.5, 92.3, 76.1, 95.0]
	// pct_change(1) at index 1: (92.3 - 88.5) / 88.5 ≈ 0.04294
	v, _ := s.GetFloat64(1)
	expected := (92.3 - 88.5) / 88.5
	if math.Abs(v-expected) > 1e-6 {
		t.Errorf("expected ~%f at index 1, got %f", expected, v)
	}
}

func TestCumNamespace(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("sum", func(t *testing.T) {
		s, err := Col("age").Cum().Sum().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// age = [25, 30, 35, 40] → cumsum = [25, 55, 90, 130]
		if v, _ := s.GetInt64(3); v != 130 {
			t.Errorf("expected 130, got %d", v)
		}
	})

	t.Run("prod", func(t *testing.T) {
		// Use small values to avoid overflow
		df, err := dataframe.New(series.NewInt64("x", []int64{2, 3, 4}))
		if err != nil {
			t.Fatal(err)
		}
		c := &Context{DF: df}
		s, err := Col("x").Cum().Prod().Evaluate(c)
		if err != nil {
			t.Fatal(err)
		}
		// [2, 3, 4] → cumprod = [2, 6, 24]
		if v, _ := s.GetInt64(2); v != 24 {
			t.Errorf("expected 24, got %d", v)
		}
	})

	t.Run("min", func(t *testing.T) {
		s, err := Col("age").Cum().Min().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// age = [25, 30, 35, 40] → cummin = [25, 25, 25, 25]
		if v, _ := s.GetInt64(3); v != 25 {
			t.Errorf("expected 25, got %d", v)
		}
	})

	t.Run("max", func(t *testing.T) {
		s, err := Col("age").Cum().Max().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// age = [25, 30, 35, 40] → cummax = [25, 30, 35, 40]
		if v, _ := s.GetInt64(3); v != 40 {
			t.Errorf("expected 40, got %d", v)
		}
	})
}

func TestRollingNamespace(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("mean", func(t *testing.T) {
		s, err := Col("score").Rolling(2).Mean().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 4 {
			t.Fatalf("expected len 4, got %d", s.Len())
		}
		// score = [88.5, 92.3, 76.1, 95.0]
		// rolling_mean(2) at index 1: (88.5+92.3)/2 = 90.4
		v, _ := s.GetFloat64(1)
		if math.Abs(v-90.4) > 0.01 {
			t.Errorf("expected ~90.4, got %f", v)
		}
	})

	t.Run("sum", func(t *testing.T) {
		s, err := Col("score").Rolling(2).Sum().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// index 1: 88.5 + 92.3 = 180.8
		v, _ := s.GetFloat64(1)
		if math.Abs(v-180.8) > 0.01 {
			t.Errorf("expected ~180.8, got %f", v)
		}
	})

	t.Run("min", func(t *testing.T) {
		s, err := Col("score").Rolling(2).Min().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// index 1: min(88.5, 92.3) = 88.5
		v, _ := s.GetFloat64(1)
		if math.Abs(v-88.5) > 0.01 {
			t.Errorf("expected ~88.5, got %f", v)
		}
	})

	t.Run("max", func(t *testing.T) {
		s, err := Col("score").Rolling(2).Max().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// index 1: max(88.5, 92.3) = 92.3
		v, _ := s.GetFloat64(1)
		if math.Abs(v-92.3) > 0.01 {
			t.Errorf("expected ~92.3, got %f", v)
		}
	})

	t.Run("std", func(t *testing.T) {
		s, err := Col("score").Rolling(3).Std().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 4 {
			t.Fatalf("expected len 4, got %d", s.Len())
		}
	})
}

func TestMathExpressions(t *testing.T) {
	df, err := dataframe.New(
		series.NewFloat64("x", []float64{-2.5, 4.0, 9.0, 1.0}),
		series.NewInt64("y", []int64{-3, 5, -7, 10}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	t.Run("abs_float", func(t *testing.T) {
		s, err := Col("x").Abs().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != 2.5 {
			t.Errorf("expected 2.5, got %f", v)
		}
	})

	t.Run("abs_int", func(t *testing.T) {
		s, err := Col("y").Abs().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(2)
		if v != 7 {
			t.Errorf("expected 7, got %d", v)
		}
	})

	t.Run("sqrt", func(t *testing.T) {
		s, err := Col("x").Abs().Sqrt().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(2)
		if math.Abs(v-3.0) > 1e-10 {
			t.Errorf("expected 3.0, got %f", v)
		}
	})

	t.Run("log", func(t *testing.T) {
		s, err := Col("x").Abs().Log().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(1)
		if math.Abs(v-math.Log(4.0)) > 1e-10 {
			t.Errorf("expected ln(4), got %f", v)
		}
	})

	t.Run("exp", func(t *testing.T) {
		s, err := Lit(1.0).Exp().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if math.Abs(v-math.E) > 1e-10 {
			t.Errorf("expected e, got %f", v)
		}
	})

	t.Run("round", func(t *testing.T) {
		df2, _ := dataframe.New(series.NewFloat64("v", []float64{3.14159, 2.71828}))
		ctx2 := &Context{DF: df2}
		s, err := Col("v").Round(2).Evaluate(ctx2)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if math.Abs(v-3.14) > 1e-10 {
			t.Errorf("expected 3.14, got %f", v)
		}
		v2, _ := s.GetFloat64(1)
		if math.Abs(v2-2.72) > 1e-10 {
			t.Errorf("expected 2.72, got %f", v2)
		}
	})

	t.Run("floor", func(t *testing.T) {
		s, err := Col("x").Floor().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != -3.0 {
			t.Errorf("expected -3.0, got %f", v)
		}
	})

	t.Run("ceil", func(t *testing.T) {
		s, err := Col("x").Ceil().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if v != -2.0 {
			t.Errorf("expected -2.0, got %f", v)
		}
	})
}

func TestFirstLastExpr(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("first", func(t *testing.T) {
		s, err := Col("age").First().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 1 {
			t.Fatalf("expected len 1, got %d", s.Len())
		}
		v, _ := s.GetInt64(0)
		if v != 25 {
			t.Errorf("expected 25, got %d", v)
		}
	})

	t.Run("last", func(t *testing.T) {
		s, err := Col("age").Last().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 1 {
			t.Fatalf("expected len 1, got %d", s.Len())
		}
		v, _ := s.GetInt64(0)
		if v != 40 {
			t.Errorf("expected 40, got %d", v)
		}
	})
}

func TestHeadTailGatherUniqueExpr(t *testing.T) {
	_, ctx := testDF(t)

	t.Run("head", func(t *testing.T) {
		s, err := Col("age").Head(2).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 2 {
			t.Fatalf("expected len 2, got %d", s.Len())
		}
	})

	t.Run("tail", func(t *testing.T) {
		s, err := Col("age").Tail(2).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 2 {
			t.Fatalf("expected len 2, got %d", s.Len())
		}
		v, _ := s.GetInt64(0)
		if v != 35 {
			t.Errorf("expected 35, got %d", v)
		}
	})

	t.Run("gather", func(t *testing.T) {
		s, err := Col("age").Gather([]int{0, 3}).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 2 {
			t.Fatalf("expected len 2, got %d", s.Len())
		}
		v0, _ := s.GetInt64(0)
		v1, _ := s.GetInt64(1)
		if v0 != 25 || v1 != 40 {
			t.Errorf("expected [25, 40], got [%d, %d]", v0, v1)
		}
	})

	t.Run("unique", func(t *testing.T) {
		df, _ := dataframe.New(series.NewInt64("x", []int64{1, 2, 2, 3, 3, 3}))
		c := &Context{DF: df}
		s, err := Col("x").Unique().Evaluate(c)
		if err != nil {
			t.Fatal(err)
		}
		if s.Len() != 3 {
			t.Errorf("expected 3 unique values, got %d", s.Len())
		}
	})
}

func TestSortByExpr(t *testing.T) {
	df, err := dataframe.New(
		series.NewString("name", []string{"Alice", "Bob", "Charlie"}),
		series.NewInt64("score", []int64{90, 70, 80}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{DF: df}

	s, err := Col("name").SortBy(Col("score"), false).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by score ascending: Bob(70), Charlie(80), Alice(90)
	v0, _ := s.GetString(0)
	v1, _ := s.GetString(1)
	v2, _ := s.GetString(2)
	if v0 != "Bob" || v1 != "Charlie" || v2 != "Alice" {
		t.Errorf("expected [Bob, Charlie, Alice], got [%s, %s, %s]", v0, v1, v2)
	}
}

func TestStrCapitalizeZFill(t *testing.T) {
	df, _ := dataframe.New(series.NewString("s", []string{"hello world", "FOO", "123"}))
	ctx := &Context{DF: df}

	t.Run("capitalize", func(t *testing.T) {
		s, err := Col("s").Str().Capitalize().Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetString(0)
		if v != "Hello world" {
			t.Errorf("expected 'Hello world', got %q", v)
		}
		v2, _ := s.GetString(1)
		if v2 != "Foo" {
			t.Errorf("expected 'Foo', got %q", v2)
		}
	})

	t.Run("zfill", func(t *testing.T) {
		s, err := Col("s").Str().ZFill(6).Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetString(2)
		if v != "000123" {
			t.Errorf("expected '000123', got %q", v)
		}
	})
}

func TestStrToDatetime(t *testing.T) {
	df, _ := dataframe.New(series.NewString("dates", []string{
		"2024-01-15", "2024-06-30", "2024-12-25",
	}))
	ctx := &Context{DF: df}

	s, err := Col("dates").Str().ToDatetime("2006-01-02").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.DataType() != dtype.DateTime {
		t.Errorf("expected DateTime, got %s", s.DataType())
	}
	if s.Len() != 3 {
		t.Fatalf("expected len 3, got %d", s.Len())
	}
}

func TestDtEpochAndTotalSeconds(t *testing.T) {
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	df, _ := dataframe.New(series.NewDateTime("ts", []int64{ts.UnixMicro()}))
	ctx := &Context{DF: df}

	t.Run("epoch_s", func(t *testing.T) {
		s, err := Col("ts").Dt().Epoch("s").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(0)
		if v != ts.Unix() {
			t.Errorf("expected %d, got %d", ts.Unix(), v)
		}
	})

	t.Run("epoch_ms", func(t *testing.T) {
		s, err := Col("ts").Dt().Epoch("ms").Evaluate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetInt64(0)
		if v != ts.UnixMilli() {
			t.Errorf("expected %d, got %d", ts.UnixMilli(), v)
		}
	})

	t.Run("total_seconds", func(t *testing.T) {
		// 5.5 seconds = 5_500_000 microseconds
		df2, _ := dataframe.New(series.NewDuration("dur", []int64{5_500_000}))
		ctx2 := &Context{DF: df2}
		s, err := Col("dur").Dt().TotalSeconds().Evaluate(ctx2)
		if err != nil {
			t.Fatal(err)
		}
		v, _ := s.GetFloat64(0)
		if math.Abs(v-5.5) > 1e-10 {
			t.Errorf("expected 5.5, got %f", v)
		}
	})
}

func TestTransformUsedColumns(t *testing.T) {
	e := Col("x").Shift(1)
	cols := e.UsedColumns()
	if len(cols) != 1 || cols[0] != "x" {
		t.Errorf("expected [x], got %v", cols)
	}

	e2 := Col("y").Cum().Sum()
	cols2 := e2.UsedColumns()
	if len(cols2) != 1 || cols2[0] != "y" {
		t.Errorf("expected [y], got %v", cols2)
	}

	e3 := Col("z").Rolling(3).Mean()
	cols3 := e3.UsedColumns()
	if len(cols3) != 1 || cols3[0] != "z" {
		t.Errorf("expected [z], got %v", cols3)
	}

	if Col("a").Diff(1).IsConstant() {
		t.Error("diff should not be constant")
	}
	if Col("a").PctChange(1).IsConstant() {
		t.Error("pct_change should not be constant")
	}
}
