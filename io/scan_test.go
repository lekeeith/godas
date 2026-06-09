package io

import (
	"path/filepath"
	"testing"
)

func TestScanParquetBasic(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).Collect()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	rows, _ := result.Shape()
	if rows != 3 {
		t.Fatalf("rows = %d, want 3", rows)
	}
}

func TestScanParquetSelect(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan_sel.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).Select("name", "score").Collect()
	if err != nil {
		t.Fatal(err)
	}
	_, cols := result.Shape()
	if cols != 2 {
		t.Fatalf("cols = %d, want 2", cols)
	}
}

func TestScanParquetFilter(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan_filter.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).Filter("age", ">", 28).Collect()
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestScanParquetLimit(t *testing.T) {
	csv := "x\n1\n2\n3\n4\n5\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan_limit.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).Limit(2).Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
}

func TestScanParquetChained(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\ndave,28,95.0\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan_chain.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).
		Select("name", "age", "score").
		Filter("age", ">=", 28).
		Limit(2).
		Collect()
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestScanParquetStringFilter(t *testing.T) {
	csv := "name,age\nalice,25\nbob,30\ncharlie,35\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "scan_str.parquet")
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(path).Filter("name", "==", "bob").Collect()
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", result.Len())
	}
	if result.Col("name").String(0) != "bob" {
		t.Errorf("name[0] = %q", result.Col("name").String(0))
	}
}

func TestScanParquetNotFound(t *testing.T) {
	_, err := Scan("/nonexistent/file.parquet").Collect()
	if err == nil {
		t.Error("expected error for missing file")
	}
}
