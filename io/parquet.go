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

// ReadParquetFile reads a Parquet file into a DataFrame.
func ReadParquetFile(path string) (*arrow.ArrowDataFrame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return readParquetBytes(data)
}

func readParquetBytes(data []byte) (*arrow.ArrowDataFrame, error) {
	r := bytes.NewReader(data)
	pf, err := parquet.OpenFile(r, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open parquet: %w", err)
	}

	numRows := pf.NumRows()
	schema := pf.Schema()
	colNames := extractColumnNames(schema)
	numCols := len(colNames)

	// Use non-generic Reader
	reader := parquet.NewReader(pf)
	defer reader.Close()

	// Read all rows in batches
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
			return nil, fmt.Errorf("read parquet rows: %w", err)
		}
	}

	alloc := memory.NewGoAllocator()
	cols := make([]*arrow.ArrowSeries, numCols)

	for j := 0; j < numCols; j++ {
		vals := make([]parquet.Value, len(allRows))
		for i, row := range allRows {
			if j < len(row) {
				vals[i] = row[j]
			}
		}
		cols[j] = buildParquetColumn(colNames[j], vals, alloc)
	}

	return arrow.NewDataFrame(cols...), nil
}

func extractColumnNames(schema *parquet.Schema) []string {
	var names []string
	for _, field := range schema.Fields() {
		collectLeafNames(field, &names)
	}
	return names
}

func collectLeafNames(field parquet.Field, names *[]string) {
	if field.Leaf() {
		*names = append(*names, field.Name())
	} else {
		for _, child := range field.Fields() {
			collectLeafNames(child, names)
		}
	}
}

func buildParquetColumn(name string, values []parquet.Value, alloc memory.Allocator) *arrow.ArrowSeries {
	numRows := len(values)
	if numRows == 0 {
		bldr := array.NewInt64Builder(alloc)
		s := arrow.NewArrowSeries(name, bldr.NewArray(), nil)
		bldr.Release()
		return s
	}

	dt := inferParquetType(values)

	switch dt {
	case core.BOOL:
		bldr := array.NewBooleanBuilder(alloc)
		bldr.Resize(numRows)
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
		bldr.Resize(numRows)
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
		bldr.Resize(numRows)
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
		bldr.Resize(numRows)
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

func inferParquetType(values []parquet.Value) core.DType {
	sampleSize := len(values)
	if sampleSize > 100 {
		sampleSize = 100
	}
	for i := 0; i < sampleSize; i++ {
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

// WriteParquetFile writes a DataFrame to a Parquet file.
func WriteParquetFile(df *arrow.ArrowDataFrame, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	schema := buildParquetSchema(df)

	// Write to buffer first, then flush to file
	buf := parquet.NewBuffer(schema)
	numRows, _ := df.Shape()
	colNames := df.Columns()

	for i := 0; i < numRows; i++ {
		row := make(map[string]any, len(colNames))
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
		if err := buf.Write(row); err != nil {
			return fmt.Errorf("write parquet buffer: %w", err)
		}
	}

	// Flush buffer to file
	writer := parquet.NewWriter(f, schema)
	defer writer.Close()

	if _, err := writer.WriteRowGroup(buf); err != nil {
		return fmt.Errorf("write parquet: %w", err)
	}

	return nil
}

func buildParquetSchema(df *arrow.ArrowDataFrame) *parquet.Schema {
	colNames := df.Columns()
	nodes := make(parquet.Group, len(colNames))
	for _, name := range colNames {
		dt := df.Col(name).Dtype()
		var node parquet.Node
		switch dt {
		case core.BOOL:
			node = parquet.Optional(parquet.Leaf(parquet.BooleanType))
		case core.INT8, core.INT16, core.INT32:
			node = parquet.Optional(parquet.Leaf(parquet.Int32Type))
		case core.INT64, core.UINT8, core.UINT16, core.UINT32, core.UINT64:
			node = parquet.Optional(parquet.Leaf(parquet.Int64Type))
		case core.FLOAT32:
			node = parquet.Optional(parquet.Leaf(parquet.FloatType))
		case core.FLOAT64:
			node = parquet.Optional(parquet.Leaf(parquet.DoubleType))
		default:
			node = parquet.Optional(parquet.String())
		}
		nodes[name] = node
	}
	return parquet.NewSchema("godans", nodes)
}
