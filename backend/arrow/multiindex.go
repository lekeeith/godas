package arrow

import (
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// MultiIndex is a multi-level index for rows or columns.
type MultiIndex struct {
	levels [][]string // level[0] = first level values, level[1] = second level, etc.
	labels [][]int    // labels[level][row] = index into levels[level]
	names  []string   // name for each level
}

// NewMultiIndex creates a MultiIndex from levels and labels.
func NewMultiIndex(levels [][]string, labels [][]int, names []string) *MultiIndex {
	return &MultiIndex{levels: levels, labels: labels, names: names}
}

// NewMultiIndexFromTuples creates a MultiIndex from a slice of tuples.
func NewMultiIndexFromTuples(tuples [][]string, names []string) *MultiIndex {
	if len(tuples) == 0 {
		return &MultiIndex{names: names}
	}
	nLevels := len(tuples[0])
	levels := make([][]string, nLevels)
	levelMaps := make([]map[string]int, nLevels)
	// labels[level][row] = index into levels[level]
	labels := make([][]int, nLevels)

	for i := range levels {
		levels[i] = make([]string, 0)
		levelMaps[i] = make(map[string]int)
		labels[i] = make([]int, len(tuples))
	}

	for rowIdx, tuple := range tuples {
		for lvl, val := range tuple {
			if _, exists := levelMaps[lvl][val]; !exists {
				levelMaps[lvl][val] = len(levels[lvl])
				levels[lvl] = append(levels[lvl], val)
			}
			labels[lvl][rowIdx] = levelMaps[lvl][val]
		}
	}

	return &MultiIndex{levels: levels, labels: labels, names: names}
}

// Len returns the number of entries.
func (mi *MultiIndex) Len() int {
	if len(mi.labels) == 0 || len(mi.labels[0]) == 0 {
		return 0
	}
	return len(mi.labels[0])
}

// NLevels returns the number of index levels.
func (mi *MultiIndex) NLevels() int {
	return len(mi.levels)
}

// Names returns the level names.
func (mi *MultiIndex) Names() []string {
	return mi.names
}

// Get returns the tuple at position i as a slice of strings.
func (mi *MultiIndex) Get(i int) []string {
	if i < 0 || i >= mi.Len() {
		panic(fmt.Sprintf("MultiIndex: index %d out of range [0:%d)", i, mi.Len()))
	}
	tuple := make([]string, len(mi.levels))
	for lvl, level := range mi.levels {
		if i < len(mi.labels[lvl]) {
			tuple[lvl] = level[mi.labels[lvl][i]]
		}
	}
	return tuple
}

// GetLevel returns the value at position i for the given level.
func (mi *MultiIndex) GetLevel(level, i int) string {
	if level < 0 || level >= len(mi.levels) {
		return ""
	}
	if i < 0 || i >= len(mi.labels[level]) {
		return ""
	}
	return mi.levels[level][mi.labels[level][i]]
}

// String returns a string representation.
func (mi *MultiIndex) String() string {
	var b strings.Builder
	for i := 0; i < mi.Len(); i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		tuple := mi.Get(i)
		b.WriteString(fmt.Sprintf("(%s)", strings.Join(tuple, ", ")))
	}
	return b.String()
}

// ToIndex converts the MultiIndex to a simple string Index (by joining tuple values).
func (mi *MultiIndex) ToIndex() core.Index {
	vals := make([]string, mi.Len())
	for i := 0; i < mi.Len(); i++ {
		vals[i] = strings.Join(mi.Get(i), "/")
	}
	return core.NewStringIndex(vals)
}

// Xs returns cross-section: rows matching the given value at the specified level.
func (mi *MultiIndex) Xs(level int, value string) []int {
	indices := make([]int, 0)
	for i := 0; i < mi.Len(); i++ {
		if mi.GetLevel(level, i) == value {
			indices = append(indices, i)
		}
	}
	return indices
}

// Slice returns a subset of the MultiIndex.
func (mi *MultiIndex) Slice(start, end int) *MultiIndex {
	newLabels := make([][]int, len(mi.levels))
	for lvl := range mi.levels {
		newLabels[lvl] = mi.labels[lvl][start:end]
	}
	return &MultiIndex{levels: mi.levels, labels: newLabels, names: mi.names}
}

// SwapLevel swaps two levels in the MultiIndex.
func (mi *MultiIndex) SwapLevel(i, j int) *MultiIndex {
	if i < 0 || i >= len(mi.levels) || j < 0 || j >= len(mi.levels) {
		return mi
	}
	newLevels := make([][]string, len(mi.levels))
	newLabels := make([][]int, len(mi.levels))
	newNames := make([]string, len(mi.names))
	copy(newLevels, mi.levels)
	copy(newLabels, mi.labels)
	copy(newNames, mi.names)
	newLevels[i], newLevels[j] = newLevels[j], newLevels[i]
	newLabels[i], newLabels[j] = newLabels[j], newLabels[i]
	newNames[i], newNames[j] = newNames[j], newNames[i]
	return &MultiIndex{levels: newLevels, labels: newLabels, names: newNames}
}

// DropLevel removes a level from the MultiIndex.
func (mi *MultiIndex) DropLevel(level int) *MultiIndex {
	if level < 0 || level >= len(mi.levels) || len(mi.levels) <= 1 {
		return mi
	}
	newLevels := make([][]string, 0, len(mi.levels)-1)
	newLabels := make([][]int, 0, len(mi.levels)-1)
	newNames := make([]string, 0, len(mi.names)-1)
	for i := range mi.levels {
		if i != level {
			newLevels = append(newLevels, mi.levels[i])
			newLabels = append(newLabels, mi.labels[i])
			if i < len(mi.names) {
				newNames = append(newNames, mi.names[i])
			}
		}
	}
	return &MultiIndex{levels: newLevels, labels: newLabels, names: newNames}
}

// RenameLevel renames a level in the MultiIndex.
func (mi *MultiIndex) RenameLevel(level int, name string) *MultiIndex {
	if level < 0 || level >= len(mi.names) {
		return mi
	}
	newNames := make([]string, len(mi.names))
	copy(newNames, mi.names)
	newNames[level] = name
	return &MultiIndex{levels: mi.levels, labels: mi.labels, names: newNames}
}

// GetLevelValues returns the values at the given level as a string slice.
func (mi *MultiIndex) GetLevelValues(level int) []string {
	if level < 0 || level >= len(mi.levels) {
		return nil
	}
	vals := make([]string, mi.Len())
	for i := 0; i < mi.Len(); i++ {
		vals[i] = mi.GetLevel(level, i)
	}
	return vals
}

// --- DataFrame MultiIndex level operations ---

// MultiIndexSwapLevel swaps two levels of the MultiIndex.
func (df *ArrowDataFrame) MultiIndexSwapLevel(i, j int) *ArrowDataFrame {
	if df.multiIndex == nil {
		return df
	}
	rdf := df.copy()
	rdf.multiIndex = df.multiIndex.SwapLevel(i, j)
	rdf.index = rdf.multiIndex.ToIndex()
	return rdf
}

// MultiIndexDropLevel removes a level from the MultiIndex.
func (df *ArrowDataFrame) MultiIndexDropLevel(level int) *ArrowDataFrame {
	if df.multiIndex == nil {
		return df
	}
	rdf := df.copy()
	rdf.multiIndex = df.multiIndex.DropLevel(level)
	rdf.index = rdf.multiIndex.ToIndex()
	return rdf
}

// MultiIndexRenameLevel renames a level of the MultiIndex.
func (df *ArrowDataFrame) MultiIndexRenameLevel(level int, name string) *ArrowDataFrame {
	if df.multiIndex == nil {
		return df
	}
	rdf := df.copy()
	rdf.multiIndex = df.multiIndex.RenameLevel(level, name)
	return rdf
}

// MultiIndexGetLevelValues returns the values at the given level as a Series.
func (df *ArrowDataFrame) MultiIndexGetLevelValues(level int) core.Series {
	if df.multiIndex == nil {
		return nil
	}
	vals := df.multiIndex.GetLevelValues(level)
	levelName := fmt.Sprintf("level_%d", level)
	if level < len(df.multiIndex.names) && df.multiIndex.names[level] != "" {
		levelName = df.multiIndex.names[level]
	}
	return NewStringSeries(levelName, vals, nil)
}

// GroupByLevel groups by the given index level and aggregates.
func (df *ArrowDataFrame) GroupByLevel(level int, aggs map[string]core.AggFunc) core.DataFrame {
	if df.multiIndex == nil {
		return df
	}
	// Build group key from the level values
	groups := make(map[string][]int)
	for i := 0; i < df.Len(); i++ {
		key := df.multiIndex.GetLevel(level, i)
		groups[key] = append(groups[key], i)
	}

	alloc := memory.NewGoAllocator()

	// Build level key column
	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	keyBldr := array.NewStringBuilder(alloc)
	keyBldr.Resize(len(groupKeys))
	for _, k := range groupKeys {
		keyBldr.Append(k)
	}
	keySeries := NewArrowSeries("_group_key", keyBldr.NewArray(), nil)

	// Aggregation columns
	var resultSeries []*ArrowSeries
	resultSeries = append(resultSeries, keySeries)
	for colName, fn := range aggs {
		s := df.Col(colName).(*ArrowSeries)
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(len(groups))
		for _, k := range groupKeys {
			vals := make([]float64, 0, len(groups[k]))
			for _, idx := range groups[k] {
				if s.NotNull(idx) {
					vals = append(vals, s.Float(idx))
				}
			}
			bldr.Append(applyAgg(fn, vals))
		}
		resultSeries = append(resultSeries, NewArrowSeries(colName+"_"+fn.String(), bldr.NewArray(), nil))
	}
	return NewDataFrame(resultSeries...)
}

// copy creates a shallow copy of the DataFrame.
func (df *ArrowDataFrame) copy() *ArrowDataFrame {
	newCols := make([]*ArrowSeries, len(df.columns))
	copy(newCols, df.columns)
	newColMap := make(map[string]int, len(df.colMap))
	for k, v := range df.colMap {
		newColMap[k] = v
	}
	rdf := &ArrowDataFrame{
		columns:    newCols,
		colMap:     newColMap,
		index:      df.index,
		multiIndex: df.multiIndex,
	}
	return rdf
}

// --- DataFrame MultiIndex support ---

// SetMultiIndex sets a MultiIndex from multiple columns.
func (df *ArrowDataFrame) SetMultiIndex(colNames []string) *ArrowDataFrame {
	rows, _ := df.Shape()
	tuples := make([][]string, rows)
	for i := 0; i < rows; i++ {
		tuple := make([]string, len(colNames))
		for j, name := range colNames {
			s := df.Col(name).(*ArrowSeries)
			if s.IsNull(i) {
				tuple[j] = "<nil>"
			} else {
				switch s.Dtype() {
				case core.BOOL:
					tuple[j] = fmt.Sprintf("%v", s.Bool(i))
				case core.FLOAT32, core.FLOAT64:
					tuple[j] = fmt.Sprintf("%g", s.Float(i))
				case core.STRING:
					tuple[j] = s.String(i)
				default:
					tuple[j] = fmt.Sprintf("%d", s.Int(i))
				}
			}
		}
		tuples[i] = tuple
	}

	mi := NewMultiIndexFromTuples(tuples, colNames)

	// Drop the index columns from the DataFrame
	result := df.DropCols(colNames)
	// Set the MultiIndex as the index
	rdf := result.(*ArrowDataFrame)
	rdf.index = mi.ToIndex()
	rdf.multiIndex = mi
	return rdf
}

// Xs returns rows matching a cross-section value at the given level.
func (df *ArrowDataFrame) Xs(level int, value string) core.DataFrame {
	// Use MultiIndex directly if available
	if df.multiIndex != nil {
		indices := df.multiIndex.Xs(level, value)
		return df.Take(indices)
	}
	// Fallback: parse flattened index
	indices := make([]int, 0)
	rows, _ := df.Shape()
	for i := 0; i < rows; i++ {
		idxVal := df.index.Get(i)
		parts := strings.Split(idxVal, "/")
		if level < len(parts) && parts[level] == value {
			indices = append(indices, i)
		}
	}
	return df.Take(indices)
}

// --- DateTimeIndex ---

// NewDateTimeIndexFromSeries creates a DateTimeIndex from a timestamp series.
func NewDateTimeIndexFromSeries(s *ArrowSeries) core.DateTimeIndex {
	times := make([]time.Time, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			times[i] = time.Unix(0, s.Int(i))
		}
	}
	return *core.NewDateTimeIndex(times)
}
