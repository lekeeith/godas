package arrow

import (
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// Where replaces values where cond is false with `other`.
// cond: boolean Series, other: replacement value or Series.
func (s *ArrowSeries) Where(cond core.Series, other interface{}) core.Series {
	c := cond.(*ArrowSeries)
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if c.NotNull(i) && c.Bool(i) {
			// Keep original
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				copyValue(bldr, s, i)
			}
		} else {
			// Replace with other
			if other == nil {
				bldr.AppendNull()
			} else {
				appendValue(bldr, s.Dtype(), other)
			}
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// Mask replaces values where cond is true with `other`.
func (s *ArrowSeries) Mask(cond core.Series, other interface{}) core.Series {
	c := cond.(*ArrowSeries)
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if c.NotNull(i) && c.Bool(i) {
			// Replace with other
			if other == nil {
				bldr.AppendNull()
			} else {
				appendValue(bldr, s.Dtype(), other)
			}
		} else {
			// Keep original
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				copyValue(bldr, s, i)
			}
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// DataFrame.Where replaces values where cond is false.
func (df *ArrowDataFrame) Where(cond core.Series, other interface{}) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		series[i] = df.Col(name).(*ArrowSeries).Where(cond, other).(*ArrowSeries)
	}
	return NewDataFrame(series...)
}

// DataFrame.Mask replaces values where cond is true.
func (df *ArrowDataFrame) Mask(cond core.Series, other interface{}) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		series[i] = df.Col(name).(*ArrowSeries).Mask(cond, other).(*ArrowSeries)
	}
	return NewDataFrame(series...)
}
