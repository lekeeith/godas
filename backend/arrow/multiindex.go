package arrow

import (
	"fmt"
	"strings"
	"time"

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
	return rdf
}

// Xs returns rows matching a cross-section value at the given level.
func (df *ArrowDataFrame) Xs(level int, value string) core.DataFrame {
	// Try to use the index if it's a MultiIndex
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
