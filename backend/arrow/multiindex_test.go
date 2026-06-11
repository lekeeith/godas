package arrow

import (
	"testing"

	"github.com/lekeeith/godas/core"
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

func TestMultiIndexSwapLevel(t *testing.T) {
	tuples := [][]string{
		{"eng", "senior"},
		{"eng", "junior"},
		{"sales", "senior"},
	}
	mi := NewMultiIndexFromTuples(tuples, []string{"dept", "level"})
	swapped := mi.SwapLevel(0, 1)
	if swapped.names[0] != "level" || swapped.names[1] != "dept" {
		t.Errorf("SwapLevel names = %v", swapped.names)
	}
	if swapped.GetLevel(0, 0) != "senior" || swapped.GetLevel(1, 0) != "eng" {
		t.Errorf("SwapLevel values = %s/%s", swapped.GetLevel(0, 0), swapped.GetLevel(1, 0))
	}
}

func TestMultiIndexDropLevel(t *testing.T) {
	tuples := [][]string{
		{"eng", "senior"},
		{"eng", "junior"},
	}
	mi := NewMultiIndexFromTuples(tuples, []string{"dept", "level"})
	dropped := mi.DropLevel(1)
	if dropped.NLevels() != 1 {
		t.Fatalf("DropLevel NLevels = %d, want 1", dropped.NLevels())
	}
	if dropped.GetLevel(0, 0) != "eng" {
		t.Errorf("DropLevel[0] = %q, want eng", dropped.GetLevel(0, 0))
	}
}

func TestMultiIndexRenameLevel(t *testing.T) {
	tuples := [][]string{{"a", "1"}}
	mi := NewMultiIndexFromTuples(tuples, []string{"x", "y"})
	renamed := mi.RenameLevel(0, "renamed")
	if renamed.names[0] != "renamed" || renamed.names[1] != "y" {
		t.Errorf("RenameLevel names = %v", renamed.names)
	}
}

func TestMultiIndexGetLevelValues(t *testing.T) {
	tuples := [][]string{
		{"eng", "senior"},
		{"eng", "junior"},
		{"sales", "senior"},
	}
	mi := NewMultiIndexFromTuples(tuples, []string{"dept", "level"})
	vals := mi.GetLevelValues(0)
	if len(vals) != 3 {
		t.Fatalf("GetLevelValues len = %d, want 3", len(vals))
	}
	if vals[0] != "eng" || vals[2] != "sales" {
		t.Errorf("GetLevelValues = %v", vals)
	}
}

func TestDataFrameMultiIndexOps(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewStringSeries("level", []string{"senior", "junior", "senior", "junior"}, nil),
		NewFloat64Series("salary", []float64{100, 80, 90, 70}, nil),
	)
	mdf := df.SetMultiIndex([]string{"dept", "level"})

	// Test MultiIndex is preserved
	rdf := mdf
	if rdf.multiIndex == nil {
		t.Fatal("multiIndex should be non-nil after SetMultiIndex")
	}

	// Test Xs uses multiIndex
	eng := rdf.Xs(0, "eng")
	if eng.Len() != 2 {
		t.Fatalf("Xs eng.Len() = %d, want 2", eng.Len())
	}

	// Test GroupByLevel
	grouped := rdf.GroupByLevel(0, map[string]core.AggFunc{"salary": core.AggMean})
	if grouped.Len() != 2 {
		t.Fatalf("GroupByLevel.Len() = %d, want 2", grouped.Len())
	}

	// Test Unstack
	unstacked := rdf.Unstack(0)
	if unstacked.Len() != 2 { // junior, senior
		t.Fatalf("Unstack.Len() = %d, want 2", unstacked.Len())
	}
}
