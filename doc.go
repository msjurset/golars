// Package golars provides a high-performance DataFrame library for Go,
// inspired by Python's Polars. It offers columnar data storage, a composable
// expression DSL, lazy evaluation with query optimization, SQL querying, and
// comprehensive I/O support for CSV, JSON, Parquet, and Excel formats.
//
// # Import
//
//	import "github.com/msjurset/golars"
//
// All public types and functions are re-exported at the top level. Internal
// sub-packages use Go's internal/ convention and cannot be imported directly.
//
// # Key Design Principles
//
// Immutability: DataFrames and Series are immutable after construction. Every
// transformation returns a new value, making concurrent reads safe without
// synchronization. The underlying array storage is shared where possible to
// minimize copies.
//
// Expression DSL: Rather than embedding logic in method chains on DataFrames,
// golars uses a composable expression system. Expressions like [Col], [Lit],
// and [When] build up computation trees that can be evaluated against a
// DataFrame context, enabling reuse, composition, and optimization.
//
// Lazy Evaluation: The [Lazy] function wraps a DataFrame in a [LazyFrame] that
// records operations as a logical plan. The plan is optimized (predicate
// pushdown, projection pruning) and only executed when [LazyFrame.Collect] is
// called.
//
// # Constructing DataFrames
//
// Create typed Series and combine them into a DataFrame:
//
//	df, err := golars.NewDataFrame(
//	    golars.NewInt64Series("id", []int64{1, 2, 3}),
//	    golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
//	    golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1}),
//	)
//
// Series with nullable values use explicit validity bitmaps:
//
//	s := golars.NewInt64SeriesWithValidity("val", []int64{10, 0, 30}, []bool{true, false, true})
//
// # Selecting, Filtering, and Sorting
//
//	selected, err := df.Select("name", "score")
//	dropped, err := df.Drop("id")
//	sorted, err := df.Sort("score", true) // descending
//
// Filtering uses the expression system to produce a boolean mask:
//
//	ctx := &golars.ExprContext{DF: df}
//	mask, err := golars.Col("score").Gt(golars.Lit(80.0)).Evaluate(ctx)
//	filtered, err := df.Filter(mask)
//
// # Expressions
//
// Expressions are composable building blocks for column-level computation:
//
//	// Arithmetic
//	golars.Col("price").Mul(golars.Lit(1.1)).Alias("with_tax")
//
//	// Comparison and logical
//	golars.Col("age").Gte(golars.Lit(18)).And(golars.Col("active").Eq(golars.Lit(true)))
//
//	// Aggregation
//	golars.Col("score").Mean()
//	golars.Col("revenue").Sum().Over("region") // window function
//
//	// Conditional
//	golars.When(golars.Col("score").Gt(golars.Lit(90))).
//	    Then(golars.Lit("A")).
//	    Otherwise(golars.Lit("B"))
//
// # GroupBy and Aggregation
//
//	grouped, err := df.GroupBy("department")
//	result, err := grouped.Agg(map[string]golars.AggFunc{
//	    "salary": golars.AggMean,
//	    "id":     golars.AggCount,
//	})
//
// Available aggregation functions: [AggSum], [AggMean], [AggMin], [AggMax],
// [AggCount], [AggFirst], [AggLast].
//
// # Joins
//
//	joined, err := left.Join(right, []string{"id"}, golars.InnerJoin)
//
// Supported join types: [InnerJoin], [LeftJoin], [RightJoin], [FullJoin],
// [SemiJoin], [AntiJoin], [CrossJoin].
//
// # Reshaping
//
//	pivoted, err := df.Pivot("name", "subject", "score")
//	melted, err := df.Unpivot([]string{"id"}, []string{"math", "english"})
//	transposed, err := df.Transpose()
//
// # Lazy Evaluation
//
// Build a query plan and execute it in one optimized pass:
//
//	result, err := golars.Lazy(df).
//	    Filter(golars.Col("age").Gt(golars.Lit(25))).
//	    Sort("score", true).
//	    Head(10).
//	    Collect()
//
// Inspect the logical plan with Explain or ExplainOptimized:
//
//	plan := golars.Lazy(df).Filter(expr).Sort("col", false).Explain()
//
// # I/O
//
// Read and write CSV, JSON, NDJSON, Parquet, and Excel files. All readers
// accept file paths; most also offer io.Reader variants for streaming:
//
//	df, err := golars.ReadCSV("data.csv", golars.WithSeparator('\t'))
//	df, err := golars.ReadCSVFromReader(r, golars.WithHasHeader(false))
//	err = golars.WriteCSV(df, w)
//
//	df, err = golars.ReadParquet("data.parquet")
//	err = golars.WriteParquetFile(df, "out.parquet")
//
//	df, err = golars.ReadJSON("data.json")
//	df, err = golars.ReadExcel("data.xlsx")
//
// CSV options: [WithSeparator], [WithNullValues], [WithCSVColumns],
// [WithInferSchemaLength], [WithSkipRows], [WithNRows], [WithHasHeader].
//
// # SQL
//
// Register DataFrames and query them with standard SQL syntax:
//
//	ctx := golars.NewSQLContext()
//	ctx.Register("users", df)
//	result, err := ctx.Execute("SELECT name, AVG(score) FROM users GROUP BY name")
//
// # Series Operations
//
// Series provide rich per-column operations:
//
//	s.Sum()              // aggregation (returns float64, bool)
//	s.Mean()
//	s.RollingMean(3)     // rolling window (returns *Series)
//	s.CumSum()           // cumulative operations
//	s.Shift(1)           // shift values
//	s.Str().ToUpper()    // string namespace methods
//	s.Cast(golars.Float64)
//
// # Row-Level Operations
//
//	row := df.Row(0)             // map[string]any
//	rows := df.ToMaps()          // []map[string]any
//	s := df.MapRows("sum", fn)   // apply function per row
package golars
