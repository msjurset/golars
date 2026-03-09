package golars_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/msjurset/golars"
)

func TestNewDataFrame(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if df.Height() != 4 {
		t.Errorf("expected height 4, got %d", df.Height())
	}
	if df.Width() != 3 {
		t.Errorf("expected width 3, got %d", df.Width())
	}
	output := df.String()
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected Alice in output:\n%s", output)
	}
}

func TestExpressions(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Filter using expression
	ctx := &golars.ExprContext{DF: df}
	mask, err := golars.Col("age").Gt(golars.Lit(30)).Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := df.Filter(mask)
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Height() != 2 {
		t.Errorf("expected 2 rows after filter, got %d", filtered.Height())
	}

	// Arithmetic expression
	curvedExpr := golars.Col("score").Mul(golars.Lit(1.1)).Alias("curved")
	curved, err := curvedExpr.Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if curved.Name() != "curved" {
		t.Errorf("expected name 'curved', got %q", curved.Name())
	}

	// When/Then/Otherwise
	tierExpr := golars.When(golars.Col("age").Gt(golars.Lit(35))).
		Then(golars.Lit("senior")).
		Otherwise(golars.Lit("junior"))
	tier, err := tierExpr.Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := tier.GetString(0)
	if !ok || v != "junior" {
		t.Errorf("expected junior for age 25, got %q", v)
	}
	v, ok = tier.GetString(3)
	if !ok || v != "senior" {
		t.Errorf("expected senior for age 40, got %q", v)
	}
}

func TestCSVRoundTrip(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{25, 30}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := golars.WriteCSV(df, &buf); err != nil {
		t.Fatal(err)
	}

	df2, err := golars.ReadCSVFromReader(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if df2.Height() != 2 || df2.Width() != 3 {
		t.Errorf("round-trip: expected 2x3, got %dx%d", df2.Height(), df2.Width())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{25, 30}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := golars.WriteJSON(df, &buf); err != nil {
		t.Fatal(err)
	}

	df2, err := golars.ReadJSONFromReader(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if df2.Height() != 2 {
		t.Errorf("round-trip: expected 2 rows, got %d", df2.Height())
	}
}

func TestDataFrameOperations(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewInt64Series("a", []int64{3, 1, 2}),
		golars.NewStringSeries("b", []string{"x", "y", "z"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Select
	selected, err := df.Select("b")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Width() != 1 {
		t.Errorf("expected 1 column, got %d", selected.Width())
	}

	// Sort
	sorted, err := df.Sort("a", false)
	if err != nil {
		t.Fatal(err)
	}
	col, _ := sorted.Column("a")
	v, _ := col.GetInt64(0)
	if v != 1 {
		t.Errorf("expected 1 after ascending sort, got %d", v)
	}

	// Describe
	desc := df.Describe()
	if desc == nil {
		t.Error("Describe returned nil")
	}
}

func TestConcatDataFrames(t *testing.T) {
	df1, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{1, 2}),
	)
	df2, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{3, 4}),
	)

	combined, err := golars.ConcatDataFrames(df1, df2)
	if err != nil {
		t.Fatal(err)
	}
	if combined.Height() != 4 {
		t.Errorf("expected 4 rows, got %d", combined.Height())
	}
}

func TestGroupByIntegration(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("group", []string{"a", "a", "b", "b"}),
		golars.NewInt64Series("val", []int64{10, 20, 30, 40}),
	)
	if err != nil {
		t.Fatal(err)
	}

	grouped, err := df.GroupBy("group")
	if err != nil {
		t.Fatal(err)
	}

	result, err := grouped.Agg(map[string]golars.AggFunc{
		"val": golars.AggSum,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Height() != 2 {
		t.Fatalf("expected 2 groups, got %d", result.Height())
	}
}

func TestJoinIntegration(t *testing.T) {
	left, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)
	right, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{2, 3, 4}),
		golars.NewFloat64Series("score", []float64{90.0, 85.0, 95.0}),
	)

	result, err := left.Join(right, []string{"id"}, golars.InnerJoin)
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("inner join: expected 2 rows, got %d", result.Height())
	}
}

func TestPivotIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Alice", "Bob", "Bob"}),
		golars.NewStringSeries("subject", []string{"math", "english", "math", "english"}),
		golars.NewInt64Series("score", []int64{90, 85, 70, 95}),
	)

	pivoted, err := df.Pivot("name", "subject", "score")
	if err != nil {
		t.Fatal(err)
	}
	if pivoted.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", pivoted.Height())
	}
	if pivoted.Width() != 3 {
		t.Errorf("expected 3 columns (name + 2 subjects), got %d", pivoted.Width())
	}
}

func TestSeriesRolling(t *testing.T) {
	s := golars.NewFloat64Series("x", []float64{1, 2, 3, 4, 5})

	rm := s.RollingMean(3)
	if rm == nil {
		t.Fatal("RollingMean returned nil")
	}
	if rm.Len() != 5 {
		t.Errorf("expected length 5, got %d", rm.Len())
	}
	// First 2 should be null, third should be mean(1,2,3) = 2.0
	if !rm.IsNull(0) || !rm.IsNull(1) {
		t.Error("first 2 values should be null")
	}
	v, ok := rm.GetFloat64(2)
	if !ok || v != 2.0 {
		t.Errorf("rolling mean at index 2: got %f, want 2.0", v)
	}
}

func TestSeriesShift(t *testing.T) {
	s := golars.NewInt64Series("x", []int64{10, 20, 30})
	shifted := s.Shift(1)
	if !shifted.IsNull(0) {
		t.Error("shifted[0] should be null")
	}
	v, ok := shifted.GetInt64(1)
	if !ok || v != 10 {
		t.Errorf("shifted[1] = %d, want 10", v)
	}
}

func TestSeriesCumSum(t *testing.T) {
	s := golars.NewInt64Series("x", []int64{1, 2, 3, 4})
	cs := s.CumSum()
	if cs == nil {
		t.Fatal("CumSum returned nil")
	}
	expected := []int64{1, 3, 6, 10}
	for i, want := range expected {
		v, ok := cs.GetInt64(i)
		if !ok || v != want {
			t.Errorf("cumsum[%d] = %d, want %d", i, v, want)
		}
	}
}

func TestMapRows(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("a", []int64{1, 2, 3}),
		golars.NewInt64Series("b", []int64{10, 20, 30}),
	)

	result := df.MapRows("sum", func(row map[string]any) any {
		a := row["a"].(int64)
		b := row["b"].(int64)
		return a + b
	})
	if result.Len() != 3 {
		t.Fatalf("expected 3 rows, got %d", result.Len())
	}
	v, ok := result.GetInt64(0)
	if !ok || v != 11 {
		t.Errorf("sum[0] = %d, want 11", v)
	}
}

func TestLazyIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)

	result, err := golars.Lazy(df).
		Filter(golars.Col("age").Gt(golars.Lit(25))).
		Sort("score", true).
		Head(2).
		Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
	col, _ := result.Column("score")
	v, _ := col.GetFloat64(0)
	if v != 95.0 {
		t.Errorf("first score after desc sort: got %f, want 95.0", v)
	}
}

func TestSQLIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3, 4}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)

	ctx := golars.NewSQLContext()
	ctx.Register("users", df)

	result, err := ctx.Execute("SELECT name, score FROM users WHERE score > 80 ORDER BY score DESC LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Height() != 2 {
		t.Errorf("expected 2 rows, got %d", result.Height())
	}
	col, _ := result.Column("score")
	v, _ := col.GetFloat64(0)
	if v != 95.0 {
		t.Errorf("top score: got %f, want 95.0", v)
	}
}

func TestLazyExplain(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("a", []int64{1, 2, 3}),
	)

	plan := golars.Lazy(df).
		Filter(golars.Col("a").Gt(golars.Lit(1))).
		Sort("a", false).
		Explain()

	if !strings.Contains(plan, "SORT") || !strings.Contains(plan, "FILTER") {
		t.Errorf("explain plan missing expected nodes:\n%s", plan)
	}
}

func TestParquetIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)

	dir := t.TempDir()
	path := dir + "/test.parquet"

	if err := golars.WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	df2, err := golars.ReadParquet(path)
	if err != nil {
		t.Fatal(err)
	}
	if df2.Height() != 3 || df2.Width() != 3 {
		t.Errorf("expected 3x3, got %dx%d", df2.Height(), df2.Width())
	}
}

func TestExcelIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{25, 30}),
	)

	dir := t.TempDir()
	path := dir + "/test.xlsx"

	if err := golars.WriteExcelFile(df, path); err != nil {
		t.Fatal(err)
	}

	df2, err := golars.ReadExcel(path)
	if err != nil {
		t.Fatal(err)
	}
	if df2.Height() != 2 || df2.Width() != 2 {
		t.Errorf("expected 2x2, got %dx%d", df2.Height(), df2.Width())
	}
}

func TestRowsIterator(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewInt64Series("age", []int64{25, 30, 35}),
	)

	var names []string
	var ages []int64
	for row := range df.Rows() {
		name, err := row.String("name")
		if err != nil {
			t.Fatal(err)
		}
		age, err := row.Int64("age")
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
		ages = append(ages, age)
	}

	if len(names) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(names))
	}
	if names[0] != "Alice" || names[2] != "Charlie" {
		t.Errorf("unexpected names: %v", names)
	}
	if ages[1] != 30 {
		t.Errorf("expected age 30, got %d", ages[1])
	}
}

func TestWindowOverIntegration(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("group", []string{"a", "a", "b", "b"}),
		golars.NewFloat64Series("score", []float64{10, 20, 30, 40}),
	)

	ctx := &golars.ExprContext{DF: df}
	result, err := golars.Col("score").Mean().Over("group").Evaluate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 4 {
		t.Fatalf("expected 4 rows, got %d", result.Len())
	}

	// Group "a" mean should be 15.0, group "b" mean should be 35.0
	v0, _ := result.GetFloat64(0)
	v2, _ := result.GetFloat64(2)
	if v0 != 15.0 {
		t.Errorf("expected group 'a' mean 15.0, got %f", v0)
	}
	if v2 != 35.0 {
		t.Errorf("expected group 'b' mean 35.0, got %f", v2)
	}
}

func TestGroupByAgg(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("group", []string{"a", "a", "b", "b", "b"}),
		golars.NewFloat64Series("value", []float64{10, 20, 30, 40, 50}),
	)
	if err != nil {
		t.Fatal(err)
	}

	g, err := df.GroupBy("group")
	if err != nil {
		t.Fatal(err)
	}

	result, err := golars.GroupByAgg(g,
		golars.Col("value").Sum().Alias("total"),
		golars.Col("value").Mean().Alias("avg"),
		golars.Col("value").Count().Alias("cnt"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Height() != 2 {
		t.Errorf("expected 2 groups, got %d", result.Height())
	}
	if result.Width() != 4 { // group + total + avg + cnt
		t.Errorf("expected 4 columns, got %d", result.Width())
	}

	// Check totals
	total, err := result.Column("total")
	if err != nil {
		t.Fatal(err)
	}
	v0, _ := total.GetFloat64(0) // group "a": 10+20=30
	v1, _ := total.GetFloat64(1) // group "b": 30+40+50=120
	if v0 != 30 {
		t.Errorf("expected total 30 for group a, got %g", v0)
	}
	if v1 != 120 {
		t.Errorf("expected total 120 for group b, got %g", v1)
	}
}

// ---------------------------------------------------------------------------
// Phase F: Parquet Snappy + Context Cancellation
// ---------------------------------------------------------------------------

func TestParquetSnappy(t *testing.T) {
	df, err := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewInt64Series("age", []int64{25, 30, 35}),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Write with snappy compression
	path := t.TempDir() + "/test_snappy.parquet"
	err = golars.WriteParquetFile(df, path, golars.WithParquetCompression("snappy"))
	if err != nil {
		t.Fatal(err)
	}

	// Read back
	df2, err := golars.ReadParquet(path)
	if err != nil {
		t.Fatal(err)
	}

	if df2.Height() != 3 {
		t.Errorf("expected 3 rows, got %d", df2.Height())
	}

	col, _ := df2.Column("name")
	v, _ := col.GetString(0)
	if v != "Alice" {
		t.Errorf("expected Alice, got %q", v)
	}
}

func TestContextCancellation(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{1, 2, 3}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	lf := golars.Lazy(df).Select(golars.Col("x"))
	_, err := lf.CollectWithContext(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func ExampleNewDataFrame() {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("age", []int64{25, 30, 35}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)
	fmt.Println(df.Height(), df.Width())
	// Output: 3 2
}
