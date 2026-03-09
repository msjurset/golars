# Golars

[![Go Reference](https://pkg.go.dev/badge/github.com/msjurseth/golars.svg)](https://pkg.go.dev/github.com/msjurseth/golars)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A high-performance DataFrame library for Go, modeled after [Polars](https://pola.rs).

Golars brings the power of columnar data processing to Go with zero external dependencies, an expression-based API, lazy evaluation with query optimization, and comprehensive I/O support.

```go
import "github.com/msjurseth/golars"
```

## Features

- **Columnar storage** with offset-based strings and null bitmaps
- **Expression DSL** — composable, type-safe column operations
- **Lazy evaluation** — query optimization with predicate/projection pushdown
- **GroupBy & Join** — hash-based grouping and all 7 join types
- **Window functions** — SQL-style `Over(partitionBy...)` expressions
- **SQL interface** — register DataFrames and query with SQL
- **I/O** — CSV, JSON, NDJSON, Parquet, Excel (.xlsx), `database/sql`
- **Zero dependencies** — pure Go standard library
- **Go 1.24+** — uses generics and range-over-func iterators

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/msjurseth/golars"
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
```

### Expressions

Expressions are the heart of golars. They compose into computation trees that can be evaluated eagerly or deferred for lazy optimization.

```go
// Filter with expressions
ctx := &golars.ExprContext{DF: df}
mask, _ := golars.Col("age").Gt(golars.Lit(30)).Evaluate(ctx)
filtered, _ := df.Filter(mask)

// Arithmetic
curved, _ := golars.Col("score").Mul(golars.Lit(1.1)).Alias("curved").Evaluate(ctx)
df2, _ := df.WithColumn(curved)

// Conditional (When/Then/Otherwise)
tier, _ := golars.When(golars.Col("age").Gt(golars.Lit(35))).
    Then(golars.Lit("senior")).
    Otherwise(golars.Lit("junior")).
    Evaluate(ctx)
```

### GroupBy & Aggregation

```go
grouped, _ := df.GroupBy("department")
result, _ := grouped.Agg(map[string]golars.AggFunc{
    "salary": golars.AggMean,
    "bonus":  golars.AggSum,
    "name":   golars.AggCount,
})
```

Available aggregations: `AggSum`, `AggMean`, `AggMin`, `AggMax`, `AggCount`, `AggFirst`, `AggLast`.

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

### Lazy Evaluation

Lazy evaluation records operations as a query plan, then optimizes and executes when you call `Collect()`:

```go
result, err := golars.Lazy(df).
    Filter(golars.Col("score").Gt(golars.Lit(80.0))).
    Sort("score", true).
    Head(10).
    Collect()
```

Inspect the query plan:
```go
fmt.Println(golars.Lazy(df).
    Filter(golars.Col("score").Gt(golars.Lit(80.0))).
    Sort("score", true).
    Explain())
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

Supports: SELECT, FROM, WHERE, JOIN, GROUP BY, ORDER BY, LIMIT.

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

// Rolling windows
rm := s.RollingMean(3)
rs := s.RollingSum(3)

// Cumulative
cs := s.CumSum()

// Shifting
shifted := s.Shift(1)
diff := s.Diff(1)
pct := s.PctChange(1)
```

### String Operations

```go
names := golars.NewStringSeries("name", []string{"Alice Smith", "Bob Jones", "Charlie Brown"})

contains := names.Str().Contains("Smith")     // Boolean Series
upper := names.Str().ToUpper()                 // "ALICE SMITH", ...
first := names.Str().Split(" ", 0)             // "Alice", "Bob", "Charlie"
lengths := names.Str().Lengths()               // 11, 9, 13
trimmed := names.Str().Trim()
replaced := names.Str().Replace("Smith", "Lee")
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

// DataFrame null handling
clean := df.DropNulls()
filled, _ := df.FillNull(map[string]any{"score": 0.0})
```

### Type Casting

```go
ints := golars.NewInt64Series("x", []int64{1, 2, 3})
floats, _ := ints.Cast(golars.Float64)
strings, _ := ints.Cast(golars.String)
```

### Reshaping

```go
// Pivot (long to wide)
pivoted, _ := df.Pivot("name", "subject", "score")

// Unpivot/Melt (wide to long)
melted, _ := df.Unpivot([]string{"id"}, []string{"math", "english"})

// Transpose
transposed, _ := df.Transpose()
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

## Concurrency

- All read operations on DataFrame and Series are safe for concurrent use
- Mutations return new values (copy-on-write semantics)
- Internal operations parallelize across `GOMAXPROCS` for large datasets

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
- Zero external dependencies

## Acknowledgments

Golars is inspired by [Polars](https://pola.rs), the blazing-fast DataFrame library for Python and Rust created by Ritchie Vink. The API design, expression system, and lazy evaluation concepts draw directly from Polars' excellent architecture.

## License

MIT — see [LICENSE](LICENSE) for details.
