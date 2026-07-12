# Golars

[![Go Reference](https://pkg.go.dev/badge/github.com/msjurset/golars.svg)](https://pkg.go.dev/github.com/msjurset/golars)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A high-performance DataFrame library for Go, modeled after [Polars](https://pola.rs).

Golars brings the power of columnar data processing to Go with an expression-based API, lazy evaluation with query optimization, and comprehensive I/O support.

```go
import "github.com/msjurset/golars"
```

## Features

- **Columnar storage** with offset-based strings and null bitmaps
- **Expression DSL** — composable, type-safe column operations with math, string, temporal, cumulative, and rolling namespaces
- **Lazy evaluation** — query optimization with predicate pushdown, projection pushdown, constant folding, and CSE
- **GroupBy & Join** — hash-based grouping (map-based and expression-based) and all 7 join types
- **Window functions** — SQL-style `Over(partitionBy...)` expressions
- **SQL interface** — register DataFrames and query with standard SQL, including full `JOIN` and table alias resolution (`SELECT e.name FROM emps e`)
- **I/O** — CSV, JSON, NDJSON, Parquet (with Snappy compression), Excel (.xlsx), `database/sql`
- **Temporal types** — Date, DateTime, Time, Duration with `.Dt()` namespace
- **Go 1.24+** — uses generics and range-over-func iterators

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/msjurset/golars"
)

func main() {
    // Create a DataFrame
    df, err := golars.NewDataFrame(
        golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana"}),
        golars.NewInt64Series("age", []int64{25, 30, 35, 40}),
        golars.NewFloat64Series("score", []float64{88.5, 92.3, 76.1, 95.0}),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(df)
}
```

Output:
```
┌─────────┬─────┬───────┐
│ name    │ age │ score │
│ ---     │ --- │ ---   │
│ str     │ i64 │ f64   │
╞═════════╪═════╪═══════╡
│ Alice   │  25 │ 88.50 │
│ Bob     │  30 │ 92.30 │
│ Charlie │  35 │ 76.10 │
│ Diana   │  40 │ 95.00 │
└─────────┴─────┴───────┘
```

## Core Concepts

### DataFrames

An immutable collection of named, typed columns (Series). All operations return new DataFrames.

```go
// Select columns
selected, _ := df.Select("name", "score")

// Drop columns
trimmed, _ := df.Drop("age")

// Add a computed column
df2, _ := df.WithColumn(golars.NewBooleanSeries("senior", []bool{false, false, true, true}))

// Sort
sorted, _ := df.Sort("score", true) // descending

// Head / Tail / Slice
first2 := df.Head(2)
last2 := df.Tail(2)
middle := df.Slice(1, 3)

// Sampling
sample := df.Sample(2, 42)
fraction := df.SampleFraction(0.5, 42)
```

### Expressions

Expressions are the heart of golars. They compose into computation trees that can be evaluated eagerly or deferred for lazy optimization.

```go
// Filter with expressions
ctx := &golars.ExprContext{DF: df}
mask, _ := golars.Col("age").Gt(golars.Lit(30)).Evaluate(ctx)
filtered, _ := df.Filter(mask)

// Arithmetic (Add, Sub, Mul, Div, Mod, Pow)
curved, _ := golars.Col("score").Mul(golars.Lit(1.1)).Alias("curved").Evaluate(ctx)
df2, _ := df.WithColumn(curved)

// Conditional (When/Then/Otherwise)
tier, _ := golars.When(golars.Col("age").Gt(golars.Lit(35))).
    Then(golars.Lit("senior")).
    Otherwise(golars.Lit("junior")).
    Evaluate(ctx)
```

### Numeric Math

All common math functions are available as expression methods:

```go
ctx := &golars.ExprContext{DF: df}

// Absolute value
abs, _ := golars.Col("profit").Abs().Evaluate(ctx)

// Square root, log, exp
sqrt, _ := golars.Col("variance").Sqrt().Evaluate(ctx)
logVal, _ := golars.Col("x").Log().Evaluate(ctx)
expVal, _ := golars.Col("x").Exp().Evaluate(ctx)

// Rounding
rounded, _ := golars.Col("price").Round(2).Evaluate(ctx)
floored, _ := golars.Col("price").Floor().Evaluate(ctx)
ceiled, _ := golars.Col("price").Ceil().Evaluate(ctx)
```

### Predicates

```go
// Membership testing
inSet, _ := golars.Col("status").IsIn("active", "pending").Evaluate(ctx)

// Range checking
inRange, _ := golars.Col("age").IsBetween(golars.Lit(18), golars.Lit(65)).Evaluate(ctx)
```

### Selection & Sorting

```go
// First/Last aggregation
first, _ := golars.Col("name").First().Evaluate(ctx)
last, _ := golars.Col("name").Last().Evaluate(ctx)

// Head/Tail on expressions
top3, _ := golars.Col("score").Head(3).Evaluate(ctx)
bot3, _ := golars.Col("score").Tail(3).Evaluate(ctx)

// Gather by indices
picked, _ := golars.Col("name").Gather([]int{0, 2, 4}).Evaluate(ctx)

// Unique values
uniq, _ := golars.Col("category").Unique().Evaluate(ctx)

// Sort one column by another
sorted, _ := golars.Col("name").SortBy(golars.Col("score"), true).Evaluate(ctx)
```

### GroupBy & Aggregation

```go
// Map-based aggregation
grouped, _ := df.GroupBy("department")
result, _ := grouped.Agg(map[string]golars.AggFunc{
    "salary": golars.AggMean,
    "bonus":  golars.AggSum,
    "name":   golars.AggCount,
})

// Expression-based aggregation (Polars-style)
result, _ = golars.GroupByAgg(grouped,
    golars.Col("salary").Mean().Alias("avg_salary"),
    golars.Col("salary").Sum().Alias("total_salary"),
    golars.Col("name").Count().Alias("headcount"),
)
```

Available aggregations: `Sum`, `Mean`, `Min`, `Max`, `Count`, `First`, `Last`, `Std`, `Var`, `NUnique`, `Median`, `Quantile(p)`.

### Joins

All 7 join types are supported:

```go
result, _ := left.Join(right, []string{"id"}, golars.InnerJoin)
result, _ = left.Join(right, []string{"id"}, golars.LeftJoin)
result, _ = left.Join(right, []string{"id"}, golars.RightJoin)
result, _ = left.Join(right, []string{"id"}, golars.FullJoin)
result, _ = left.Join(right, []string{"id"}, golars.SemiJoin)
result, _ = left.Join(right, []string{"id"}, golars.AntiJoin)
result, _ = left.Join(right, []string{"id"}, golars.CrossJoin)
```

### Window Functions

SQL-style window functions compute per-partition aggregates broadcast back to the original row order:

```go
ctx := &golars.ExprContext{DF: df}
avgByDept, _ := golars.Col("salary").Mean().Over("department").Evaluate(ctx)
```

### Row Transforms

```go
// Shift values by n positions
shifted, _ := golars.Col("price").Shift(1).Evaluate(ctx)

// Element-wise difference
diff, _ := golars.Col("price").Diff(1).Evaluate(ctx)

// Percentage change
pctChg, _ := golars.Col("price").PctChange(1).Evaluate(ctx)

// Rank
ranked, _ := golars.Col("score").Rank().Evaluate(ctx)
```

### Cumulative Operations

```go
cumSum, _ := golars.Col("revenue").Cum().Sum().Evaluate(ctx)
cumProd, _ := golars.Col("growth").Cum().Prod().Evaluate(ctx)
cumMin, _ := golars.Col("price").Cum().Min().Evaluate(ctx)
cumMax, _ := golars.Col("price").Cum().Max().Evaluate(ctx)
```

### Rolling Windows

```go
rollMean, _ := golars.Col("price").Rolling(7).Mean().Evaluate(ctx)
rollSum, _ := golars.Col("volume").Rolling(7).Sum().Evaluate(ctx)
rollMin, _ := golars.Col("price").Rolling(7).Min().Evaluate(ctx)
rollMax, _ := golars.Col("price").Rolling(7).Max().Evaluate(ctx)
rollStd, _ := golars.Col("price").Rolling(7).Std().Evaluate(ctx)
```

### Lazy Evaluation

Lazy evaluation records operations as a query plan, then optimizes and executes when you call `Collect()`:

```go
result, err := golars.Lazy(df).
    Filter(golars.Col("score").Gt(golars.Lit(80.0))).
    Sort("score", true).
    Head(10).
    Collect()
```

The optimizer applies:
- **Predicate pushdown** — pushes filters closer to data sources
- **Projection pushdown** — only reads needed columns
- **Constant folding** — evaluates constant expressions at plan time
- **Common subexpression elimination** — avoids redundant computation
- **Projection merging** — combines adjacent select nodes

Inspect the query plan:
```go
fmt.Println(golars.Lazy(df).
    Filter(golars.Col("score").Gt(golars.Lit(80.0))).
    Explain())
```

Lazy I/O scans — only materialize data when collected:
```go
result, _ := golars.ScanCSV("large.csv").
    Filter(golars.Col("status").Eq(golars.Lit("active"))).
    Select(golars.Col("name"), golars.Col("email")).
    Collect()

result, _ = golars.ScanParquet("data.parquet").
    Filter(golars.Col("year").Gt(golars.Lit(2020))).
    Collect()
```

### SQL Interface

Register DataFrames and query them with SQL:

```go
ctx := golars.NewSQLContext()
ctx.Register("users", df)

result, err := ctx.Execute(`
    SELECT name, score
    FROM users
    WHERE age > 30
    ORDER BY score DESC
    LIMIT 5
`)
```

Supports: SELECT, FROM, WHERE, JOIN, GROUP BY, ORDER BY, LIMIT, aggregate functions, string functions.

### Row Iteration

Use Go 1.23+ range-over-func iterators:

```go
for row := range df.Rows() {
    name, _ := row.String("name")
    age, _ := row.Int64("age")
    fmt.Printf("%s: %d\n", name, age)
}
```

### Series Operations

Series support rich analytics:

```go
s := golars.NewFloat64Series("x", []float64{1, 2, 3, 4, 5})

// Aggregation
sum, _ := s.Sum()
mean, _ := s.Mean()
std, _ := s.Std()
median := ... // via expression: Col("x").Median()

// Rolling windows
rm := s.RollingMean(3)
rs := s.RollingSum(3)

// Cumulative
cs := s.CumSum()
cp := s.CumProd()

// Shifting
shifted := s.Shift(1)
diff := s.Diff(1)
pct := s.PctChange(1)

// Ranking
ranked := s.Rank("average") // "average", "min", "max", "dense", "ordinal"

// Interpolation
filled := s.Interpolate("linear") // "linear", "forward_fill", "backfill"
```

### String Operations

The `.Str()` namespace provides string operations on both Series and expressions:

```go
names := golars.NewStringSeries("name", []string{"Alice Smith", "Bob Jones", "Charlie Brown"})

// Checks
contains := names.Str().Contains("Smith")      // Boolean Series
starts := names.Str().StartsWith("Alice")      // Boolean Series
ends := names.Str().EndsWith("Brown")          // Boolean Series

// Transformations
upper := names.Str().ToUpper()                  // "ALICE SMITH", ...
lower := names.Str().ToLower()                  // "alice smith", ...
capped := names.Str().Capitalize()              // "Alice smith", ...

// Extraction
first := names.Str().Split(" ", 0)              // "Alice", "Bob", "Charlie"
sub := names.Str().Slice(0, 5)                  // "Alice", "Bob J", "Charl"
lengths := names.Str().Lengths()                // 11, 9, 13

// Manipulation
trimmed := names.Str().Trim()
replaced := names.Str().Replace("Smith", "Lee")
padded := names.Str().Pad(20, "right", ' ')
zfilled := names.Str().ZFill(15)
stripped := names.Str().Strip(" ")

// Regex
extracted := names.Str().Extract(`(\w+)`, 1)    // First word
count := names.Str().CountMatches(`[aeiou]`)    // Vowel count

// Parse to DateTime
dates := golars.NewStringSeries("d", []string{"2024-01-15", "2024-06-30"})
dt := dates.Str().ToDatetime("2006-01-02")      // DateTime Series
```

### Temporal Operations

The `.Dt()` namespace provides temporal operations on Date, DateTime, Time, and Duration Series:

```go
// Component extraction
year := golars.Col("timestamp").Dt().Year()
month := golars.Col("timestamp").Dt().Month()
day := golars.Col("timestamp").Dt().Day()
hour := golars.Col("timestamp").Dt().Hour()
weekday := golars.Col("timestamp").Dt().Weekday()
quarter := golars.Col("timestamp").Dt().Quarter()
dayOfYear := golars.Col("timestamp").Dt().DayOfYear()
isoWeek := golars.Col("timestamp").Dt().IsoWeek()

// Truncation
truncated := golars.Col("timestamp").Dt().Truncate("1d")  // "1h", "1d", "1mo", "1y"

// Formatting
formatted := golars.Col("timestamp").Dt().Strftime("2006-01-02")

// Offset
shifted := golars.Col("timestamp").Dt().OffsetBy("7d")    // "1d", "2mo", "-1y", "3h"

// Epoch conversion
epochSec := golars.Col("timestamp").Dt().Epoch("s")       // "s", "ms", "us", "ns"

// Duration total seconds
totalSec := golars.Col("duration").Dt().TotalSeconds()     // Float64
```

### Name Namespace

Transform expression output names:

```go
prefixed := golars.Col("score").Name().Prefix("raw_")   // "raw_score"
suffixed := golars.Col("score").Name().Suffix("_v2")     // "score_v2"
mapped := golars.Col("score").Name().Map(strings.ToUpper) // "SCORE"
```

### Null Handling

```go
// Create Series with nulls
s := golars.NewInt64SeriesWithValidity("x", []int64{1, 0, 3}, []bool{true, false, true})

s.IsNull(1)     // true
s.NullCount()   // 1
s.HasNulls()    // true

dropped := s.DropNulls()            // [1, 3]
filled := s.FillNullInt64(0)        // [1, 0, 3]

// Expression null handling
isNull := golars.Col("x").IsNull()
isNotNull := golars.Col("x").IsNotNull()
filled := golars.Col("x").FillNull(golars.Lit(0))

// DataFrame null handling
clean := df.DropNulls()
filled, _ := df.FillNull(map[string]any{"score": 0.0})
```

### Type Casting

```go
ints := golars.NewInt64Series("x", []int64{1, 2, 3})
floats, _ := ints.Cast(golars.Float64)
strings, _ := ints.Cast(golars.String)

// TryCast returns null instead of error on failure
safe, _ := s.TryCast(golars.Int64)

// Expression cast
casted := golars.Col("x").Cast(golars.Float64)
tryCasted := golars.Col("x").TryCast(golars.Float64)
```

### Reshaping

```go
// Pivot (long to wide)
pivoted, _ := df.Pivot("name", "subject", "score")

// Unpivot/Melt (wide to long)
melted, _ := df.Unpivot([]string{"id"}, []string{"math", "english"})

// Transpose
transposed, _ := df.Transpose()

// Explode
exploded, _ := df.Explode("tags")
```

## I/O

### CSV

```go
// Read
df, _ := golars.ReadCSV("data.csv")
df, _ = golars.ReadCSV("data.tsv", golars.WithSeparator('\t'))
df, _ = golars.ReadCSV("data.csv",
    golars.WithNullValues([]string{"NA", ""}),
    golars.WithCSVColumns([]string{"name", "age"}),
    golars.WithInferSchemaLength(1000),
    golars.WithSkipRows(1),
    golars.WithNRows(100),
)

// Read with context (cancellation support)
df, _ = golars.ReadCSVWithContext(ctx, "data.csv")

// Write
golars.WriteCSVFile(df, "output.csv")

// io.Reader / io.Writer integration
golars.WriteCSV(df, os.Stdout)
df, _ = golars.ReadCSVFromReader(reader)
```

### JSON / NDJSON

```go
df, _ := golars.ReadJSON("data.json")
df, _ = golars.ReadNDJSON("data.ndjson")

golars.WriteJSONFile(df, "output.json")
golars.WriteNDJSONFile(df, "output.ndjson")
```

### Parquet

```go
df, _ := golars.ReadParquet("data.parquet")
golars.WriteParquetFile(df, "output.parquet")

// With Snappy compression
golars.WriteParquetFile(df, "compressed.parquet", golars.WithParquetCompression("snappy"))
```

### Excel

```go
df, _ := golars.ReadExcel("data.xlsx")
golars.WriteExcelFile(df, "output.xlsx")
```

### Database

```go
import "database/sql"

db, _ := sql.Open("sqlite3", "data.db")
df, _ := golars.ReadSQL(db, "SELECT * FROM users WHERE age > ?", 25)
```

## Data Types

| Type | Go Storage | Constructor |
|------|-----------|-------------|
| Int8/16/32/64 | intN | `NewIntNSeries` |
| UInt8/16/32/64 | uintN | `NewUIntNSeries` |
| Float32/64 | floatN | `NewFloatNSeries` |
| Boolean | bool | `NewBooleanSeries` |
| String | offset-based | `NewStringSeries` |
| Date | int32 (days since epoch) | `NewDateSeries` |
| DateTime | int64 (microseconds since epoch) | `NewDateTimeSeries` |
| Time | int64 (nanoseconds since midnight) | `NewTimeSeries` |
| Duration | int64 (microseconds) | `NewDurationSeries` |

Convenience constructors from `time.Time`:
```go
dates := golars.NewDateSeriesFromTime("d", []time.Time{...})
timestamps := golars.NewDateTimeSeriesFromTime("ts", []time.Time{...})
```

## Concurrency

- All read operations on DataFrame and Series are safe for concurrent use
- Mutations return new values (copy-on-write semantics)
- Internal operations parallelize across `GOMAXPROCS` for large datasets
- Context-aware operations support cancellation via `context.Context`

## Benchmarks

Run benchmarks:
```sh
go test -bench=. -benchtime=3s ./bench/
```

Compare with Python Polars:
```sh
python bench/bench_polars.py
```

## Requirements

- Go 1.24+

## Acknowledgments

Golars is inspired by [Polars](https://pola.rs), the blazing-fast DataFrame library for Python and Rust created by Ritchie Vink. The API design, expression system, and lazy evaluation concepts draw directly from Polars' excellent architecture.

## License

MIT — see [LICENSE](LICENSE) for details.
