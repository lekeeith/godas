package arrow

import (
	"math"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

// --- Arithmetic Operations ---

func (s *ArrowSeries) Add(other core.Series) core.Series {
	return binaryOp(s, other.(*ArrowSeries), func(a, b float64) float64 { return a + b })
}

func (s *ArrowSeries) AddScalar(v float64) core.Series {
	return scalarOp(s, v, func(a, b float64) float64 { return a + b })
}

func (s *ArrowSeries) Sub(other core.Series) core.Series {
	return binaryOp(s, other.(*ArrowSeries), func(a, b float64) float64 { return a - b })
}

func (s *ArrowSeries) SubScalar(v float64) core.Series {
	return scalarOp(s, v, func(a, b float64) float64 { return a - b })
}

func (s *ArrowSeries) Mul(other core.Series) core.Series {
	return binaryOp(s, other.(*ArrowSeries), func(a, b float64) float64 { return a * b })
}

func (s *ArrowSeries) MulScalar(v float64) core.Series {
	return scalarOp(s, v, func(a, b float64) float64 { return a * b })
}

func (s *ArrowSeries) Div(other core.Series) core.Series {
	return binaryOp(s, other.(*ArrowSeries), func(a, b float64) float64 {
		if b == 0 {
			return math.NaN()
		}
		return a / b
	})
}

func (s *ArrowSeries) DivScalar(v float64) core.Series {
	if v == 0 {
		return scalarOp(s, math.NaN(), func(a, b float64) float64 { return b })
	}
	return scalarOp(s, v, func(a, b float64) float64 { return a / b })
}

func (s *ArrowSeries) Mod(other core.Series) core.Series {
	return binaryOp(s, other.(*ArrowSeries), func(a, b float64) float64 {
		if b == 0 {
			return math.NaN()
		}
		return math.Mod(a, b)
	})
}

func (s *ArrowSeries) Neg() core.Series {
	return scalarOp(s, -1, func(a, b float64) float64 { return a * b })
}

func (s *ArrowSeries) Abs() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(math.Abs(s.Float(i)))
		}
	}
	return NewArrowSeries(s.Name()+"_abs", bldr.NewArray(), s.Index())
}

// --- Comparison Operations ---

func (s *ArrowSeries) Eq(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a == b })
}

func (s *ArrowSeries) Ne(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a != b })
}

func (s *ArrowSeries) Lt(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a < b })
}

func (s *ArrowSeries) Le(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a <= b })
}

func (s *ArrowSeries) Gt(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a > b })
}

func (s *ArrowSeries) Ge(other core.Series) core.Series {
	return compareOp(s, other.(*ArrowSeries), func(a, b float64) bool { return a >= b })
}

func (s *ArrowSeries) EqScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a == b })
}

func (s *ArrowSeries) NeScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a != b })
}

func (s *ArrowSeries) LtScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a < b })
}

func (s *ArrowSeries) LeScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a <= b })
}

func (s *ArrowSeries) GtScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a > b })
}

func (s *ArrowSeries) GeScalar(v float64) core.Series {
	return compareScalarOp(s, v, func(a, b float64) bool { return a >= b })
}

// --- Logic Operations ---

func (s *ArrowSeries) And(other core.Series) core.Series {
	return logicOp(s, other.(*ArrowSeries), func(a, b bool) bool { return a && b })
}

func (s *ArrowSeries) Or(other core.Series) core.Series {
	return logicOp(s, other.(*ArrowSeries), func(a, b bool) bool { return a || b })
}

func (s *ArrowSeries) Not() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(!s.Bool(i))
		}
	}
	return NewArrowSeries(s.Name()+"_not", bldr.NewArray(), s.Index())
}

// --- Internal helpers ---

func binaryOp(a, b *ArrowSeries, fn func(float64, float64) float64) *ArrowSeries {
	n := a.Len()
	if b.Len() < n {
		n = b.Len()
	}
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		if a.IsNull(i) || b.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(a.Float(i), b.Float(i)))
		}
	}
	return NewArrowSeries(a.Name()+"_op", bldr.NewArray(), a.Index().Slice(0, n))
}

func scalarOp(s *ArrowSeries, v float64, fn func(float64, float64) float64) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.Float(i), v))
		}
	}
	return NewArrowSeries(s.Name()+"_op", bldr.NewArray(), s.Index())
}

func compareOp(a, b *ArrowSeries, fn func(float64, float64) bool) *ArrowSeries {
	n := a.Len()
	if b.Len() < n {
		n = b.Len()
	}
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		if a.IsNull(i) || b.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(a.Float(i), b.Float(i)))
		}
	}
	return NewArrowSeries(a.Name()+"_cmp", bldr.NewArray(), a.Index().Slice(0, n))
}

func compareScalarOp(s *ArrowSeries, v float64, fn func(float64, float64) bool) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.Float(i), v))
		}
	}
	return NewArrowSeries(s.Name()+"_cmp", bldr.NewArray(), s.Index())
}

func logicOp(a, b *ArrowSeries, fn func(bool, bool) bool) *ArrowSeries {
	n := a.Len()
	if b.Len() < n {
		n = b.Len()
	}
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		if a.IsNull(i) || b.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(a.Bool(i), b.Bool(i)))
		}
	}
	return NewArrowSeries(a.Name()+"_logic", bldr.NewArray(), a.Index().Slice(0, n))
}
