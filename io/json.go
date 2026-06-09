package io

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/backend/arrow"
	"github.com/lekeeith/godas/core"
)

// ReadJSON reads a JSON array of objects into a DataFrame.
// Each object becomes a row; keys become column names.
func ReadJSON(data string) (*arrow.ArrowDataFrame, error) {
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &rows); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return buildFromMaps(rows)
}

// ReadJSONFile reads a JSON file into a DataFrame.
func ReadJSONFile(path string) (*arrow.ArrowDataFrame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ReadJSON(string(data))
}

// ReadJSONLines reads newline-delimited JSON (NDJSON) into a DataFrame.
func ReadJSONLines(data string) (*arrow.ArrowDataFrame, error) {
	var rows []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("parse json line: %w", err)
		}
		rows = append(rows, obj)
	}
	return buildFromMaps(rows)
}

// ReadJSONLinesFile reads an NDJSON file into a DataFrame.
func ReadJSONLinesFile(path string) (*arrow.ArrowDataFrame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ReadJSONLines(string(data))
}

// WriteJSON writes a DataFrame to a JSON array of objects string.
func WriteJSON(df *arrow.ArrowDataFrame) (string, error) {
	rows, cols := df.Shape()
	colNames := df.Columns()
	result := make([]map[string]interface{}, rows)

	for i := 0; i < rows; i++ {
		row := make(map[string]interface{}, cols)
		for _, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				row[name] = nil
				continue
			}
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
		result[i] = row
	}

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(b), nil
}

// WriteJSONFile writes a DataFrame to a JSON file.
func WriteJSONFile(df *arrow.ArrowDataFrame, path string) error {
	s, err := WriteJSON(df)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), 0644)
}

// WriteJSONLines writes a DataFrame to NDJSON format.
func WriteJSONLines(df *arrow.ArrowDataFrame) (string, error) {
	rows, cols := df.Shape()
	colNames := df.Columns()
	var b strings.Builder

	for i := 0; i < rows; i++ {
		row := make(map[string]interface{}, cols)
		for _, name := range colNames {
			s := df.Col(name)
			if s.IsNull(i) {
				row[name] = nil
				continue
			}
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
		line, err := json.Marshal(row)
		if err != nil {
			return "", fmt.Errorf("marshal json line: %w", err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// WriteJSONLinesFile writes a DataFrame to an NDJSON file.
func WriteJSONLinesFile(df *arrow.ArrowDataFrame, path string) error {
	s, err := WriteJSONLines(df)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), 0644)
}

// buildFromMaps constructs a DataFrame from a slice of maps.
func buildFromMaps(rows []map[string]interface{}) (*arrow.ArrowDataFrame, error) {
	if len(rows) == 0 {
		return arrow.NewDataFrame(), nil
	}

	// Collect all column names (preserving first-seen order)
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

	// Infer types for each column
	colTypes := inferJSONTypes(rows, colOrder)

	alloc := memory.NewGoAllocator()
	series := make([]*arrow.ArrowSeries, len(colOrder))

	for j, name := range colOrder {
		dt := colTypes[j]
		switch dt {
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
			series[j] = arrow.NewArrowSeries(name, bldr.NewArray(), nil)
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
			series[j] = arrow.NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()

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
			series[j] = arrow.NewArrowSeries(name, bldr.NewArray(), nil)
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
			series[j] = arrow.NewArrowSeries(name, bldr.NewArray(), nil)
			bldr.Release()
		}
	}

	return arrow.NewDataFrame(series...), nil
}

// inferJSONTypes determines the DType for each column from JSON values.
func inferJSONTypes(rows []map[string]interface{}, cols []string) []core.DType {
	types := make([]core.DType, len(cols))
	for j, name := range cols {
		types[j] = inferJSONColType(rows, name)
	}
	return types
}

func inferJSONColType(rows []map[string]interface{}, col string) core.DType {
	boolCount, intCount, floatCount, total := 0, 0, 0, 0
	sampleSize := len(rows)
	if sampleSize > 100 {
		sampleSize = 100
	}

	for i := 0; i < sampleSize; i++ {
		v, ok := rows[i][col]
		if !ok || v == nil {
			continue
		}
		total++

		switch v.(type) {
		case bool:
			boolCount++
		case float64:
			// JSON numbers are float64; check if it's actually an integer
			fv := v.(float64)
			if fv == float64(int64(fv)) {
				intCount++
			} else {
				floatCount++
			}
		case string:
			// try to parse
			s := v.(string)
			if _, err := strconv.ParseInt(s, 10, 64); err == nil {
				intCount++
			} else if _, err := strconv.ParseFloat(s, 64); err == nil {
				floatCount++
			}
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

func toInt64Val(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	case string:
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func toFloat64Val(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}
