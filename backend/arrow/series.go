package arrow

import (
	"fmt"
	"strings"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// ArrowSeries implements core.Series backed by an Arrow array.
type ArrowSeries struct {
	name  string
	arr   arrow.Array
	index core.Index
}

// NewArrowSeries creates a new ArrowSeries.
func NewArrowSeries(name string, arr arrow.Array, index core.Index) *ArrowSeries {
	if index == nil {
		index = core.NewDefaultIndex(arr.Len())
	}
	return &ArrowSeries{name: name, arr: arr, index: index}
}

func (s *ArrowSeries) Name() string       { return s.name }
func (s *ArrowSeries) Len() int           { return s.arr.Len() }
func (s *ArrowSeries) Index() core.Index  { return s.index }
func (s *ArrowSeries) NullCount() int     { return s.arr.NullN() }
func (s *ArrowSeries) IsNull(i int) bool  { return s.arr.IsNull(i) }
func (s *ArrowSeries) NotNull(i int) bool { return s.arr.IsValid(i) }

func (s *ArrowSeries) Dtype() core.DType {
	return ArrowToDType(s.arr.DataType())
}

func (s *ArrowSeries) Bool(i int) bool {
	if s.IsNull(i) {
		return false
	}
	if b, ok := s.arr.(*array.Boolean); ok {
		return b.Value(i)
	}
	return false
}

func (s *ArrowSeries) Int(i int) int64 {
	if s.IsNull(i) {
		return 0
	}
	switch a := s.arr.(type) {
	case *array.Int8:
		return int64(a.Value(i))
	case *array.Int16:
		return int64(a.Value(i))
	case *array.Int32:
		return int64(a.Value(i))
	case *array.Int64:
		return a.Value(i)
	case *array.Uint8:
		return int64(a.Value(i))
	case *array.Uint16:
		return int64(a.Value(i))
	case *array.Uint32:
		return int64(a.Value(i))
	case *array.Uint64:
		return int64(a.Value(i))
	case *array.Float32:
		return int64(a.Value(i))
	case *array.Float64:
		return int64(a.Value(i))
	}
	return 0
}

func (s *ArrowSeries) Float(i int) float64 {
	if s.IsNull(i) {
		return 0
	}
	switch a := s.arr.(type) {
	case *array.Int8:
		return float64(a.Value(i))
	case *array.Int16:
		return float64(a.Value(i))
	case *array.Int32:
		return float64(a.Value(i))
	case *array.Int64:
		return float64(a.Value(i))
	case *array.Uint8:
		return float64(a.Value(i))
	case *array.Uint16:
		return float64(a.Value(i))
	case *array.Uint32:
		return float64(a.Value(i))
	case *array.Uint64:
		return float64(a.Value(i))
	case *array.Float32:
		return float64(a.Value(i))
	case *array.Float64:
		return a.Value(i)
	}
	return 0
}

func (s *ArrowSeries) String(i int) string {
	if s.IsNull(i) {
		return ""
	}
	if a, ok := s.arr.(*array.String); ok {
		return a.Value(i)
	}
	return fmt.Sprintf("%v", s.arr.(*array.Int64).Value(i))
}

func (s *ArrowSeries) Head(n int) core.Series {
	if n > s.Len() {
		n = s.Len()
	}
	sliced := array.NewSlice(s.arr, 0, int64(n))
	defer sliced.Release()
	return NewArrowSeries(s.name, array.MakeFromData(sliced.Data()), s.index.Slice(0, n))
}

func (s *ArrowSeries) Tail(n int) core.Series {
	if n > s.Len() {
		n = s.Len()
	}
	start := s.Len() - n
	sliced := array.NewSlice(s.arr, int64(start), int64(s.Len()))
	defer sliced.Release()
	return NewArrowSeries(s.name, array.MakeFromData(sliced.Data()), s.index.Slice(start, s.Len()))
}

func (s *ArrowSeries) Slice(start, end int) core.Series {
	sliced := array.NewSlice(s.arr, int64(start), int64(end))
	defer sliced.Release()
	return NewArrowSeries(s.name, array.MakeFromData(sliced.Data()), s.index.Slice(start, end))
}

func (s *ArrowSeries) Filter(mask []bool) core.Series {
	if len(mask) != s.Len() {
		// Defensive: return empty series instead of panic
		alloc := memory.NewGoAllocator()
		bldr := array.NewBuilder(alloc, s.arr.DataType())
		defer bldr.Release()
		return NewArrowSeries(s.name, bldr.NewArray(), core.NewDefaultIndex(0))
	}
	indices := make([]int, 0)
	for i, m := range mask {
		if m {
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

func (s *ArrowSeries) Take(indices []int) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	defer bldr.Release()
	bldr.Resize(len(indices))

	for _, idx := range indices {
		if s.IsNull(idx) {
			bldr.AppendNull()
		} else {
			switch s.arr.(type) {
			case *array.Boolean:
				bldr.(*array.BooleanBuilder).Append(s.Bool(idx))
			case *array.Int8:
				bldr.(*array.Int8Builder).Append(int8(s.Int(idx)))
			case *array.Int16:
				bldr.(*array.Int16Builder).Append(int16(s.Int(idx)))
			case *array.Int32:
				bldr.(*array.Int32Builder).Append(int32(s.Int(idx)))
			case *array.Int64:
				bldr.(*array.Int64Builder).Append(s.Int(idx))
			case *array.Uint8:
				bldr.(*array.Uint8Builder).Append(uint8(s.Int(idx)))
			case *array.Uint16:
				bldr.(*array.Uint16Builder).Append(uint16(s.Int(idx)))
			case *array.Uint32:
				bldr.(*array.Uint32Builder).Append(uint32(s.Int(idx)))
			case *array.Uint64:
				bldr.(*array.Uint64Builder).Append(uint64(s.Int(idx)))
			case *array.Float32:
				bldr.(*array.Float32Builder).Append(float32(s.Float(idx)))
			case *array.Float64:
				bldr.(*array.Float64Builder).Append(s.Float(idx))
			case *array.String:
				bldr.(*array.StringBuilder).Append(s.String(idx))
			}
		}
	}

	newIndex := s.reindex(indices)
	return NewArrowSeries(s.name, bldr.NewArray(), newIndex)
}

func (s *ArrowSeries) ToSlice() []interface{} {
	result := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			result[i] = nil
			continue
		}
		switch s.arr.(type) {
		case *array.Boolean:
			result[i] = s.Bool(i)
		case *array.Int8, *array.Int16, *array.Int32, *array.Int64,
			*array.Uint8, *array.Uint16, *array.Uint32, *array.Uint64:
			result[i] = s.Int(i)
		case *array.Float32, *array.Float64:
			result[i] = s.Float(i)
		case *array.String:
			result[i] = s.String(i)
		default:
			result[i] = nil
		}
	}
	return result
}

func (s *ArrowSeries) Copy() core.Series {
	return NewArrowSeries(s.name, array.MakeFromData(s.arr.Data()), s.index.Copy())
}

func (s *ArrowSeries) SetName(name string) core.Series {
	return NewArrowSeries(name, s.arr, s.index)
}

// reindex builds a new index from the original index using the given positions.
func (s *ArrowSeries) reindex(positions []int) core.Index {
	switch s.index.(type) {
	case *core.RangeIndex:
		return core.NewRangeIndex(0, len(positions))
	default:
		strs := make([]string, len(positions))
		for i, p := range positions {
			strs[i] = s.index.Get(p)
		}
		return core.NewStringIndex(strs)
	}
}

// GoString returns a human-readable representation of the series.
func (s *ArrowSeries) GoString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Series[%s] (%s) len=%d\n", s.name, s.Dtype(), s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			fmt.Fprintf(&b, "  %s: null\n", s.index.Get(i))
		} else {
			switch s.arr.(type) {
			case *array.Boolean:
				fmt.Fprintf(&b, "  %s: %v\n", s.index.Get(i), s.Bool(i))
			case *array.Float32, *array.Float64:
				fmt.Fprintf(&b, "  %s: %g\n", s.index.Get(i), s.Float(i))
			case *array.String:
				fmt.Fprintf(&b, "  %s: %q\n", s.index.Get(i), s.String(i))
			default:
				fmt.Fprintf(&b, "  %s: %d\n", s.index.Get(i), s.Int(i))
			}
		}
	}
	return b.String()
}

// Fmt returns a formatted table string.
// If elements ≤ 20, shows all; otherwise shows top 10 + bottom 3.
func (s *ArrowSeries) Fmt() string {
	total := s.Len()
	if total <= 20 {
		return s.Display(total, 0)
	}
	return s.Display(10, 3)
}

// Display returns a formatted table string showing top elements and bottom elements.
func (s *ArrowSeries) Display(top, bottom int) string {
	if top <= 0 {
		top = 5
	}
	if bottom <= 0 {
		bottom = 5
	}

	total := s.Len()
	showTop := top
	showBottom := bottom
	truncated := total > showTop+showBottom
	if showTop > total {
		showTop = total
		showBottom = 0
		truncated = false
	} else if showTop+showBottom > total {
		showBottom = total - showTop
		truncated = false
	}

	// Index column width
	idxW := strWidth(fmt.Sprintf("%d", total-1))
	if idxW < 3 {
		idxW = 3
	}

	// Value column width
	const maxValW = 30
	valW := strWidth(s.name)
	if valW < 4 {
		valW = 4
	}
	checkRows := func(start, end int) {
		for i := start; i < end; i++ {
			if s.IsNull(i) {
				continue
			}
			var v string
			switch s.arr.(type) {
			case *array.Boolean:
				v = fmt.Sprintf("%v", s.Bool(i))
			case *array.Float32, *array.Float64:
				v = fmt.Sprintf("%g", s.Float(i))
			case *array.String:
				v = s.String(i)
			default:
				v = fmt.Sprintf("%d", s.Int(i))
			}
			if vw := strWidth(v); vw > valW {
				valW = vw
			}
		}
	}
	checkRows(0, showTop)
	if truncated && showBottom > 0 {
		checkRows(total-showBottom, total)
	}

	// Fit within terminal width
	maxW := terminalWidth()
	needW := idxW + valW + 7 // "│ " + idx + " │ " + val + " │"
	if needW > maxW {
		valW = maxW - idxW - 7
		if valW < 4 {
			valW = 4
		}
	}

	formatVal := func(i int) string {
		if s.IsNull(i) {
			return ""
		}
		switch s.arr.(type) {
		case *array.Boolean:
			return fmt.Sprintf("%v", s.Bool(i))
		case *array.Float32, *array.Float64:
			return fmt.Sprintf("%g", s.Float(i))
		case *array.String:
			return s.String(i)
		default:
			return fmt.Sprintf("%d", s.Int(i))
		}
	}

	var b strings.Builder

	// Header
	b.WriteString("┌" + strings.Repeat("─", idxW+2) + "┬─" + strings.Repeat("─", valW) + "─┐\n")
	b.WriteString("│ " + padLeft("", idxW) + " │ " + padRight(truncate(s.name, valW), valW) + " │\n")
	b.WriteString("├" + strings.Repeat("─", idxW+2) + "┼─" + strings.Repeat("·", valW) + "─┤\n")

	writeRow := func(i int) {
		v := truncate(formatVal(i), valW)
		b.WriteString("│ " + padLeft(fmt.Sprintf("%d", i), idxW) + " │ " + padRight(v, valW) + " │\n")
	}

	for i := 0; i < showTop; i++ {
		writeRow(i)
	}

	if truncated && showBottom > 0 {
		b.WriteString("│ " + padLeft("...", idxW) + " │ " + padRight("...", valW) + " │\n")
		for i := total - showBottom; i < total; i++ {
			writeRow(i)
		}
	}

	b.WriteString("└" + strings.Repeat("─", idxW+2) + "┴─" + strings.Repeat("─", valW) + "─┘\n")
	b.WriteString(fmt.Sprintf("%s: %d elements", s.name, total))
	if truncated {
		b.WriteString(fmt.Sprintf(" (showing %d+%d of %d)", showTop, showBottom, total))
	}

	return b.String()
}
