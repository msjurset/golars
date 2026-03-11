# Benchmarks — Golars vs Polars

**Date:** March 10, 2026

## About This Comparison

[Polars](https://pola.rs) is one of the fastest DataFrame libraries available, built in Rust with extensive use of SIMD vectorization, multi-threaded execution, and memory-mapped I/O. It sets a high bar for performance in the data processing space.

Golars aims to bring similar DataFrame capabilities to Go. This benchmark comparison is meant to provide an honest look at where Golars stands relative to Polars, highlighting both strengths and areas where Polars' mature Rust implementation has clear advantages.

### Methodology

- All benchmarks run on the same machine in sequence with no other workloads
- Go benchmarks use `testing.B` with the standard `go test -bench` framework
- Python/Polars benchmarks use `time.perf_counter_ns()` with median-of-5 timing
- Each benchmark is designed to measure the same logical operation in both libraries
- Results reflect end-to-end wall-clock time including all allocations and setup
- The comparison uses the largest common scale for each benchmark (typically 100K or 1M rows)

### Important Caveats

- **Creation benchmarks favor Go.** Go's `Series`/`DataFrame` constructors take ownership of pre-allocated slices (zero-copy), while Polars must copy data across the Python-Rust boundary. This is a fundamental API difference, not a performance comparison of the core engines.
- **Polars benefits from SIMD.** Rust's LLVM backend auto-vectorizes tight loops, giving Polars significant advantages in hash-based operations (GroupBy, Join, Unique) and filtering. Go's compiler does not auto-vectorize.
- **Polars uses chunked arrays.** Operations like vertical concatenation are O(1) in Polars (append a chunk reference) vs O(n) in Golars (must copy data). This is a fundamental architectural difference.
- **Python overhead varies.** Some Polars benchmarks include Python-to-Rust FFI overhead, which may add a small constant cost to operations that are otherwise instantaneous in Rust.

### Environment

| Component | Version |
|-----------|---------|
| Machine | Apple M1 Max (10 cores, ARM64) |
| Go | 1.25.5 darwin/arm64 |
| Python | 3.12.5 |
| Polars | 1.21.0 |
| Golars | latest (this repo) |

---

## Results

### Golars Wins

Operations where Golars is faster than Polars.

| Benchmark | Scale | Golars | Polars | Ratio | Notes |
|-----------|-------|--------|--------|-------|-------|
| Series Create Int64 | 1M | 40 ns | 8.4 ms | **Go 211,000x** | Go takes slice ownership; Polars copies across FFI |
| Series Create Float64 | 1M | 43 ns | 5.7 ms | **Go 133,000x** | Same — not a fair engine comparison |
| Series Create String | 1M | 5.7 ms | 45.6 ms | **Go 8.0x** | Go builds offset array; Polars copies + validates UTF-8 |
| DataFrame Create | 1M | 265 ns | 58.2 ms | **Go 220,000x** | Go wraps existing Series; Polars copies column data |
| DataFrame Select | 10K | 314 ns | 10.9 us | **Go 35x** | Go returns column references; Polars clones |
| CumSum | 1M | 806 us | 2.9 ms | **Go 3.6x** | |
| CumMax | 1M | 1.1 ms | 2.9 ms | **Go 2.6x** | |
| Concat Horizontal | 100K | 343 ns | 1.0 us | **Go 2.9x** | |
| Str Contains | 1M | 6.5 ms | 10.9 ms | **Go 1.7x** | |
| Filter Gt90pct | 1M | 158 us | 221 us | **Go 1.4x** | Sparse filter — few rows pass |
| Str Lengths | 1M | 3.4 ms | 3.9 ms | **Go 1.2x** | |

### Near Parity

Operations where performance is within ~2x.

| Benchmark | Scale | Golars | Polars | Ratio |
|-----------|-------|--------|--------|-------|
| Lazy Chain | 100K | 699 us | 701 us | **1.0x** |
| Expr Str Contains | 100K | 1.1 ms | 1.1 ms | **1.0x** |
| DataFrame Sort | 1M | 5.2 ms | 4.9 ms | **1.1x** |
| Expr Arithmetic | 1M | 839 us | 682 us | **1.2x** |
| Expr Comparison | 1M | 518 us | 422 us | **1.2x** |
| JSON Write | 10K | 1.3 ms | 1.0 ms | **1.2x** |
| Unique String | 1M | 8.1 ms | 6.4 ms | **1.3x** |
| Series Max | 1M | 161 us | 104 us | **1.5x** |
| JSON Read | 10K | 2.5 ms | 1.5 ms | **1.6x** |
| Parquet Write | 100K | 9.0 ms | 5.1 ms | **1.8x** |
| Sort Float64 | 1M | 1.6 ms | 826 us | **2.0x** |

### Moderate Gaps

Operations where Polars is 2–5x faster.

| Benchmark | Scale | Golars | Polars | Ratio |
|-----------|-------|--------|--------|-------|
| Series Sum | 1M | 257 us | 106 us | **2.4x** |
| Window Mean Over | 100K | 2.1 ms | 972 us | **2.2x** |
| Str ToUpper | 1M | 43.9 ms | 17.9 ms | **2.4x** |
| Parquet Read | 100K | 2.5 ms | 1.1 ms | **2.4x** |
| Window Sum Over | 100K | 2.3 ms | 891 us | **2.6x** |
| Lazy Filter+Sort | 1M | 4.1 ms | 1.6 ms | **2.6x** |
| Rolling Mean | 1M | 8.8 ms | 3.3 ms | **2.6x** |
| Sort String | 1M | 32.0 ms | 11.2 ms | **2.9x** |
| GroupBy MultiAgg | 100K | 2.1 ms | 731 us | **2.9x** |
| Lazy GroupBy+Agg | 100K | 2.1 ms | 707 us | **3.0x** |
| Filter Gt50pct | 1M | 745 us | 235 us | **3.2x** |
| Join Inner 10% | 100K | 1.8 ms | 558 us | **3.2x** |
| NUnique | 1M | 20.9 ms | 6.1 ms | **3.4x** |
| Filter Expr Eval | 1M | 980 us | 289 us | **3.4x** |
| SortBy MultiCol | 100K | 10.5 ms | 3.1 ms | **3.4x** |
| GroupBy 10K groups | 1M | 19.9 ms | 5.3 ms | **3.7x** |
| Rolling Sum | 1M | 9.0 ms | 2.3 ms | **3.9x** |
| Expr Conditional | 100K | 200 us | 49 us | **4.1x** |
| Pivot | 10K | 16.3 ms | 3.6 ms | **4.6x** |

### Larger Gaps

Operations where Polars is 5x+ faster, generally due to SIMD vectorization, native Rust I/O, or architectural differences.

| Benchmark | Scale | Golars | Polars | Ratio | Primary Cause |
|-----------|-------|--------|--------|-------|---------------|
| Unique Int64 | 1M | 4.5 ms | 823 us | **5.5x** | SIMD hash probing |
| Join Left 10% | 100K | 3.9 ms | 699 us | **5.6x** | SIMD hash tables |
| Join Left 50% | 100K | 6.7 ms | 1.2 ms | **5.8x** | SIMD hash tables |
| GroupBy 100 groups | 1M | 17.7 ms | 3.0 ms | **5.9x** | SIMD hash tables |
| GroupBy 5 groups | 1M | 18.5 ms | 2.8 ms | **6.6x** | SIMD hash tables + multithreading |
| Join Inner 50% | 100K | 5.7 ms | 823 us | **6.9x** | SIMD hash tables |
| CSV Write | 100K | 25.3 ms | 2.7 ms | **9.5x** | Native Rust I/O pipeline |
| CSV Read | 100K | 12.1 ms | 1.1 ms | **10.8x** | Native Rust I/O pipeline |
| Concat Vertical | 100K | 414 us | 34 us | **12.1x** | Zero-copy chunk append vs data copy |

---

## Analysis

### Where Golars Does Well

Golars performs strongly in operations that benefit from Go's zero-copy slice semantics (creation, select, horizontal concat), sequential scan patterns (cumulative operations, string search), and workloads where the overhead of Python FFI erodes Polars' Rust advantage (JSON I/O, lazy chain evaluation).

### Where Polars Excels

Polars has significant advantages in hash-intensive operations (GroupBy, Join, Unique) where SIMD-accelerated hash tables provide 5–7x throughput gains that are difficult to match without hardware-specific instructions. CSV I/O benefits from Polars' fully native Rust pipeline with zero Python involvement. Vertical concatenation is architecturally O(1) in Polars due to its chunked array design, while Golars must copy data.

### Root Causes of Gaps

| Factor | Impact | Affected Operations |
|--------|--------|---------------------|
| **SIMD auto-vectorization** | 2–5x per loop | Hashing, filtering, aggregation |
| **Multi-threaded execution** | 2–4x on multi-core | GroupBy, large scans |
| **Chunked arrays** | O(1) vs O(n) | Concat vertical |
| **Native Rust I/O** | 5–10x | CSV read/write |
| **GC pressure** | 10–30% overhead | Large allocations |

### Reproducing These Benchmarks

```bash
# Go benchmarks
cd bench
go test -bench=. -benchmem -count=3 -timeout 600s

# Polars benchmarks (requires Python 3.12+ and polars)
pip install polars
python3 bench_polars.py
```

Both benchmark suites are included in the `bench/` directory of this repository.
