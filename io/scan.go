package io

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/backend/arrow"
	"github.com/lekeeith/godas/core"
	"github.com/parquet-go/parquet-go"
)

// ScanParquet lazily reads a Parquet file with projection and predicate pushdown.
type ScanParquet struct {
	path    string
	columns []string
	filters []scanFilter
	limit   int
}

type scanFilter struct {
	col string
	op  string
	val interface{}
}

// Scan creates a lazy Parquet scanner.
func Scan(path string) *ScanParquet {
	return &ScanParquet{path: path}
}

// Select specifies columns to read (projection pushdown).
func (s *ScanParquet) Select(columns ...string) *ScanParquet {
	s.columns = columns
	return s
}

// Filter adds a predicate filter.
func (s *ScanParquet) Filter(col, op string, val interface{}) *ScanParquet {
	s.filters = append(s.filters, scanFilter{col: col, op: op, val: val})
	return s
}

// Limit limits returned rows.
func (s *ScanParquet) Limit(n int) *ScanParquet {
	s.limit = n
	return s
}

// Collect executes the scan and returns a DataFrame.
func (s *ScanParquet) Collect() (*arrow.ArrowDataFrame, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}

	pf, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open parquet: %w", err)
	}

	numRows := pf.NumRows()
	schema := pf.Schema()
	allColNames := extractScanColumnNames(schema)

	// Projection pushdown
	colNames := allColNames
	if len(s.columns) > 0 {
		want := make(map[string]bool)
		for _, c := range s.columns {
			want[c] = true
		}
		colNames = make([]string, 0)
		for _, c := range allColNames {
			if want[c] {
				colNames = append(colNames, c)
			}
		}
	}

	reader := parquet.NewReader(pf)
	defer reader.Close()

	allRows := make([]parquet.Row, 0, numRows)
	buf := make([]parquet.Row, 256)
	for {
		n, err := reader.ReadRows(buf)
		if n > 0 {
			rows := make([]parquet.Row, n)
			for i := 0; i < n; i++ {
				row := make(parquet.Row, len(buf[i]))
				copy(row, buf[i])
				rows[i] = row
			}
			allRows = append(allRows, rows...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read rows: %w", err)
		}
	}

	colIdxMap := make(map[string]int)
	for i, c := range allColNames {
		colIdxMap[c] = i
	}

	alloc := memory.NewGoAllocator()
	series := make([]*arrow.ArrowSeries, len(colNames))

	for j, name := range colNames {
		idx := colIdxMap[name]
		vals := make([]parquet.Value, len(allRows))
		for i, row := range allRows {
			if idx < len(row) {
				vals[i] = row[idx]
			}
		}
		series[j] = buildScanColumn(name, vals, alloc)
	}

	df := arrow.NewDataFrame(series...)

	// Predicate pushdown (post-read filter)
	for _, f := range s.filters {
		mask := applyScanFilter(df, f)
		df = df.Filter(mask).(*arrow.ArrowDataFrame)
	}

	if s.limit > 0 && df.Len() > s.limit {
		df = df.Slice(0, s.limit).(*arrow.ArrowDataFrame)
	}

	return df, nil
}

func extractScanColumnNames(schema *parquet.Schema) []string {
	var names []string
	for _, field := range schema.Fields() {
		collectScanLeaf(field, &names)
	}
	return names
}

func collectScanLeaf(field parquet.Field, names *[]string) {
	if field.Leaf() {
		*names = append(*names, field.Name())
	} else {
		for _, child := range field.Fields() {
			collectScanLeaf(child, names)
		}
	}
}

func buildScanColumn(name string, values []parquet.Value, alloc memory.Allocator) *arrow.ArrowSeries {
	n := len(values)
	if n == 0 {
		bldr := array.NewInt64Builder(alloc)
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	}

	dt := inferScanType(values)
	switch dt {
	case core.BOOL:
		bldr := array.NewBooleanBuilder(alloc)
		bldr.Resize(n)
		for _, v := range values {
			if v.IsNull() {
				bldr.AppendNull()
			} else {
				bldr.Append(v.Boolean())
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	case core.INT64:
		bldr := array.NewInt64Builder(alloc)
		bldr.Resize(n)
		for _, v := range values {
			if v.IsNull() {
				bldr.AppendNull()
			} else {
				bldr.Append(v.Int64())
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	case core.FLOAT64:
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(n)
		for _, v := range values {
			if v.IsNull() {
				bldr.AppendNull()
			} else {
				bldr.Append(v.Double())
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	default:
		bldr := array.NewStringBuilder(alloc)
		bldr.Resize(n)
		for _, v := range values {
			if v.IsNull() {
				bldr.AppendNull()
			} else {
				bldr.Append(v.String())
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	}
}

func inferScanType(values []parquet.Value) core.DType {
	sample := len(values)
	if sample > 100 {
		sample = 100
	}
	for i := 0; i < sample; i++ {
		if values[i].IsNull() {
			continue
		}
		switch values[i].Kind() {
		case parquet.Boolean:
			return core.BOOL
		case parquet.Int32, parquet.Int64:
			return core.INT64
		case parquet.Float, parquet.Double:
			return core.FLOAT64
		default:
			return core.STRING
		}
	}
	return core.STRING
}

func applyScanFilter(df *arrow.ArrowDataFrame, f scanFilter) []bool {
	rows, _ := df.Shape()
	mask := make([]bool, rows)
	s := df.Col(f.col)
	for i := 0; i < rows; i++ {
		if s.IsNull(i) {
			mask[i] = false
			continue
		}
		var cmp int
		switch v := f.val.(type) {
		case float64:
			sv := s.Float(i)
			if sv < v {
				cmp = -1
			} else if sv > v {
				cmp = 1
			}
		case int:
			sv := s.Int(i)
			iv := int64(v)
			if sv < iv {
				cmp = -1
			} else if sv > iv {
				cmp = 1
			}
		case int64:
			sv := s.Int(i)
			if sv < v {
				cmp = -1
			} else if sv > v {
				cmp = 1
			}
		case string:
			sv := s.String(i)
			if sv < v {
				cmp = -1
			} else if sv > v {
				cmp = 1
			}
		default:
			mask[i] = true
			continue
		}
		switch f.op {
		case "==":
			mask[i] = cmp == 0
		case "!=":
			mask[i] = cmp != 0
		case ">":
			mask[i] = cmp > 0
		case "<":
			mask[i] = cmp < 0
		case ">=":
			mask[i] = cmp >= 0
		case "<=":
			mask[i] = cmp <= 0
		default:
			mask[i] = true
		}
	}
	return mask
}
