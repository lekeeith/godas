# godas - Pandas for Go: Feature Plan

## 已完成 ✅

### 数据结构 (5/5)
- [x] Series — 一维带标签数组
- [x] DataFrame — 二维带标签表格
- [x] Index — RangeIndex/Int64/StringIndex/DateTimeIndex/MultiIndex
- [x] DType — bool/int8-64/uint8-64/float32/64/string/timestamp/duration
- [x] SeriesBuilder — 增量构建

### I/O (5/5)
- [x] CSV — ReadCSV/WriteCSV + 类型推断
- [x] JSON — ReadJSON/WriteJSON + NDJSON
- [x] Parquet — ReadParquetFile/WriteParquetFile
- [x] Excel — ReadExcelFile/WriteExcelFile (excelize)

### 选择索引 (8/8)
- [x] Col/SelectCols/DropCols — 按列名访问
- [x] Slice/Take — 按位置访问
- [x] Filter — 布尔索引
- [x] IsIn — 值匹配过滤
- [x] BetweenTime — 时间范围过滤
- [x] Query — 字符串表达式筛选
- [x] xs() — 跨层选取 (MultiIndex)
- [x] Where/Mask — 条件替换

### 数据清洗 (10/10)
- [x] IsNull/NotNull/NullCount
- [x] DropNA + FillNA (ffill/bfill/interpolate)
- [x] Replace/Rename
- [x] Duplicated/DropDuplicates
- [x] AsType/ToNumeric/ConvertDtypes
- [x] Clip — 异常值截断

### 转换 (10/10)
- [x] Apply/Map/Transform/ApplyRows
- [x] Pipe — 管道操作
- [x] Pivot/PivotTable/Melt/Stack
- [x] GetDummies — One-Hot 编码
- [x] Cut/QCut — 分箱
- [x] Explode — 列表展开
- [x] Factorize — 编码

### 合并 (5/5)
- [x] MergeOn (inner/left/right/outer/cross)
- [x] Join — 基于索引
- [x] Concat — 纵向/横向拼接
- [x] CombineFirst/Update/Compare

### GroupBy (7/7)
- [x] GroupByGroups + Agg (11 种聚合)
- [x] GroupBy.Transform/Filter/Apply
- [x] Rolling/Expanding/EWM
- [x] Resample — 时间重采样

### 时间序列 (10/10)
- [x] .dt 访问器 (Year/Month/Day/Hour/Minute/Second/DayOfWeek/Quarter/Week)
- [x] DateRange/DateRangeEnd
- [x] Shift/PctChange/Diff
- [x] CumSum/CumProd/CumMax/CumMin
- [x] TA() — 时间算术
- [x] TzLocalize/TzConvert/BetweenTime/AtTime
- [x] AsFreq — 频率转换

### 统计 (14/14)
- [x] Describe/Info/ValueCounts/NUnique/Unique
- [x] NLargest/NSmallest/Quantile/Rank
- [x] Corr/Cov/CorrMatrix
- [x] SortBy/Mode/Skew/Kurt/MemoryUsage

### 算术 (10/10)
- [x] Add/Sub/Mul/Div/Mod + Scalar 变体
- [x] Eq/Ne/Lt/Le/Gt/Ge + Scalar 变体
- [x] And/Or/Not — 逻辑
- [x] Neg/Abs/PromoteDType

### 字符串 .str (30+)
- [x] Upper/Lower/Title/SwapCase
- [x] Strip/LStrip/RStrip
- [x] Contains/StartsWith/EndsWith
- [x] Replace/ReplaceRegex/Split/Extract/ExtractAll
- [x] Count/Len/Pad/ZFill/Cat
- [x] IsNumeric/IsAlpha/IsDigit/IsEmpty

### 高级
- [x] MultiIndex — 多级层次化索引

---

## ⚠️ Pandas 剩余项

### 🟡 高级特性
- [x] **Categorical** — 分类类型 (有序/无序, `.cat` 访问器)
- [ ] **Sparse** — 稀疏数据结构 (节省内存)
- [ ] **ExtensionArray** — 自定义数据类型扩展接口

### 🟢 可视化
- [ ] **.plot()** — 可视化集成 (需引入 `gonum.org/v1/plot`)
- [ ] **.style** — DataFrame 样式渲染 (HTML 输出)

---

## 🚀 Polars 特性 — 按优先级实现

### Phase 1: 性能优化 ✅
- [x] **并行 Transform/Agg** — Go goroutine 自动并行化列操作
- [x] **并行 GroupBy** — 分组聚合自动分片并行
- [x] **ScanParquet** — 惰性扫描，不全量加载

### Phase 2: 表达式系统 ✅
- [x] **Expr 类型** — 可序列化的表达式描述 (col("x").Add(1).Filter(>5))
- [x] **Lazy DataFrame** — `df.Lazy()` 构建查询计划
- [x] **查询优化器** — Select/Filter/WithColumn/Sort/Limit/GroupBy
- [x] **Collect()** — 执行惰性计划，返回结果

### Phase 3: 高级 Join ✅
- [x] **Asof Join** — 时间序列近似匹配
- [x] **Semi Join** — 只返回左表匹配行
- [x] **Anti Join** — 返回左表不匹配行
- [x] **Cross Join** — 笛卡尔积

### Phase 4: 时间序列增强 ✅
- [x] **GroupByDynamic** — 动态时间窗口分组
- [x] **GroupBy Rolling Time** — 时间滚动窗口分组

### Phase 5: 类型扩展 ✅
- [x] **Categorical** — 字典编码分类类型 (有序/无序, .cat 访问器)

### Phase 6: SQL 接口 ✅
- [x] **SQL 引擎** — 内置 SQL 解析器 (SELECT/WHERE/GROUP BY/ORDER BY/LIMIT/JOIN)
- [x] **数据库连接器** — MySQL/PostgreSQL/SQLite (database/sql)
- [x] **ReadSQL/WriteSQL** — 从数据库读写 DataFrame

---

### 统计

| 指标 | 值 |
|------|-----|
| Pandas 功能点 | ~85 |
| Pandas 已完成 | ~85 |
| Pandas 覆盖率 | **~100%** |
| Polars 特性 | Phase 1-6 全部完成 |
| 测试数 | **277** |
| Go 文件 | **59** |
| Commit | **32** |
