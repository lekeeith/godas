package io

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCSV(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92.0\ncharlie,35,76.5\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	rows, cols := df.Shape()
	if rows != 3 || cols != 3 {
		t.Fatalf("Shape() = (%d,%d), want (3,3)", rows, cols)
	}

	names := df.Columns()
	if names[0] != "name" || names[1] != "age" || names[2] != "score" {
		t.Errorf("Columns() = %v", names)
	}

	// Check types
	dtypes := df.Dtypes()
	if dtypes[1].String() != "int64" {
		t.Errorf("age dtype = %s, want int64", dtypes[1])
	}
	if dtypes[2].String() != "float64" {
		t.Errorf("score dtype = %s, want float64", dtypes[2])
	}

	// Check values
	if df.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q, want %q", df.Col("name").String(0), "alice")
	}
	if df.Col("age").Int(1) != 30 {
		t.Errorf("age[1] = %d, want 30", df.Col("age").Int(1))
	}
	if df.Col("score").Float(2) != 76.5 {
		t.Errorf("score[2] = %g, want 76.5", df.Col("score").Float(2))
	}
}

func TestReadCSVBool(t *testing.T) {
	csv := "id,active\n1,true\n2,false\n3,yes\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	dtypes := df.Dtypes()
	if dtypes[1].String() != "bool" {
		t.Errorf("active dtype = %s, want bool", dtypes[1])
	}
	if !df.Col("active").Bool(0) {
		t.Error("active[0] should be true")
	}
	if df.Col("active").Bool(1) {
		t.Error("active[1] should be false")
	}
}

func TestReadCSVWithNulls(t *testing.T) {
	csv := "x,y\n1,hello\n,world\n3,\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if df.Col("x").NullCount() != 1 {
		t.Errorf("x nulls = %d, want 1", df.Col("x").NullCount())
	}
	if df.Col("y").NullCount() != 1 {
		t.Errorf("y nulls = %d, want 1", df.Col("y").NullCount())
	}
}

func TestReadCSVMixedTypes(t *testing.T) {
	csv := "name,value\nfoo,100\nbar,200.5\nbaz,hello\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	// value column has mixed types → should be string
	dtypes := df.Dtypes()
	if dtypes[1].String() != "string" {
		t.Errorf("value dtype = %s, want string (mixed types)", dtypes[1])
	}
}

func TestReadCSVFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	csv := "a,b\n1,2\n3,4\n"
	if err := os.WriteFile(path, []byte(csv), 0644); err != nil {
		t.Fatal(err)
	}
	df, err := ReadCSVFile(path)
	if err != nil {
		t.Fatalf("ReadCSVFile: %v", err)
	}
	rows, _ := df.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestReadCSVEmptyError(t *testing.T) {
	_, err := ReadCSV("")
	if err == nil {
		t.Error("expected error for empty CSV")
	}
	_, err = ReadCSV("name,age\n")
	if err == nil {
		t.Error("expected error for header-only CSV")
	}
}

func TestWriteCSV(t *testing.T) {
	csv := "name,age,score\nalice,25,88.5\nbob,30,92\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	out := WriteCSV(df)
	if len(out) == 0 {
		t.Fatal("WriteCSV returned empty")
	}
	// Should contain header and data
	if out[:4] != "name" {
		t.Errorf("header = %q", out[:10])
	}
}

func TestWriteCSVFile(t *testing.T) {
	csv := "x,y\n1,2\n"
	df, err := ReadCSV(csv)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	if err := WriteCSVFile(df, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestInferColumnType(t *testing.T) {
	rows := [][]string{
		{"1", "3.14", "true", "hello"},
		{"2", "2.72", "false", "world"},
		{"3", "1.41", "yes", "foo"},
	}
	if got := inferColumnType(rows, 0); got.String() != "int64" {
		t.Errorf("col 0: got %s, want int64", got)
	}
	if got := inferColumnType(rows, 1); got.String() != "float64" {
		t.Errorf("col 1: got %s, want float64", got)
	}
	if got := inferColumnType(rows, 2); got.String() != "bool" {
		t.Errorf("col 2: got %s, want bool", got)
	}
	if got := inferColumnType(rows, 3); got.String() != "string" {
		t.Errorf("col 3: got %s, want string", got)
	}
}
