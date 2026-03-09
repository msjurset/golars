package expr

import (
	"math"
	"testing"

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
