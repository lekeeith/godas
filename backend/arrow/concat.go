package arrow

import (
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
)

// ConcatMode specifies how to concatenate DataFrames.
type ConcatMode int

const (
	ConcatRows ConcatMode = iota // 纵向拼接 (axis=0)
	ConcatCols                   // 横向拼接 (axis=1)
)

// Concat concatenates multiple DataFrames.
// mode: ConcatRows (纵向, same columns) or ConcatCols (横向, same rows).
func Concat(dfs []*ArrowDataFrame, mode ConcatMode) *ArrowDataFrame {
	if len(dfs) == 0 {
		return NewDataFrame()
	}
	if len(dfs) == 1 {
		return dfs[0]
	}

	switch mode {
	case ConcatCols:
		return concatCols(dfs)
	default:
		return concatRows(dfs)
	}
}

func concatRows(dfs []*ArrowDataFrame) *ArrowDataFrame {
	// Use columns from first DataFrame as reference
	colNames := dfs[0].Columns()
	alloc := memory.NewGoAllocator()

	// Calculate total rows
	totalRows := 0
	for _, df := range dfs {
		totalRows += df.Len()
	}

	series := make([]*ArrowSeries, len(colNames))
	for j, name := range colNames {
		// Collect all values for this column
		bldr := array.NewBuilder(alloc, dfs[0].Col(name).(*ArrowSeries).arr.DataType())
		bldr.Resize(totalRows)
		for _, df := range dfs {
			col := df.Col(name).(*ArrowSeries)
			for i := 0; i < col.Len(); i++ {
				if col.IsNull(i) {
					bldr.AppendNull()
				} else {
					copyValue(bldr, col, i)
				}
			}
		}
		series[j] = NewArrowSeries(name, bldr.NewArray(), nil)
	}
	return NewDataFrame(series...)
}

func concatCols(dfs []*ArrowDataFrame) *ArrowDataFrame {
	var allCols []*ArrowSeries
	for _, df := range dfs {
		for _, name := range df.Columns() {
			allCols = append(allCols, df.Col(name).(*ArrowSeries))
		}
	}
	return NewDataFrame(allCols...)
}
