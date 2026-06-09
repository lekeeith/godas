package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godans/godans/core"
)

func TestReadJSON(t *testing.T) {
	data := `[{"name":"alice","age":25,"score":88.5},{"name":"bob","age":30,"score":92}]`
	df, err := ReadJSON(data)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	rows, cols := df.Shape()
	if rows != 2 || cols != 3 {
		t.Fatalf("Shape() = (%d,%d), want (2,3)", rows, cols)
	}
	if df.Col("name").String(0) != "alice" {
		t.Errorf("name[0] = %q", df.Col("name").String(0))
	}
	if df.Col("age").Int(1) != 30 {
		t.Errorf("age[1] = %d", df.Col("age").Int(1))
	}
}

func TestReadJSONWithNulls(t *testing.T) {
	data := `[{"x":1,"y":"hello"},{"x":null,"y":"world"}]`
	df, err := ReadJSON(data)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if df.Col("x").NullCount() != 1 {
		t.Errorf("x nulls = %d, want 1", df.Col("x").NullCount())
	}
}

func TestReadJSONEmpty(t *testing.T) {
	data := `[]`
	df, err := ReadJSON(data)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	rows, _ := df.Shape()
	if rows != 0 {
		t.Errorf("rows = %d, want 0", rows)
	}
}

func TestReadJSONLines(t *testing.T) {
	data := "{\"a\":1,\"b\":\"x\"}\n{\"a\":2,\"b\":\"y\"}\n{\"a\":3,\"b\":\"z\"}\n"
	df, err := ReadJSONLines(data)
	if err != nil {
		t.Fatalf("ReadJSONLines: %v", err)
	}
	rows, _ := df.Shape()
	if rows != 3 {
		t.Fatalf("rows = %d, want 3", rows)
	}
	if df.Col("a").Int(2) != 3 {
		t.Errorf("a[2] = %d, want 3", df.Col("a").Int(2))
	}
}

func TestReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := `[{"x":10},{"x":20}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	df, err := ReadJSONFile(path)
	if err != nil {
		t.Fatalf("ReadJSONFile: %v", err)
	}
	if df.Col("x").Int(0) != 10 {
		t.Errorf("x[0] = %d", df.Col("x").Int(0))
	}
}

func TestWriteJSON(t *testing.T) {
	csv := "name,age\nalice,25\nbob,30\n"
	df, _ := ReadCSV(csv)

	out, err := WriteJSON(df)
	if err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}

	// Round-trip: read back
	df2, err := ReadJSON(out)
	if err != nil {
		t.Fatalf("ReadJSON roundtrip: %v", err)
	}
	if df2.Col("name").String(0) != "alice" {
		t.Errorf("roundtrip name[0] = %q", df2.Col("name").String(0))
	}
}

func TestWriteJSONLines(t *testing.T) {
	csv := "x,y\n1,hello\n2,world\n"
	df, _ := ReadCSV(csv)

	out, err := WriteJSONLines(df)
	if err != nil {
		t.Fatalf("WriteJSONLines: %v", err)
	}
	df2, err := ReadJSONLines(out)
	if err != nil {
		t.Fatalf("ReadJSONLines roundtrip: %v", err)
	}
	rows, _ := df2.Shape()
	if rows != 2 {
		t.Fatalf("roundtrip rows = %d, want 2", rows)
	}
}

func TestWriteJSONFile(t *testing.T) {
	csv := "a,b\n1,2\n"
	df, _ := ReadCSV(csv)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	if err := WriteJSONFile(df, path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Error("empty file")
	}
}

func TestInferJSONColType(t *testing.T) {
	rows := []map[string]interface{}{
		{"a": float64(1), "b": float64(3.14), "c": true, "d": "hello"},
		{"a": float64(2), "b": float64(2.72), "c": false, "d": "world"},
	}
	if got := inferJSONColType(rows, "a"); got != core.INT64 {
		t.Errorf("a: got %s, want int64", got)
	}
	if got := inferJSONColType(rows, "b"); got != core.FLOAT64 {
		t.Errorf("b: got %s, want float64", got)
	}
	if got := inferJSONColType(rows, "c"); got != core.BOOL {
		t.Errorf("c: got %s, want bool", got)
	}
	if got := inferJSONColType(rows, "d"); got != core.STRING {
		t.Errorf("d: got %s, want string", got)
	}
}
