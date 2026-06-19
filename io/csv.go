package io

import (
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
)

// CSVOptions configures CSV reading behaviour.
// Zero values use sensible defaults: Comma=0 (meaning ','), SkipLines=0, HasHeader=false (meaning true).
//
// Because bool zero-value is false, we flip the semantics: the field is actually "NoHeader".
// NoHeader=false (zero) → first row is a header (the common case).
// NoHeader=true → no header row, columns are named col0, col1, ...
type CSVOptions struct {
	Comma     rune // field delimiter; 0 defaults to ','
	SkipLines int  // skip first N lines before the header (e.g. metadata/comment lines)
	NoHeader  bool // true = first row is data, not a header
}

var defaultCSVOptions = CSVOptions{
	Comma: ',',
}

func resolveCSVOptions(opts []CSVOptions) CSVOptions {
	if len(opts) == 0 {
		return defaultCSVOptions
	}
	o := opts[0]
	if o.Comma == 0 {
		o.Comma = ','
	}
	return o
}

// ReadCSV reads a CSV string into a DataFrame with automatic type inference.
func ReadCSV(data string, opts ...CSVOptions) (*arrow.ArrowDataFrame, error) {
	opt := resolveCSVOptions(opts)
	r := csv.NewReader(strings.NewReader(data))
	r.Comma = opt.Comma
	return readCSVReader(r, opt)
}

// ReadCSVFile reads a CSV file into a DataFrame.
func ReadCSVFile(path string, opts ...CSVOptions) (*arrow.ArrowDataFrame, error) {
	opt := resolveCSVOptions(opts)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = opt.Comma
	return readCSVReader(r, opt)
}

func readCSVReader(r *csv.Reader, opt CSVOptions) (*arrow.ArrowDataFrame, error) {
	// Allow variable field counts (e.g. after SkipLines, comment lines may differ)
	r.FieldsPerRecord = -1

	// Skip leading lines (metadata, comments, etc.)
	for i := 0; i < opt.SkipLines; i++ {
		if _, err := r.Read(); err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("csv has fewer lines than SkipLines=%d", opt.SkipLines)
			}
			return nil, fmt.Errorf("skip line %d: %w", i+1, err)
		}
	}

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	if !opt.NoHeader {
		// First row is header
		if len(records) < 2 {
			return nil, fmt.Errorf("csv must have at least a header and one data row")
		}
		headers := records[0]
		rows := records[1:]
		numCols := len(headers)
		numRows := len(rows)
		colTypes := inferTypes(rows, numCols)
		alloc := memory.NewGoAllocator()

		cols := make([]*arrow.ArrowSeries, numCols)
		for j := 0; j < numCols; j++ {
			cols[j] = buildColumn(headers[j], colTypes[j], rows, j, numRows, alloc)
		}
		return arrow.NewDataFrame(cols...), nil
	}

	// No header: generate column names col0, col1, ...
	if len(records) == 0 {
		return nil, fmt.Errorf("csv has no data rows")
	}
	rows := records
	numCols := len(rows[0])
	numRows := len(rows)
	colTypes := inferTypes(rows, numCols)
	alloc := memory.NewGoAllocator()

	cols := make([]*arrow.ArrowSeries, numCols)
	for j := 0; j < numCols; j++ {
		name := fmt.Sprintf("col%d", j)
		cols[j] = buildColumn(name, colTypes[j], rows, j, numRows, alloc)
	}
	return arrow.NewDataFrame(cols...), nil
}

// buildColumn constructs a single-column ArrowSeries for one CSV column.
func buildColumn(name string, dt core.DType, rows [][]string, col, numRows int, alloc memory.Allocator) *arrow.ArrowSeries {
	switch dt {
	case core.INT64:
		bldr := array.NewInt64Builder(alloc)
		bldr.Resize(numRows)
		for i := 0; i < numRows; i++ {
			if v, err := strconv.ParseInt(strings.TrimSpace(rows[i][col]), 10, 64); err == nil {
				bldr.Append(v)
			} else {
				bldr.AppendNull()
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s

	case core.FLOAT64:
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(numRows)
		for i := 0; i < numRows; i++ {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rows[i][col]), 64); err == nil {
				bldr.Append(v)
			} else {
				bldr.AppendNull()
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s

	case core.BOOL:
		bldr := array.NewBooleanBuilder(alloc)
		bldr.Resize(numRows)
		for i := 0; i < numRows; i++ {
			val := strings.TrimSpace(strings.ToLower(rows[i][col]))
			switch val {
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
		bldr.Resize(numRows)
		for i := 0; i < numRows; i++ {
			if rows[i][col] == "" {
				bldr.AppendNull()
			} else {
				bldr.Append(rows[i][col])
			}
		}
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	}
}

// inferTypes determines the DType for each column.
func inferTypes(rows [][]string, numCols int) []core.DType {
	types := make([]core.DType, numCols)
	for j := 0; j < numCols; j++ {
		types[j] = inferColumnType(rows, j)
	}
	return types
}

func inferColumnType(rows [][]string, col int) core.DType {
	boolCount, intCount, floatCount, total := 0, 0, 0, 0
	sampleSize := len(rows)
	if sampleSize > 100 {
		sampleSize = 100
	}
	for i := 0; i < sampleSize; i++ {
		val := strings.TrimSpace(rows[i][col])
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

// WriteCSV writes a DataFrame to a CSV string.
func WriteCSV(df *arrow.ArrowDataFrame) string {
	return df.ToCSV()
}

// WriteCSVFile writes a DataFrame to a CSV file.
func WriteCSVFile(df *arrow.ArrowDataFrame, path string) error {
	return os.WriteFile(path, []byte(df.ToCSV()), 0644)
}

// CSVWriter writes DataFrame chunks to a CSV file incrementally.
// Header is written on the first chunk; subsequent chunks append rows.
type CSVWriter struct {
	file       *os.File
	writer     *csv.Writer
	headerDone bool
}

// NewCSVWriter creates a streaming CSV writer.
func NewCSVWriter(path string) (*CSVWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	w := csv.NewWriter(f)
	return &CSVWriter{file: f, writer: w}, nil
}

// WriteChunk appends a DataFrame chunk to the CSV file.
func (cw *CSVWriter) WriteChunk(df *arrow.ArrowDataFrame) error {
	colNames := df.Columns()

	// Write header on first chunk
	if !cw.headerDone {
		if err := cw.writer.Write(colNames); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		cw.headerDone = true
	}

	numRows, _ := df.Shape()
	record := make([]string, len(colNames))
	for i := 0; i < numRows; i++ {
		for j, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				record[j] = ""
			} else {
				record[j] = s.String(i)
			}
		}
		if err := cw.writer.Write(record); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	cw.writer.Flush()
	return cw.writer.Error()
}

// Close flushes and closes the file.
func (cw *CSVWriter) Close() error {
	cw.writer.Flush()
	if err := cw.writer.Error(); err != nil {
		return err
	}
	return cw.file.Close()
}
