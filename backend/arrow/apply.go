package arrow

import (
	"fmt"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// --- Series Apply/Map/Transform ---

// ApplyFunc is a function that transforms a single value.
// Input is interface{} (nil for null), output is interface{} (nil for null).
type ApplyFunc func(val interface{}) interface{}

// MapFloatFunc transforms a float64 to float64.
type MapFloatFunc func(float64) float64

// MapStringFunc transforms a string to string.
type MapStringFunc func(string) string

// MapBoolFunc transforms a bool to bool.
type MapBoolFunc func(bool) bool

// Apply applies a function to each element of the series.
func (s *ArrowSeries) Apply(fn ApplyFunc) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc) // Store results as string initially
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		var val interface{}
		if s.IsNull(i) {
			val = nil
		} else {
			switch s.Dtype() {
			case core.BOOL:
				val = s.Bool(i)
			case core.FLOAT32, core.FLOAT64:
				val = s.Float(i)
			case core.STRING:
				val = s.String(i)
			default:
				val = s.Int(i)
			}
		}

		result := fn(val)
		if result == nil {
			bldr.AppendNull()
		} else {
			bldr.Append(fmt.Sprintf("%v", result))
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// MapFloat applies a float64->float64 function to each element.
func (s *ArrowSeries) MapFloat(fn MapFloatFunc) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.Float(i)))
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// MapString applies a string->string function to each element.
func (s *ArrowSeries) MapString(fn MapStringFunc) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.String(i)))
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// MapBool applies a bool->bool function to each element.
func (s *ArrowSeries) MapBool(fn MapBoolFunc) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.Bool(i)))
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// --- DataFrame Apply/Transform ---

// ColApplyFunc transforms a column (Series) into a new Series.
type ColApplyFunc func(col core.Series) core.Series

// RowApplyFunc transforms a row (map of column name to value) into a new row.
type RowApplyFunc func(row map[string]interface{}) map[string]interface{}

// ApplyCols applies a function to each column.
func (df *ArrowDataFrame) ApplyCols(fn ColApplyFunc) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		result := fn(df.Col(name))
		series[i] = result.(*ArrowSeries)
	}
	return NewDataFrame(series...)
}

// Transform applies a function to each numeric column, leaving non-numeric columns unchanged.
func (df *ArrowDataFrame) Transform(fn MapFloatFunc) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		col := df.Col(name)
		if col.Dtype().IsNumeric() {
			series[i] = col.(*ArrowSeries).MapFloat(fn).(*ArrowSeries)
		} else {
			series[i] = col.(*ArrowSeries)
		}
	}
	return NewDataFrame(series...)
}

// ApplyRows applies a function to each row, producing a new DataFrame.
func (df *ArrowDataFrame) ApplyRows(fn RowApplyFunc) core.DataFrame {
	rows, _ := df.Shape()
	colNames := df.Columns()

	// Apply function to each row
	newRows := make([]map[string]interface{}, rows)
	for i := 0; i < rows; i++ {
		row := make(map[string]interface{}, len(colNames))
		for _, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				row[name] = nil
			} else {
				switch s.Dtype() {
				case core.BOOL:
					row[name] = s.Bool(i)
				case core.FLOAT32, core.FLOAT64:
					row[name] = s.Float(i)
				case core.STRING:
					row[name] = s.String(i)
				default:
					row[name] = s.Int(i)
				}
			}
		}
		newRows[i] = fn(row)
	}

	// Build new DataFrame from transformed rows
	return buildFromMapsHelper(newRows)
}

// buildFromMapsHelper builds a DataFrame from maps (used by ApplyRows).
func buildFromMapsHelper(rows []map[string]interface{}) *ArrowDataFrame {
	if len(rows) == 0 {
		return NewDataFrame()
	}

	// Collect column names
	colOrder := make([]string, 0)
	seen := make(map[string]bool)
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				colOrder = append(colOrder, k)
			}
		}
	}

	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, len(colOrder))

	for j, name := range colOrder {
		// Infer type from first non-nil value
		dt := core.STRING
		for _, row := range rows {
			if v, ok := row[name]; ok && v != nil {
				switch v.(type) {
				case bool:
					dt = core.BOOL
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					dt = core.INT64
				case float32, float64:
					dt = core.FLOAT64
				}
				break
			}
		}

		switch dt {
		case core.BOOL:
			bldr := array.NewBooleanBuilder(alloc)
			bldr.Resize(len(rows))
			for _, row := range rows {
				if v, ok := row[name]; ok && v != nil {
					if bv, ok := v.(bool); ok {
						bldr.Append(bv)
					} else {
						bldr.AppendNull()
					}
				} else {
					bldr.AppendNull()
				}
			}
			series[j] = NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()

		case core.INT64:
			bldr := array.NewInt64Builder(alloc)
			bldr.Resize(len(rows))
			for _, row := range rows {
				if v, ok := row[name]; ok && v != nil {
					bldr.Append(toInt64Val(v))
				} else {
					bldr.AppendNull()
				}
			}
			series[j] = NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()

		case core.FLOAT64:
			bldr := array.NewFloat64Builder(alloc)
			bldr.Resize(len(rows))
			for _, row := range rows {
				if v, ok := row[name]; ok && v != nil {
					bldr.Append(toFloat64Val(v))
				} else {
					bldr.AppendNull()
				}
			}
			series[j] = NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()

		default:
			bldr := array.NewStringBuilder(alloc)
			bldr.Resize(len(rows))
			for _, row := range rows {
				if v, ok := row[name]; ok && v != nil {
					bldr.Append(fmt.Sprintf("%v", v))
				} else {
					bldr.AppendNull()
				}
			}
			series[j] = NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()
		}
	}

	return NewDataFrame(series...)
}

// --- Helper type conversions (shared with io package) ---

func toInt64Val(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	}
	return 0
}

func toFloat64Val(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	}
	return 0
}
