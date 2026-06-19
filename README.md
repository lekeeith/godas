# godas

**Pandas for Go** — A high-performance DataFrame library built on Apache Arrow.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-197%20passed-brightgreen)](#testing)

godas brings the familiar pandas API to Go, using Apache Arrow's columnar memory format as the backend for zero-copy, cache-friendly data processing.

---

## Features at a Glance

| Category | Features | Coverage |
|----------|----------|----------|
| Data Structures | Series, DataFrame, Index (Range/Int64/String/DateTime) | ✅ 100% |
| I/O | CSV, JSON, NDJSON, Parquet, **ScanCSV (lazy/streaming)** | ✅ 90% |
| Selection | Col, Slice, Filter, IsIn, Query, Head/Tail | ✅ 87% |
| Cleaning | DropNA, FillNA (ffill/bfill/interpolate), Clip, Duplicated | ✅ 90% |
| Transform | Apply, Map, Pivot, Melt, GetDummies, Cut/QCut, Explode | ✅ 100% |
| Merge | MergeOn (inner/left/right/outer), Join, Concat, Compare | ✅ 83% |
| GroupBy | Agg (11 funcs), Transform, Filter, Apply, Rolling, Resample | ✅ 100% |
| Time Series | .dt accessor, Shift, Diff, CumSum, Resample, TZ, TA() | ✅ 100% |
| Statistics | Describe, Rank, Quantile, Corr/Cov, Skew/Kurt, Mode | ✅ 100% |
| Arithmetic | Add/Sub/Mul/Div/Mod, Comparison, Logic, Type Promotion | ✅ 100% |
| Strings (.str) | 30+ methods (Contains, Replace, Split, Extract, ...) | ✅ 100% |
| Windows | Rolling, Expanding, EWM (Mean/Sum/Std/Min/Max/Median/Var) | ✅ 100% |
| **Performance** | **Arrow SIMD Filter, zero-copy Slice, streaming ScanCSV** | ✅ NEW |

**Overall: ~95% coverage (81/85 pandas features), 197 unit tests, 42 Go files, ~7000 LOC**

---

## Installation

```bash
go get github.com/lekeeith/godas
```

---

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/lekeeith/godas/backend/arrow"
    "github.com/lekeeith/godas/core"
)

func main() {
    // Create a DataFrame from Series
    name := arrow.NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil)
    age := arrow.NewInt64Series("age", []int64{25, 30, 35}, nil)
    score := arrow.NewFloat64Series("score", []float64{88.5, 92.0, 76.5}, nil)
    df := arrow.NewDataFrame(name, age, score)

    // fmt.Println 自动输出带边框的表格
    fmt.Println(df)
    // ┌─────┬─────────┬──────┬───────┐
    // │     │ name    │ age  │ score │
    // ├─────┼─·······─┼─····─┼─·····─┤
    // │   0 │ alice   │ 25   │ 88.5  │
    // │   1 │ bob     │ 30   │ 92    │
    // │   2 │ charlie │ 35   │ 76.5  │
    // └─────┴─────────┴──────┴───────┘

    // Info() 和 Describe() 也输出带边框的表格
    fmt.Println(df.Info())
    fmt.Println(df.Describe())

    // Filter rows
    mask := make([]bool, df.Len())
    for i := 0; i < df.Len(); i++ {
        mask[i] = age.Int(i) > 26
    }
    filtered := df.Filter(mask)

    // GroupBy + Agg
    dept := arrow.NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil)
    salary := arrow.NewFloat64Series("salary", []float64{100, 120, 80}, nil)
    df2 := arrow.NewDataFrame(dept, salary)
    result := df2.Agg([]string{"dept"}, map[string]core.AggFunc{"salary": core.AggMean})
    fmt.Println(result)
}
```

### Read from CSV

```go
df, err := io.ReadCSVFile("data.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Println(df)
```

### ScanCSV — Lazy Scanning for Large Files

For large CSV files (GB+), use `ScanCSV` for streaming with predicate/projection pushdown:

```go
// Collect: load all matched rows into memory
df, err := io.ScanCSVFile("huge.csv").
    Filter("citycode", "==", "320600").
    Filter("fre", "==", "hourly").
    Select("time", "citycode", "poll", "value").
    Collect()

// ForEach: stream in chunks, callback per chunk (memory = O(chunkSize × cols))
processed, err := io.ScanCSVFile("huge.csv").
    Filter("citycode", "==", "320600").
    ForEach(10000, func(chunk *arrow.ArrowDataFrame) error {
        // process chunk (e.g., insert to DB)
        return db.BatchInsert(chunk)
    })

// Resume after error: skip already-processed rows
scan.Offset(processed).ForEach(10000, insertFn)
```

### Read from JSON / Parquet

```go
df, err := io.ReadJSON(`[{"name":"alice","age":25},{"name":"bob","age":30}]`)
df, err := io.ReadParquetFile("data.parquet")
```

### Write to File

```go
df.WriteJSONFile("output.json")       // JSON array
df.WriteJSONLinesFile("output.ndjson") // NDJSON (one object per line)
io.WriteCSVFile(df, "output.csv")
```

---

## Performance

### Filter Optimization (3-layer)

`DataFrame.Filter` uses three optimization layers:

| Layer | When | Complexity |
|-------|------|-----------|
| Zero-copy Slice | Contiguous indices | O(1), no data copy |
| Arrow SIMD | Large non-contiguous | C++ vectorized filter |
| Manual Take | Fallback | Direct `a.Value(idx)` |

### Benchmark Results

```
cpu: 13th Gen Intel(R) Core(TM) i5-13600KF

BenchmarkFilter/rows=10000        508 μs/op     120 KB/op
BenchmarkFilter/rows=100000      4792 μs/op     787 KB/op
BenchmarkFilter/rows=500000     24057 μs/op    3834 KB/op

BenchmarkTakeContiguous/rows=10000      2.4 μs/op    4.8 KB/op  ← O(1) zero-copy
BenchmarkTakeContiguous/rows=100000     2.0 μs/op    4.8 KB/op
BenchmarkTakeContiguous/rows=500000     1.8 μs/op    4.8 KB/op

BenchmarkTakeScattered/rows=10000     107 μs/op    153 KB/op
BenchmarkTakeScattered/rows=100000    982 μs/op   1352 KB/op
BenchmarkTakeScattered/rows=500000   5726 μs/op   7472 KB/op
```

### ScanCSV Performance (3.67M rows, 123 columns)

| Method | Time | Memory |
|--------|------|--------|
| `ReadCSVFile` + Filter | 42s | O(all data) |
| `ScanCSV.Collect` | 8s | O(matched rows) |
| `ScanCSV.ForEach` | 7.8s | O(chunkSize) |

Run benchmarks:

```bash
go test ./backend/arrow/ -bench=. -benchmem -count=1
```

---

## Architecture

```
godas/
├── core/                    # Interface layer (backend-agnostic)
│   ├── dtype.go             # DType enum + helpers
│   ├── index.go             # Index interface (Range/Int64/String/DateTime)
│   ├── series.go            # Series interface
│   ├── dataframe.go         # DataFrame interface
│   ├── types.go             # JoinType, AggFunc enums
│   ├── arithmetic.go        # Arithmetic/Comparison/Logic interfaces + PromoteDType
│   └── dt_accessor.go       # DateTimeAccessor interface + ResampleRule
│
├── backend/arrow/           # Arrow implementation (18 source files)
│   ├── series.go            # ArrowSeries: core Series impl
│   ├── dataframe.go         # ArrowDataFrame: core DataFrame impl + String()/Fmt()/Display()
│   ├── builder.go           # SeriesBuilder: incremental construction
│   ├── benchmark_test.go    # Performance benchmarks
│   ├── ops.go               # Sort, merge, groupby internals
│   ├── utils.go             # DType ↔ Arrow type conversion
│   ├── arithmetic.go        # Add/Sub/Mul/Div/Mod, comparison, logic
│   ├── time_arithmetic.go   # TA(): timestamp + duration operations
│   ├── dt_accessor.go       # .dt accessor (Year/Month/Day/...)
│   ├── resample.go          # Resample, Shift, Diff, CumSum, DateRange
│   ├── apply.go             # Apply/Map/Transform/ApplyRows
│   ├── selection.go         # IsIn, ValueCounts, Duplicated, Unique
│   ├── concat.go            # Concat (rows/cols)
│   ├── fill.go              # ffill, bfill, interpolate
│   ├── astype.go            # AsType, ToNumeric
│   ├── str_accessor.go      # .str accessor (30+ string methods)
│   ├── stats.go             # Rank, Quantile, Corr/Cov, NLargest/NSmallest
│   ├── rolling.go           # Rolling, Expanding, EWM windows
│   ├── extras.go            # Pivot, Melt, Query, Clip, Cut, GetDummies, ...
│   ├── parallel.go          # ParallelFilter, ParallelAgg
│   ├── lazy.go              # LazyFrame (query plan builder)
│   └── expr.go              # Expression DSL
│
└── io/                      # I/O layer
    ├── csv.go               # CSV read/write + type inference
    ├── json.go              # JSON/NDJSON read/write
    ├── parquet.go           # Parquet read/write (parquet-go)
    ├── excel.go             # Excel (.xlsx) read/write
    ├── scan.go              # ScanCSV (lazy/streaming) + ScanParquet
    └── database.go          # Database connector
```

---

## Why Arrow?

### Columnar vs Row Storage

```
Row storage (Go slices):
Row 0: name="alice" age=25 score=88  → scattered memory access
Row 1: name="bob"   age=30 score=92

Column storage (Arrow):
name:  [alice, bob, carol]   → contiguous, CPU cache friendly
age:   [25, 30, 28]          → SIMD vectorizable
score: [88, 92, 76]          → zero-copy slice
```

**3-10x faster** for analytics (aggregation, filter, statistics).

### Null Handling

Arrow's validity bitmap: `n/8` bytes vs Go's `[]bool` = `n` bytes. **87.5% memory savings** for null markers.

### Zero-Copy

```go
sliced := array.NewSlice(arr, start, end)  // O(1), no data copy
filtered := compute.Filter(arr, mask)       // SIMD-accelerated
```

---

## API Reference

### DataFrame

| Operation | Method |
|-----------|--------|
| **Create** | `NewDataFrame(series...)` |
| **Shape** | `Shape()`, `Len()`, `Columns()`, `Dtypes()` |
| **Column** | `Col(name)`, `SelectCols(names)`, `DropCols(names)` |
| **Row** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` |
| **Display** | `String()`, `Fmt()`, `Display(top,bottom)`, `Info()`, `Describe()` |
| **Mutate** | `WithColumn(name,s)`, `Rename(mapping)`, `SetIndex(name)`, `ResetIndex()` |
| **Clean** | `DropNA()`, `FillNA(value)`, `FillNAMethod("ffill")` |
| **Sort** | `SortBy(names, ascending)` |
| **Merge** | `MergeOn(other, on, how)`, `Join(other, how)` |
| **GroupBy** | `GroupByGroups(names)`, `Agg(groupCols, aggs)` |
| **I/O** | `ToCSV()`, `WriteJSONFile(path)`, `WriteJSONLinesFile(path)` |

### Series

| Operation | Method |
|-----------|--------|
| **Create** | `NewInt64Series`, `NewFloat64Series`, `NewStringSeries`, `NewBoolSeries` |
| **Access** | `Name()`, `Len()`, `Dtype()`, `Index()`, `NullCount()` |
| **Element** | `Bool(i)`, `Int(i)`, `Float(i)`, `String(i)`, `IsNull(i)` |
| **Slice** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` |
| **Display** | `String()`, `Fmt()`, `Display(top,bottom)` |

### ScanCSV (Lazy Scanner)

| Method | Description |
|--------|-------------|
| `ScanCSVFile(path)` | Create lazy scanner |
| `Filter(col, op, val)` | Predicate pushdown (==, !=, >, <, >=, <=) |
| `Select(columns...)` | Projection pushdown |
| `Limit(n)` | Limit rows |
| `Offset(n)` | Skip first N matched rows (resume) |
| `Collect()` | Execute and return DataFrame |
| `ForEach(chunkSize, fn)` | Stream in chunks, callback per chunk |

### I/O

| Format | Read | Write |
|--------|------|-------|
| CSV | `ReadCSV(data)`, `ReadCSVFile(path)` | `WriteCSV(df)`, `WriteCSVFile(df, path)` |
| CSV Streaming | `ScanCSVFile(path).Filter(...).Collect()` | — |
| CSV Chunked | `ScanCSVFile(path).Filter(...).ForEach(n, fn)` | — |
| JSON | `ReadJSON(data)`, `ReadJSONFile(path)` | `WriteJSON(df)`, `WriteJSONFile(df, path)` |
| NDJSON | `ReadJSONLines(data)`, `ReadJSONLinesFile(path)` | `WriteJSONLines(df)`, `WriteJSONLinesFile(df, path)` |
| Parquet | `ReadParquetFile(path)` | `WriteParquetFile(df, path)` |
| Excel | `ReadExcelFile(path, sheet)` | `WriteExcelFile(df, path)` |

---

## Testing

```bash
# Run all tests
go test ./... -count=1

# Run benchmarks
go test ./backend/arrow/ -bench=. -benchmem -count=1

# Run specific package
go test ./backend/arrow/ -v -run "TestFilter"
```

**197 tests across 3 packages**, all passing.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/apache/arrow/go/v18` | Arrow columnar memory format + compute SIMD |
| `github.com/parquet-go/parquet-go` | Parquet file I/O |
| `github.com/xuri/excelize/v2` | Excel (.xlsx) I/O |

No CGO required. Pure Go.

---

## License

MIT
