package bench

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/msjurseth/golars"
)

// ---------------------------------------------------------------------------
// Data generation helpers
// ---------------------------------------------------------------------------

func makeInt64Slice(n int) []int64 {
	d := make([]int64, n)
	for i := range d {
		d[i] = int64(i)
	}
	return d
}

func makeFloat64Slice(n int) []float64 {
	d := make([]float64, n)
	for i := range d {
		d[i] = float64(i) * 1.1
	}
	return d
}

var cycleStrings = []string{"alpha", "beta", "gamma", "delta", "epsilon"}

func makeStringSlice(n int) []string {
	d := make([]string, n)
	for i := range d {
		d[i] = cycleStrings[i%len(cycleStrings)]
	}
	return d
}

func makeGroupSlice(n, nGroups int) []string {
	d := make([]string, n)
	for i := range d {
		d[i] = "g" + strconv.Itoa(i%nGroups)
	}
	return d
}

func makeInt64Series(name string, n int) *golars.Series {
	return golars.NewInt64Series(name, makeInt64Slice(n))
}

func makeFloat64Series(name string, n int) *golars.Series {
	return golars.NewFloat64Series(name, makeFloat64Slice(n))
}

func makeStringSeries(name string, n int) *golars.Series {
	return golars.NewStringSeries(name, makeStringSlice(n))
}

func makeGroupSeries(name string, n, nGroups int) *golars.Series {
	return golars.NewStringSeries(name, makeGroupSlice(n, nGroups))
}

func makeDF(n int) *golars.DataFrame {
	df, _ := golars.NewDataFrame(
		makeInt64Series("id", n),
		makeFloat64Series("value", n),
		makeStringSeries("group", n),
	)
	return df
}

func makeDFWithGroups(n, nGroups int) *golars.DataFrame {
	df, _ := golars.NewDataFrame(
		makeInt64Series("id", n),
		makeFloat64Series("value", n),
		makeGroupSeries("group", n, nGroups),
	)
	return df
}

func makeWideDF(n, nCols int) *golars.DataFrame {
	cols := make([]*golars.Series, nCols)
	for i := range cols {
		cols[i] = makeFloat64Series(fmt.Sprintf("col_%d", i), n)
	}
	df, _ := golars.NewDataFrame(cols...)
	return df
}

// Standard scales used across benchmarks.
var scales = []struct {
	name string
	n    int
}{
	{"1K", 1_000},
	{"10K", 10_000},
	{"100K", 100_000},
	{"1M", 1_000_000},
	{"10M", 10_000_000},
}

// Smaller scales for expensive operations (join, pivot, IO).
var smallScales = []struct {
	name string
	n    int
}{
	{"1K", 1_000},
	{"10K", 10_000},
	{"100K", 100_000},
}

// ═══════════════════════════════════════════════════════════════════════════
// 1. Series Creation
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesCreate(b *testing.B) {
	for _, sc := range scales {
		b.Run("Int64/"+sc.name, func(b *testing.B) {
			data := makeInt64Slice(sc.n)
			b.ResetTimer()
			for b.Loop() {
				golars.NewInt64Series("x", data)
			}
		})
		b.Run("Float64/"+sc.name, func(b *testing.B) {
			data := makeFloat64Slice(sc.n)
			b.ResetTimer()
			for b.Loop() {
				golars.NewFloat64Series("x", data)
			}
		})
		b.Run("String/"+sc.name, func(b *testing.B) {
			data := makeStringSlice(sc.n)
			b.ResetTimer()
			for b.Loop() {
				golars.NewStringSeries("x", data)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 2. Series Aggregation
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesAgg(b *testing.B) {
	aggs := []struct {
		name string
		fn   func(s *golars.Series)
	}{
		{"Sum", func(s *golars.Series) { s.Sum() }},
		{"Mean", func(s *golars.Series) { s.Mean() }},
		{"Min", func(s *golars.Series) { s.Min() }},
		{"Max", func(s *golars.Series) { s.Max() }},
	}
	for _, agg := range aggs {
		for _, sc := range scales {
			b.Run(agg.name+"/"+sc.name, func(b *testing.B) {
				s := makeFloat64Series("x", sc.n)
				b.ResetTimer()
				for b.Loop() {
					agg.fn(s)
				}
			})
		}
	}
}

func BenchmarkSeriesNUnique(b *testing.B) {
	for _, sc := range scales[:4] { // up to 1M
		b.Run(sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.NUnique()
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 3. Series Sort
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesSort(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("Float64/Asc/"+sc.name, func(b *testing.B) {
			s := makeFloat64Series("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Sort(false)
			}
		})
		b.Run("Float64/Desc/"+sc.name, func(b *testing.B) {
			s := makeFloat64Series("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Sort(true)
			}
		})
		b.Run("String/Asc/"+sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Sort(false)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 4. Series Rolling
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesRollingMean(b *testing.B) {
	windows := []int{10, 100}
	for _, w := range windows {
		for _, sc := range scales[:4] {
			b.Run(fmt.Sprintf("W%d/%s", w, sc.name), func(b *testing.B) {
				s := makeFloat64Series("x", sc.n)
				b.ResetTimer()
				for b.Loop() {
					s.RollingMean(w)
				}
			})
		}
	}
}

func BenchmarkSeriesRollingSum(b *testing.B) {
	windows := []int{10, 100}
	for _, w := range windows {
		for _, sc := range scales[:4] {
			b.Run(fmt.Sprintf("W%d/%s", w, sc.name), func(b *testing.B) {
				s := makeFloat64Series("x", sc.n)
				b.ResetTimer()
				for b.Loop() {
					s.RollingSum(w)
				}
			})
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 5. Series Cumulative
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesCumSum(b *testing.B) {
	for _, sc := range scales {
		b.Run(sc.name, func(b *testing.B) {
			s := makeInt64Series("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.CumSum()
			}
		})
	}
}

func BenchmarkSeriesCumMax(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			s := makeFloat64Series("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.CumMax()
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 6. Series String Ops
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesStrContains(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Str().Contains("alp")
			}
		})
	}
}

func BenchmarkSeriesStrToUpper(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Str().ToUpper()
			}
		})
	}
}

func BenchmarkSeriesStrLengths(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Str().Lengths()
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 7. DataFrame Creation
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameCreate(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("3Col/"+sc.name, func(b *testing.B) {
			c1 := makeInt64Series("id", sc.n)
			c2 := makeFloat64Series("value", sc.n)
			c3 := makeStringSeries("group", sc.n)
			b.ResetTimer()
			for b.Loop() {
				golars.NewDataFrame(c1, c2, c3)
			}
		})
	}
	// Wide DataFrame
	for _, width := range []int{10, 50, 100} {
		b.Run(fmt.Sprintf("%dCol/10K", width), func(b *testing.B) {
			cols := make([]*golars.Series, width)
			for i := range cols {
				cols[i] = makeFloat64Series(fmt.Sprintf("c%d", i), 10_000)
			}
			b.ResetTimer()
			for b.Loop() {
				golars.NewDataFrame(cols...)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 8. DataFrame Filter
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameFilter(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("Gt50pct/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			threshold := float64(sc.n) * 0.5 * 1.1
			mask, _ := golars.Col("value").Gt(golars.Lit(threshold)).Evaluate(ctx)
			b.ResetTimer()
			for b.Loop() {
				df.Filter(mask)
			}
		})
		b.Run("Gt90pct/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			threshold := float64(sc.n) * 0.9 * 1.1
			mask, _ := golars.Col("value").Gt(golars.Lit(threshold)).Evaluate(ctx)
			b.ResetTimer()
			for b.Loop() {
				df.Filter(mask)
			}
		})
	}
}

func BenchmarkDataFrameFilterExprEval(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			threshold := float64(sc.n) * 0.5 * 1.1
			filterExpr := golars.Col("value").Gt(golars.Lit(threshold))
			b.ResetTimer()
			for b.Loop() {
				mask, _ := filterExpr.Evaluate(ctx)
				df.Filter(mask)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 9. DataFrame Sort
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameSort(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("SingleCol/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				df.Sort("value", false)
			}
		})
	}
}

func BenchmarkDataFrameSortBy(b *testing.B) {
	for _, sc := range scales[:3] { // up to 100K for multi-col sort
		b.Run("MultiCol/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				df.SortBy([]string{"group", "value"}, []bool{false, true})
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 10. DataFrame GroupBy
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameGroupBy(b *testing.B) {
	groupCounts := []struct {
		name    string
		nGroups int
	}{
		{"5grp", 5},
		{"100grp", 100},
		{"10Kgrp", 10_000},
	}
	for _, gc := range groupCounts {
		for _, sc := range scales[:4] {
			if gc.nGroups > sc.n {
				continue
			}
			b.Run(fmt.Sprintf("Sum/%s/%s", gc.name, sc.name), func(b *testing.B) {
				df := makeDFWithGroups(sc.n, gc.nGroups)
				b.ResetTimer()
				for b.Loop() {
					g, _ := df.GroupBy("group")
					g.Agg(map[string]golars.AggFunc{"value": golars.AggSum})
				}
			})
		}
	}
	// Multiple aggregations at once
	for _, sc := range scales[:3] {
		b.Run("MultiAgg/5grp/"+sc.name, func(b *testing.B) {
			df := makeDFWithGroups(sc.n, 5)
			b.ResetTimer()
			for b.Loop() {
				g, _ := df.GroupBy("group")
				g.Agg(map[string]golars.AggFunc{
					"value": golars.AggSum,
					"id":    golars.AggMean,
				})
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 11. DataFrame Join
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameJoin(b *testing.B) {
	ratios := []struct {
		name      string
		rightFrac float64
	}{
		{"10pctRight", 0.10},
		{"50pctRight", 0.50},
	}
	joinTypes := []struct {
		name string
		jt   golars.JoinType
	}{
		{"Inner", golars.InnerJoin},
		{"Left", golars.LeftJoin},
	}
	for _, jt := range joinTypes {
		for _, r := range ratios {
			for _, sc := range smallScales {
				rightN := int(float64(sc.n) * r.rightFrac)
				if rightN < 1 {
					rightN = 1
				}
				b.Run(fmt.Sprintf("%s/%s/%s", jt.name, r.name, sc.name), func(b *testing.B) {
					left := makeDF(sc.n)
					right, _ := golars.NewDataFrame(
						makeInt64Series("id", rightN),
						makeFloat64Series("rval", rightN),
					)
					b.ResetTimer()
					for b.Loop() {
						left.Join(right, []string{"id"}, jt.jt)
					}
				})
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 12. DataFrame Select / Drop
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFrameSelect(b *testing.B) {
	widths := []int{10, 50, 100}
	for _, w := range widths {
		b.Run(fmt.Sprintf("Select3of%d/10K", w), func(b *testing.B) {
			df := makeWideDF(10_000, w)
			b.ResetTimer()
			for b.Loop() {
				df.Select("col_0", "col_1", "col_2")
			}
		})
		b.Run(fmt.Sprintf("Drop3of%d/10K", w), func(b *testing.B) {
			df := makeWideDF(10_000, w)
			b.ResetTimer()
			for b.Loop() {
				df.Drop("col_0", "col_1", "col_2")
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 13. DataFrame Pivot
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkDataFramePivot(b *testing.B) {
	pivotScales := []struct {
		name    string
		n       int
		nGroups int
	}{
		{"1K_5grp", 1_000, 5},
		{"10K_5grp", 10_000, 5},
		{"10K_100grp", 10_000, 100},
	}
	for _, ps := range pivotScales {
		b.Run(ps.name, func(b *testing.B) {
			df, _ := golars.NewDataFrame(
				makeInt64Series("id", ps.n),
				makeGroupSeries("cat", ps.n, ps.nGroups),
				makeFloat64Series("val", ps.n),
			)
			b.ResetTimer()
			for b.Loop() {
				df.Pivot("id", "cat", "val")
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 14. Lazy Evaluation
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkLazyFilterSort(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			threshold := float64(sc.n) * 0.5 * 1.1
			b.ResetTimer()
			for b.Loop() {
				golars.Lazy(df).
					Filter(golars.Col("value").Gt(golars.Lit(threshold))).
					Sort("value", true).
					Collect()
			}
		})
	}
}

func BenchmarkLazyGroupByAgg(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDFWithGroups(sc.n, 5)
			b.ResetTimer()
			for b.Loop() {
				golars.Lazy(df).
					GroupBy("group").
					Agg(map[string]golars.AggFunc{"value": golars.AggSum}).
					Collect()
			}
		})
	}
}

func BenchmarkLazyChain(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDFWithGroups(sc.n, 10)
			threshold := float64(sc.n) * 0.3 * 1.1
			b.ResetTimer()
			for b.Loop() {
				golars.Lazy(df).
					Filter(golars.Col("value").Gt(golars.Lit(threshold))).
					Sort("value", false).
					Head(100).
					Collect()
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 15. I/O
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkIOCSVWrite(b *testing.B) {
	for _, sc := range smallScales {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				var buf bytes.Buffer
				golars.WriteCSV(df, &buf)
			}
		})
	}
}

func BenchmarkIOCSVRead(b *testing.B) {
	for _, sc := range smallScales {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			var buf bytes.Buffer
			golars.WriteCSV(df, &buf)
			csvData := buf.Bytes()
			b.ResetTimer()
			for b.Loop() {
				r := bytes.NewReader(csvData)
				golars.ReadCSVFromReader(r)
			}
		})
	}
}

func BenchmarkIOJSONWrite(b *testing.B) {
	for _, sc := range smallScales[:2] { // JSON is slower, limit to 1K/10K
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				var buf bytes.Buffer
				golars.WriteJSON(df, &buf)
			}
		})
	}
}

func BenchmarkIOJSONRead(b *testing.B) {
	for _, sc := range smallScales[:2] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			var buf bytes.Buffer
			golars.WriteJSON(df, &buf)
			jsonData := buf.Bytes()
			b.ResetTimer()
			for b.Loop() {
				r := bytes.NewReader(jsonData)
				golars.ReadJSONFromReader(r)
			}
		})
	}
}

func BenchmarkIOParquetWrite(b *testing.B) {
	for _, sc := range smallScales {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				var buf bytes.Buffer
				golars.WriteParquet(df, &buf)
			}
		})
	}
}

func BenchmarkIOParquetRead(b *testing.B) {
	for _, sc := range smallScales {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			var buf bytes.Buffer
			golars.WriteParquet(df, &buf)
			pqData := buf.Bytes()
			b.ResetTimer()
			for b.Loop() {
				r := bytes.NewReader(pqData)
				golars.ReadParquetFromReader(r)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 16. Window Functions
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkWindowMeanOver(b *testing.B) {
	groupCounts := []struct {
		name    string
		nGroups int
	}{
		{"5grp", 5},
		{"100grp", 100},
	}
	for _, gc := range groupCounts {
		for _, sc := range scales[:3] {
			b.Run(fmt.Sprintf("%s/%s", gc.name, sc.name), func(b *testing.B) {
				df := makeDFWithGroups(sc.n, gc.nGroups)
				ctx := &golars.ExprContext{DF: df}
				windowExpr := golars.Col("value").Mean().Over("group")
				b.ResetTimer()
				for b.Loop() {
					windowExpr.Evaluate(ctx)
				}
			})
		}
	}
}

func BenchmarkWindowSumOver(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run("5grp/"+sc.name, func(b *testing.B) {
			df := makeDFWithGroups(sc.n, 5)
			ctx := &golars.ExprContext{DF: df}
			windowExpr := golars.Col("value").Sum().Over("group")
			b.ResetTimer()
			for b.Loop() {
				windowExpr.Evaluate(ctx)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 17. Expression Evaluation
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkExprArithmetic(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("AddMul/"+sc.name, func(b *testing.B) {
			df, _ := golars.NewDataFrame(
				makeFloat64Series("a", sc.n),
				makeFloat64Series("b", sc.n),
			)
			ctx := &golars.ExprContext{DF: df}
			// (a + b) * lit(2.0)
			e := golars.Col("a").Add(golars.Col("b")).Mul(golars.Lit(2.0))
			b.ResetTimer()
			for b.Loop() {
				e.Evaluate(ctx)
			}
		})
	}
}

func BenchmarkExprComparison(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("GtAndLt/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			low := float64(sc.n) * 0.25 * 1.1
			high := float64(sc.n) * 0.75 * 1.1
			e := golars.Col("value").Gt(golars.Lit(low)).And(golars.Col("value").Lt(golars.Lit(high)))
			b.ResetTimer()
			for b.Loop() {
				e.Evaluate(ctx)
			}
		})
	}
}

func BenchmarkExprConditional(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run("WhenThenOtherwise/"+sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			threshold := float64(sc.n) * 0.5 * 1.1
			e := golars.When(golars.Col("value").Gt(golars.Lit(threshold))).
				Then(golars.Lit(1)).
				Otherwise(golars.Lit(0))
			b.ResetTimer()
			for b.Loop() {
				e.Evaluate(ctx)
			}
		})
	}
}

func BenchmarkExprStrContains(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run(sc.name, func(b *testing.B) {
			df := makeDF(sc.n)
			ctx := &golars.ExprContext{DF: df}
			e := golars.Col("group").Str().Contains("alp")
			b.ResetTimer()
			for b.Loop() {
				e.Evaluate(ctx)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Bonus: Series Unique
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkSeriesUnique(b *testing.B) {
	for _, sc := range scales[:4] {
		b.Run("String/"+sc.name, func(b *testing.B) {
			s := makeStringSeries("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Unique()
			}
		})
		b.Run("Int64/"+sc.name, func(b *testing.B) {
			s := makeInt64Series("x", sc.n)
			b.ResetTimer()
			for b.Loop() {
				s.Unique()
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Bonus: Concat
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkConcatVertical(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run(sc.name, func(b *testing.B) {
			df1 := makeDF(sc.n)
			df2 := makeDF(sc.n)
			b.ResetTimer()
			for b.Loop() {
				golars.ConcatDataFrames(df1, df2)
			}
		})
	}
}

func BenchmarkConcatHorizontal(b *testing.B) {
	for _, sc := range scales[:3] {
		b.Run(sc.name, func(b *testing.B) {
			df1, _ := golars.NewDataFrame(makeFloat64Series("a", sc.n))
			df2, _ := golars.NewDataFrame(makeFloat64Series("b", sc.n))
			b.ResetTimer()
			for b.Loop() {
				golars.ConcatDataFramesHorizontal(df1, df2)
			}
		})
	}
}
