# godans - Pandas for Go: Feature Plan

## 已完成 ✅

- [x] 数据结构: Series, DataFrame, Index (Range/Int64/String/DateTime)
- [x] 数据类型: bool, int8-64, uint8-64, float32/64, string, timestamp
- [x] I/O: CSV, JSON, NDJSON, Parquet 读写
- [x] 选择索引: Col/SelectCols/DropCols, Slice/Take, Filter (布尔索引)
- [x] 数据清洗: IsNull/NotNull/NullCount, DropNA, FillNA, Rename
- [x] 转换: Apply, MapFloat/MapString/MapBool, Transform, ApplyRows
- [x] 合并: MergeOn (inner/left/right/outer), Join (基于索引)
- [x] GroupBy: GroupByGroups, Agg (sum/mean/median/std/var/min/max/count/first/last/nunique)
- [x] 时间序列: .dt 访问器, Resample, Shift, PctChange, Diff, CumSum/CumProd/CumMax/CumMin
- [x] 时间算术: TA() (timestamp+duration, Before/After, 单位转换)
- [x] 统计: Describe, Info, SortBy
- [x] 算术: Add/Sub/Mul/Div/Mod, 比较 (Eq/Ne/Lt/Le/Gt/Ge), 逻辑 (And/Or/Not), Neg/Abs

---

## 高优先级 🔴

- [ ] **concat** — 纵向/横向拼接 DataFrame
- [ ] **value_counts** — 值计数 (Series.value_counts)
- [ ] **isin** — 值匹配过滤 (Series.isin, DataFrame.isin)
- [ ] **duplicated / drop_duplicates** — 去重
- [ ] **interpolate / ffill / bfill** — 缺失值插补 (线性/前向/后向)
- [ ] **astype** — 类型转换 (int→float, string→int 等)
- [ ] **to_numeric / to_datetime** — 类型解析
- [ ] **.str 访问器** — 字符串处理 (contains/replace/split/strip/startswith/endswith/upper/lower/len/count/extract)
- [ ] **nlargest / nsmallest** — Top N
- [ ] **quantile** — 分位数
- [ ] **rank** — 排名 (min/dense/first/average)
- [ ] **corr / cov** — 相关性/协方差矩阵
- [ ] **rolling / expanding / ewm** — 滚动窗口 (mean/sum/std/min/max/apply)

## 中优先级 🟡

- [ ] **pivot / melt / stack / unstack** — 数据重塑
- [ ] **get_dummies** — One-Hot 编码
- [ ] **cut / qcut** — 分箱
- [ ] **GroupBy.transform / filter / apply** — 组内变换/过滤/应用
- [ ] **pipe** — 管道操作
- [ ] **combine_first / update** — 合并填补
- [ ] **where / mask** — 条件替换
- [ ] **query** — 字符串表达式筛选
- [ ] **factorize** — 编码
- [ ] **explode** — 列表展开
- [ ] **mode** — 众数
- [ ] **skew / kurt** — 偏度/峰度
- [ ] **memory_usage** — 内存占用

## 低优先级 🟢

- [ ] **Categorical** 类型
- [ ] **Sparse** 稀疏数据
- [ ] **ExtensionArray** 自定义类型
- [ ] **.style** 样式渲染
- [ ] **.plot()** 可视化集成
- [ ] **SQL** 查询接口
