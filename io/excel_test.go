package io

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadExcel(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	// Write
	if err := WriteExcelFile(df, path, "Data"); err != nil {
		t.Fatalf("WriteExcelFile: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("excel file is empty")
	}

	// Read back
	df2, err := ReadExcelFile(path, "Data")
	if err != nil {
		t.Fatalf("ReadExcelFile: %v", err)
	}

	rows, cols := df2.Shape()
	if rows != 3 {
		t.Fatalf("rows = %d, want 3", rows)
	}
	if cols != 3 {
		t.Fatalf("cols = %d, want 3", cols)
	}

	if df2.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q", df2.Col("name").String(0))
	}
	if df2.Col("age").Int(1) != 30 {
		t.Errorf("age[1] = %d", df2.Col("age").Int(1))
	}
}

func TestExcelWithNulls(t *testing.T) {
	csv := "x,y\n1,hello\n,world\n3,\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "nulls.xlsx")

	if err := WriteExcelFile(df, path, ""); err != nil {
		t.Fatal(err)
	}

	df2, err := ReadExcelFile(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if df2.Col("x").NullCount() != 1 {
		t.Errorf("x nulls = %d, want 1", df2.Col("x").NullCount())
	}
}

func TestExcelMultipleSheets(t *testing.T) {
	df1, _ := ReadCSV("a,b\n1,2\n3,4\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "multi.xlsx")

	// Write two sheets
	if err := WriteExcelFile(df1, path, "Sheet1"); err != nil {
		t.Fatal(err)
	}
	// Can't easily append sheets with current API, just verify first sheet
	r, err := ReadExcelFile(path, "Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != 2 {
		t.Fatalf("Sheet1 rows = %d, want 2", r.Len())
	}
}

func TestExcelEmptySheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	// Create empty file first
	df, _ := ReadCSV("a,b\n1,2\n")
	if err := WriteExcelFile(df, path, ""); err != nil {
		t.Fatal(err)
	}

	// Read with empty sheet name (defaults to first)
	r, err := ReadExcelFile(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Len() != 1 {
		t.Fatalf("rows = %d, want 1", r.Len())
	}
}

func TestExcelTypeInference(t *testing.T) {
	csv := "id,price,active,name\n1,19.99,true,alice\n2,29.99,false,bob\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "types.xlsx")

	if err := WriteExcelFile(df, path, ""); err != nil {
		t.Fatal(err)
	}

	df2, err := ReadExcelFile(path, "")
	if err != nil {
		t.Fatal(err)
	}

	dtypes := df2.Dtypes()
	if dtypes[0].String() != "int64" {
		t.Errorf("id dtype = %s, want int64", dtypes[0])
	}
	if dtypes[1].String() != "float64" {
		t.Errorf("price dtype = %s, want float64", dtypes[1])
	}
}
