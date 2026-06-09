# godans

**Pandas for Go** — A high-performance DataFrame library built on Apache Arrow.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-180%20passed-brightgreen)](#testing)

godans brings the familiar pandas API to Go, using Apache Arrow's columnar memory format as the backend for zero-copy, cache-friendly data processing.

---

## Features at a Glance

| Category | Features | Coverage |
|----------|----------|----------|
| Data Structures | Series, DataFrame, Index (Range/Int64/String/DateTime) | ✅ 100% |
| I/O | CSV, JSON, NDJSON, Parquet | ✅ 80% |
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

**Overall: ~95% coverage (81/85 pandas features), 180 unit tests, 42 Go files, ~7000 LOC**

---

## Installation

```bash
go get github.com/godans/godans
```

---

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/godans/godans/backend/arrow"
    "github.com/godans/godans/core"
)

func main() {
    // Create a DataFrame from Series
    name := arrow.NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil)
    age := arrow.NewInt64Series("age", []int64{25, 30, 35}, nil)
    score := arrow.NewFloat64Series("score", []float64{88.5, 92.0, 76.5}, nil)
    df := arrow.NewDataFrame(name, age, score)

    fmt.Println(df.Info())
    // DataFrame: 3 rows x 3 columns
    // Column          DType      Non-Null
    // name            string     3
    // age             int64      3
    // score           float64    3

    // Filter rows
    filtered := df.Query("age > 26")
    fmt.Println(filtered.Col("name").String(0)) // "bob"

    // GroupBy + Agg
    dept := arrow.NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil)
    salary := arrow.NewFloat64Series("salary", []float64{100, 120, 80}, nil)
    df2 := arrow.NewDataFrame(dept, salary)
    result := df2.Agg([]string{"dept"}, map[string]core.AggFunc{"salary": core.AggMean})
    fmt.Println(result.ToCSV())
    // dept,salary_mean
    // eng,110
    // sales,80
}
```

### Read from CSV

```go
df, err := io.ReadCSVFile("data.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Println(df.Describe().ToCSV())
```

### Read from JSON

```go
data := `[{"name":"alice","age":25},{"name":"bob","age":30}]`
df, err := io.ReadJSON(data)
```

### Read from Parquet

```go
df, err := io.ReadParquetFile("data.parquet")
```

---

## Architecture

```
godans/
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
│   ├── dataframe.go         # ArrowDataFrame: core DataFrame impl
│   ├── builder.go           # SeriesBuilder: incremental construction
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
│   └── extras.go            # Pivot, Melt, Query, Clip, Cut, GetDummies, ...
│
└── io/                      # I/O layer
    ├── csv.go               # CSV read/write + type inference
    ├── json.go              # JSON/NDJSON read/write
    └── parquet.go           # Parquet read/write (parquet-go)
```

### Design Principles

1. **Interface-first**: `core/` defines pure Go interfaces (`Series`, `DataFrame`, `Index`), backend-agnostic
2. **Arrow-native**: `backend/arrow/` implements everything using Apache Arrow columnar arrays
3. **Zero-copy where possible**: Arrow's memory model enables zero-copy slicing, filtering, and IPC
4. **Null-aware**: Every operation propagates nulls correctly via Arrow's validity bitmap
5. **Go-idiomatic**: Uses Go closures for `Apply`/`Map` instead of string expressions

---

## API Reference

### Series

| Operation | Method | File | Line |
|-----------|--------|------|------|
| **Create** | `NewInt64Series`, `NewFloat64Series`, `NewStringSeries`, `NewBoolSeries` | `builder.go` | 153-197 |
| **Create with nulls** | `NewInt64SeriesWithNulls`, `NewSeriesBuilder` | `builder.go` | 197, 20 |
| **Timestamp** | `NewTimestampSeries`, `DateRange`, `DateRangeEnd` | `resample.go` | 287-315 |
| **Access** | `Name()`, `Len()`, `Dtype()`, `Index()`, `NullCount()` | `series.go` | 28-31 |
| **Element** | `Bool(i)`, `Int(i)`, `Float(i)`, `String(i)`, `IsNull(i)` | `series.go` | 39-115 |
| **Slice** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` | `series.go` | 117-197 |
| **Copy** | `Copy()`, `SetName(name)`, `ToSlice()` | `series.go` | 198-228 |

### Arithmetic (`arithmetic.go`)

| Operation | Method | Description |
|-----------|--------|-------------|
| `Add` / `AddScalar` | `s.Add(other)` | Element-wise addition |
| `Sub` / `SubScalar` | `s.Sub(other)` | Element-wise subtraction |
| `Mul` / `MulScalar` | `s.Mul(other)` | Element-wise multiplication |
| `Div` / `DivScalar` | `s.Div(other)` | Element-wise division (NaN on /0) |
| `Mod` | `s.Mod(other)` | Modulo |
| `Neg` / `Abs` | `s.Neg()` | Negation / Absolute value |
| `Eq` `Ne` `Lt` `Le` `Gt` `Ge` | `s.Gt(other)` | Comparison → bool Series |
| `And` `Or` `Not` | `s.And(other)` | Logical operations |

### Statistics (`stats.go`, `extras.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| `NLargest(n)` | Top N values | `stats.go:13` |
| `NSmallest(n)` | Bottom N values | `stats.go:22` |
| `Rank(method)` | Ranking (average/min/max/first/dense) | `stats.go:57` |
| `Quantile(p)` | Quantile value | `stats.go:156` |
| `Corr(a, b)` | Pearson correlation | `stats.go:191` |
| `Cov(a, b)` | Covariance | `stats.go:196` |
| `CorrMatrix(df)` | Full correlation matrix | `stats.go:257` |
| `Skew()` | Skewness | `extras.go:581` |
| `Kurt()` | Excess kurtosis | `extras.go:607` |
| `Mode()` | Most frequent value(s) | `extras.go:510` |

### DataFrame

| Operation | Method | File:Line |
|-----------|--------|-----------|
| **Create** | `NewDataFrame(series...)` | `dataframe.go:20` |
| **Shape** | `Shape()`, `Len()`, `Columns()`, `Dtypes()` | `dataframe.go:35-63` |
| **Column** | `Col(name)`, `SelectCols(names)`, `DropCols(names)` | `dataframe.go:76-103` |
| **Row** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` | `dataframe.go:105-143` |
| **Info** | `Info()`, `Describe()`, `MemoryUsage()` | `dataframe.go:145-167`, `extras.go:635` |
| **Mutate** | `WithColumn(name,s)`, `Rename(mapping)`, `SetIndex(name)`, `ResetIndex()` | `dataframe.go:188-282` |
| **Clean** | `DropNA()`, `FillNA(value)`, `FillNAMethod("ffill")` | `dataframe.go:204-236`, `fill.go:172` |
| **Sort** | `SortBy(names, ascending)` | `dataframe.go:284` |
| **Merge** | `MergeOn(other, on, how)`, `Join(other, how)` | `dataframe.go:298-309` |
| **Concat** | `Concat(dfs, mode)` | `concat.go:18` |
| **GroupBy** | `GroupByGroups(names)`, `Agg(groupCols, aggs)` | `dataframe.go:311-385` |
| **GroupBy ext** | `GroupByTransform`, `GroupByFilter`, `GroupByApply` | `dataframe.go:423-492` |
| **I/O** | `ToCSV()` | `dataframe.go:387` |

### Selection & Cleaning (`selection.go`, `extras.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| `IsIn(values)` | Value matching | `selection.go:14` (Series), `268` (DataFrame) |
| `ValueCounts()` | Count occurrences | `selection.go:56` |
| `NUnique()` / `Unique()` | Unique values | `selection.go:115`, `138` |
| `Duplicated(keep)` / `DropDuplicates(keep)` | Dedup | `selection.go:188`, `254` |
| `Clip(lo, hi)` | Clamp values | `extras.go:18` |
| `ConvertDtypes()` | Auto-infer types | `extras.go:42` |
| `Query(expr)` | String expression filter | `extras.go:104` |

### Transform (`extras.go`, `apply.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| `Apply(fn)` | Generic value transform | `apply.go:27` |
| `MapFloat(fn)` / `MapString(fn)` / `MapBool(fn)` | Typed map | `apply.go:61-108` |
| `Transform(fn)` | Numeric column transform | `apply.go:131` |
| `ApplyRows(fn)` | Row-wise transform | `apply.go:146` |
| `Pipe(fn)` | Function chaining | `extras.go:669` |
| `Pivot(index, cols, vals)` | Wide format | `extras.go:177` |
| `PivotTable(...)` | Pivot with aggregation | `extras.go:245` |
| `Melt(idVars, valueVars)` | Long format | `extras.go:312` |
| `GetDummies(df, cols)` | One-hot encoding | `extras.go:676` |
| `Cut(s, bins)` / `QCut(s, q)` | Binning | `extras.go:707`, `745` |
| `Explode(sep)` | Split list to rows | `extras.go:784` |
| `Factorize(s)` | Integer encoding | `extras.go:817` |

### Fill & Type (`fill.go`, `astype.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| `FillForward()` | Forward fill (ffill) | `fill.go:10` |
| `FillBackward()` | Backward fill (bfill) | `fill.go:54` |
| `Interpolate()` | Linear interpolation | `fill.go:110` |
| `AsType(dt)` | Type conversion | `astype.go:13` |
| `ToNumeric(s)` | String → float64 | `astype.go:120` |

### Time Series (`dt_accessor.go`, `resample.go`, `time_arithmetic.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| **.dt accessor** | `s.DT().Year()`, `.Month()`, `.Day()`, `.Hour()`, `.Minute()`, `.Second()` | `dt_accessor.go:21-26` |
| | `.DayOfWeek()`, `.DayOfYear()`, `.Quarter()`, `.Week()`, `.Unix()` | `dt_accessor.go:27-31` |
| | `.Date()`, `.Time()` | `dt_accessor.go:33-39` |
| | `.Floor(d)`, `.Ceil(d)`, `.Round(d)` | `dt_accessor.go:41-59` |
| **Shift** | `s.Shift(n)` — lag/lead | `dt_accessor.go:133` |
| **Change** | `s.PctChange(periods)`, `s.Diff(periods)` | `dt_accessor.go:153`, `179` |
| **Cumulative** | `s.CumSum()`, `CumProd()`, `CumMax()`, `CumMin()` | `dt_accessor.go:199-277` |
| **Resample** | `ResampleDataFrame(df, timeCol, rule)` | `resample.go:151` |
| **TZ** | `TzLocalize(s, loc)`, `TzConvert(s, loc)` | `resample.go:233`, `252` |
| **Filter** | `BetweenTime(s, start, end)`, `AtTime(s, h, m)` | `resample.go:257`, `272` |
| **TA()** | `s.TA().AddDuration(d)`, `.SubDuration(d)`, `.SubTimestamps(other)` | `time_arithmetic.go:35-76` |
| | `.DurationAdd/Sub/Mul/Div(other)`, `.Before(other)`, `.After(other)` | `time_arithmetic.go:97-175` |
| | `.ToDays()`, `.ToHours()`, `.ToMinutes()`, `.ToSeconds()` | `time_arithmetic.go:180-202` |

### Windows (`rolling.go`)

| Operation | Method | File:Line |
|-----------|--------|-----------|
| **Rolling** | `s.Rolling(window).Mean()`, `.Sum()`, `.Std()`, `.Min()`, `.Max()` | `rolling.go:30-95` |
| | `.Count()`, `.Median()`, `.Var()`, `.Apply(fn)` | `rolling.go:96-135` |
| | `.MinPeriods(n)` — minimum observations | `rolling.go:25` |
| **Expanding** | `s.Expanding().Mean()`, `.Sum()`, `.Min()`, `.Max()`, `.Std()` | `rolling.go:189-243` |
| **EWM** | `s.EWMAlpha(alpha).Mean()`, `s.EWMSpan(span).Mean()` | `rolling.go:278-315` |
| **GroupBy** | `df.GroupByRolling(groupCol, valCol, window, fn)` | `rolling.go:321` |

### String Accessor (`str_accessor.go`)

Access via `s.Str().Method()`:

| Category | Methods |
|----------|---------|
| Case | `Upper()`, `Lower()`, `Title()`, `SwapCase()` |
| Trim | `Strip()`, `LStrip()`, `RStrip()` |
| Search | `Contains(pattern)`, `StartsWith(prefix)`, `EndsWith(suffix)` |
| Replace | `Replace(old, new)`, `ReplaceRegex(pattern, repl)` |
| Split | `Split(sep, n)`, `Extract(pattern)`, `ExtractAll(pattern)` |
| Info | `Len()`, `Count(pattern)`, `Cat(sep)` |
| Pad | `Pad(width, char, side)`, `ZFill(width)` |
| Test | `IsNumeric()`, `IsAlpha()`, `IsDigit()`, `IsEmpty()` |

### I/O (`io/`)

| Format | Read | Write | File |
|--------|------|-------|------|
| CSV | `ReadCSV(data)`, `ReadCSVFile(path)` | `WriteCSV(df)`, `WriteCSVFile(df, path)` | `csv.go` |
| JSON | `ReadJSON(data)`, `ReadJSONFile(path)` | `WriteJSON(df)`, `WriteJSONFile(df, path)` | `json.go` |
| NDJSON | `ReadJSONLines(data)`, `ReadJSONLinesFile(path)` | `WriteJSONLines(df)`, `WriteJSONLinesFile(df, path)` | `json.go` |
| Parquet | `ReadParquetFile(path)` | `WriteParquetFile(df, path)` | `parquet.go` |

All readers include **automatic type inference** (int64/float64/bool/string) with 80% threshold over 100-row sample.

---

## Why Arrow?

| Aspect | Go Slices | Apache Arrow (godans) |
|--------|-----------|----------------------|
| Memory layout | Row-oriented, scattered | Columnar, contiguous |
| Null handling | Extra `[]bool` bitmap | Native validity bitmap |
| Cache efficiency | Poor for column ops | CPU-cache friendly |
| Zero-copy slice | Impossible | Native `array.NewSlice` |
| SIMD potential | None | Arrow Compute integration |
| IPC / Serialization | Full copy | Zero-copy via Arrow IPC |
| Ecosystem | Isolated | DuckDB, Polars, DataFusion, Flight |

---

## Testing

```bash
# Run all tests
go test ./... -count=1

# Run with verbose output
go test ./... -v -count=1

# Run specific package
go test ./backend/arrow/ -v -run "TestRolling"
```

**180 tests across 3 packages**, all passing.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/apache/arrow/go/v18` | Arrow columnar memory format |
| `github.com/parquet-go/parquet-go` | Parquet file I/O |

No CGO required. Pure Go.

---

## Roadmap

See [plan.md](plan.md) for the complete feature checklist.

**Remaining (~5%):**
- Excel I/O (requires `excelize`)
- MultiIndex (multi-level row/column index)
- `.plot()` visualization (requires `gonum/plot`)
- Categorical / Sparse types

---

## License

MIT
