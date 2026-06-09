package arrow

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// computeStats returns count, mean, std, min, q25, q50, q75, max for sorted or unsorted data.
func computeStats(vals []float64) (mean, std, min, q25, q50, q75, max float64) {
	n := len(vals)
	if n == 0 {
		return
	}
	// Sort a copy
	sorted := make([]float64, n)
	copy(sorted, vals)
	sort.Float64s(sorted)

	min = sorted[0]
	max = sorted[n-1]

	// Mean
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	mean = sum / float64(n)

	// Std (population)
	var ss float64
	for _, v := range sorted {
		d := v - mean
		ss += d * d
	}
	std = math.Sqrt(ss / float64(n))

	// Quantiles
	q25 = percentile(sorted, 0.25)
	q50 = percentile(sorted, 0.50)
	q75 = percentile(sorted, 0.75)
	return
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	pos := p * float64(n-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

// sortDataFrame sorts indices in-place by the given column names.
func sortDataFrame(indices []int, df *ArrowDataFrame, names []string, ascending []bool) {
	sort.SliceStable(indices, func(a, b int) bool {
		for i, name := range names {
			s := df.Col(name).(*ArrowSeries)
			asc := true
			if i < len(ascending) {
				asc = ascending[i]
			}
			cmp := compareValues(s, indices[a], indices[b])
			if cmp < 0 {
				return asc
			}
			if cmp > 0 {
				return !asc
			}
		}
		return false
	})
}

func compareValues(s *ArrowSeries, a, b int) int {
	if s.IsNull(a) && s.IsNull(b) {
		return 0
	}
	if s.IsNull(a) {
		return -1
	}
	if s.IsNull(b) {
		return 1
	}
	switch s.arr.(type) {
	case *array.Boolean:
		va, vb := s.Bool(a), s.Bool(b)
		if va == vb {
			return 0
		}
		if !va {
			return -1
		}
		return 1
	case *array.Float32, *array.Float64:
		va, vb := s.Float(a), s.Float(b)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case *array.String:
		va, vb := s.String(a), s.String(b)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	default:
		va, vb := s.Int(a), s.Int(b)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	}
}

// applyAgg applies an aggregation function to a slice of values.
func applyAgg(fn core.AggFunc, vals []float64) float64 {
	n := len(vals)
	if n == 0 {
		return math.NaN()
	}
	switch fn {
	case core.AggSum:
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	case core.AggMean:
		return sumF(vals) / float64(n)
	case core.AggMedian:
		sorted := make([]float64, n)
		copy(sorted, vals)
		sort.Float64s(sorted)
		return percentile(sorted, 0.5)
	case core.AggMin:
		return minF(vals)
	case core.AggMax:
		return maxF(vals)
	case core.AggCount:
		return float64(n)
	case core.AggStd:
		_, std, _, _, _, _, _ := computeStats(vals)
		return std
	case core.AggVar:
		_, std, _, _, _, _, _ := computeStats(vals)
		return std * std
	case core.AggFirst:
		return vals[0]
	case core.AggLast:
		return vals[n-1]
	case core.AggNUnique:
		seen := make(map[float64]bool)
		for _, v := range vals {
			seen[v] = true
		}
		return float64(len(seen))
	default:
		return math.NaN()
	}
}

func sumF(vals []float64) float64 {
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s
}

func minF(vals []float64) float64 {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxF(vals []float64) float64 {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// seriesIndex implements core.Index backed by a Series (for SetIndex).
type seriesIndex struct {
	values *ArrowSeries
}

func (idx *seriesIndex) Len() int            { return idx.values.Len() }
func (idx *seriesIndex) Get(i int) string    { return idx.values.String(i) }
func (idx *seriesIndex) Type() core.DType    { return idx.values.Dtype() }
func (idx *seriesIndex) Slice(start, end int) core.Index {
	return &seriesIndex{values: idx.values.Slice(start, end).(*ArrowSeries)}
}
func (idx *seriesIndex) Copy() core.Index {
	return &seriesIndex{values: idx.values.Copy().(*ArrowSeries)}
}

// mergeOnIndex joins two DataFrames on their indices.
func mergeOnIndex(left, right *ArrowDataFrame, how core.JoinType) *ArrowDataFrame {
	switch how {
	case core.Inner:
		return innerJoinIndex(left, right)
	case core.Left:
		return leftJoinIndex(left, right)
	default:
		// For now, fallback to left join
		return leftJoinIndex(left, right)
	}
}

func innerJoinIndex(left, right *ArrowDataFrame) *ArrowDataFrame {
	// Build right index lookup
	rightIdx := make(map[string]int)
	rowsR, _ := right.Shape()
	for i := 0; i < rowsR; i++ {
		rightIdx[right.index.Get(i)] = i
	}
	rowsL, _ := left.Shape()
	var lIndices, rIndices []int
	for i := 0; i < rowsL; i++ {
		key := left.index.Get(i)
		if j, ok := rightIdx[key]; ok {
			lIndices = append(lIndices, i)
			rIndices = append(rIndices, j)
		}
	}
	return joinByIndices(left, right, lIndices, rIndices)
}

func leftJoinIndex(left, right *ArrowDataFrame) *ArrowDataFrame {
	rightIdx := make(map[string]int)
	rowsR, _ := right.Shape()
	for i := 0; i < rowsR; i++ {
		rightIdx[right.index.Get(i)] = i
	}
	rowsL, _ := left.Shape()
	var lIndices, rIndices []int
	var rHasMatch []bool
	for i := 0; i < rowsL; i++ {
		key := left.index.Get(i)
		lIndices = append(lIndices, i)
		if j, ok := rightIdx[key]; ok {
			rIndices = append(rIndices, j)
			rHasMatch = append(rHasMatch, true)
		} else {
			rIndices = append(rIndices, -1)
			rHasMatch = append(rHasMatch, false)
		}
	}
	return joinByIndicesWithNulls(left, right, lIndices, rIndices, rHasMatch)
}

// mergeOnColumns joins two DataFrames on the given column names.
func mergeOnColumns(left, right *ArrowDataFrame, on []string, how core.JoinType) *ArrowDataFrame {
	// Build right lookup by join key
	rowsR, _ := right.Shape()
	rightIdx := make(map[string][]int)
	for i := 0; i < rowsR; i++ {
		key := joinKey(right, on, i)
		rightIdx[key] = append(rightIdx[key], i)
	}

	rowsL, _ := left.Shape()
	var lIndices, rIndices []int

	for i := 0; i < rowsL; i++ {
		key := joinKey(left, on, i)
		if matches, ok := rightIdx[key]; ok {
			for _, j := range matches {
				lIndices = append(lIndices, i)
				rIndices = append(rIndices, j)
			}
		} else if how == core.Left || how == core.Outer {
			lIndices = append(lIndices, i)
			rIndices = append(rIndices, -1)
		}
	}

	if how == core.Outer {
		// Add unmatched right rows
		matched := make(map[int]bool)
		for _, j := range rIndices {
			if j >= 0 {
				matched[j] = true
			}
		}
		for i := 0; i < rowsR; i++ {
			if !matched[i] {
				lIndices = append(lIndices, -1)
				rIndices = append(rIndices, i)
			}
		}
	}

	return joinByIndicesWithNullsLR(left, right, on, lIndices, rIndices)
}

func joinKey(df *ArrowDataFrame, cols []string, row int) string {
	var key strings.Builder
	for i, c := range cols {
		if i > 0 {
			key.WriteByte(0) // separator unlikely in data
		}
		s := df.Col(c).(*ArrowSeries)
		if s.IsNull(row) {
			key.WriteString("<nil>")
		} else {
			switch s.Dtype() {
			case core.STRING:
				key.WriteString(s.String(row))
			case core.FLOAT32, core.FLOAT64:
				fmt.Fprintf(&key, "%g", s.Float(row))
			default:
				fmt.Fprintf(&key, "%d", s.Int(row))
			}
		}
	}
	return key.String()
}

// joinByIndices builds a merged DataFrame from matched index pairs (inner join).
func joinByIndices(left, right *ArrowDataFrame, lIdx, rIdx []int) *ArrowDataFrame {
	var series []*ArrowSeries
	for _, c := range left.columns {
		series = append(series, c.Take(lIdx).(*ArrowSeries))
	}
	// Add right columns (skip those that conflict with left)
	leftNames := make(map[string]bool)
	for _, c := range left.columns {
		leftNames[c.Name()] = true
	}
	for _, c := range right.columns {
		name := c.Name()
		if leftNames[name] {
			name = name + "_right"
		}
		taken := c.Take(rIdx).(*ArrowSeries)
		series = append(series, NewArrowSeries(name, taken.arr, taken.index))
	}
	return NewDataFrame(series...)
}

// joinByIndicesWithNulls builds a merged DataFrame allowing null right rows (left join).
func joinByIndicesWithNulls(left, right *ArrowDataFrame, lIdx, rIdx []int, rHasMatch []bool) *ArrowDataFrame {
	var series []*ArrowSeries
	for _, c := range left.columns {
		series = append(series, c.Take(lIdx).(*ArrowSeries))
	}
	leftNames := make(map[string]bool)
	for _, c := range left.columns {
		leftNames[c.Name()] = true
	}
	for _, c := range right.columns {
		name := c.Name()
		if leftNames[name] {
			name = name + "_right"
		}
		bldr := array.NewBuilder(memory.NewGoAllocator(), c.arr.DataType())
		bldr.Resize(len(rIdx))
		for i, j := range rIdx {
			if j < 0 || !rHasMatch[i] {
				bldr.AppendNull()
			} else {
				copyValue(bldr, c, j)
			}
		}
		series = append(series, NewArrowSeries(name, bldr.NewArray(), nil))
		bldr.Release()
	}
	return NewDataFrame(series...)
}

// joinByIndicesWithNullsLR builds a merged DataFrame for column-based joins.
func joinByIndicesWithNullsLR(left, right *ArrowDataFrame, on []string, lIdx, rIdx []int) *ArrowDataFrame {
	var series []*ArrowSeries
	// Join key columns (from left)
	for _, name := range on {
		s := left.Col(name).(*ArrowSeries)
		series = append(series, s.Take(lIdx).(*ArrowSeries))
	}
	// Left non-key columns
	keySet := make(map[string]bool, len(on))
	for _, k := range on {
		keySet[k] = true
	}
	for _, c := range left.columns {
		if !keySet[c.Name()] {
			series = append(series, c.Take(lIdx).(*ArrowSeries))
		}
	}
	// Right non-key columns
	for _, c := range right.columns {
		if !keySet[c.Name()] {
			name := c.Name()
			if _, exists := left.colMap[name]; exists {
				name = name + "_right"
			}
			bldr := array.NewBuilder(memory.NewGoAllocator(), c.arr.DataType())
			bldr.Resize(len(rIdx))
			for _, j := range rIdx {
				if j < 0 {
					bldr.AppendNull()
				} else {
					copyValue(bldr, c, j)
				}
			}
			series = append(series, NewArrowSeries(name, bldr.NewArray(), nil))
			bldr.Release()
		}
	}
	return NewDataFrame(series...)
}

// appendValue appends a Go value to a builder.
func appendValue(bldr array.Builder, dt core.DType, v interface{}) {
	if v == nil {
		bldr.AppendNull()
		return
	}
	switch dt {
	case core.BOOL:
		if bv, ok := v.(bool); ok {
			bldr.(*array.BooleanBuilder).Append(bv)
		} else {
			bldr.AppendNull()
		}
	case core.INT8, core.INT16, core.INT32, core.INT64:
		if iv, ok := toInt64(v); ok {
			switch bd := bldr.(type) {
			case *array.Int8Builder:
				bd.Append(int8(iv))
			case *array.Int16Builder:
				bd.Append(int16(iv))
			case *array.Int32Builder:
				bd.Append(int32(iv))
			case *array.Int64Builder:
				bd.Append(iv)
			}
		} else {
			bldr.AppendNull()
		}
	case core.UINT8, core.UINT16, core.UINT32, core.UINT64:
		if iv, ok := toInt64(v); ok {
			switch bd := bldr.(type) {
			case *array.Uint8Builder:
				bd.Append(uint8(iv))
			case *array.Uint16Builder:
				bd.Append(uint16(iv))
			case *array.Uint32Builder:
				bd.Append(uint32(iv))
			case *array.Uint64Builder:
				bd.Append(uint64(iv))
			}
		} else {
			bldr.AppendNull()
		}
	case core.FLOAT32, core.FLOAT64:
		if fv, ok := toFloat64(v); ok {
			switch bd := bldr.(type) {
			case *array.Float32Builder:
				bd.Append(float32(fv))
			case *array.Float64Builder:
				bd.Append(fv)
			}
		} else {
			bldr.AppendNull()
		}
	case core.STRING:
		bldr.(*array.StringBuilder).Append(fmt.Sprintf("%v", v))
	default:
		bldr.AppendNull()
	}
}

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int8:
		return int64(val), true
	case int16:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint8:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	case float32:
		return int64(val), true
	case float64:
		return int64(val), true
	}
	return 0, false
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	}
	return 0, false
}

// copyValue copies a single value from a series into a builder.
func copyValue(bldr array.Builder, s *ArrowSeries, i int) {
	if s.IsNull(i) {
		bldr.AppendNull()
		return
	}
	switch s.arr.(type) {
	case *array.Boolean:
		bldr.(*array.BooleanBuilder).Append(s.Bool(i))
	case *array.Int8:
		bldr.(*array.Int8Builder).Append(int8(s.Int(i)))
	case *array.Int16:
		bldr.(*array.Int16Builder).Append(int16(s.Int(i)))
	case *array.Int32:
		bldr.(*array.Int32Builder).Append(int32(s.Int(i)))
	case *array.Int64:
		bldr.(*array.Int64Builder).Append(s.Int(i))
	case *array.Uint8:
		bldr.(*array.Uint8Builder).Append(uint8(s.Int(i)))
	case *array.Uint16:
		bldr.(*array.Uint16Builder).Append(uint16(s.Int(i)))
	case *array.Uint32:
		bldr.(*array.Uint32Builder).Append(uint32(s.Int(i)))
	case *array.Uint64:
		bldr.(*array.Uint64Builder).Append(uint64(s.Int(i)))
	case *array.Float32:
		bldr.(*array.Float32Builder).Append(float32(s.Float(i)))
	case *array.Float64:
		bldr.(*array.Float64Builder).Append(s.Float(i))
	case *array.String:
		bldr.(*array.StringBuilder).Append(s.String(i))
	default:
		bldr.AppendNull()
	}
}
