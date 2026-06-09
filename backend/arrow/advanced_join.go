package arrow

import (
	"sort"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// AsofJoin performs an as-of join: for each row in the left DataFrame,
// find the most recent row in the right DataFrame where the on column
// is <= the left value. Useful for time series alignment.
//
// Example: align trades (left) with quotes (right) by timestamp,
// finding the most recent quote before each trade.
func (df *ArrowDataFrame) AsofJoin(other core.DataFrame, on string, by []string) core.DataFrame {
	o := other.(*ArrowDataFrame)
	leftOn := df.Col(on).(*ArrowSeries)
	rightOn := o.Col(on).(*ArrowSeries)

	// Sort right by on column for binary search
	rightIndices := make([]int, rightOn.Len())
	for i := range rightIndices {
		rightIndices[i] = i
	}
	sort.Slice(rightIndices, func(a, b int) bool {
		return rightOn.Float(rightIndices[a]) <= rightOn.Float(rightIndices[b])
	})

	// For each left row, find the matching right row
	matchIdx := make([]int, df.Len())
	for i := 0; i < df.Len(); i++ {
		if leftOn.IsNull(i) {
			matchIdx[i] = -1
			continue
		}
		leftVal := leftOn.Float(i)

		// If by columns specified, also match on those
		if len(by) > 0 {
			matchIdx[i] = findAsofWithBy(o, rightOn, rightIndices, leftVal, df, by, i)
		} else {
			matchIdx[i] = findAsof(rightOn, rightIndices, leftVal)
		}
	}

	return mergeAsof(df, o, matchIdx, on)
}

func findAsof(rightOn *ArrowSeries, rightIndices []int, leftVal float64) int {
	// Binary search for the last right value <= leftVal
	lo, hi := 0, len(rightIndices)-1
	result := -1
	for lo <= hi {
		mid := (lo + hi) / 2
		midIdx := rightIndices[mid]
		if rightOn.Float(midIdx) <= leftVal {
			result = midIdx
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return result
}

func findAsofWithBy(right *ArrowDataFrame, rightOn *ArrowSeries, rightIndices []int, leftVal float64, left *ArrowDataFrame, by []string, leftRow int) int {
	// Find the last right row where on <= leftVal AND by columns match
	for i := len(rightIndices) - 1; i >= 0; i-- {
		rIdx := rightIndices[i]
		if rightOn.Float(rIdx) > leftVal {
			continue
		}
		// Check by columns
		match := true
		for _, b := range by {
			lv := left.Col(b).String(leftRow)
			rv := right.Col(b).String(rIdx)
			if lv != rv {
				match = false
				break
			}
		}
		if match {
			return rIdx
		}
	}
	return -1
}

func mergeAsof(left, right *ArrowDataFrame, matchIdx []int, on string) *ArrowDataFrame {
	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, 0)

	// Left columns
	for _, name := range left.Columns() {
		s := left.Col(name).(*ArrowSeries)
		series = append(series, s)
	}

	// Right columns (excluding the join key)
	for _, name := range right.Columns() {
		if name == on {
			continue
		}
		s := right.Col(name).(*ArrowSeries)
		bldr := array.NewBuilder(alloc, s.arr.DataType())
		bldr.Resize(len(matchIdx))
		for _, idx := range matchIdx {
			if idx < 0 || s.IsNull(idx) {
				bldr.AppendNull()
			} else {
				copyValue(bldr, s, idx)
			}
		}
		series = append(series, NewArrowSeries(name, bldr.NewArray(), left.Index()))
	}

	return NewDataFrame(series...)
}

// SemiJoin returns only rows from the left DataFrame where there is a match
// in the right DataFrame on the specified columns.
func (df *ArrowDataFrame) SemiJoin(other core.DataFrame, on []string) core.DataFrame {
	o := other.(*ArrowDataFrame)

	// Build lookup from right
	rightSet := make(map[string]bool)
	rowsR, _ := o.Shape()
	for i := 0; i < rowsR; i++ {
		key := buildJoinKey(o, on, i)
		rightSet[key] = true
	}

	// Filter left rows
	rowsL, _ := df.Shape()
	indices := make([]int, 0)
	for i := 0; i < rowsL; i++ {
		key := buildJoinKey(df, on, i)
		if rightSet[key] {
			indices = append(indices, i)
		}
	}
	return df.Take(indices)
}

// AntiJoin returns only rows from the left DataFrame where there is NO match
// in the right DataFrame on the specified columns.
func (df *ArrowDataFrame) AntiJoin(other core.DataFrame, on []string) core.DataFrame {
	o := other.(*ArrowDataFrame)

	// Build lookup from right
	rightSet := make(map[string]bool)
	rowsR, _ := o.Shape()
	for i := 0; i < rowsR; i++ {
		key := buildJoinKey(o, on, i)
		rightSet[key] = true
	}

	// Filter left rows where NO match
	rowsL, _ := df.Shape()
	indices := make([]int, 0)
	for i := 0; i < rowsL; i++ {
		key := buildJoinKey(df, on, i)
		if !rightSet[key] {
			indices = append(indices, i)
		}
	}
	return df.Take(indices)
}

func buildJoinKey(df *ArrowDataFrame, cols []string, row int) string {
	var key string
	for i, name := range cols {
		if i > 0 {
			key += "\x00"
		}
		s := df.Col(name).(*ArrowSeries)
		if s.IsNull(row) {
			key += "<nil>"
		} else {
			switch s.Dtype() {
			case core.BOOL:
				if s.Bool(row) {
					key += "1"
				} else {
					key += "0"
				}
			case core.STRING:
				key += s.String(row)
			case core.FLOAT32, core.FLOAT64:
				key += formatFloat(s.Float(row))
			default:
				key += formatInt(s.Int(row))
			}
		}
	}
	return key
}

// CrossJoin returns the Cartesian product of two DataFrames.
func (df *ArrowDataFrame) CrossJoin(other core.DataFrame) core.DataFrame {
	o := other.(*ArrowDataFrame)
	rowsL, _ := df.Shape()
	rowsR, _ := o.Shape()
	total := rowsL * rowsR

	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, 0)

	// Left columns (repeated)
	for _, name := range df.Columns() {
		s := df.Col(name).(*ArrowSeries)
		bldr := array.NewBuilder(alloc, s.arr.DataType())
		bldr.Resize(total)
		for i := 0; i < rowsL; i++ {
			for j := 0; j < rowsR; j++ {
				if s.IsNull(i) {
					bldr.AppendNull()
				} else {
					copyValue(bldr, s, i)
				}
			}
		}
		series = append(series, NewArrowSeries(name, bldr.NewArray(), nil))
	}

	// Right columns
	for _, name := range o.Columns() {
		s := o.Col(name).(*ArrowSeries)
		bldr := array.NewBuilder(alloc, s.arr.DataType())
		bldr.Resize(total)
		for i := 0; i < rowsL; i++ {
			for j := 0; j < rowsR; j++ {
				if s.IsNull(j) {
					bldr.AppendNull()
				} else {
					copyValue(bldr, s, j)
				}
			}
		}
		series = append(series, NewArrowSeries(name, bldr.NewArray(), nil))
	}

	return NewDataFrame(series...)
}
