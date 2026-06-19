# godas

**Go 语言的 Pandas** — 基于 Apache Arrow 的高性能 DataFrame 库。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-197%20passed-brightgreen)](#测试)

godas 将熟悉的 pandas API 带入 Go，使用 Apache Arrow 列式内存格式作为后端，实现零拷贝、缓存友好的数据处理。

[English](README.md) | 中文

---

## 功能一览

| 类别 | 功能 | 覆盖率 |
|------|------|--------|
| 数据结构 | Series, DataFrame, Index (Range/Int64/String/DateTime) | ✅ 100% |
| I/O | CSV, JSON, NDJSON, Parquet, **ScanCSV (惰性/流式)** | ✅ 90% |
| 选择 | Col, Slice, Filter, IsIn, Query, Head/Tail | ✅ 87% |
| 清洗 | DropNA, FillNA (ffill/bfill/interpolate), Clip, Duplicated | ✅ 90% |
| 变换 | Apply, Map, Pivot, Melt, GetDummies, Cut/QCut, Explode | ✅ 100% |
| 合并 | MergeOn (inner/left/right/outer), Join, Concat, Compare | ✅ 83% |
| 分组聚合 | Agg (11 种聚合), Transform, Filter, Apply, Rolling, Resample | ✅ 100% |
| 时间序列 | .dt 访问器, Shift, Diff, CumSum, Resample, TZ, TA() | ✅ 100% |
| 统计 | Describe, Rank, Quantile, Corr/Cov, Skew/Kurt, Mode | ✅ 100% |
| 算术 | Add/Sub/Mul/Div/Mod, 比较, 逻辑, 类型提升 | ✅ 100% |
| 字符串 (.str) | 30+ 方法 (Contains, Replace, Split, Extract, ...) | ✅ 100% |
| 窗口函数 | Rolling, Expanding, EWM (Mean/Sum/Std/Min/Max/Median/Var) | ✅ 100% |
| **性能** | **Arrow SIMD Filter, 零拷贝 Slice, 流式 ScanCSV** | ✅ NEW |

**总体: ~95% 覆盖率 (81/85 pandas 特性), 197 个单元测试, 42 个 Go 文件, ~7000 行代码**

---

## 安装

```bash
go get github.com/lekeeith/godas
```

---

## 快速开始

```go
package main

import (
    "fmt"

    "github.com/lekeeith/godas/backend/arrow"
    "github.com/lekeeith/godas/core"
)

func main() {
    // 从 Series 创建 DataFrame
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

    // 过滤行
    mask := make([]bool, df.Len())
    for i := 0; i < df.Len(); i++ {
        mask[i] = age.Int(i) > 26
    }
    filtered := df.Filter(mask)

    // 分组聚合
    dept := arrow.NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil)
    salary := arrow.NewFloat64Series("salary", []float64{100, 120, 80}, nil)
    df2 := arrow.NewDataFrame(dept, salary)
    result := df2.Agg([]string{"dept"}, map[string]core.AggFunc{"salary": core.AggMean})
    fmt.Println(result)
}
```

### 读取 CSV

```go
df, err := io.ReadCSVFile("data.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Println(df)
```

#### CSV 选项

`ReadCSV` 和 `ReadCSVFile` 支持可选的 `CSVOptions` 参数，用于自定义分隔符、跳过行数、表头控制：

```go
// 分号分隔的文件
df, err := io.ReadCSVFile("data.csv", io.CSVOptions{Comma: ';'})

// Tab 分隔，跳过顶部 2 行注释/元数据
df, err := io.ReadCSVFile("data.tsv", io.CSVOptions{Comma: '\t', SkipLines: 2})

// 无表头模式 — 列名自动生成为 col0, col1, ...
df, err := io.ReadCSV("alice,25\nbob,30\n", io.CSVOptions{NoHeader: true})
```

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Comma` | `rune` | `','` | 字段分隔符 (`';'`、`'\t'` 等) |
| `SkipLines` | `int` | `0` | 跳过前 N 行 (元数据/注释) |
| `NoHeader` | `bool` | `false` | 无表头行，自动生成 `col0`、`col1`、... |

### ScanCSV — 大文件惰性扫描

对于大型 CSV 文件 (GB 级别)，使用 `ScanCSV` 进行流式读取，支持谓词下推和投影下推：

```go
// Collect: 将所有匹配行加载到内存
df, err := io.ScanCSVFile("huge.csv").
    Filter("citycode", "==", "320600").
    Filter("fre", "==", "hourly").
    Select("time", "citycode", "poll", "value").
    Collect()

// 分号分隔的文件，顶部有 3 行注释
df, err := io.ScanCSVFile("data.csv").
    Delimiter(';').
    SkipLines(3).
    Filter("age", ">", "25").
    Collect()

// ForEach: 分块流式处理，每块调用回调 (内存 = O(chunkSize × 列数))
processed, err := io.ScanCSVFile("huge.csv").
    Delimiter('\t').              // TSV 文件
    SkipLines(1).                // 跳过注释行
    Filter("citycode", "==", "320600").
    ForEach(10000, func(chunk *arrow.ArrowDataFrame) error {
        // 处理每一块 (例如写入数据库)
        return db.BatchInsert(chunk)
    })

// 断点续传：跳过已处理的行
scan.Offset(processed).ForEach(10000, insertFn)
```

### 读取 JSON / Parquet

```go
df, err := io.ReadJSON(`[{"name":"alice","age":25},{"name":"bob","age":30}]`)
df, err := io.ReadParquetFile("data.parquet")
```

### 写入文件

```go
df.WriteJSONFile("output.json")        // JSON 数组
df.WriteJSONLinesFile("output.ndjson")  // NDJSON (每行一个对象)
io.WriteCSVFile(df, "output.csv")
```

---

## 性能

### 过滤优化 (三层)

`DataFrame.Filter` 使用三层优化：

| 层级 | 场景 | 复杂度 |
|------|------|--------|
| 零拷贝切片 | 连续索引 | O(1)，无数据拷贝 |
| Arrow SIMD | 大量非连续索引 | C++ 向量化过滤 |
| 手动 Take | 兜底 | 直接 `a.Value(idx)` |

### 基准测试结果

```
cpu: 13th Gen Intel(R) Core(TM) i5-13600KF

BenchmarkFilter/rows=10000        508 μs/op     120 KB/op
BenchmarkFilter/rows=100000      4792 μs/op     787 KB/op
BenchmarkFilter/rows=500000     24057 μs/op    3834 KB/op

BenchmarkTakeContiguous/rows=10000      2.4 μs/op    4.8 KB/op  ← O(1) 零拷贝
BenchmarkTakeContiguous/rows=100000     2.0 μs/op    4.8 KB/op
BenchmarkTakeContiguous/rows=500000     1.8 μs/op    4.8 KB/op

BenchmarkTakeScattered/rows=10000     107 μs/op    153 KB/op
BenchmarkTakeScattered/rows=100000    982 μs/op   1352 KB/op
BenchmarkTakeScattered/rows=500000   5726 μs/op   7472 KB/op
```

### ScanCSV 性能 (367 万行, 123 列)

| 方法 | 耗时 | 内存 |
|------|------|------|
| `ReadCSVFile` + Filter | 42s | O(全部数据) |
| `ScanCSV.Collect` | 8s | O(匹配行) |
| `ScanCSV.ForEach` | 7.8s | O(chunkSize) |

运行基准测试：

```bash
go test ./backend/arrow/ -bench=. -benchmem -count=1
```

---

## 架构

```
godas/
├── core/                    # 接口层 (后端无关)
│   ├── dtype.go             # DType 枚举 + 辅助函数
│   ├── index.go             # Index 接口 (Range/Int64/String/DateTime)
│   ├── series.go            # Series 接口
│   ├── dataframe.go         # DataFrame 接口
│   ├── types.go             # JoinType, AggFunc 枚举
│   ├── arithmetic.go        # 算术/比较/逻辑接口 + PromoteDType
│   └── dt_accessor.go       # DateTimeAccessor 接口 + ResampleRule
│
├── backend/arrow/           # Arrow 实现 (18 个源文件)
│   ├── series.go            # ArrowSeries: 核心 Series 实现
│   ├── dataframe.go         # ArrowDataFrame: 核心 DataFrame 实现 + String()/Fmt()/Display()
│   ├── builder.go           # SeriesBuilder: 增量构建
│   ├── benchmark_test.go    # 性能基准测试
│   ├── ops.go               # 排序、合并、分组内部实现
│   ├── utils.go             # DType ↔ Arrow 类型转换
│   ├── arithmetic.go        # Add/Sub/Mul/Div/Mod, 比较, 逻辑
│   ├── time_arithmetic.go   # TA(): 时间戳 + 时间段运算
│   ├── dt_accessor.go       # .dt 访问器 (Year/Month/Day/...)
│   ├── resample.go          # Resample, Shift, Diff, CumSum, DateRange
│   ├── apply.go             # Apply/Map/Transform/ApplyRows
│   ├── selection.go         # IsIn, ValueCounts, Duplicated, Unique
│   ├── concat.go            # Concat (行/列拼接)
│   ├── fill.go              # ffill, bfill, interpolate
│   ├── astype.go            # AsType, ToNumeric
│   ├── str_accessor.go      # .str 访问器 (30+ 字符串方法)
│   ├── stats.go             # Rank, Quantile, Corr/Cov, NLargest/NSmallest
│   ├── rolling.go           # Rolling, Expanding, EWM 窗口函数
│   ├── extras.go            # Pivot, Melt, Query, Clip, Cut, GetDummies, ...
│   ├── parallel.go          # ParallelFilter, ParallelAgg
│   ├── lazy.go              # LazyFrame (查询计划构建器)
│   └── expr.go              # 表达式 DSL
│
└── io/                      # I/O 层
    ├── csv.go               # CSV 读写 + 类型推断
    ├── json.go              # JSON/NDJSON 读写
    ├── parquet.go           # Parquet 读写 (parquet-go)
    ├── excel.go             # Excel (.xlsx) 读写
    ├── scan.go              # ScanCSV (惰性/流式) + ScanParquet
    └── database.go          # 数据库连接器
```

---

## 为什么选择 Arrow?

### 列式 vs 行式存储

```
行式存储 (Go slices):
Row 0: name="alice" age=25 score=88  → 内存分散访问
Row 1: name="bob"   age=30 score=92

列式存储 (Arrow):
name:  [alice, bob, carol]   → 连续内存, CPU 缓存友好
age:   [25, 30, 28]          → SIMD 可向量化
score: [88, 92, 76]          → 零拷贝切片
```

分析场景 (聚合、过滤、统计) **快 3-10 倍**。

### 空值处理

Arrow 的有效性位图: `n/8` 字节 vs Go 的 `[]bool` = `n` 字节。空值标记 **节省 87.5% 内存**。

### 零拷贝

```go
sliced := array.NewSlice(arr, start, end)  // O(1), 无数据拷贝
filtered := compute.Filter(arr, mask)       // SIMD 加速
```

---

## API 参考

### DataFrame

| 操作 | 方法 |
|------|------|
| **创建** | `NewDataFrame(series...)` |
| **形状** | `Shape()`, `Len()`, `Columns()`, `Dtypes()` |
| **列操作** | `Col(name)`, `SelectCols(names)`, `DropCols(names)` |
| **行操作** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` |
| **显示** | `String()`, `Fmt()`, `Display(top,bottom)`, `Info()`, `Describe()` |
| **修改** | `WithColumn(name,s)`, `Rename(mapping)`, `SetIndex(name)`, `ResetIndex()` |
| **清洗** | `DropNA()`, `FillNA(value)`, `FillNAMethod("ffill")` |
| **排序** | `SortBy(names, ascending)` |
| **合并** | `MergeOn(other, on, how)`, `Join(other, how)` |
| **分组** | `GroupByGroups(names)`, `Agg(groupCols, aggs)` |
| **I/O** | `ToCSV()`, `WriteJSONFile(path)`, `WriteJSONLinesFile(path)` |

### Series

| 操作 | 方法 |
|------|------|
| **创建** | `NewInt64Series`, `NewFloat64Series`, `NewStringSeries`, `NewBoolSeries` |
| **访问** | `Name()`, `Len()`, `Dtype()`, `Index()`, `NullCount()` |
| **元素** | `Bool(i)`, `Int(i)`, `Float(i)`, `String(i)`, `IsNull(i)` |
| **切片** | `Head(n)`, `Tail(n)`, `Slice(start,end)`, `Filter(mask)`, `Take(indices)` |
| **显示** | `String()`, `Fmt()`, `Display(top,bottom)` |

### ScanCSV (惰性扫描器)

| 方法 | 说明 |
|------|------|
| `ScanCSVFile(path)` | 创建惰性扫描器 |
| `Delimiter(c)` | 设置字段分隔符 (默认 `,`) |
| `SkipLines(n)` | 跳过表头前 N 行 |
| `Filter(col, op, val)` | 谓词下推 (==, !=, >, <, >=, <=) |
| `Select(columns...)` | 投影下推 |
| `Limit(n)` | 限制返回行数 |
| `Offset(n)` | 跳过前 N 个匹配行 (断点续传) |
| `Collect()` | 执行并返回 DataFrame |
| `ForEach(chunkSize, fn)` | 分块流式处理，每块调用回调 |

### I/O

| 格式 | 读取 | 写入 |
|------|------|------|
| CSV | `ReadCSV(data, ...CSVOptions)`, `ReadCSVFile(path, ...CSVOptions)` | `WriteCSV(df)`, `WriteCSVFile(df, path)` |
| CSV 流式 | `ScanCSVFile(path).Delimiter(';').SkipLines(n).Filter(...).Collect()` | — |
| CSV 分块 | `ScanCSVFile(path).Delimiter(';').Filter(...).ForEach(n, fn)` | — |
| JSON | `ReadJSON(data)`, `ReadJSONFile(path)` | `WriteJSON(df)`, `WriteJSONFile(df, path)` |
| NDJSON | `ReadJSONLines(data)`, `ReadJSONLinesFile(path)` | `WriteJSONLines(df)`, `WriteJSONLinesFile(df, path)` |
| Parquet | `ReadParquetFile(path)` | `WriteParquetFile(df, path)` |
| Excel | `ReadExcelFile(path, sheet)` | `WriteExcelFile(df, path)` |

---

## 测试

```bash
# 运行所有测试
go test ./... -count=1

# 运行基准测试
go test ./backend/arrow/ -bench=. -benchmem -count=1

# 运行指定包的测试
go test ./backend/arrow/ -v -run "TestFilter"
```

**3 个包共 197 个测试**，全部通过。

---

## 依赖

| 包 | 用途 |
|----|------|
| `github.com/apache/arrow/go/v18` | Arrow 列式内存格式 + 计算 SIMD |
| `github.com/parquet-go/parquet-go` | Parquet 文件读写 |
| `github.com/xuri/excelize/v2` | Excel (.xlsx) 读写 |

无需 CGO，纯 Go 实现。

---

## 许可证

MIT
