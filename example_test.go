package golars_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/msjurset/golars"
)

func ExampleNewDataFrame_basic() {
	df, err := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1}),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("rows=%d cols=%d\n", df.Height(), df.Width())

	col, _ := df.Column("name")
	for i := 0; i < col.Len(); i++ {
		v, _ := col.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// rows=3 cols=3
	// Alice
	// Bob
	// Charlie
}

func ExampleDataFrame_Filter() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
	)

	ctx := &golars.ExprContext{DF: df}
	mask, _ := golars.Col("age").Gt(golars.Lit(30)).Evaluate(ctx)
	filtered, _ := df.Filter(mask)

	fmt.Println("rows:", filtered.Height())
	col, _ := filtered.Column("name")
	for i := 0; i < col.Len(); i++ {
		v, _ := col.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// rows: 2
	// Charlie
	// Diana
}

func ExampleDataFrame_Sort() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Charlie", "Alice", "Bob"}),
		golars.NewFloat64Series("score", []float64{76.1, 88.5, 92.3}),
	)

	sorted, _ := df.Sort("score", false) // ascending
	col, _ := sorted.Column("name")
	for i := 0; i < col.Len(); i++ {
		v, _ := col.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// Charlie
	// Alice
	// Bob
}

func ExampleDataFrame_Select() {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1}),
	)

	selected, _ := df.Select("name", "score")
	fmt.Println("cols:", selected.Width())

	nameCol, _ := selected.Column("name")
	scoreCol, _ := selected.Column("score")
	for i := 0; i < selected.Height(); i++ {
		name, _ := nameCol.GetString(i)
		score, _ := scoreCol.GetFloat64(i)
		fmt.Printf("%s: %.1f\n", name, score)
	}
	// Output:
	// cols: 2
	// Alice: 88.5
	// Bob: 92.3
	// Charlie: 76.1
}

func ExampleDataFrame_GroupBy() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}),
		golars.NewInt64Series("salary", []int64{100, 120, 80, 90}),
	)

	grouped, _ := df.GroupBy("dept")
	result, _ := grouped.Agg(map[string]golars.AggFunc{
		"salary": golars.AggSum,
	})

	// Sort result for deterministic output.
	result, _ = result.Sort("dept", false)

	deptCol, _ := result.Column("dept")
	salCol, _ := result.Column("salary")
	for i := 0; i < result.Height(); i++ {
		dept, _ := deptCol.GetString(i)
		sal, _ := salCol.GetFloat64(i)
		fmt.Printf("%s: %.0f\n", dept, sal)
	}
	// Output:
	// eng: 220
	// sales: 170
}

func ExampleDataFrame_Join() {
	employees, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)
	scores, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{2, 3, 4}),
		golars.NewFloat64Series("score", []float64{92.3, 76.1, 95.0}),
	)

	joined, _ := employees.Join(scores, []string{"id"}, golars.InnerJoin)
	fmt.Println("rows:", joined.Height())

	nameCol, _ := joined.Column("name")
	scoreCol, _ := joined.Column("score")
	for i := 0; i < joined.Height(); i++ {
		name, _ := nameCol.GetString(i)
		score, _ := scoreCol.GetFloat64(i)
		fmt.Printf("%s: %.1f\n", name, score)
	}
	// Output:
	// rows: 2
	// Bob: 92.3
	// Charlie: 76.1
}

func ExampleDataFrame_WithColumn() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("item", []string{"Widget", "Gadget", "Gizmo"}),
		golars.NewFloat64Series("price", []float64{10.0, 25.0, 15.0}),
	)

	// Compute a new column using expressions.
	ctx := &golars.ExprContext{DF: df}
	taxed, _ := golars.Col("price").Mul(golars.Lit(1.1)).Alias("with_tax").Evaluate(ctx)
	df2, _ := df.WithColumn(taxed)

	fmt.Println("cols:", df2.Width())
	col, _ := df2.Column("with_tax")
	for i := 0; i < col.Len(); i++ {
		v, _ := col.GetFloat64(i)
		fmt.Printf("%.2f\n", v)
	}
	// Output:
	// cols: 3
	// 11.00
	// 27.50
	// 16.50
}

func ExampleSeries_aggregation() {
	s := golars.NewFloat64Series("values", []float64{10, 20, 30, 40, 50})

	sum, _ := s.Sum()
	mean, _ := s.Mean()
	min, _ := s.Min()
	max, _ := s.Max()

	fmt.Printf("sum=%.0f mean=%.0f min=%.0f max=%.0f\n", sum, mean, min, max)
	fmt.Println("count:", s.Count())
	fmt.Println("nunique:", s.NUnique())
	// Output:
	// sum=150 mean=30 min=10 max=50
	// count: 5
	// nunique: 5
}

func ExampleSeries_rolling() {
	s := golars.NewFloat64Series("x", []float64{1, 2, 3, 4, 5})

	rm := s.RollingMean(3)
	for i := 0; i < rm.Len(); i++ {
		if rm.IsNull(i) {
			fmt.Println("null")
		} else {
			v, _ := rm.GetFloat64(i)
			fmt.Printf("%.1f\n", v)
		}
	}
	// Output:
	// null
	// null
	// 2.0
	// 3.0
	// 4.0
}

func ExampleSeries_Str() {
	s := golars.NewStringSeries("words", []string{"Hello World", "Foo Bar", "Go Lang"})

	upper := s.Str().ToUpper()
	for i := 0; i < upper.Len(); i++ {
		v, _ := upper.GetString(i)
		fmt.Println(v)
	}

	contains := s.Str().Contains("oo")
	for i := 0; i < contains.Len(); i++ {
		v, _ := contains.GetBool(i)
		fmt.Println(v)
	}
	// Output:
	// HELLO WORLD
	// FOO BAR
	// GO LANG
	// false
	// true
	// false
}

func ExampleSeries_nullHandling() {
	s := golars.NewInt64SeriesWithValidity("data",
		[]int64{10, 0, 30, 0, 50},
		[]bool{true, false, true, false, true},
	)

	fmt.Println("nulls:", s.NullCount())
	fmt.Println("has_nulls:", s.HasNulls())

	// Drop nulls.
	clean := s.DropNulls()
	fmt.Println("after drop:", clean.Len())

	// Fill nulls.
	filled := s.FillNullInt64(0)
	for i := 0; i < filled.Len(); i++ {
		v, _ := filled.GetInt64(i)
		fmt.Print(v, " ")
	}
	fmt.Println()
	// Output:
	// nulls: 2
	// has_nulls: true
	// after drop: 3
	// 10 0 30 0 50
}

func ExampleWhen() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewFloat64Series("score", []float64{95.0, 72.0, 88.0}),
	)

	ctx := &golars.ExprContext{DF: df}
	grade, _ := golars.When(golars.Col("score").Gte(golars.Lit(90.0))).
		Then(golars.Lit("A")).
		Otherwise(golars.Lit("B")).
		Evaluate(ctx)

	for i := 0; i < grade.Len(); i++ {
		v, _ := grade.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// A
	// B
	// B
}

func ExampleLazy() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)

	result, _ := golars.Lazy(df).
		Filter(golars.Col("age").Gt(golars.Lit(25))).
		Sort("score", true). // descending
		Head(2).
		Collect()

	nameCol, _ := result.Column("name")
	scoreCol, _ := result.Column("score")
	for i := 0; i < result.Height(); i++ {
		name, _ := nameCol.GetString(i)
		score, _ := scoreCol.GetFloat64(i)
		fmt.Printf("%s: %.1f\n", name, score)
	}
	// Output:
	// Diana: 95.0
	// Bob: 92.3
}

func ExampleSQLContext() {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3, 4}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
		golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
	)

	ctx := golars.NewSQLContext()
	ctx.Register("students", df)

	result, _ := ctx.Execute("SELECT name, score FROM students WHERE score > 80 ORDER BY score DESC LIMIT 3")

	nameCol, _ := result.Column("name")
	scoreCol, _ := result.Column("score")
	for i := 0; i < result.Height(); i++ {
		name, _ := nameCol.GetString(i)
		score, _ := scoreCol.GetFloat64(i)
		fmt.Printf("%s: %.1f\n", name, score)
	}
	// Output:
	// Diana: 95.0
	// Bob: 92.3
	// Alice: 88.5
}

func ExampleReadCSV() {
	// Write CSV data to a buffer, then read it back with options.
	csvData := "name|age|active\nAlice|25|true\nBob|30|false\n"
	reader := strings.NewReader(csvData)

	df, err := golars.ReadCSVFromReader(reader, golars.WithSeparator('|'))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("rows=%d cols=%d\n", df.Height(), df.Width())

	// Round-trip to CSV with default separator.
	var buf bytes.Buffer
	_ = golars.WriteCSV(df, &buf)
	fmt.Print(buf.String())
	// Output:
	// rows=2 cols=3
	// name,age,active
	// Alice,25,true
	// Bob,30,false
}

func ExampleDataFrame_Pivot() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("student", []string{"Alice", "Alice", "Bob", "Bob"}),
		golars.NewStringSeries("subject", []string{"math", "english", "math", "english"}),
		golars.NewInt64Series("grade", []int64{90, 85, 70, 95}),
	)

	pivoted, _ := df.Pivot("student", "subject", "grade")
	fmt.Printf("rows=%d cols=%d\n", pivoted.Height(), pivoted.Width())

	// The pivoted DataFrame has columns: student, math, english.
	studentCol, _ := pivoted.Column("student")
	mathCol, _ := pivoted.Column("math")
	englishCol, _ := pivoted.Column("english")
	for i := 0; i < pivoted.Height(); i++ {
		student, _ := studentCol.GetString(i)
		math, _ := mathCol.GetInt64(i)
		english, _ := englishCol.GetInt64(i)
		fmt.Printf("%s: math=%d english=%d\n", student, math, english)
	}
	// Output:
	// rows=2 cols=3
	// Alice: math=90 english=85
	// Bob: math=70 english=95
}

func ExampleDataFrame_MapRows() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("first", []string{"Jane", "John"}),
		golars.NewStringSeries("last", []string{"Doe", "Smith"}),
	)

	full := df.MapRows("full_name", func(row map[string]any) any {
		return row["first"].(string) + " " + row["last"].(string)
	})

	for i := 0; i < full.Len(); i++ {
		v, _ := full.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// Jane Doe
	// John Smith
}

func ExampleDataFrame_Rows() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{25, 30}),
	)

	for row := range df.Rows() {
		name, _ := row.String("name")
		age, _ := row.Int64("age")
		fmt.Printf("%s is %d\n", name, age)
	}
	// Output:
	// Alice is 25
	// Bob is 30
}

func ExampleDataFrame_Describe() {
	df, _ := golars.NewDataFrame(
		golars.NewFloat64Series("x", []float64{1, 2, 3, 4, 5}),
	)

	desc := df.Describe()
	col, _ := desc.Column("x")
	mean, _ := col.GetFloat64(1) // row 1 = mean
	fmt.Printf("mean: %.1f\n", mean)
	// Output:
	// mean: 3.0
}

func ExampleConcatDataFrames() {
	df1, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{1, 2}),
	)
	df2, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{3, 4}),
	)

	combined, _ := golars.ConcatDataFrames(df1, df2)
	fmt.Println(combined.Height())
	// Output:
	// 4
}

func ExampleSeries_CumSum() {
	s := golars.NewInt64Series("x", []int64{1, 2, 3, 4, 5})
	cs := s.CumSum()
	for i := 0; i < cs.Len(); i++ {
		v, _ := cs.GetInt64(i)
		fmt.Printf("%d ", v)
	}
	// Output:
	// 1 3 6 10 15
}

func ExampleSeries_Cast() {
	ints := golars.NewInt64Series("x", []int64{1, 2, 3})
	strs, _ := ints.Cast(golars.String)
	for i := 0; i < strs.Len(); i++ {
		v, _ := strs.GetString(i)
		fmt.Printf("%q ", v)
	}
	// Output:
	// "1" "2" "3"
}

func Example_windowOver() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("team", []string{"A", "A", "B", "B"}),
		golars.NewFloat64Series("score", []float64{10, 20, 30, 40}),
	)

	ctx := &golars.ExprContext{DF: df}
	teamMean, _ := golars.Col("score").Mean().Over("team").Evaluate(ctx)

	teamCol, _ := df.Column("team")
	for i := 0; i < teamMean.Len(); i++ {
		team, _ := teamCol.GetString(i)
		mean, _ := teamMean.GetFloat64(i)
		fmt.Printf("%s: %.1f\n", team, mean)
	}
	// Output:
	// A: 15.0
	// A: 15.0
	// B: 35.0
	// B: 35.0
}

func Example_mathExpressions() {
	df, _ := golars.NewDataFrame(
		golars.NewFloat64Series("x", []float64{-2.5, 4.0, 9.0, 1.0}),
	)
	ctx := &golars.ExprContext{DF: df}

	// Abs
	abs, _ := golars.Col("x").Abs().Evaluate(ctx)
	for i := 0; i < abs.Len(); i++ {
		v, _ := abs.GetFloat64(i)
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%.1f", v)
	}
	fmt.Println()

	// Round
	df2, _ := golars.NewDataFrame(golars.NewFloat64Series("v", []float64{3.14159, 2.71828}))
	ctx2 := &golars.ExprContext{DF: df2}
	rounded, _ := golars.Col("v").Round(2).Evaluate(ctx2)
	for i := 0; i < rounded.Len(); i++ {
		v, _ := rounded.GetFloat64(i)
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%.2f", v)
	}
	fmt.Println()
	// Output:
	// 2.5 4.0 9.0 1.0
	// 3.14 2.72
}

func Example_cumulativeRolling() {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{1, 2, 3, 4, 5}),
	)
	ctx := &golars.ExprContext{DF: df}

	// Cumulative sum
	cumSum, _ := golars.Col("x").Cum().Sum().Evaluate(ctx)
	for i := 0; i < cumSum.Len(); i++ {
		v, _ := cumSum.GetInt64(i)
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%d", v)
	}
	fmt.Println()

	// Rolling mean (window=3)
	rollMean, _ := golars.Col("x").Cast(golars.Float64).Rolling(3).Mean().Evaluate(ctx)
	for i := 0; i < rollMean.Len(); i++ {
		if i > 0 {
			fmt.Print(" ")
		}
		if rollMean.IsNull(i) {
			fmt.Print("null")
		} else {
			v, _ := rollMean.GetFloat64(i)
			fmt.Printf("%.1f", v)
		}
	}
	fmt.Println()
	// Output:
	// 1 3 6 10 15
	// null null 2.0 3.0 4.0
}

func Example_shiftDiffPctChange() {
	df, _ := golars.NewDataFrame(
		golars.NewFloat64Series("price", []float64{100, 105, 103, 110}),
	)
	ctx := &golars.ExprContext{DF: df}

	// Percentage change
	pct, _ := golars.Col("price").PctChange(1).Evaluate(ctx)
	for i := 0; i < pct.Len(); i++ {
		if pct.IsNull(i) {
			fmt.Print("null ")
		} else {
			v, _ := pct.GetFloat64(i)
			fmt.Printf("%.4f ", v)
		}
	}
	fmt.Println()
	// Output:
	// null 0.0500 -0.0190 0.0680
}

func Example_sortBy() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewInt64Series("score", []int64{90, 70, 80}),
	)
	ctx := &golars.ExprContext{DF: df}

	// Sort names by score (ascending)
	sorted, _ := golars.Col("name").SortBy(golars.Col("score"), false).Evaluate(ctx)
	for i := 0; i < sorted.Len(); i++ {
		v, _ := sorted.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// Bob
	// Charlie
	// Alice
}

func Example_strCapitalize() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("word", []string{"hello world", "FOO BAR", "go lang"}),
	)
	ctx := &golars.ExprContext{DF: df}

	capped, _ := golars.Col("word").Str().Capitalize().Evaluate(ctx)
	for i := 0; i < capped.Len(); i++ {
		v, _ := capped.GetString(i)
		fmt.Println(v)
	}
	// Output:
	// Hello world
	// Foo bar
	// Go lang
}

func Example_firstLastExpr() {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)
	ctx := &golars.ExprContext{DF: df}

	first, _ := golars.Col("name").First().Evaluate(ctx)
	last, _ := golars.Col("name").Last().Evaluate(ctx)

	v1, _ := first.GetString(0)
	v2, _ := last.GetString(0)
	fmt.Printf("first=%s last=%s\n", v1, v2)
	// Output:
	// first=Alice last=Charlie
}
