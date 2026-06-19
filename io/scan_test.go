package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lekeeith/godas/backend/arrow"
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

func TestScanCSVDelimiter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "semi.csv")
	data := "name;age;score\nalice;25;88.5\nbob;30;92.0\ncharlie;35;76.5\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanCSVFile(path).Delimiter(';').Collect()
	if err != nil {
		t.Fatalf("ScanCSV: %v", err)
	}
	rows, cols := result.Shape()
	if rows != 3 || cols != 3 {
		t.Fatalf("Shape() = (%d,%d), want (3,3)", rows, cols)
	}
	if result.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q, want alice", result.Col("name").String(0))
	}
}

func TestScanCSVSkipLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skip.csv")
	data := "# generated\n# date: 2026-01-01\nname,age\nalice,25\nbob,30\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanCSVFile(path).SkipLines(2).Collect()
	if err != nil {
		t.Fatalf("ScanCSV: %v", err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
	if result.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q, want alice", result.Col("name").String(0))
	}
}

func TestScanCSVDelimiterAndSkipLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "combo.csv")
	data := "# comment\nname;age\nalice;25\nbob;30\ncharlie;35\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanCSVFile(path).Delimiter(';').SkipLines(1).Filter("age", ">", "28").Collect()
	if err != nil {
		t.Fatalf("ScanCSV: %v", err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2 (charlie and bob)", rows)
	}
}

func TestScanCSVDelimiterForEach(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "each.csv")
	data := "x;y\n1;10\n2;20\n3;30\n4;40\n5;50\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	total, err := ScanCSVFile(path).Delimiter(';').ForEach(2, func(chunk *arrow.ArrowDataFrame) error {
		if chunk.Col("x").String(0) == "" {
			t.Error("unexpected empty x")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ForEach: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
}

func TestScanCSVSkipLinesForEach(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skip_each.csv")
	data := "## header comment\nname,age\na,1\nb,2\nc,3\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	total, err := ScanCSVFile(path).SkipLines(1).ForEach(2, func(chunk *arrow.ArrowDataFrame) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ForEach: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}
