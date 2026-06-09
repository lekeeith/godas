package arrow

import (
	"testing"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

func sampleDF() *ArrowDataFrame {
	name := NewStringSeries("name", []string{"alice", "bob", "charlie", "dave"}, nil)
	age := NewInt64Series("age", []int64{25, 30, 35, 40}, nil)
	score := NewFloat64Series("score", []float64{88.5, 92.0, 76.5, 95.0}, nil)
	return NewDataFrame(name, age, score)
}

func TestNewDataFrame(t *testing.T) {
	df := sampleDF()
	rows, cols := df.Shape()
	if rows != 4 || cols != 3 {
		t.Fatalf("Shape() = (%d,%d), want (4,3)", rows, cols)
	}
	names := df.Columns()
	if len(names) != 3 || names[0] != "name" || names[1] != "age" || names[2] != "score" {
		t.Errorf("Columns() = %v", names)
	}
}

func TestDataFrameCol(t *testing.T) {
	df := sampleDF()
	s := df.Col("age")
	if s.Len() != 4 {
		t.Fatalf("Col.Len() = %d, want 4", s.Len())
	}
	if s.Int(0) != 25 {
		t.Errorf("Col.Int(0) = %d, want 25", s.Int(0))
	}
}

func TestDataFrameSelectCols(t *testing.T) {
	df := sampleDF()
	sub := df.SelectCols([]string{"name", "score"})
	_, cols := sub.Shape()
	if cols != 2 {
		t.Fatalf("SelectCols.Shape() cols = %d, want 2", cols)
	}
}

func TestDataFrameDropCols(t *testing.T) {
	df := sampleDF()
	sub := df.DropCols([]string{"age"})
	_, cols := sub.Shape()
	if cols != 2 {
		t.Fatalf("DropCols.Shape() cols = %d, want 2", cols)
	}
}

func TestDataFrameHead(t *testing.T) {
	df := sampleDF()
	h := df.Head(2)
	rows, _ := h.Shape()
	if rows != 2 {
		t.Fatalf("Head.Len() = %d, want 2", rows)
	}
	if h.Col("name").String(0) != "alice" {
		t.Errorf("Head name[0] = %q", h.Col("name").String(0))
	}
}

func TestDataFrameTail(t *testing.T) {
	df := sampleDF()
	ta := df.Tail(2)
	rows, _ := ta.Shape()
	if rows != 2 {
		t.Fatalf("Tail.Len() = %d, want 2", rows)
	}
	if ta.Col("name").String(0) != "charlie" {
		t.Errorf("Tail name[0] = %q", ta.Col("name").String(0))
	}
}

func TestDataFrameSlice(t *testing.T) {
	df := sampleDF()
	sub := df.Slice(1, 3)
	rows, _ := sub.Shape()
	if rows != 2 {
		t.Fatalf("Slice.Len() = %d, want 2", rows)
	}
	if sub.Col("name").String(0) != "bob" {
		t.Errorf("Slice name[0] = %q", sub.Col("name").String(0))
	}
}

func TestDataFrameFilter(t *testing.T) {
	df := sampleDF()
	mask := []bool{true, false, true, false}
	f := df.Filter(mask)
	rows, _ := f.Shape()
	if rows != 2 {
		t.Fatalf("Filter.Len() = %d, want 2", rows)
	}
	if f.Col("name").String(0) != "alice" || f.Col("name").String(1) != "charlie" {
		t.Errorf("Filter names: %s, %s", f.Col("name").String(0), f.Col("name").String(1))
	}
}

func TestDataFrameDescribe(t *testing.T) {
	df := sampleDF()
	desc := df.Describe()
	_, cols := desc.Shape()
	if cols != 2 { // age and score are numeric
		t.Fatalf("Describe cols = %d, want 2", cols)
	}
	// Check that we have 8 stat rows
	rows, _ := desc.Shape()
	if rows != 8 {
		t.Fatalf("Describe rows = %d, want 8", rows)
	}
}

func TestDataFrameDropNA(t *testing.T) {
	name := NewStringSeriesWithNulls("name", []string{"a", "b", "c"}, []bool{true, false, true})
	age := NewInt64SeriesWithNulls("age", []int64{1, 2, 3}, []bool{true, true, false}, nil)
	df := NewDataFrame(name, age)
	clean := df.DropNA()
	rows, _ := clean.Shape()
	if rows != 1 {
		t.Fatalf("DropNA.Len() = %d, want 1", rows)
	}
}

func TestDataFrameFillNA(t *testing.T) {
	age := NewInt64SeriesWithNulls("age", []int64{1, 2, 3}, []bool{true, false, true}, nil)
	df := NewDataFrame(age)
	filled := df.FillNA(int64(99))
	if filled.Col("age").Int(1) != 99 {
		t.Errorf("FillNA value = %d, want 99", filled.Col("age").Int(1))
	}
}

func TestDataFrameRename(t *testing.T) {
	df := sampleDF()
	renamed := df.Rename(map[string]string{"name": "person", "age": "years"})
	cols := renamed.Columns()
	if cols[0] != "person" || cols[1] != "years" {
		t.Errorf("Rename columns = %v", cols)
	}
}

func TestDataFrameWithColumn(t *testing.T) {
	df := sampleDF()
	gpa := NewFloat64Series("gpa", []float64{3.5, 3.8, 3.2, 3.9}, nil)
	df2 := df.WithColumn("gpa", gpa)
	_, cols := df2.Shape()
	if cols != 4 {
		t.Fatalf("WithColumn cols = %d, want 4", cols)
	}
	// Replace existing column
	age2 := NewInt64Series("age", []int64{26, 31, 36, 41}, nil)
	df3 := df2.WithColumn("age", age2)
	if df3.Col("age").Int(0) != 26 {
		t.Errorf("WithColumn replace age[0] = %d, want 26", df3.Col("age").Int(0))
	}
}

func TestDataFrameSortBy(t *testing.T) {
	df := sampleDF()
	sorted := df.SortBy([]string{"age"}, []bool{false}) // descending
	if sorted.Col("name").String(0) != "dave" {
		t.Errorf("SortBy desc first = %q, want dave", sorted.Col("name").String(0))
	}
	if sorted.Col("age").Int(0) != 40 {
		t.Errorf("SortBy desc age[0] = %d, want 40", sorted.Col("age").Int(0))
	}
}

func TestDataFrameJoin(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("key", []string{"a", "b", "c"}, nil),
		NewInt64Series("val1", []int64{1, 2, 3}, nil),
	)
	right := NewDataFrame(
		NewStringSeries("key", []string{"b", "c", "d"}, nil),
		NewInt64Series("val2", []int64{20, 30, 40}, nil),
	)
	merged := left.MergeOn(right, []string{"key"}, core.Inner)
	rows, _ := merged.Shape()
	if rows != 2 {
		t.Fatalf("InnerMerge rows = %d, want 2", rows)
	}
	if merged.Col("val1").Int(0) != 2 || merged.Col("val2").Int(0) != 20 {
		t.Errorf("InnerMerge values wrong: val1=%d val2=%d", merged.Col("val1").Int(0), merged.Col("val2").Int(0))
	}
}

func TestDataFrameGroupByAgg(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90}, nil),
	)
	result := df.Agg([]string{"dept"}, map[string]core.AggFunc{"salary": core.AggMean})
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("Agg rows = %d, want 2", rows)
	}
	_, cols := result.Shape()
	if cols != 2 { // dept + salary_mean
		t.Fatalf("Agg cols = %d, want 2", cols)
	}
}

func TestDataFrameToCSV(t *testing.T) {
	df := sampleDF()
	csv := df.ToCSV()
	if len(csv) == 0 {
		t.Fatal("ToCSV() returned empty")
	}
	// Check header
	if csv[:4] != "name" {
		t.Errorf("CSV header = %q", csv[:10])
	}
}

func TestDataFrameGroupByGroups(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil),
		NewInt64Series("id", []int64{1, 2, 3}, nil),
	)
	groups := df.GroupByGroups([]string{"dept"})
	if len(groups) != 2 {
		t.Fatalf("GroupByGroups len = %d, want 2", len(groups))
	}
	if len(groups["eng"]) != 2 {
		t.Errorf("eng group len = %d, want 2", len(groups["eng"]))
	}
}

// helper
func NewStringSeriesWithNulls(name string, data []string, valid []bool) *ArrowSeries {
	b := array.NewStringBuilder(memory.NewGoAllocator())
	b.Resize(len(data))
	for i, v := range data {
		if i < len(valid) && !valid[i] {
			b.AppendNull()
		} else {
			b.Append(v)
		}
	}
	return NewArrowSeries(name, b.NewArray(), nil)
}
