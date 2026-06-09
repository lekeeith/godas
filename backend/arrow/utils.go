package arrow

import (
	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// ArrowToDType converts an Arrow DataType to a core.DType.
func ArrowToDType(dt arrow.DataType) core.DType {
	switch dt.ID() {
	case arrow.BOOL:
		return core.BOOL
	case arrow.INT8:
		return core.INT8
	case arrow.INT16:
		return core.INT16
	case arrow.INT32:
		return core.INT32
	case arrow.INT64:
		return core.INT64
	case arrow.UINT8:
		return core.UINT8
	case arrow.UINT16:
		return core.UINT16
	case arrow.UINT32:
		return core.UINT32
	case arrow.UINT64:
		return core.UINT64
	case arrow.FLOAT32:
		return core.FLOAT32
	case arrow.FLOAT64:
		return core.FLOAT64
	case arrow.STRING:
		return core.STRING
	case arrow.TIMESTAMP:
		return core.TIMESTAMP
	default:
		return core.STRING
	}
}

// DTypeToArrow converts a core.DType to an Arrow DataType.
func DTypeToArrow(dt core.DType) arrow.DataType {
	switch dt {
	case core.BOOL:
		return arrow.FixedWidthTypes.Boolean
	case core.INT8:
		return arrow.PrimitiveTypes.Int8
	case core.INT16:
		return arrow.PrimitiveTypes.Int16
	case core.INT32:
		return arrow.PrimitiveTypes.Int32
	case core.INT64:
		return arrow.PrimitiveTypes.Int64
	case core.UINT8:
		return arrow.PrimitiveTypes.Uint8
	case core.UINT16:
		return arrow.PrimitiveTypes.Uint16
	case core.UINT32:
		return arrow.PrimitiveTypes.Uint32
	case core.UINT64:
		return arrow.PrimitiveTypes.Uint64
	case core.FLOAT32:
		return arrow.PrimitiveTypes.Float32
	case core.FLOAT64:
		return arrow.PrimitiveTypes.Float64
	case core.STRING:
		return arrow.BinaryTypes.String
	default:
		return arrow.BinaryTypes.String
	}
}

// buildInt64Array creates an Arrow Int64 array from a Go slice.
func buildInt64Array(values []int) arrow.Array {
	b := array.NewInt64Builder(memory.NewGoAllocator())
	defer b.Release()
	b.Resize(len(values))
	for _, v := range values {
		b.Append(int64(v))
	}
	return b.NewArray()
}
