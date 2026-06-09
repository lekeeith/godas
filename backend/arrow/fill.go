package arrow

import (
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// FillForward propagates last valid value forward (ffill).
func (s *ArrowSeries) FillForward() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(s.Len())

	var lastValid interface{}
	hasValid := false

	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			copyValue(bldr, s, i)
			switch s.Dtype() {
			case core.BOOL:
				lastValid = s.Bool(i)
			case core.STRING:
				lastValid = s.String(i)
			case core.FLOAT32, core.FLOAT64:
				lastValid = s.Float(i)
			default:
				lastValid = s.Int(i)
			}
			hasValid = true
		} else if hasValid {
			switch s.Dtype() {
			case core.BOOL:
				bldr.(*array.BooleanBuilder).Append(lastValid.(bool))
			case core.STRING:
				bldr.(*array.StringBuilder).Append(lastValid.(string))
			case core.FLOAT32:
				bldr.(*array.Float32Builder).Append(float32(lastValid.(float64)))
			case core.FLOAT64:
				bldr.(*array.Float64Builder).Append(lastValid.(float64))
			default:
				bldr.(*array.Int64Builder).Append(lastValid.(int64))
			}
		} else {
			bldr.AppendNull()
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// FillBackward propagates next valid value backward (bfill).
func (s *ArrowSeries) FillBackward() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(s.Len())

	// Scan backward to find next valid value for each position
	nextValid := make([]interface{}, s.Len())
	hasNext := make([]bool, s.Len())
	var current interface{}
	found := false

	for i := s.Len() - 1; i >= 0; i-- {
		if s.NotNull(i) {
			switch s.Dtype() {
			case core.BOOL:
				current = s.Bool(i)
			case core.STRING:
				current = s.String(i)
			case core.FLOAT32, core.FLOAT64:
				current = s.Float(i)
			default:
				current = s.Int(i)
			}
			found = true
		}
		if found {
			nextValid[i] = current
			hasNext[i] = true
		}
	}

	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			copyValue(bldr, s, i)
		} else if hasNext[i] {
			switch s.Dtype() {
			case core.BOOL:
				bldr.(*array.BooleanBuilder).Append(nextValid[i].(bool))
			case core.STRING:
				bldr.(*array.StringBuilder).Append(nextValid[i].(string))
			case core.FLOAT32:
				bldr.(*array.Float32Builder).Append(float32(nextValid[i].(float64)))
			case core.FLOAT64:
				bldr.(*array.Float64Builder).Append(nextValid[i].(float64))
			default:
				bldr.(*array.Int64Builder).Append(nextValid[i].(int64))
			}
		} else {
			bldr.AppendNull()
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// Interpolate fills null values using linear interpolation.
func (s *ArrowSeries) Interpolate() core.Series {
	if s.Dtype() != core.INT64 && s.Dtype() != core.FLOAT64 && s.Dtype() != core.FLOAT32 {
		// Non-numeric: fallback to forward fill
		return s.FillForward()
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	// Collect valid points
	type point struct {
		idx int
		val float64
	}
	valid := make([]point, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			valid = append(valid, point{i, s.Float(i)})
		}
	}

	if len(valid) == 0 {
		for i := 0; i < s.Len(); i++ {
			bldr.AppendNull()
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
	}

	vIdx := 0
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			bldr.Append(s.Float(i))
			vIdx++
		} else {
			// Find surrounding valid points
			var left, right *point
			if vIdx > 0 {
				left = &valid[vIdx-1]
			}
			if vIdx < len(valid) {
				right = &valid[vIdx]
			}

			if left == nil && right == nil {
				bldr.AppendNull()
			} else if left == nil {
				bldr.Append(right.val)
			} else if right == nil {
				bldr.Append(left.val)
			} else {
				// Linear interpolation
				t := float64(i-left.idx) / float64(right.idx-left.idx)
				bldr.Append(left.val + t*(right.val-left.val))
			}
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// DataFrame FillNA with method: "ffill", "bfill", "interpolate".
func (df *ArrowDataFrame) FillNAMethod(method string) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		col := df.Col(name).(*ArrowSeries)
		switch method {
		case "ffill":
			series[i] = col.FillForward().(*ArrowSeries)
		case "bfill":
			series[i] = col.FillBackward().(*ArrowSeries)
		case "interpolate":
			series[i] = col.Interpolate().(*ArrowSeries)
		default:
			series[i] = col
		}
	}
	return NewDataFrame(series...)
}
