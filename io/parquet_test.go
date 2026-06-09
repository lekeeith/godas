package io

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadParquet(t *testing.T) {
	// Build a DataFrame from CSV
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	// Write to Parquet
	if err := WriteParquetFile(df, path); err != nil {
		t.Fatalf("WriteParquetFile: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("parquet file is empty")
	}

	// Read back
	df2, err := ReadParquetFile(path)
	if err != nil {
		t.Fatalf("ReadParquetFile: %v", err)
	}

	rows, cols := df2.Shape()
	if rows != 3 {
		t.Fatalf("rows = %d, want 3", rows)
	}
	if cols != 3 {
		t.Fatalf("cols = %d, want 3", cols)
	}

	// Check values
	if df2.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q, want %q", df2.Col("name").String(0), "alice")
	}
	if df2.Col("age").Int(1) != 30 {
		t.Errorf("age[1] = %d, want 30", df2.Col("age").Int(1))
	}
	if df2.Col("score").Float(2) != 76.5 {
		t.Errorf("score[2] = %g, want 76.5", df2.Col("score").Float(2))
	}
}

func TestParquetWithNulls(t *testing.T) {
	csv := "x,y\n1,hello\n,world\n3,\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "nulls.parquet")

	if err := WriteParquetFile(df, path); err != nil {
		t.Fatalf("WriteParquetFile: %v", err)
	}

	df2, err := ReadParquetFile(path)
	if err != nil {
		t.Fatalf("ReadParquetFile: %v", err)
	}

	// Check nulls preserved
	if df2.Col("x").NullCount() != 1 {
		t.Errorf("x nulls = %d, want 1", df2.Col("x").NullCount())
	}
	if df2.Col("y").NullCount() != 1 {
		t.Errorf("y nulls = %d, want 1", df2.Col("y").NullCount())
	}
}

func TestParquetSingleColumn(t *testing.T) {
	csv := "val\n10\n20\n30\n"
	df, _ := ReadCSV(csv)

	dir := t.TempDir()
	path := filepath.Join(dir, "single.parquet")

	if err := WriteParquetFile(df, path); err != nil {
		t.Fatalf("WriteParquetFile: %v", err)
	}

	df2, err := ReadParquetFile(path)
	if err != nil {
		t.Fatalf("ReadParquetFile: %v", err)
	}

	if df2.Col("val").Int(0) != 10 {
		t.Errorf("val[0] = %d, want 10", df2.Col("val").Int(0))
	}
}
