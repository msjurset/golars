#!/usr/bin/env python3
"""
Polars benchmark suite for comparison with golars Go benchmarks.

Run:  python bench_polars.py
Deps: pip install polars
"""

import io
import time
import sys

import polars as pl
import numpy as np

# ---------------------------------------------------------------------------
# Data generation (mirrors Go helpers)
# ---------------------------------------------------------------------------

CYCLE_STRINGS = ["alpha", "beta", "gamma", "delta", "epsilon"]


def make_int64(n: int) -> list[int]:
    return list(range(n))


def make_float64(n: int) -> list[float]:
    return [i * 1.1 for i in range(n)]


def make_strings(n: int) -> list[str]:
    return [CYCLE_STRINGS[i % 5] for i in range(n)]


def make_groups(n: int, n_groups: int) -> list[str]:
    return [f"g{i % n_groups}" for i in range(n)]


def make_df(n: int) -> pl.DataFrame:
    return pl.DataFrame(
        {
            "id": make_int64(n),
            "value": make_float64(n),
            "group": make_strings(n),
        }
    )


def make_df_groups(n: int, n_groups: int) -> pl.DataFrame:
    return pl.DataFrame(
        {
            "id": make_int64(n),
            "value": make_float64(n),
            "group": make_groups(n, n_groups),
        }
    )


def make_wide_df(n: int, n_cols: int) -> pl.DataFrame:
    return pl.DataFrame({f"col_{i}": make_float64(n) for i in range(n_cols)})


# ---------------------------------------------------------------------------
# Benchmark runner
# ---------------------------------------------------------------------------

RESULTS: list[tuple[str, int, float]] = []


def bench(name: str, scale: int, fn, iterations: int | None = None):
    """Run fn() multiple times and record median ns/op."""
    if iterations is None:
        # Auto-select iterations based on scale
        if scale >= 10_000_000:
            iterations = 3
        elif scale >= 1_000_000:
            iterations = 5
        elif scale >= 100_000:
            iterations = 10
        elif scale >= 10_000:
            iterations = 30
        else:
            iterations = 100

    # Warmup
    fn()

    times = []
    for _ in range(iterations):
        t0 = time.perf_counter_ns()
        fn()
        t1 = time.perf_counter_ns()
        times.append(t1 - t0)

    times.sort()
    median_ns = times[len(times) // 2]
    RESULTS.append((name, scale, median_ns))


# ---------------------------------------------------------------------------
# Scales
# ---------------------------------------------------------------------------

SCALES = [
    ("1K", 1_000),
    ("10K", 10_000),
    ("100K", 100_000),
    ("1M", 1_000_000),
    ("10M", 10_000_000),
]

SMALL_SCALES = SCALES[:3]  # 1K, 10K, 100K

# ═══════════════════════════════════════════════════════════════════════════
# 1. Series Creation
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_create():
    for label, n in SCALES:
        idata = make_int64(n)
        bench(f"SeriesCreate/Int64/{label}", n, lambda d=idata: pl.Series("x", d))

        fdata = make_float64(n)
        bench(f"SeriesCreate/Float64/{label}", n, lambda d=fdata: pl.Series("x", d))

        sdata = make_strings(n)
        bench(f"SeriesCreate/String/{label}", n, lambda d=sdata: pl.Series("x", d))


# ═══════════════════════════════════════════════════════════════════════════
# 2. Series Aggregation
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_agg():
    for agg_name, method in [("Sum", "sum"), ("Mean", "mean"), ("Min", "min"), ("Max", "max")]:
        for label, n in SCALES:
            s = pl.Series("x", make_float64(n))
            bench(f"SeriesAgg/{agg_name}/{label}", n, lambda s=s, m=method: getattr(s, m)())

    for label, n in SCALES[:4]:
        s = pl.Series("x", make_strings(n))
        bench(f"SeriesNUnique/{label}", n, lambda s=s: s.n_unique())


# ═══════════════════════════════════════════════════════════════════════════
# 3. Series Sort
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_sort():
    for label, n in SCALES[:4]:
        s = pl.Series("x", make_float64(n))
        bench(f"SeriesSort/Float64/Asc/{label}", n, lambda s=s: s.sort())
        bench(f"SeriesSort/Float64/Desc/{label}", n, lambda s=s: s.sort(descending=True))

        ss = pl.Series("x", make_strings(n))
        bench(f"SeriesSort/String/Asc/{label}", n, lambda s=ss: s.sort())


# ═══════════════════════════════════════════════════════════════════════════
# 4. Series Rolling
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_rolling():
    for w in [10, 100]:
        for label, n in SCALES[:4]:
            s = pl.Series("x", make_float64(n))
            bench(
                f"SeriesRollingMean/W{w}/{label}",
                n,
                lambda s=s, w=w: s.rolling_mean(window_size=w),
            )
            bench(
                f"SeriesRollingSum/W{w}/{label}",
                n,
                lambda s=s, w=w: s.rolling_sum(window_size=w),
            )


# ═══════════════════════════════════════════════════════════════════════════
# 5. Series Cumulative
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_cumulative():
    for label, n in SCALES:
        s = pl.Series("x", make_int64(n))
        bench(f"SeriesCumSum/{label}", n, lambda s=s: s.cum_sum())

    for label, n in SCALES[:4]:
        s = pl.Series("x", make_float64(n))
        bench(f"SeriesCumMax/{label}", n, lambda s=s: s.cum_max())


# ═══════════════════════════════════════════════════════════════════════════
# 6. Series String Ops
# ═══════════════════════════════════════════════════════════════════════════


def bench_series_str():
    for label, n in SCALES[:4]:
        s = pl.Series("x", make_strings(n))
        bench(f"SeriesStrContains/{label}", n, lambda s=s: s.str.contains("alp"))
        bench(f"SeriesStrToUpper/{label}", n, lambda s=s: s.str.to_uppercase())
        bench(f"SeriesStrLengths/{label}", n, lambda s=s: s.str.len_chars())


# ═══════════════════════════════════════════════════════════════════════════
# 7. DataFrame Creation
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_create():
    for label, n in SCALES[:4]:
        c1, c2, c3 = make_int64(n), make_float64(n), make_strings(n)
        bench(
            f"DataFrameCreate/3Col/{label}",
            n,
            lambda c1=c1, c2=c2, c3=c3: pl.DataFrame({"id": c1, "value": c2, "group": c3}),
        )

    for width in [10, 50, 100]:
        cols = {f"c{i}": make_float64(10_000) for i in range(width)}
        bench(
            f"DataFrameCreate/{width}Col/10K",
            10_000,
            lambda c=cols: pl.DataFrame(c),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 8. DataFrame Filter
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_filter():
    for label, n in SCALES[:4]:
        df = make_df(n)
        t50 = n * 0.5 * 1.1
        t90 = n * 0.9 * 1.1
        bench(f"DataFrameFilter/Gt50pct/{label}", n, lambda df=df, t=t50: df.filter(pl.col("value") > t))
        bench(f"DataFrameFilter/Gt90pct/{label}", n, lambda df=df, t=t90: df.filter(pl.col("value") > t))

    for label, n in SCALES[:4]:
        df = make_df(n)
        t50 = n * 0.5 * 1.1
        bench(
            f"DataFrameFilterExprEval/{label}",
            n,
            lambda df=df, t=t50: df.filter(pl.col("value") > t),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 9. DataFrame Sort
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_sort():
    for label, n in SCALES[:4]:
        df = make_df(n)
        bench(f"DataFrameSort/SingleCol/{label}", n, lambda df=df: df.sort("value"))

    for label, n in SCALES[:3]:
        df = make_df(n)
        bench(
            f"DataFrameSortBy/MultiCol/{label}",
            n,
            lambda df=df: df.sort(["group", "value"], descending=[False, True]),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 10. DataFrame GroupBy
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_groupby():
    for gc_name, n_groups in [("5grp", 5), ("100grp", 100), ("10Kgrp", 10_000)]:
        for label, n in SCALES[:4]:
            if n_groups > n:
                continue
            df = make_df_groups(n, n_groups)
            bench(
                f"DataFrameGroupBy/Sum/{gc_name}/{label}",
                n,
                lambda df=df: df.group_by("group").agg(pl.col("value").sum()),
            )

    for label, n in SCALES[:3]:
        df = make_df_groups(n, 5)
        bench(
            f"DataFrameGroupBy/MultiAgg/5grp/{label}",
            n,
            lambda df=df: df.group_by("group").agg(
                pl.col("value").sum(), pl.col("id").mean()
            ),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 11. DataFrame Join
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_join():
    for jt_name, how in [("Inner", "inner"), ("Left", "left")]:
        for r_name, frac in [("10pctRight", 0.10), ("50pctRight", 0.50)]:
            for label, n in SMALL_SCALES:
                rn = max(1, int(n * frac))
                left = make_df(n)
                right = pl.DataFrame({"id": make_int64(rn), "rval": make_float64(rn)})
                bench(
                    f"DataFrameJoin/{jt_name}/{r_name}/{label}",
                    n,
                    lambda l=left, r=right, h=how: l.join(r, on="id", how=h),
                )


# ═══════════════════════════════════════════════════════════════════════════
# 12. DataFrame Select / Drop
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_select():
    for width in [10, 50, 100]:
        df = make_wide_df(10_000, width)
        bench(
            f"DataFrameSelect/Select3of{width}/10K",
            10_000,
            lambda df=df: df.select("col_0", "col_1", "col_2"),
        )
        bench(
            f"DataFrameSelect/Drop3of{width}/10K",
            10_000,
            lambda df=df: df.drop("col_0", "col_1", "col_2"),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 13. DataFrame Pivot
# ═══════════════════════════════════════════════════════════════════════════


def bench_df_pivot():
    for ps_name, n, ng in [("1K_5grp", 1000, 5), ("10K_5grp", 10000, 5), ("10K_100grp", 10000, 100)]:
        df = pl.DataFrame(
            {
                "id": make_int64(n),
                "cat": make_groups(n, ng),
                "val": make_float64(n),
            }
        )
        bench(
            f"DataFramePivot/{ps_name}",
            n,
            lambda df=df: df.pivot(on="cat", index="id", values="val"),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 14. Lazy Evaluation
# ═══════════════════════════════════════════════════════════════════════════


def bench_lazy():
    for label, n in SCALES[:4]:
        df = make_df(n)
        t = n * 0.5 * 1.1
        bench(
            f"LazyFilterSort/{label}",
            n,
            lambda df=df, t=t: df.lazy()
            .filter(pl.col("value") > t)
            .sort("value", descending=True)
            .collect(),
        )

    for label, n in SCALES[:3]:
        df = make_df_groups(n, 5)
        bench(
            f"LazyGroupByAgg/{label}",
            n,
            lambda df=df: df.lazy()
            .group_by("group")
            .agg(pl.col("value").sum())
            .collect(),
        )

    for label, n in SCALES[:3]:
        df = make_df_groups(n, 10)
        t = n * 0.3 * 1.1
        bench(
            f"LazyChain/{label}",
            n,
            lambda df=df, t=t: df.lazy()
            .filter(pl.col("value") > t)
            .sort("value")
            .head(100)
            .collect(),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 15. I/O
# ═══════════════════════════════════════════════════════════════════════════


def bench_io():
    for label, n in SMALL_SCALES:
        df = make_df(n)

        bench(f"IOCSVWrite/{label}", n, lambda df=df: df.write_csv(io.BytesIO()))

        csv_bytes = df.write_csv().encode()
        bench(
            f"IOCSVRead/{label}",
            n,
            lambda b=csv_bytes: pl.read_csv(io.BytesIO(b)),
        )

    for label, n in SMALL_SCALES[:2]:
        df = make_df(n)
        bench(f"IOJSONWrite/{label}", n, lambda df=df: df.write_json())

        jb = df.write_json().encode()
        bench(
            f"IOJSONRead/{label}",
            n,
            lambda b=jb: pl.read_json(io.BytesIO(b)),
        )

    for label, n in SMALL_SCALES:
        df = make_df(n)
        bench(f"IOParquetWrite/{label}", n, lambda df=df: df.write_parquet(io.BytesIO()))

        buf = io.BytesIO()
        df.write_parquet(buf)
        pq_bytes = buf.getvalue()
        bench(
            f"IOParquetRead/{label}",
            n,
            lambda b=pq_bytes: pl.read_parquet(io.BytesIO(b)),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 16. Window Functions
# ═══════════════════════════════════════════════════════════════════════════


def bench_window():
    for gc_name, ng in [("5grp", 5), ("100grp", 100)]:
        for label, n in SCALES[:3]:
            df = make_df_groups(n, ng)
            bench(
                f"WindowMeanOver/{gc_name}/{label}",
                n,
                lambda df=df: df.with_columns(pl.col("value").mean().over("group")),
            )

    for label, n in SCALES[:3]:
        df = make_df_groups(n, 5)
        bench(
            f"WindowSumOver/5grp/{label}",
            n,
            lambda df=df: df.with_columns(pl.col("value").sum().over("group")),
        )


# ═══════════════════════════════════════════════════════════════════════════
# 17. Expression Evaluation
# ═══════════════════════════════════════════════════════════════════════════


def bench_expr():
    for label, n in SCALES[:4]:
        df = pl.DataFrame({"a": make_float64(n), "b": make_float64(n)})
        bench(
            f"ExprArithmetic/AddMul/{label}",
            n,
            lambda df=df: df.select((pl.col("a") + pl.col("b")) * 2.0),
        )

    for label, n in SCALES[:4]:
        df = make_df(n)
        lo = n * 0.25 * 1.1
        hi = n * 0.75 * 1.1
        bench(
            f"ExprComparison/GtAndLt/{label}",
            n,
            lambda df=df, lo=lo, hi=hi: df.select(
                (pl.col("value") > lo) & (pl.col("value") < hi)
            ),
        )

    for label, n in SCALES[:3]:
        df = make_df(n)
        t = n * 0.5 * 1.1
        bench(
            f"ExprConditional/WhenThenOtherwise/{label}",
            n,
            lambda df=df, t=t: df.select(
                pl.when(pl.col("value") > t).then(1).otherwise(0)
            ),
        )

    for label, n in SCALES[:3]:
        df = make_df(n)
        bench(
            f"ExprStrContains/{label}",
            n,
            lambda df=df: df.select(pl.col("group").str.contains("alp")),
        )


# ═══════════════════════════════════════════════════════════════════════════
# Bonus: Unique
# ═══════════════════════════════════════════════════════════════════════════


def bench_unique():
    for label, n in SCALES[:4]:
        s = pl.Series("x", make_strings(n))
        bench(f"SeriesUnique/String/{label}", n, lambda s=s: s.unique())

        si = pl.Series("x", make_int64(n))
        bench(f"SeriesUnique/Int64/{label}", n, lambda s=si: s.unique())


# ═══════════════════════════════════════════════════════════════════════════
# Bonus: Concat
# ═══════════════════════════════════════════════════════════════════════════


def bench_concat():
    for label, n in SCALES[:3]:
        df1 = make_df(n)
        df2 = make_df(n)
        bench(f"ConcatVertical/{label}", n, lambda d1=df1, d2=df2: pl.concat([d1, d2]))

        a = pl.DataFrame({"a": make_float64(n)})
        b_df = pl.DataFrame({"b": make_float64(n)})
        bench(
            f"ConcatHorizontal/{label}",
            n,
            lambda a=a, b=b_df: pl.concat([a, b], how="horizontal"),
        )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    print(f"Polars version: {pl.__version__}")
    print(f"Python version: {sys.version.split()[0]}")
    print()

    runners = [
        ("Series Creation", bench_series_create),
        ("Series Aggregation", bench_series_agg),
        ("Series Sort", bench_series_sort),
        ("Series Rolling", bench_series_rolling),
        ("Series Cumulative", bench_series_cumulative),
        ("Series String Ops", bench_series_str),
        ("DataFrame Creation", bench_df_create),
        ("DataFrame Filter", bench_df_filter),
        ("DataFrame Sort", bench_df_sort),
        ("DataFrame GroupBy", bench_df_groupby),
        ("DataFrame Join", bench_df_join),
        ("DataFrame Select/Drop", bench_df_select),
        ("DataFrame Pivot", bench_df_pivot),
        ("Lazy Evaluation", bench_lazy),
        ("I/O", bench_io),
        ("Window Functions", bench_window),
        ("Expressions", bench_expr),
        ("Series Unique", bench_unique),
        ("Concat", bench_concat),
    ]

    for section, fn in runners:
        print(f"Running {section}...", flush=True)
        fn()

    # Print results
    print()
    print(f"{'Benchmark':<60} {'Scale':>8} {'ns/op':>14}")
    print("-" * 84)
    for name, scale, ns in RESULTS:
        ns_str = f"{ns:>14,}"
        print(f"{name:<60} {scale:>8,} {ns_str}")

    print(f"\nTotal benchmarks: {len(RESULTS)}")


if __name__ == "__main__":
    main()
