package arrow

import (
	"fmt"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

// --- Series.isin ---

// IsIn returns a boolean Series where true means the value exists in the given set.
func (s *ArrowSeries) IsIn(values []interface{}) core.Series {
	// Build lookup set
	set := make(map[interface{}]bool, len(values))
	for _, v := range values {
		set[v] = true
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			var key interface{}
			switch s.Dtype() {
			case core.BOOL:
				key = s.Bool(i)
			case core.STRING:
				key = s.String(i)
			case core.FLOAT32, core.FLOAT64:
				key = s.Float(i)
			default:
				key = s.Int(i)
			}
			bldr.Append(set[key])
		}
	}

	return NewArrowSeries(s.Name()+"_in", bldr.NewArray(), s.Index())
}

// --- Series.value_counts ---

// ValueCountsResult holds the result of value_counts.
type ValueCountsResult struct {
	Values core.Series
	Counts core.Series
}

// ValueCounts counts the occurrences of each unique value.
func (s *ArrowSeries) ValueCounts() *ValueCountsResult {
	counts := make(map[interface{}]int)
	order := make([]interface{}, 0)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		if _, exists := counts[key]; !exists {
			order = append(order, key)
		}
		counts[key]++
	}

	alloc := memory.NewGoAllocator()

	// Build values series
	valBldr := array.NewBuilder(alloc, s.arr.DataType())
	valBldr.Resize(len(order))
	for _, v := range order {
		switch s.Dtype() {
		case core.BOOL:
			valBldr.(*array.BooleanBuilder).Append(v.(bool))
		case core.STRING:
			valBldr.(*array.StringBuilder).Append(v.(string))
		case core.FLOAT32:
			valBldr.(*array.Float32Builder).Append(float32(v.(float64)))
		case core.FLOAT64:
			valBldr.(*array.Float64Builder).Append(v.(float64))
		default:
			valBldr.(*array.Int64Builder).Append(v.(int64))
		}
	}

	// Build counts series
	cntBldr := array.NewInt64Builder(alloc)
	cntBldr.Resize(len(order))
	for _, v := range order {
		cntBldr.Append(int64(counts[v]))
	}

	return &ValueCountsResult{
		Values: NewArrowSeries(s.Name(), valBldr.NewArray(), nil),
		Counts: NewArrowSeries("count", cntBldr.NewArray(), nil),
	}
}

// NUnique returns the number of unique values.
func (s *ArrowSeries) NUnique() int {
	seen := make(map[interface{}]bool)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		seen[key] = true
	}
	return len(seen)
}

// Unique returns a Series of unique values.
func (s *ArrowSeries) Unique() core.Series {
	seen := make(map[interface{}]bool)
	order := make([]interface{}, 0)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		if !seen[key] {
			seen[key] = true
			order = append(order, key)
		}
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(len(order))
	for _, v := range order {
		switch s.Dtype() {
		case core.BOOL:
			bldr.(*array.BooleanBuilder).Append(v.(bool))
		case core.STRING:
			bldr.(*array.StringBuilder).Append(v.(string))
		case core.FLOAT32:
			bldr.(*array.Float32Builder).Append(float32(v.(float64)))
		case core.FLOAT64:
			bldr.(*array.Float64Builder).Append(v.(float64))
		default:
			bldr.(*array.Int64Builder).Append(v.(int64))
		}
	}

	return NewArrowSeries(s.Name()+"_unique", bldr.NewArray(), nil)
}

// --- Series.duplicated ---

// Duplicated returns a boolean Series marking duplicate values.
// keep: "first" marks first occurrence as false, "last" marks last as false, "none" marks all duplicates as true.
func (s *ArrowSeries) Duplicated(keep string) core.Series {
	seen := make(map[interface{}]int) // key -> first index
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())

	// First pass: find duplicates
	type entry struct {
		key interface{}
		idx int
	}
	entries := make([]entry, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			entries[i] = entry{nil, i}
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		entries[i] = entry{key, i}
		if _, exists := seen[key]; !exists {
			seen[key] = i
		}
	}

	// Track last occurrence for keep="last"
	lastSeen := make(map[interface{}]int)
	for i := s.Len() - 1; i >= 0; i-- {
		if entries[i].key != nil {
			if _, exists := lastSeen[entries[i].key]; !exists {
				lastSeen[entries[i].key] = i
			}
		}
	}

	for i, e := range entries {
		if e.key == nil {
			bldr.Append(false) // nulls are never duplicates
			continue
		}
		firstIdx := seen[e.key]
		lastIdx := lastSeen[e.key]

		switch keep {
		case "first":
			bldr.Append(i != firstIdx)
		case "last":
			bldr.Append(i != lastIdx)
		default: // "none"
			bldr.Append(firstIdx != lastIdx) // true if appears more than once
		}
	}

	return NewArrowSeries(s.Name()+"_dup", bldr.NewArray(), s.Index())
}

// DropDuplicates returns a Series with duplicates removed.
func (s *ArrowSeries) DropDuplicates(keep string) core.Series {
	dup := s.Duplicated(keep).(*ArrowSeries)
	indices := make([]int, 0)
	for i := 0; i < dup.Len(); i++ {
		if !dup.Bool(i) {
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

// --- DataFrame.isin ---

// IsIn returns a boolean DataFrame where true means the value exists in the given set per column.
func (df *ArrowDataFrame) IsIn(values map[string][]interface{}) core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		if vals, ok := values[name]; ok {
			series[i] = df.Col(name).(*ArrowSeries).IsIn(vals).(*ArrowSeries)
		} else {
			// No filter for this column → all false
			alloc := memory.NewGoAllocator()
			bldr := array.NewBooleanBuilder(alloc)
			bldr.Resize(df.Len())
			for j := 0; j < df.Len(); j++ {
				bldr.Append(false)
			}
			series[i] = NewArrowSeries(name, bldr.NewArray(), nil)
		}
	}
	return NewDataFrame(series...)
}

// --- DataFrame.duplicated ---

// Duplicated returns a boolean Series marking duplicate rows based on the given columns.
func (df *ArrowDataFrame) Duplicated(cols []string, keep string) core.Series {
	rows, _ := df.Shape()

	// Build composite key for each row
	keys := make([]string, rows)
	for i := 0; i < rows; i++ {
		var key string
		for j, name := range cols {
			s := df.Col(name).(*ArrowSeries)
			if j > 0 {
				key += "\x00"
			}
			if s.IsNull(i) {
				key += "<nil>"
			} else {
				switch s.Dtype() {
				case core.BOOL:
					key += fmt.Sprintf("%v", s.Bool(i))
				case core.STRING:
					key += s.String(i)
				case core.FLOAT32, core.FLOAT64:
					key += fmt.Sprintf("%g", s.Float(i))
				default:
					key += fmt.Sprintf("%d", s.Int(i))
				}
			}
		}
		keys[i] = key
	}

	seen := make(map[string]int)
	lastSeen := make(map[string]int)
	for i, k := range keys {
		if _, exists := seen[k]; !exists {
			seen[k] = i
		}
	}
	for i := len(keys) - 1; i >= 0; i-- {
		if _, exists := lastSeen[keys[i]]; !exists {
			lastSeen[keys[i]] = i
		}
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(rows)
	for i, k := range keys {
		firstIdx := seen[k]
		lastIdx := lastSeen[k]
		switch keep {
		case "first":
			bldr.Append(i != firstIdx)
		case "last":
			bldr.Append(i != lastIdx)
		default:
			bldr.Append(firstIdx != lastIdx)
		}
	}
	return NewArrowSeries("duplicated", bldr.NewArray(), df.Index())
}

// DropDuplicates returns a DataFrame with duplicate rows removed.
func (df *ArrowDataFrame) DropDuplicates(cols []string, keep string) core.DataFrame {
	dup := df.Duplicated(cols, keep).(*ArrowSeries)
	indices := make([]int, 0)
	for i := 0; i < dup.Len(); i++ {
		if !dup.Bool(i) {
			indices = append(indices, i)
		}
	}
	return df.Take(indices)
}
