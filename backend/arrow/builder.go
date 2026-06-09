package arrow

import (
	"fmt"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// SeriesBuilder constructs an ArrowSeries incrementally.
type SeriesBuilder struct {
	name  string
	alloc memory.Allocator
	bldr  array.Builder
	index core.Index
}

// NewSeriesBuilder creates a builder for the given dtype.
func NewSeriesBuilder(name string, dt core.DType, alloc memory.Allocator) *SeriesBuilder {
	if alloc == nil {
		alloc = memory.NewGoAllocator()
	}
	return &SeriesBuilder{
		name:  name,
		alloc: alloc,
		bldr:  newBuilder(dt, alloc),
	}
}

func newBuilder(dt core.DType, alloc memory.Allocator) array.Builder {
	switch dt {
	case core.BOOL:
		return array.NewBooleanBuilder(alloc)
	case core.INT8:
		return array.NewInt8Builder(alloc)
	case core.INT16:
		return array.NewInt16Builder(alloc)
	case core.INT32:
		return array.NewInt32Builder(alloc)
	case core.INT64:
		return array.NewInt64Builder(alloc)
	case core.UINT8:
		return array.NewUint8Builder(alloc)
	case core.UINT16:
		return array.NewUint16Builder(alloc)
	case core.UINT32:
		return array.NewUint32Builder(alloc)
	case core.UINT64:
		return array.NewUint64Builder(alloc)
	case core.FLOAT32:
		return array.NewFloat32Builder(alloc)
	case core.FLOAT64:
		return array.NewFloat64Builder(alloc)
	case core.STRING:
		return array.NewStringBuilder(alloc)
	default:
		return array.NewStringBuilder(alloc)
	}
}

// AppendBool appends a bool value.
func (b *SeriesBuilder) AppendBool(v bool) {
	b.bldr.(*array.BooleanBuilder).Append(v)
}

// AppendInt appends an int64 value (cast to the target dtype).
func (b *SeriesBuilder) AppendInt(v int64) {
	switch bd := b.bldr.(type) {
	case *array.Int8Builder:
		bd.Append(int8(v))
	case *array.Int16Builder:
		bd.Append(int16(v))
	case *array.Int32Builder:
		bd.Append(int32(v))
	case *array.Int64Builder:
		bd.Append(v)
	case *array.Uint8Builder:
		bd.Append(uint8(v))
	case *array.Uint16Builder:
		bd.Append(uint16(v))
	case *array.Uint32Builder:
		bd.Append(uint32(v))
	case *array.Uint64Builder:
		bd.Append(uint64(v))
	case *array.Float32Builder:
		bd.Append(float32(v))
	case *array.Float64Builder:
		bd.Append(float64(v))
	case *array.StringBuilder:
		bd.Append(formatInt(v))
	}
}

// AppendFloat appends a float64 value.
func (b *SeriesBuilder) AppendFloat(v float64) {
	switch bd := b.bldr.(type) {
	case *array.Float32Builder:
		bd.Append(float32(v))
	case *array.Float64Builder:
		bd.Append(v)
	case *array.Int64Builder:
		bd.Append(int64(v))
	case *array.StringBuilder:
		bd.Append(formatFloat(v))
	}
}

// AppendString appends a string value.
func (b *SeriesBuilder) AppendString(v string) {
	b.bldr.(*array.StringBuilder).Append(v)
}

// AppendNull appends a null value.
func (b *SeriesBuilder) AppendNull() {
	b.bldr.AppendNull()
}

// Len returns the current number of appended values.
func (b *SeriesBuilder) Len() int {
	return b.bldr.Len()
}

// Build finalizes and returns an ArrowSeries. The builder is reset after this call.
func (b *SeriesBuilder) Build() *ArrowSeries {
	arr := b.bldr.NewArray()
	idx := b.index
	if idx == nil {
		idx = core.NewDefaultIndex(arr.Len())
	}
	return NewArrowSeries(b.name, arr, idx)
}

// SetIndex sets the index for the resulting series.
func (b *SeriesBuilder) SetIndex(idx core.Index) *SeriesBuilder {
	b.index = idx
	return b
}

// --- helper formatters ---

func formatInt(v int64) string {
	return fmt.Sprintf("%d", v)
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%g", v)
}

// --- convenience constructors ---

// NewInt64Series creates an Int64 series from a Go slice.
func NewInt64Series(name string, data []int64, index core.Index) *ArrowSeries {
	b := array.NewInt64Builder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(data))
	for _, v := range data {
		b.Append(v)
	}
	return NewArrowSeries(name, b.NewArray(), index)
}

// NewFloat64Series creates a Float64 series from a Go slice.
func NewFloat64Series(name string, data []float64, index core.Index) *ArrowSeries {
	b := array.NewFloat64Builder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(data))
	for _, v := range data {
		b.Append(v)
	}
	return NewArrowSeries(name, b.NewArray(), index)
}

// NewStringSeries creates a String series from a Go slice.
func NewStringSeries(name string, data []string, index core.Index) *ArrowSeries {
	b := array.NewStringBuilder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(data))
	for _, v := range data {
		b.Append(v)
	}
	return NewArrowSeries(name, b.NewArray(), index)
}

// NewBoolSeries creates a Bool series from a Go slice.
func NewBoolSeries(name string, data []bool, index core.Index) *ArrowSeries {
	b := array.NewBooleanBuilder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(data))
	for _, v := range data {
		b.Append(v)
	}
	return NewArrowSeries(name, b.NewArray(), index)
}

// NewInt64SeriesWithNulls creates an Int64 series with null bitmap.
func NewInt64SeriesWithNulls(name string, data []int64, valid []bool, index core.Index) *ArrowSeries {
	b := array.NewInt64Builder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(data))
	for i, v := range data {
		if i < len(valid) && !valid[i] {
			b.AppendNull()
		} else {
			b.Append(v)
		}
	}
	return NewArrowSeries(name, b.NewArray(), index)
}
