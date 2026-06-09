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

## Why Arrow? — 底层架构决策分析

### 为什么选择 Apache Arrow 作为后端

godans 的核心设计决策是使用 [Apache Arrow](https://arrow.apache.org/) 的列式内存格式作为底层存储，而不是传统的 Go 切片或自定义结构。这个决策基于以下深层考量：

#### 1. 列式存储 vs 行式存储

```
行式存储 (Go 切片 / 结构体数组):
┌─────────────────────────────────────┐
│ Row 0: name="alice" age=25 score=88 │
│ Row 1: name="bob"   age=30 score=92 │
│ Row 2: name="carol" age=28 score=76 │
└─────────────────────────────────────┘
→ 计算某一列的均值需要跳跃访问内存

列式存储 (Arrow):
┌───────────┬───────────┬───────────┐
│ name      │ age       │ score     │
│ [alice,   │ [25,30,28]│ [88,92,76]│
│  bob,carol]│           │           │
└───────────┴───────────┴───────────┘
→ 计算 score 均值：连续内存，CPU cache 极友好
```

**性能差异**：列式布局在分析场景（聚合、过滤、统计）下比行式快 **3-10 倍**，因为：
- CPU 缓存行（64B）可以一次加载多个同类型值
- 无指针追逐，无间接寻址
- SIMD 向量化友好

#### 2. Null 处理

```go
// Go 切片方案：需要额外一列
type Column struct {
    values []float64
    valid  []bool  // 额外内存分配，额外 cache miss
}

// Arrow 方案：紧凑的 bitmap
// validity bitmap: [1, 0, 1, 1, 0] → 只需 1 bit/值
// 数据区:          [1.5, _, 3.0, 4.2, _] → null 位置不占数据空间
```

Arrow 的 validity bitmap 只需 `n/8` 字节，而 Go 切片方案需要 `n` 字节的 `[]bool`。对于百万行数据，节省 ~87.5% 的 null 标记内存。

#### 3. 零拷贝切片与过滤

```go
// Go 切片：Filter 必须复制数据
func Filter(s []float64, mask []bool) []float64 {
    result := make([]float64, 0)  // 新分配
    for i, v := range s {
        if mask[i] {
            result = append(result, v)  // 逐个复制
        }
    }
    return result
}

// Arrow：切片只需调整 offset/length，零拷贝
sliced := array.NewSlice(arr, start, end)  // O(1)，无内存分配
```

#### 4. 与 Arrow 生态的互操作

godans 使用 Arrow 格式后，可以零成本对接：
- **DuckDB**：直接查询 Arrow 表，无需序列化
- **Polars**：通过 Arrow IPC 交换数据
- **DataFusion**：Rust 查询引擎，Arrow 原生
- **Flight RPC**：跨进程/跨机器零拷贝传输

### Arrow 方案的优缺点

| 维度 | 优点 | 缺点 |
|------|------|------|
| **内存效率** | 列式连续存储，紧凑的 null bitmap | 小数据集有 builder 开销 |
| **计算性能** | CPU cache 友好，SIMD 潜力 | Go 编译器自动向量化能力弱 |
| **零拷贝** | Slice/Filter/IPC 零拷贝 | 需要理解 buffer 生命周期 |
| **互操作** | 零成本对接 DuckDB/Polars/Flight | 与非 Arrow 系统交互需序列化 |
| **类型系统** | 丰富的物理类型 (8/16/32/64位整数/浮点/字符串/时间) | Go 泛型不成熟，需 type switch |
| **GC 压力** | 大块连续内存，GC 扫描快 | 大量小 array 时 GC 压力增大 |
| **调试** | Arrow 提供 String() 方法 | 二进制布局不如切片直观 |
| **依赖** | 纯 Go 实现 (v14+ 无 CGO) | 依赖体积较大 (~10MB) |
| **学习曲线** | 文档完善，社区活跃 | 需理解 builder/array/buffer 三层模型 |

### 与 Go 切片方案的权衡

| 场景 | Go 切片更优 | Arrow 更优 |
|------|------------|-----------|
| 小数据 (<1000行) | ✅ 无构建开销 | |
| 大数据 (>10万行) | | ✅ 列式计算快 3-10x |
| 频繁增删行 | ✅ 切片更灵活 | |
| 聚合/统计/过滤 | | ✅ 连续内存 + SIMD |
| 零依赖部署 | ✅ 无外部依赖 | |
| 与数据生态互操作 | | ✅ DuckDB/Polars/Flight |
| 简单 CRUD | ✅ 直观 | |
| 时间序列/窗口函数 | | ✅ 列式天然适合 |

**结论**：godans 选择 Arrow 是因为目标场景是**数据分析**（聚合、统计、过滤、窗口函数），这正是列式存储的优势领域。如果目标是简单的键值存储或 CRUD，Go 切片会更合适。

---

## Polars 参考 — 可借鉴的设计理念

[Polars](https://pola.rs/) 是用 Rust 实现的高性能 DataFrame 库，被广泛认为是 Pandas 的现代替代品。以下是其核心设计理念和可借鉴之处：

### 1. 惰性执行 (Lazy Evaluation)

```python
# Polars 的惰性 API
result = (
    df.lazy()
    .filter(pl.col("age") > 25)
    .group_by("dept")
    .agg(pl.col("salary").mean())
    .collect()  # 此时才执行，且自动优化
)
```

**核心思想**：所有操作先构建表达式图（expression graph），再统一优化后执行。

**优化策略**：
- **谓词下推 (Predicate Pushdown)**：把过滤条件推到数据源层，减少读取量
- **投影下推 (Projection Pushdown)**：只读取需要的列，减少 I/O
- **公共子表达式消除 (CSE)**：重复计算只执行一次
- **并行化**：自动将独立操作分配到多线程

**godans 可借鉴**：当前 godans 是即时执行（eager），未来可加 `Lazy()` 模式，构建查询计划后优化执行。

### 2. 表达式系统 (Expression System)

```python
# Polars 表达式：声明式、可组合、可优化
df.select([
    pl.col("name").str.to_uppercase(),
    pl.col("age").clip(0, 100),
    (pl.col("salary") / pl.col("hours")).alias("hourly_rate"),
])
```

**核心思想**：表达式不是立即执行的函数调用，而是一个可组合的描述对象。引擎拿到完整表达式后可以：
- 自动并行化独立列的操作
- 推断输出类型
- 合并多次遍历为单次遍历

**godans 可借鉴**：当前 godans 用 Go 闭包实现 `Apply`/`Map`，表达式不可序列化、不可优化。可考虑引入表达式 DSL。

### 3. 多线程并行

Polars 使用 Rust 的 `rayon` 线程池，自动将列操作并行化：
- 每列独立计算，天然可并行
- GroupBy 后的聚合自动分片并行
- Join 使用多线程 hash table 构建

**godans 可借鉴**：Go 的 goroutine 天然适合并行，可在 `DataFrame.Transform`、`GroupBy.Agg` 等操作中自动并行化。

### 4. 内存映射与流式处理

```python
# Polars 流式处理：数据不用全加载到内存
result = (
    pl.scan_parquet("huge_file.parquet")  # 不加载数据
    .filter(pl.col("date") > "2024-01-01")
    .group_by("category")
    .agg(pl.col("value").sum())
    .collect(streaming=True)  # 流式执行
)
```

**godans 可借鉴**：当前 `ReadParquetFile` 全量加载，大数据集会 OOM。可加 `ScanParquet` 惰性扫描。

### 5. 数据类型系统

Polars 的类型比 Pandas 更丰富且严格：

| Polars 类型 | Pandas 对应 | 说明 |
|------------|------------|------|
| `Categorical` | `category` | 有序/无序分类，字典编码 |
| `List(T)` | 无原生支持 | 嵌套列表，每行可不同长度 |
| `Struct` | 无原生支持 | 嵌套结构体 |
| `Array(T, N)` | 无原生支持 | 固定长度数组 |
| `Enum` | 无 | 有序枚举 |
| `Date` / `Datetime` / `Duration` | `datetime64` | 严格的时间类型 |
| `Decimal` | `object` | 精确十进制 |

**godans 可借鉴**：当前缺少 `Categorical`、`List`、`Struct` 类型。这些对数据分析很有用。

### 6. Join 算法

Polars 实现了多种 Join 算法并自动选择：

| 算法 | 适用场景 | 复杂度 |
|------|---------|--------|
| Hash Join | 等值连接，通用 | O(n+m) |
| Sort-Merge Join | 已排序数据 | O(n log n) |
| Cross Join | 笛卡尔积 | O(n×m) |
| Asof Join | 时间序列近似匹配 | O(n log m) |
| Semi Join | 只返回左表匹配行 | O(n+m) |
| Anti Join | 返回左表不匹配行 | O(n+m) |

**godans 可借鉴**：当前只有 Hash Join 思路。可加 Asof Join（时间序列场景非常有用）和 Semi/Anti Join。

### 7. 时间序列支持

Polars 的时间序列功能比 Pandas 更强大：
- `group_by_dynamic`：动态时间窗口分组
- `group_by_rolling`：滚动窗口分组
- `asof_join`：时间序列近似匹配
- 时区感知的 `Datetime` 类型
- `truncate`：时间截断到指定精度

**godans 可借鉴**：已有 `Resample` 和 `Rolling`，但缺少 `group_by_dynamic` 和 `asof_join`。

### 8. SQL 接口

Polars 内置 SQL 引擎，可以直接用 SQL 查询 DataFrame：

```python
df = pl.DataFrame({"a": [1, 2, 3], "b": [4, 5, 6]})
result = pl.sql("SELECT a, SUM(b) FROM df GROUP BY a").collect()
```

**godans 可借鉴**：可集成 DuckDB 的 Go 绑定，提供 SQL 查询接口。

### Polars vs godans 对比

| 特性 | Polars (Rust) | godans (Go) | 差距 |
|------|--------------|-------------|------|
| 惰性执行 | ✅ 核心特性 | ❌ 未实现 | 高优先级 |
| 表达式系统 | ✅ 可组合 | ❌ 用闭包替代 | 中优先级 |
| 多线程并行 | ✅ 自动 | ❌ 单线程 | 高优先级 |
| 流式处理 | ✅ streaming | ❌ 全量加载 | 中优先级 |
| Categorical | ✅ | ❌ | 中优先级 |
| List/Struct | ✅ | ❌ | 低优先级 |
| Asof Join | ✅ | ❌ | 中优先级 |
| SQL 接口 | ✅ | ❌ | 低优先级 |
| Arrow 内存 | ✅ | ✅ | 已持平 |
| 类型安全 | ✅ Rust 类型系统 | ✅ Go 接口 | 已持平 |
| I/O 格式 | CSV/JSON/Parquet/IPC/AVRO | CSV/JSON/Parquet/Excel | 基本持平 |

### 建议的下一步演进路径

```
Phase 1: 性能优化
├── 多线程并行 Transform/Agg (Go goroutine)
├── ScanParquet 惰性扫描
└── Predicate pushdown 到 I/O 层

Phase 2: 表达式系统
├── 表达式 DSL (可序列化、可优化)
├── 惰性执行引擎
└── 自动查询优化

Phase 3: 高级特性
├── Categorical 类型
├── Asof Join
├── GroupBy Dynamic
└── SQL 接口 (集成 DuckDB)
```

Sources:
- [Polars Official Documentation](https://docs.pola.rs/)
- [Polars GitHub](https://github.com/pola-rs/polars)
- [Apache Arrow Go](https://pkg.go.dev/github.com/apache/arrow/go/v18)

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

**197 tests across 3 packages**, all passing.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/apache/arrow/go/v18` | Arrow columnar memory format |
| `github.com/parquet-go/parquet-go` | Parquet file I/O |
| `github.com/xuri/excelize/v2` | Excel (.xlsx) I/O |

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
