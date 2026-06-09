package arrow

import (
	"testing"
)

// --- MultiIndex ---

func TestMultiIndexFromTuples(t *testing.T) {
	tuples := [][]string{
		{"eng", "senior"},
		{"eng", "junior"},
		{"sales", "senior"},
		{"sales", "junior"},
	}
	mi := NewMultiIndexFromTuples(tuples, []string{"dept", "level"})
	if mi.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", mi.Len())
	}
	if mi.NLevels() != 2 {
		t.Fatalf("NLevels() = %d, want 2", mi.NLevels())
	}
	if mi.GetLevel(0, 0) != "eng" || mi.GetLevel(1, 0) != "senior" {
		t.Errorf("GetLevel: %s/%s", mi.GetLevel(0, 0), mi.GetLevel(1, 0))
	}
}

func TestMultiIndexXs(t *testing.T) {
	tuples := [][]string{
		{"eng", "senior"},
		{"eng", "junior"},
		{"sales", "senior"},
		{"sales", "junior"},
	}
	mi := NewMultiIndexFromTuples(tuples, []string{"dept", "level"})
	engIndices := mi.Xs(0, "eng")
	if len(engIndices) != 2 {
		t.Fatalf("Xs eng: len = %d, want 2", len(engIndices))
	}
	if engIndices[0] != 0 || engIndices[1] != 1 {
		t.Errorf("Xs eng: %v", engIndices)
	}

	seniorIndices := mi.Xs(1, "senior")
	if len(seniorIndices) != 2 {
		t.Fatalf("Xs senior: len = %d, want 2", len(seniorIndices))
	}
}

func TestDataFrameSetMultiIndex(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil),
		NewStringSeries("level", []string{"senior", "junior", "senior"}, nil),
		NewFloat64Series("salary", []float64{100, 80, 90}, nil),
	)
	result := df.SetMultiIndex([]string{"dept", "level"})
	cols := result.Columns()
	if len(cols) != 1 || cols[0] != "salary" {
		t.Errorf("Columns = %v, want [salary]", cols)
	}
	// Index should be "eng/senior", "eng/junior", "sales/senior"
	if result.Index().Get(0) != "eng/senior" {
		t.Errorf("Index[0] = %q, want %q", result.Index().Get(0), "eng/senior")
	}
}

func TestDataFrameXs(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil),
		NewStringSeries("level", []string{"senior", "junior", "senior"}, nil),
		NewFloat64Series("salary", []float64{100, 80, 90}, nil),
	)
	result := df.SetMultiIndex([]string{"dept", "level"})
	eng := result.Xs(0, "eng")
	if eng.Len() != 2 {
		t.Fatalf("Xs eng.Len() = %d, want 2", eng.Len())
	}
}

func TestMultiIndexString(t *testing.T) {
	tuples := [][]string{{"a", "1"}, {"b", "2"}}
	mi := NewMultiIndexFromTuples(tuples, []string{"x", "y"})
	s := mi.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestMultiIndexToIndex(t *testing.T) {
	tuples := [][]string{{"a", "1"}, {"b", "2"}}
	mi := NewMultiIndexFromTuples(tuples, []string{"x", "y"})
	idx := mi.ToIndex()
	if idx.Len() != 2 {
		t.Fatalf("ToIndex.Len() = %d, want 2", idx.Len())
	}
	if idx.Get(0) != "a/1" {
		t.Errorf("ToIndex[0] = %q, want %q", idx.Get(0), "a/1")
	}
}

func TestMultiIndexSlice(t *testing.T) {
	tuples := [][]string{{"a", "1"}, {"b", "2"}, {"c", "3"}}
	mi := NewMultiIndexFromTuples(tuples, nil)
	sub := mi.Slice(1, 3)
	if sub.Len() != 2 {
		t.Fatalf("Slice.Len() = %d, want 2", sub.Len())
	}
	if sub.GetLevel(0, 0) != "b" {
		t.Errorf("Slice[0] = %q", sub.GetLevel(0, 0))
	}
}
