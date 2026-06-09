package arrow

import (
	"testing"

	"github.com/lekeeith/godas/core"
)

func TestParallelTransform(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"a", "b", "c"}, nil),
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	result := df.ParallelTransform(func(v float64) float64 { return v * 2 })
	if result.Col("name").String(0) != "a" {
		t.Error("name should be unchanged")
	}
	if result.Col("x").(*ArrowSeries).Float(0) != 2 {
		t.Errorf("x[0] = %g, want 2", result.Col("x").(*ArrowSeries).Float(0))
	}
	if result.Col("y").(*ArrowSeries).Float(2) != 60 {
		t.Errorf("y[2] = %g, want 60", result.Col("y").(*ArrowSeries).Float(2))
	}
}

func TestParallelApplyCols(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("a", []float64{1, 2, 3}, nil),
		NewFloat64Series("b", []float64{10, 20, 30}, nil),
	)
	result := df.ParallelApplyCols(func(col core.Series) core.Series {
		return col.(*ArrowSeries).MapFloat(func(v float64) float64 { return v + 1 })
	})
	if result.Col("a").(*ArrowSeries).Float(0) != 2 {
		t.Errorf("a[0] = %g, want 2", result.Col("a").(*ArrowSeries).Float(0))
	}
	if result.Col("b").(*ArrowSeries).Float(2) != 31 {
		t.Errorf("b[2] = %g, want 31", result.Col("b").(*ArrowSeries).Float(2))
	}
}

func TestParallelAgg(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90}, nil),
		NewFloat64Series("bonus", []float64{10, 20, 5, 15}, nil),
	)
	result := df.ParallelAgg(
		[]string{"dept"},
		map[string]core.AggFunc{"salary": core.AggMean, "bonus": core.AggSum},
	)
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestParallelFilter(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil),
	)
	mask := make([]bool, 10)
	for i := range mask {
		mask[i] = i%2 == 0
	}
	result := df.ParallelFilter(mask)
	if result.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", result.Len())
	}
	if result.Col("x").(*ArrowSeries).Float(0) != 1 {
		t.Errorf("x[0] = %g, want 1", result.Col("x").(*ArrowSeries).Float(0))
	}
}

func TestParallelInfo(t *testing.T) {
	info := ParallelInfo()
	if info == "" {
		t.Error("ParallelInfo() returned empty")
	}
}

func TestParallelTransformPreservesNonNumeric(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"a", "b"}, nil),
		NewInt64Series("id", []int64{1, 2}, nil),
	)
	result := df.ParallelTransform(func(v float64) float64 { return v * 10 })
	if result.Col("name").String(0) != "a" {
		t.Error("string column should be unchanged")
	}
	if result.Col("id").(*ArrowSeries).Float(0) != 10 {
		t.Errorf("id[0] = %g, want 10", result.Col("id").(*ArrowSeries).Float(0))
	}
}
