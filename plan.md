# godans - Pandas for Go: Feature Plan

## 已完成 ✅

### 数据结构 (5/5)
- [x] Series — 一维带标签数组
- [x] DataFrame — 二维带标签表格
- [x] Index — RangeIndex/Int64Index/StringIndex/DateTimeIndex
- [x] DType — bool/int8-64/uint8-64/float32/64/string/timestamp/duration/category
- [x] SeriesBuilder — 增量构建

### I/O (4/5)
- [x] CSV — ReadCSV/ReadCSVFile/WriteCSV/WriteCSVFile + 类型推断
- [x] JSON — ReadJSON/WriteJSON + NDJSON
- [x] Parquet — ReadParquetFile/WriteParquetFile
- [ ] Excel — read_excel/to_excel
- [x] SQL — (通过 DataFrame 构造，无独立接口)

### 选择索引 (6/8)
- [x] Col/SelectCols/DropCols — 按列名访问
- [x] Slice/Take — 按位置访问
- [x] Filter — 布尔索引
- [x] IsIn — 值匹配过滤
- [x] BetweenTime — 时间范围过滤
- [ ] .xs() — 跨层选取 (MultiIndex)
- [ ] .query() — 字符串表达式筛选
- [x] Where/Mask — 条件替换 (通过 Apply 实现)

### 数据清洗 (8/10)
- [x] IsNull/NotNull/NullCount
- [x] DropNA
- [x] FillNA + FillNAMethod (ffill/bfill/interpolate)
- [x] Replace/Rename
- [x] Duplicated/DropDuplicates
- [x] AsType/ToNumeric
- [ ] ConvertDtypes — 自动推断最佳类型
- [ ] Clip — 异常值截断

### 转换 (7/10)
- [x] Apply (Series.Apply, DataFrame.ApplyCols/ApplyRows)
- [x] MapFloat/MapString/MapBool
- [x] Transform
- [x] Pipe — 管道操作
- [x] Explode — 列表展开
- [x] GetDummies — One-Hot 编码
- [x] Cut/QCut — 分箱
- [ ] Pivot/PivotTable — 数据透视
- [ ] Melt — 宽转长
- [ ] Stack/Unstack — 层次化旋转

### 合并 (4/6)
- [x] MergeOn — inner/left/right/outer/cross join
- [x] Join — 基于索引
- [x] Concat — 纵向/横向拼接
- [x] CombineFirst — 合并填补
- [ ] Compare — 差异比较
- [x] Update — 原地更新

### GroupBy (5/7)
- [x] GroupByGroups — 分组
- [x] Agg — 11 种聚合函数
- [x] GroupBy.Transform — 组内变换
- [x] GroupBy.Filter — 组过滤
- [x] GroupBy.Apply — 组内应用
- [x] Rolling — 滚动窗口聚合
- [x] Resample — 时间重采样

### 时间序列 (9/10)
- [x] .dt 访问器 — Year/Month/Day/Hour/Minute/Second/DayOfWeek/DayOfYear/Quarter/Week
- [x] DateRange/DateRangeEnd
- [x] Resample
- [x] Shift/PctChange/Diff
- [x] CumSum/CumProd/CumMax/CumMin
- [x] TA() — 时间算术 (timestamp+duration, Before/After)
- [x] TzLocalize/TzConvert
- [x] BetweenTime/AtTime
- [x] Rolling/Expanding/EWM
- [ ] AsFreq — 频率转换

### 统计 (12/12)
- [x] Describe/Info
- [x] ValueCounts/NUnique
- [x] NLargest/NSmallest
- [x] Quantile
- [x] Rank (average/min/max/first/dense)
- [x] Corr/Cov/CorrMatrix
- [x] SortBy
- [x] Skew/Kurt
- [x] Mode

### 算术 (10/10)
- [x] Add/Sub/Mul/Div/Mod — Series-Series 和 Series-scalar
- [x] Eq/Ne/Lt/Le/Gt/Ge — 比较
- [x] And/Or/Not — 逻辑
- [x] Neg/Abs — 一元运算
- [x] PromoteDType — 类型提升

### 字符串 .str (30+)
- [x] Upper/Lower/Title/SwapCase
- [x] Strip/LStrip/RStrip
- [x] Contains/StartsWith/EndsWith
- [x] Replace/ReplaceRegex
- [x] Split/Extract/ExtractAll
- [x] Count/Len
- [x] Pad/ZFill
- [x] Cat
- [x] IsNumeric/IsAlpha/IsDigit/IsEmpty

### 时间算术 TA()
- [x] AddDuration/SubDuration
- [x] SubTimestamps
- [x] DurationAdd/DurationSub/DurationMul/DurationDiv
- [x] Before/After
- [x] ToDays/ToHours/ToMinutes/ToSeconds/ToMilliseconds

---

## 未完成 ❌

### I/O
- [ ] **Excel** — read_excel/to_excel (需要 excelize 依赖)

### 选择索引
- [ ] **xs()** — MultiIndex 跨层选取
- [ ] **query()** — 字符串表达式筛选

### 数据清洗
- [ ] **convert_dtypes** — 自动推断最佳类型
- [ ] **clip** — 异常值截断 (min/max 边界)

### 转换
- [ ] **pivot/pivot_table** — 数据透视
- [ ] **melt** — 宽转长
- [ ] **stack/unstack** — 层次化旋转

### 合并
- [ ] **compare** — 差异比较

### 时间序列
- [ ] **asfreq** — 频率转换

### 高级
- [ ] **MultiIndex** — 多级层次化索引
- [ ] **Categorical** — 分类类型
- [ ] **Sparse** — 稀疏数据
- [ ] **ExtensionArray** — 自定义类型扩展
- [ ] **.style** — 样式渲染
- [ ] **.plot()** — 可视化集成 (gonum/plot)
