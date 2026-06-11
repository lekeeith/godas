package arrow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// ArrowDataFrame implements core.DataFrame backed by Arrow arrays.
type ArrowDataFrame struct {
	columns    []*ArrowSeries
	colMap     map[string]int // name -> index
	index      core.Index
	multiIndex *MultiIndex // non-nil when SetMultiIndex was used
}

// NewDataFrame creates a DataFrame from a slice of ArrowSeries.
func NewDataFrame(series ...*ArrowSeries) *ArrowDataFrame {
	if len(series) == 0 {
		return &ArrowDataFrame{colMap: map[string]int{}}
	}
	colMap := make(map[string]int, len(series))
	for i, s := range series {
		colMap[s.Name()] = i
	}
	return &ArrowDataFrame{
		columns: series,
		colMap:  colMap,
		index:   series[0].Index(),
	}
}

func (df *ArrowDataFrame) Shape() (int, int) {
	if len(df.columns) == 0 {
		return 0, 0
	}
	return df.columns[0].Len(), len(df.columns)
}

func (df *ArrowDataFrame) Len() int {
	rows, _ := df.Shape()
	return rows
}

func (df *ArrowDataFrame) Columns() []string {
	names := make([]string, len(df.columns))
	for i, c := range df.columns {
		names[i] = c.Name()
	}
	return names
}

func (df *ArrowDataFrame) Index() core.Index {
	return df.index
}

func (df *ArrowDataFrame) Dtypes() []core.DType {
	dts := make([]core.DType, len(df.columns))
	for i, c := range df.columns {
		dts[i] = c.Dtype()
	}
	return dts
}

// colIndex returns the column index for the given name, panics if not found.
func (df *ArrowDataFrame) colIndex(name string) int {
	idx, ok := df.colMap[name]
	if !ok {
		panic(fmt.Sprintf("column %q not found", name))
	}
	return idx
}

// newDataFrameWithIndex creates a DataFrame from series with a specific index.
// Used internally to preserve the DataFrame-level index through operations.
func newDataFrameWithIndex(series []*ArrowSeries, idx core.Index) *ArrowDataFrame {
	df := NewDataFrame(series...)
	if idx != nil {
		df.index = idx
	}
	return df
}

// reindex builds a new index from the DataFrame's index using the given positions.
func (df *ArrowDataFrame) reindex(positions []int) core.Index {
	switch df.index.(type) {
	case *core.RangeIndex:
		return core.NewRangeIndex(0, len(positions))
	default:
		strs := make([]string, len(positions))
		for i, p := range positions {
			strs[i] = df.index.Get(p)
		}
		return core.NewStringIndex(strs)
	}
}

func (df *ArrowDataFrame) Col(name string) core.Series {
	return df.columns[df.colIndex(name)]
}

func (df *ArrowDataFrame) SelectCols(names []string) core.DataFrame {
	series := make([]*ArrowSeries, len(names))
	for i, n := range names {
		series[i] = df.columns[df.colIndex(n)]
	}
	return newDataFrameWithIndex(series, df.index)
}

func (df *ArrowDataFrame) DropCols(names []string) core.DataFrame {
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}
	var series []*ArrowSeries
	for _, c := range df.columns {
		if !drop[c.Name()] {
			series = append(series, c)
		}
	}
	if len(series) == 0 {
		return &ArrowDataFrame{colMap: map[string]int{}, index: df.index}
	}
	return newDataFrameWithIndex(series, df.index)
}

func (df *ArrowDataFrame) Head(n int) core.DataFrame {
	if n > df.Len() {
		n = df.Len()
	}
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		series[i] = c.Head(n).(*ArrowSeries)
	}
	return newDataFrameWithIndex(series, df.index.Slice(0, n))
}

func (df *ArrowDataFrame) Tail(n int) core.DataFrame {
	rows := df.Len()
	if n > rows {
		n = rows
	}
	start := rows - n
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		series[i] = c.Tail(n).(*ArrowSeries)
	}
	return newDataFrameWithIndex(series, df.index.Slice(start, rows))
}

func (df *ArrowDataFrame) Slice(start, end int) core.DataFrame {
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		series[i] = c.Slice(start, end).(*ArrowSeries)
	}
	return newDataFrameWithIndex(series, df.index.Slice(start, end))
}

func (df *ArrowDataFrame) Filter(mask []bool) core.DataFrame {
	// Build filtered index
	var indices []int
	for i, m := range mask {
		if m {
			indices = append(indices, i)
		}
	}
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		series[i] = c.Filter(mask).(*ArrowSeries)
	}
	newIdx := df.reindex(indices)
	return newDataFrameWithIndex(series, newIdx)
}

func (df *ArrowDataFrame) Take(indices []int) core.DataFrame {
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		series[i] = c.Take(indices).(*ArrowSeries)
	}
	newIdx := df.reindex(indices)
	return newDataFrameWithIndex(series, newIdx)
}

func (df *ArrowDataFrame) Info() string {
	var b strings.Builder
	rows, cols := df.Shape()
	fmt.Fprintf(&b, "DataFrame: %d rows x %d columns\n", rows, cols)
	fmt.Fprintf(&b, "\n%-15s %-10s %-10s\n", "Column", "DType", "Non-Null")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 35))
	for _, c := range df.columns {
		nn := c.Len() - c.NullCount()
		fmt.Fprintf(&b, "%-15s %-10s %-10d\n", c.Name(), c.Dtype(), nn)
	}
	return b.String()
}

func (df *ArrowDataFrame) Describe() core.DataFrame {
	// Collect numeric columns
	var numCols []*ArrowSeries
	for _, c := range df.columns {
		if c.Dtype().IsNumeric() {
			numCols = append(numCols, c)
		}
	}
	if len(numCols) == 0 {
		return &ArrowDataFrame{colMap: map[string]int{}}
	}

	stats := []string{"count", "mean", "std", "min", "25%", "50%", "75%", "max"}
	series := make([]*ArrowSeries, len(numCols))

	for j, c := range numCols {
		vals := make([]float64, 0, c.Len())
		for i := 0; i < c.Len(); i++ {
			if c.NotNull(i) {
				vals = append(vals, c.Float(i))
			}
		}
		n := float64(len(vals))
		mean, std, min, q25, q50, q75, max := computeStats(vals)
		data := []float64{n, mean, std, min, q25, q50, q75, max}
		series[j] = NewFloat64Series(c.Name(), data, core.NewStringIndex(stats))
	}
	return NewDataFrame(series...)
}

func (df *ArrowDataFrame) WithColumn(name string, s core.Series) core.DataFrame {
	arr := s.(*ArrowSeries).SetName(name).(*ArrowSeries)
	if idx, ok := df.colMap[name]; ok {
		// Replace
		newCols := make([]*ArrowSeries, len(df.columns))
		copy(newCols, df.columns)
		newCols[idx] = arr
		return newDataFrameWithIndex(newCols, df.index)
	}
	// Append
	newCols := make([]*ArrowSeries, len(df.columns)+1)
	copy(newCols, df.columns)
	newCols[len(df.columns)] = arr
	return newDataFrameWithIndex(newCols, df.index)
}

func (df *ArrowDataFrame) DropNA() core.DataFrame {
	// Auto-parallelize for many columns
	if len(df.columns) >= autoParallelMinCols {
		return df.ParallelDropNA()
	}
	rows, _ := df.Shape()
	mask := make([]bool, rows)
	for i := 0; i < rows; i++ {
		mask[i] = true
		for _, c := range df.columns {
			if c.IsNull(i) {
				mask[i] = false
				break
			}
		}
	}
	return df.Filter(mask)
}

func (df *ArrowDataFrame) FillNA(value interface{}) core.DataFrame {
	// Auto-parallelize for many columns
	if len(df.columns) >= autoParallelMinCols {
		return df.ParallelFillNA(value)
	}
	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, len(df.columns))
	for j, c := range df.columns {
		bldr := array.NewBuilder(alloc, c.arr.DataType())
		bldr.Resize(c.Len())
		for i := 0; i < c.Len(); i++ {
			if c.IsNull(i) {
				appendValue(bldr, c.Dtype(), value)
			} else {
				copyValue(bldr, c, i)
			}
		}
		series[j] = NewArrowSeries(c.Name(), bldr.NewArray(), c.Index())
		bldr.Release()
	}
	return newDataFrameWithIndex(series, df.index)
}

func (df *ArrowDataFrame) Rename(mapping map[string]string) core.DataFrame {
	series := make([]*ArrowSeries, len(df.columns))
	for i, c := range df.columns {
		if newName, ok := mapping[c.Name()]; ok {
			series[i] = NewArrowSeries(newName, c.arr, c.index)
		} else {
			series[i] = c
		}
	}
	return newDataFrameWithIndex(series, df.index)
}

func (df *ArrowDataFrame) SetIndex(name string) core.DataFrame {
	s := df.Col(name).(*ArrowSeries)
	idx := &seriesIndex{values: s}
	// Build new columns without the index column
	var series []*ArrowSeries
	for _, c := range df.columns {
		if c.Name() != name {
			series = append(series, c)
		}
	}
	if len(series) == 0 {
		return &ArrowDataFrame{colMap: map[string]int{}, index: idx}
	}
	result := NewDataFrame(series...)
	result.index = idx
	return result
}

func (df *ArrowDataFrame) ResetIndex() core.DataFrame {
	// Build a column from the current index
	idx := df.index
	bldr := NewSeriesBuilder("_index", core.STRING, nil)
	for i := 0; i < idx.Len(); i++ {
		bldr.AppendString(idx.Get(i))
	}
	idxSeries := bldr.Build()

	// Prepend index column
	newCols := make([]*ArrowSeries, len(df.columns)+1)
	newCols[0] = idxSeries
	copy(newCols[1:], df.columns)
	return NewDataFrame(newCols...)
}

func (df *ArrowDataFrame) SortBy(names []string, ascending []bool) core.DataFrame {
	rows, _ := df.Shape()
	if rows == 0 {
		return df
	}
	indices := make([]int, rows)
	for i := range indices {
		indices[i] = i
	}
	// Stable multi-key sort
	sortDataFrame(indices, df, names, ascending)
	return df.Take(indices)
}

func (df *ArrowDataFrame) Join(other core.DataFrame, how core.JoinType) core.DataFrame {
	return df.MergeOn(other, nil, how)
}

func (df *ArrowDataFrame) MergeOn(other core.DataFrame, on []string, how core.JoinType) core.DataFrame {
	o := other.(*ArrowDataFrame)
	if len(on) == 0 {
		// Join on index - build index maps
		return mergeOnIndex(df, o, how)
	}
	return mergeOnColumns(df, o, on, how)
}

func (df *ArrowDataFrame) GroupByGroups(names []string) map[string][]int {
	rows, _ := df.Shape()
	groups := make(map[string][]int)
	for i := 0; i < rows; i++ {
		var key strings.Builder
		for j, n := range names {
			if j > 0 {
				key.WriteByte(',')
			}
			s := df.Col(n).(*ArrowSeries)
			if s.IsNull(i) {
				key.WriteString("<nil>")
			} else {
				switch s.Dtype() {
				case core.BOOL:
					fmt.Fprintf(&key, "%v", s.Bool(i))
				case core.FLOAT32, core.FLOAT64:
					fmt.Fprintf(&key, "%g", s.Float(i))
				case core.STRING:
					key.WriteString(s.String(i))
				default:
					fmt.Fprintf(&key, "%d", s.Int(i))
				}
			}
		}
		k := key.String()
		groups[k] = append(groups[k], i)
	}
	return groups
}

func (df *ArrowDataFrame) Agg(groupCols []string, aggs map[string]core.AggFunc) core.DataFrame {
	// Auto-parallelize for many aggregation columns
	if len(aggs) >= autoParallelMinCols {
		return df.ParallelAgg(groupCols, aggs)
	}
	groups := df.GroupByGroups(groupCols)
	var resultSeries []*ArrowSeries

	alloc := memory.NewGoAllocator()

	// Group key columns
	for _, gc := range groupCols {
		s := df.Col(gc).(*ArrowSeries)
		bldr := array.NewBuilder(alloc, s.arr.DataType())
		bldr.Resize(len(groups))
		seen := make(map[string]bool)
		for k := range groups {
			if !seen[k] {
				firstIdx := groups[k][0]
				copyValue(bldr, s, firstIdx)
				seen[k] = true
			}
		}
		resultSeries = append(resultSeries, NewArrowSeries(gc, bldr.NewArray(), nil))
		bldr.Release()
	}

	// Aggregation columns
	for colName, fn := range aggs {
		s := df.Col(colName).(*ArrowSeries)
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(len(groups))
		for _, indices := range groups {
			vals := make([]float64, 0, len(indices))
			for _, idx := range indices {
				if s.NotNull(idx) {
					vals = append(vals, s.Float(idx))
				}
			}
			bldr.Append(applyAgg(fn, vals))
		}
		aggName := colName + "_" + fn.String()
		resultSeries = append(resultSeries, NewArrowSeries(aggName, bldr.NewArray(), nil))
		bldr.Release()
	}

	return NewDataFrame(resultSeries...)
}

func (df *ArrowDataFrame) ToCSV() string {
	var b strings.Builder
	// Header
	for i, c := range df.columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c.Name())
	}
	b.WriteByte('\n')
	// Rows
	rows, _ := df.Shape()
	for i := 0; i < rows; i++ {
		for j, c := range df.columns {
			if j > 0 {
				b.WriteByte(',')
			}
			if c.IsNull(i) {
				continue
			}
			switch c.Dtype() {
			case core.BOOL:
				fmt.Fprintf(&b, "%v", c.Bool(i))
			case core.FLOAT32, core.FLOAT64:
				fmt.Fprintf(&b, "%g", c.Float(i))
			case core.STRING:
				b.WriteString(c.String(i))
			default:
				fmt.Fprintf(&b, "%d", c.Int(i))
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// ToJSON serializes the DataFrame to a JSON array of objects string.
func (df *ArrowDataFrame) ToJSON() (string, error) {
	rows, _ := df.Shape()
	colNames := df.Columns()
	result := make([]map[string]interface{}, rows)

	for i := 0; i < rows; i++ {
		row := make(map[string]interface{}, len(colNames))
		for _, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				row[name] = nil
				continue
			}
			switch s.Dtype() {
			case core.BOOL:
				row[name] = s.Bool(i)
			case core.FLOAT32, core.FLOAT64:
				row[name] = s.Float(i)
			case core.STRING:
				row[name] = s.String(i)
			default:
				row[name] = s.Int(i)
			}
		}
		result[i] = row
	}

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(b), nil
}

// ToJSONLines serializes the DataFrame to NDJSON format.
func (df *ArrowDataFrame) ToJSONLines() (string, error) {
	rows, _ := df.Shape()
	colNames := df.Columns()
	var b strings.Builder

	for i := 0; i < rows; i++ {
		row := make(map[string]interface{}, len(colNames))
		for _, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				row[name] = nil
				continue
			}
			switch s.Dtype() {
			case core.BOOL:
				row[name] = s.Bool(i)
			case core.FLOAT32, core.FLOAT64:
				row[name] = s.Float(i)
			case core.STRING:
				row[name] = s.String(i)
			default:
				row[name] = s.Int(i)
			}
		}
		line, err := json.Marshal(row)
		if err != nil {
			return "", fmt.Errorf("marshal json line: %w", err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// WriteCSVFile writes the DataFrame to a CSV file.
func (df *ArrowDataFrame) WriteCSVFile(path string) error {
	return os.WriteFile(path, []byte(df.ToCSV()), 0644)
}

// WriteJSONFile writes the DataFrame to a JSON file.
func (df *ArrowDataFrame) WriteJSONFile(path string) error {
	s, err := df.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), 0644)
}

// WriteJSONLinesFile writes the DataFrame to an NDJSON file.
func (df *ArrowDataFrame) WriteJSONLinesFile(path string) error {
	s, err := df.ToJSONLines()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), 0644)
}
func (df *ArrowDataFrame) GroupByTransform(groupCols []string, targetCol string, fn func([]float64) []float64) core.DataFrame {
	groups := df.GroupByGroups(groupCols)
	target := df.Col(targetCol).(*ArrowSeries)
	result := make([]float64, df.Len())

	for _, indices := range groups {
		vals := make([]float64, len(indices))
		for i, idx := range indices {
			if target.NotNull(idx) {
				vals[i] = target.Float(idx)
			}
		}
		transformed := fn(vals)
		for i, idx := range indices {
			if i < len(transformed) {
				result[idx] = transformed[i]
			}
		}
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(df.Len())
	for _, v := range result {
		bldr.Append(v)
	}

	series := make([]*ArrowSeries, len(df.columns))
	copy(series, df.columns)
	series = append(series, NewArrowSeries(targetCol+"_transformed", bldr.NewArray(), df.Index()))
	return NewDataFrame(series...)
}

// GroupByFilter keeps only groups where the predicate returns true.
func (df *ArrowDataFrame) GroupByFilter(groupCols []string, predicate func(core.DataFrame) bool) core.DataFrame {
	groups := df.GroupByGroups(groupCols)
	keep := make([]int, 0)
	for _, indices := range groups {
		sub := df.Take(indices)
		if predicate(sub) {
			keep = append(keep, indices...)
		}
	}
	return df.Take(keep)
}

// GroupByApply applies a function to each group and concatenates the results.
func (df *ArrowDataFrame) GroupByApply(groupCols []string, fn func(core.DataFrame) core.DataFrame) core.DataFrame {
	groups := df.GroupByGroups(groupCols)
	results := make([]core.DataFrame, 0, len(groups))
	for _, indices := range groups {
		sub := df.Take(indices)
		results = append(results, fn(sub))
	}
	if len(results) == 0 {
		return NewDataFrame()
	}
	dfs := make([]*ArrowDataFrame, len(results))
	for i, r := range results {
		dfs[i] = r.(*ArrowDataFrame)
	}
	return Concat(dfs, ConcatRows)
}
