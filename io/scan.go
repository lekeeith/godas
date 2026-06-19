package io

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/backend/arrow"
	"github.com/lekeeith/godas/core"
	"github.com/parquet-go/parquet-go"
)

// ScanCSV lazily reads a CSV file with Polars-style optimizations:
//   - Projection pushdown: only read needed columns
//   - Predicate pushdown: filter during scan, not after
//   - Streaming: no full materialization, only keep matched rows
type ScanCSV struct {
	path    string
	columns []string // projection (empty = all)
	filters []scanFilter
	limit   int
	offset  int // skip first N matched rows (for resume)
}

// ScanCSVFile creates a lazy CSV scanner.
func ScanCSVFile(path string) *ScanCSV {
	return &ScanCSV{path: path}
}

// Select specifies columns to read (projection pushdown).
func (s *ScanCSV) Select(columns ...string) *ScanCSV {
	s.columns = columns
	return s
}

// Filter adds a predicate filter (col op val).
func (s *ScanCSV) Filter(col, op string, val interface{}) *ScanCSV {
	s.filters = append(s.filters, scanFilter{col: col, op: op, val: val})
	return s
}

// Limit limits returned rows.
func (s *ScanCSV) Limit(n int) *ScanCSV {
	s.limit = n
	return s
}

// Offset skips the first N matched rows (for resume after error).
func (s *ScanCSV) Offset(n int) *ScanCSV {
	s.offset = n
	return s
}

// Collect executes the lazy scan and returns a DataFrame.
// Polars-style: stream rows, filter on-the-fly, only store matched projected columns.
func (s *ScanCSV) Collect() (*arrow.ArrowDataFrame, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", s.path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	numCols := len(header)

	// Build column index map
	colIdx := make(map[string]int, numCols)
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	// Validate filter columns exist
	filterColIdx := make([]int, len(s.filters))
	filterVals := make([]string, len(s.filters))
	for i, flt := range s.filters {
		idx, ok := colIdx[flt.col]
		if !ok {
			return nil, fmt.Errorf("filter column %q not found", flt.col)
		}
		filterColIdx[i] = idx
		filterVals[i], _ = flt.val.(string)
	}

	// Determine projection columns
	projCols := s.columns
	if len(projCols) == 0 {
		projCols = make([]string, numCols)
		copy(projCols, header)
	}
	projColIdx := make([]int, len(projCols))
	for j, name := range projCols {
		idx, ok := colIdx[strings.TrimSpace(name)]
		if !ok {
			return nil, fmt.Errorf("column %q not found", name)
		}
		projColIdx[j] = idx
	}

	// Stream rows: filter on-the-fly, only store matched projected values
	// Each column collector is a []string — grow as matched rows come in
	collected := make([][]string, len(projCols))
	matchCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		// Apply all filters (predicate pushdown at scan level)
		match := true
		for i, flt := range s.filters {
			sv := strings.TrimSpace(record[filterColIdx[i]])
			switch flt.op {
			case "==":
				match = sv == filterVals[i]
			case "!=":
				match = sv != filterVals[i]
			case ">":
				match = sv > filterVals[i]
			case "<":
				match = sv < filterVals[i]
			case ">=":
				match = sv >= filterVals[i]
			case "<=":
				match = sv <= filterVals[i]
			default:
				match = sv == filterVals[i]
			}
			if !match {
				break
			}
		}
		if !match {
			continue
		}

		// Row matched — collect only projected columns
		matchCount++
		for j, idx := range projColIdx {
			collected[j] = append(collected[j], strings.TrimSpace(record[idx]))
		}

		// Apply limit
		if s.limit > 0 && matchCount >= s.limit {
			break
		}
	}

	// Build Arrow arrays from collected strings
	alloc := memory.NewGoAllocator()
	series := make([]*arrow.ArrowSeries, len(projCols))
	for j, name := range projCols {
		dt := inferColumnTypeFromStrings(collected[j])
		series[j] = buildColumnFromStrings(strings.TrimSpace(name), dt, collected[j], alloc)
	}

	return arrow.NewDataFrame(series...), nil
}

// inferColumnTypeFromStrings infers the DType from string values.
func inferColumnTypeFromStrings(vals []string) core.DType {
	boolCount, intCount, floatCount, total := 0, 0, 0, 0
	sampleSize := len(vals)
	if sampleSize > 100 {
		sampleSize = 100
	}
	for i := 0; i < sampleSize; i++ {
		val := strings.TrimSpace(vals[i])
		if val == "" {
			continue
		}
		total++
		lower := strings.ToLower(val)
		if lower == "true" || lower == "false" || lower == "yes" || lower == "no" {
			boolCount++
			continue
		}
		if _, err := strconv.ParseInt(val, 10, 64); err == nil {
			intCount++
			continue
		}
		if _, err := strconv.ParseFloat(val, 64); err == nil {
			floatCount++
			continue
		}
	}
	if total == 0 {
		return core.STRING
	}
	threshold := float64(total) * 0.8
	if float64(boolCount) >= threshold {
		return core.BOOL
	}
	if float64(intCount) >= threshold {
		return core.INT64
	}
	if float64(intCount+floatCount) >= threshold {
		return core.FLOAT64
	}
	return core.STRING
}

// buildColumnFromStrings builds an ArrowSeries from string values with type inference.
func buildColumnFromStrings(name string, dt core.DType, vals []string, alloc memory.Allocator) *arrow.ArrowSeries {
	n := len(vals)
	switch dt {
	case core.INT64:
		bldr := array.NewInt64Builder(alloc)
		bldr.Resize(n)
		for _, v := range vals {
			if v == "" {
				bldr.AppendNull()
			} else if iv, err := strconv.ParseInt(v, 10, 64); err == nil {
				bldr.Append(iv)
			} else {
				bldr.AppendNull()
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	case core.FLOAT64:
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(n)
		for _, v := range vals {
			if v == "" {
				bldr.AppendNull()
			} else if fv, err := strconv.ParseFloat(v, 64); err == nil {
				bldr.Append(fv)
			} else {
				bldr.AppendNull()
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	case core.BOOL:
		bldr := array.NewBooleanBuilder(alloc)
		bldr.Resize(n)
		for _, v := range vals {
			lower := strings.ToLower(v)
			switch lower {
			case "true", "1", "yes":
				bldr.Append(true)
			case "false", "0", "no":
				bldr.Append(false)
			default:
				bldr.AppendNull()
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	default:
		bldr := array.NewStringBuilder(alloc)
		bldr.Resize(n)
		for _, v := range vals {
			if v == "" {
				bldr.AppendNull()
			} else {
				bldr.Append(v)
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	}
}

// ForEach streams matched rows in chunks, calling fn for each chunk.
// Memory usage is O(chunkSize × projCols), independent of total file size.
// Returns (processedRows, error). On error, processedRows tells how many matched rows
// were successfully sent to fn — use Offset(processedRows) to resume from where it left off.
func (s *ScanCSV) ForEach(chunkSize int, fn func(chunk *arrow.ArrowDataFrame) error) (int, error) {
	if chunkSize <= 0 {
		chunkSize = 10000
	}

	f, err := os.Open(s.path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", s.path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	numCols := len(header)

	colIdx := make(map[string]int, numCols)
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	// Validate filter columns
	filterColIdx := make([]int, len(s.filters))
	filterVals := make([]string, len(s.filters))
	for i, flt := range s.filters {
		idx, ok := colIdx[flt.col]
		if !ok {
			return 0, fmt.Errorf("filter column %q not found", flt.col)
		}
		filterColIdx[i] = idx
		filterVals[i], _ = flt.val.(string)
	}

	// Determine projection columns
	projCols := s.columns
	if len(projCols) == 0 {
		projCols = make([]string, numCols)
		copy(projCols, header)
	}
	projColIdx := make([]int, len(projCols))
	for j, name := range projCols {
		idx, ok := colIdx[strings.TrimSpace(name)]
		if !ok {
			return 0, fmt.Errorf("column %q not found", name)
		}
		projColIdx[j] = idx
	}
	numProj := len(projCols)

	// Chunk buffer
	collected := make([][]string, numProj)
	for j := range collected {
		collected[j] = make([]string, 0, chunkSize)
	}
	matchCount := 0
	totalSent := 0
	skipped := 0 // rows skipped due to Offset

	flush := func() error {
		if matchCount == 0 {
			return nil
		}
		alloc := memory.NewGoAllocator()
		series := make([]*arrow.ArrowSeries, numProj)
		for j, name := range projCols {
			dt := inferColumnTypeFromStrings(collected[j])
			series[j] = buildColumnFromStrings(strings.TrimSpace(name), dt, collected[j], alloc)
		}
		chunk := arrow.NewDataFrame(series...)
		if err := fn(chunk); err != nil {
			return err
		}
		totalSent += matchCount
		// Reset buffers
		for j := range collected {
			collected[j] = collected[j][:0]
		}
		matchCount = 0
		return nil
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Flush what we have so far before returning error
			if matchCount > 0 {
				if ferr := flush(); ferr != nil {
					return totalSent, fmt.Errorf("read row: %w (flush also failed: %v)", err, ferr)
				}
			}
			return totalSent, fmt.Errorf("read row: %w", err)
		}

		// Predicate pushdown
		match := true
		for i, flt := range s.filters {
			sv := strings.TrimSpace(record[filterColIdx[i]])
			switch flt.op {
			case "==":
				match = sv == filterVals[i]
			case "!=":
				match = sv != filterVals[i]
			case ">":
				match = sv > filterVals[i]
			case "<":
				match = sv < filterVals[i]
			case ">=":
				match = sv >= filterVals[i]
			case "<=":
				match = sv <= filterVals[i]
			default:
				match = sv == filterVals[i]
			}
			if !match {
				break
			}
		}
		if !match {
			continue
		}

		// Skip rows due to Offset (resume support)
		if skipped < s.offset {
			skipped++
			continue
		}

		// Collect projected columns
		for j, idx := range projColIdx {
			collected[j] = append(collected[j], strings.TrimSpace(record[idx]))
		}
		matchCount++

		// Flush when chunk is full
		if matchCount >= chunkSize {
			if err := flush(); err != nil {
				return totalSent, err
			}
		}

		// Apply limit
		if s.limit > 0 && totalSent+matchCount >= s.limit {
			break
		}
	}

	// Flush remaining rows
	if err := flush(); err != nil {
		return totalSent, err
	}
	return totalSent, nil
}

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
