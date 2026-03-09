package lazy

import (
	"strings"
	"testing"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
	"github.com/msjurset/golars/internal/series"
)

func helperDF(t *testing.T) *dataframe.DataFrame {
	t.Helper()
	df, err := dataframe.New(
		series.NewInt64("a", []int64{3, 1, 4, 1, 5}),
		series.NewString("b", []string{"x", "y", "z", "w", "v"}),
		series.NewFloat64("c", []float64{1.1, 2.2, 3.3, 4.4, 5.5}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return df
}

func TestLazyCollectIdentity(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df)
	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 5 || result.Width() != 3 {
		t.Errorf("expected 5x3, got %dx%d", result.Height(), result.Width())
	}
}

func TestLazyFilter(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).
		Filter(expr.Col("a").Gt(expr.Lit(2)))

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	// a > 2: values 3, 4, 5 -> 3 rows
	if result.Height() != 3 {
		t.Errorf("expected 3 rows, got %d", result.Height())
	}
}

func TestLazySelect(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).
		Select(expr.Col("a"), expr.Col("b"))

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Width() != 2 {
		t.Errorf("expected 2 columns, got %d", result.Width())
	}
}

func TestLazySort(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).Sort("a", false)

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	col, _ := result.Column("a")
	v, _ := col.GetInt64(0)
	if v != 1 {
		t.Errorf("first value after sort: got %d, want 1", v)
	}
}

func TestLazyHead(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).Head(2)

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
}

func TestLazyDrop(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).Drop("c")

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Width() != 2 {
		t.Errorf("expected 2 columns, got %d", result.Width())
	}
}

func TestLazyRename(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).Rename("a", "alpha")

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Schema().Contains("alpha") {
		t.Error("expected column 'alpha'")
	}
}

func TestLazyGroupByAgg(t *testing.T) {
	df, _ := dataframe.New(
		series.NewString("grp", []string{"a", "a", "b", "b"}),
		series.NewInt64("val", []int64{10, 20, 30, 40}),
	)
	lf := FromDataFrame(df).
		GroupBy("grp").
		Agg(map[string]dataframe.AggFunc{"val": dataframe.AggSum})

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 groups, got %d", result.Height())
	}
}

func TestLazyJoin(t *testing.T) {
	left, _ := dataframe.New(
		series.NewInt64("id", []int64{1, 2, 3}),
		series.NewString("name", []string{"a", "b", "c"}),
	)
	right, _ := dataframe.New(
		series.NewInt64("id", []int64{2, 3, 4}),
		series.NewFloat64("val", []float64{2.0, 3.0, 4.0}),
	)

	lf := FromDataFrame(left).
		Join(FromDataFrame(right), []string{"id"}, dataframe.InnerJoin)

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
}

func TestLazyChain(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).
		Filter(expr.Col("a").Gt(expr.Lit(1))).
		Sort("a", true).
		Head(2)

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
	col, _ := result.Column("a")
	v, _ := col.GetInt64(0)
	if v != 5 {
		t.Errorf("first value after desc sort: got %d, want 5", v)
	}
}

func TestLazyUnique(t *testing.T) {
	df, _ := dataframe.New(
		series.NewInt64("a", []int64{1, 1, 2}),
		series.NewString("b", []string{"x", "x", "y"}),
	)
	lf := FromDataFrame(df).Unique()

	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 unique rows, got %d", result.Height())
	}
}

func TestLazyExplain(t *testing.T) {
	df := helperDF(t)
	lf := FromDataFrame(df).
		Filter(expr.Col("a").Gt(expr.Lit(2))).
		Sort("a", false)

	plan := lf.Explain()
	if !strings.Contains(plan, "SORT") {
		t.Error("explain should contain SORT")
	}
	if !strings.Contains(plan, "FILTER") {
		t.Error("explain should contain FILTER")
	}
	if !strings.Contains(plan, "SCAN") {
		t.Error("explain should contain SCAN")
	}
}

func TestOptimizerPredicatePushdown(t *testing.T) {
	df := helperDF(t)
	// Build: Filter(Sort(Scan(df)))
	// Optimizer should push Filter below Sort.
	lf := FromDataFrame(df).
		Sort("a", false).
		Filter(expr.Col("a").Gt(expr.Lit(2)))

	optimized := Optimize(lf.plan)
	// After optimization, Sort should be the top node with Filter below it.
	if optimized.nodeType != NodeSort {
		t.Errorf("expected top node to be Sort after pushdown, got %d", optimized.nodeType)
	}
	if optimized.input.nodeType != NodeFilter {
		t.Errorf("expected child to be Filter after pushdown, got %d", optimized.input.nodeType)
	}

	// Verify result is the same.
	result, err := lf.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 3 {
		t.Errorf("expected 3 rows, got %d", result.Height())
	}
}
