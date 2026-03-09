package sql

import (
	"testing"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/series"
)

func helperDF(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	df, err := dataframe.New(
		series.NewInt64("id", []int64{1, 2, 3, 4}),
		series.NewString("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		series.NewFloat64("score", []float64{88.5, 92.3, 76.1, 95.0}),
		series.NewString("dept", []string{"eng", "eng", "sales", "sales"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func TestSelectAll(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 4 || result.Width() != 4 {
		t.Errorf("expected 4x4, got %dx%d", result.Height(), result.Width())
	}
}

func TestSelectColumns(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT name, score FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if result.Width() != 2 {
		t.Errorf("expected 2 columns, got %d", result.Width())
	}
}

func TestWhere(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users WHERE score > 80")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 3 {
		t.Errorf("expected 3 rows (score > 80), got %d", result.Height())
	}
}

func TestWhereString(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users WHERE name = 'Alice'")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 1 {
		t.Errorf("expected 1 row, got %d", result.Height())
	}
}

func TestOrderBy(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users ORDER BY score DESC")
	if err != nil {
		t.Fatal(err)
	}
	col, _ := result.Column("score")
	v, _ := col.GetFloat64(0)
	if v != 95.0 {
		t.Errorf("first score after desc sort: got %f, want 95.0", v)
	}
}

func TestLimit(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
}

func TestGroupBy(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT dept, COUNT(id) FROM users GROUP BY dept")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 groups, got %d", result.Height())
	}
}

func TestJoin(t *testing.T) {
	ctx := NewContext()
	left, _ := dataframe.New(
		series.NewInt64("id", []int64{1, 2, 3}),
		series.NewString("name", []string{"a", "b", "c"}),
	)
	right, _ := dataframe.New(
		series.NewInt64("id", []int64{2, 3, 4}),
		series.NewFloat64("val", []float64{2.0, 3.0, 4.0}),
	)
	ctx.Register("t1", left)
	ctx.Register("t2", right)

	result, err := ctx.Execute("SELECT * FROM t1 JOIN t2 ON t1.id = t2.id")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows from inner join, got %d", result.Height())
	}
}

func TestWhereAnd(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT * FROM users WHERE score > 80 AND dept = 'eng'")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
}

func TestCombined(t *testing.T) {
	ctx := NewContext()
	ctx.Register("users", helperDF(t))

	result, err := ctx.Execute("SELECT name, score FROM users WHERE score > 80 ORDER BY score DESC LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
	if result.Width() != 2 {
		t.Errorf("expected 2 columns, got %d", result.Width())
	}
	col, _ := result.Column("score")
	v, _ := col.GetFloat64(0)
	if v != 95.0 {
		t.Errorf("top score: got %f, want 95.0", v)
	}
}

func TestTableNotFound(t *testing.T) {
	ctx := NewContext()
	_, err := ctx.Execute("SELECT * FROM missing")
	if err == nil {
		t.Fatal("expected error for missing table")
	}
}
