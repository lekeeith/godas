package arrow

import (
	"fmt"
	"strconv"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// AsType converts a Series to the given DType.
func (s *ArrowSeries) AsType(dt core.DType) core.Series {
	if s.Dtype() == dt {
		return s
	}

	alloc := memory.NewGoAllocator()

	switch dt {
	case core.BOOL:
		bldr := array.NewBooleanBuilder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				switch s.Dtype() {
				case core.INT64:
					bldr.Append(s.Int(i) != 0)
				case core.FLOAT64:
					bldr.Append(s.Float(i) != 0)
				case core.STRING:
					v, _ := strconv.ParseBool(s.String(i))
					bldr.Append(v)
				default:
					bldr.Append(s.Bool(i))
				}
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())

	case core.INT64:
		bldr := array.NewInt64Builder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				switch s.Dtype() {
				case core.BOOL:
					if s.Bool(i) {
						bldr.Append(1)
					} else {
						bldr.Append(0)
					}
				case core.FLOAT64:
					bldr.Append(int64(s.Float(i)))
				case core.STRING:
					v, _ := strconv.ParseInt(s.String(i), 10, 64)
					bldr.Append(v)
				default:
					bldr.Append(s.Int(i))
				}
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())

	case core.FLOAT64:
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				switch s.Dtype() {
				case core.BOOL:
					if s.Bool(i) {
						bldr.Append(1.0)
					} else {
						bldr.Append(0.0)
					}
				case core.INT64:
					bldr.Append(float64(s.Int(i)))
				case core.STRING:
					v, _ := strconv.ParseFloat(s.String(i), 64)
					bldr.Append(v)
				default:
					bldr.Append(s.Float(i))
				}
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())

	case core.STRING:
		bldr := array.NewStringBuilder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				switch s.Dtype() {
				case core.BOOL:
					bldr.Append(fmt.Sprintf("%v", s.Bool(i)))
				case core.FLOAT64, core.FLOAT32:
					bldr.Append(fmt.Sprintf("%g", s.Float(i)))
				default:
					bldr.Append(fmt.Sprintf("%d", s.Int(i)))
				}
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())

	default:
		return s
	}
}

// ToNumeric converts a string Series to float64. Non-parseable values become null.
func ToNumeric(s *ArrowSeries) core.Series {
	if s.Dtype().IsNumeric() {
		return s
	}
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v, err := strconv.ParseFloat(s.String(i), 64)
			if err != nil {
				bldr.AppendNull()
			} else {
				bldr.Append(v)
			}
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}
