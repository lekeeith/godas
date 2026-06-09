package io

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/backend/arrow"
	"github.com/lekeeith/godas/core"
	"github.com/xuri/excelize/v2"
)

// ReadExcelFile reads an Excel file into a DataFrame.
// sheetName: sheet to read (empty = first sheet).
func ReadExcelFile(path string, sheetName string) (*arrow.ArrowDataFrame, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer f.Close()

	if sheetName == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("no sheets found")
		}
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", sheetName, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet must have at least a header and one data row")
	}

	// First row is header
	headers := rows[0]
	dataRows := rows[1:]
	numCols := len(headers)
	numRows := len(dataRows)

	// Infer types
	colTypes := inferExcelTypes(dataRows, numCols)
	alloc := memory.NewGoAllocator()
	cols := make([]*arrow.ArrowSeries, numCols)

	for j := 0; j < numCols; j++ {
		colName := headers[j]
		dt := colTypes[j]

		switch dt {
		case core.INT64:
			bldr := array.NewInt64Builder(alloc)
			bldr.Resize(numRows)
			for i := 0; i < numRows; i++ {
				if j < len(dataRows[i]) {
					v, err := parseExcelInt(dataRows[i][j])
					if err == nil {
						bldr.Append(v)
					} else {
						bldr.AppendNull()
					}
				} else {
					bldr.AppendNull()
				}
			}
			cols[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)

		case core.FLOAT64:
			bldr := array.NewFloat64Builder(alloc)
			bldr.Resize(numRows)
			for i := 0; i < numRows; i++ {
				if j < len(dataRows[i]) {
					v, err := parseExcelFloat(dataRows[i][j])
					if err == nil {
						bldr.Append(v)
					} else {
						bldr.AppendNull()
					}
				} else {
					bldr.AppendNull()
				}
			}
			cols[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)

		case core.BOOL:
			bldr := array.NewBooleanBuilder(alloc)
			bldr.Resize(numRows)
			for i := 0; i < numRows; i++ {
				if j < len(dataRows[i]) {
					v, err := parseExcelBool(dataRows[i][j])
					if err == nil {
						bldr.Append(v)
					} else {
						bldr.AppendNull()
					}
				} else {
					bldr.AppendNull()
				}
			}
			cols[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)

		default:
			bldr := array.NewStringBuilder(alloc)
			bldr.Resize(numRows)
			for i := 0; i < numRows; i++ {
				if j < len(dataRows[i]) && dataRows[i][j] != "" {
					bldr.Append(dataRows[i][j])
				} else {
					bldr.AppendNull()
				}
			}
			cols[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)
		}
	}

	return mergeExcelCols(cols), nil
}

func mergeExcelCols(dfs []*arrow.ArrowSeries) *arrow.ArrowDataFrame {
	return arrow.NewDataFrame(dfs...)
}

// WriteExcelFile writes a DataFrame to an Excel file.
func WriteExcelFile(df *arrow.ArrowDataFrame, path string, sheetName string) error {
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	f := excelize.NewFile()
	defer f.Close()

	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	// Write headers
	colNames := df.Columns()
	for j, name := range colNames {
		cell, _ := excelize.CoordinatesToCellName(j+1, 1)
		f.SetCellValue(sheetName, cell, name)
	}

	// Write data
	rows, _ := df.Shape()
	for i := 0; i < rows; i++ {
		for j, name := range colNames {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			s := df.Col(name)
			if s.IsNull(i) {
				f.SetCellValue(sheetName, cell, nil)
			} else {
				switch s.Dtype() {
				case core.BOOL:
					f.SetCellValue(sheetName, cell, s.Bool(i))
				case core.FLOAT32, core.FLOAT64:
					f.SetCellValue(sheetName, cell, s.Float(i))
				case core.STRING:
					f.SetCellValue(sheetName, cell, s.String(i))
				default:
					f.SetCellValue(sheetName, cell, s.Int(i))
				}
			}
		}
	}

	f.SetActiveSheet(idx)
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("save excel: %w", err)
	}
	return nil
}

func inferExcelTypes(rows [][]string, numCols int) []core.DType {
	types := make([]core.DType, numCols)
	for j := 0; j < numCols; j++ {
		boolCount, intCount, floatCount, total := 0, 0, 0, 0
		sampleSize := len(rows)
		if sampleSize > 100 {
			sampleSize = 100
		}
		for i := 0; i < sampleSize; i++ {
			if j >= len(rows[i]) || rows[i][j] == "" {
				continue
			}
			total++
			lower := rows[i][j]
			if lower == "true" || lower == "false" || lower == "TRUE" || lower == "FALSE" {
				boolCount++
				continue
			}
			if _, err := parseExcelInt(rows[i][j]); err == nil {
				intCount++
				continue
			}
			if _, err := parseExcelFloat(rows[i][j]); err == nil {
				floatCount++
				continue
			}
		}
		if total == 0 {
			types[j] = core.STRING
			continue
		}
		threshold := float64(total) * 0.8
		if float64(boolCount) >= threshold {
			types[j] = core.BOOL
		} else if float64(intCount) >= threshold {
			types[j] = core.INT64
		} else if float64(intCount+floatCount) >= threshold {
			types[j] = core.FLOAT64
		} else {
			types[j] = core.STRING
		}
	}
	return types
}

func parseExcelInt(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func parseExcelFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func parseExcelBool(s string) (bool, error) {
	switch s {
	case "true", "TRUE", "1", "yes", "YES":
		return true, nil
	case "false", "FALSE", "0", "no", "NO":
		return false, nil
	default:
		return false, fmt.Errorf("not a bool: %s", s)
	}
}
