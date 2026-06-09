package arrow

import (
	"strings"
	"testing"

	"github.com/lekeeith/godas/core"
)

func TestSeriesMapFloat(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 4, 9, 16}, nil)
	result := s.MapFloat(func(v float64) float64 {
		return v * 2
	})
	expected := []float64{2, 8, 18, 32}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("MapFloat[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestSeriesMapString(t *testing.T) {
	s := NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil)
	result := s.MapString(func(v string) string {
		return strings.ToUpper(v)
	})
	if result.String(0) != "ALICE" || result.String(2) != "CHARLIE" {
		t.Errorf("MapString: %s, %s", result.String(0), result.String(2))
	}
}

func TestSeriesMapBool(t *testing.T) {
	s := NewBoolSeries("b", []bool{true, false, true}, nil)
	result := s.MapBool(func(v bool) bool {
		return !v
	})
	if result.Bool(0) || !result.Bool(1) || result.Bool(2) {
		t.Error("MapBool failed")
	}
}

func TestSeriesMapFloatWithNulls(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.FLOAT64, nil)
	bldr.AppendFloat(1)
	bldr.AppendNull()
	bldr.AppendFloat(3)
	s := bldr.Build()

	result := s.MapFloat(func(v float64) float64 {
		return v * 10
	})
	if result.Float(0) != 10 {
		t.Errorf("MapFloat[0] = %g, want 10", result.Float(0))
	}
	if !result.IsNull(1) {
		t.Error("expected null at index 1")
	}
	if result.Float(2) != 30 {
		t.Errorf("MapFloat[2] = %g, want 30", result.Float(2))
	}
}

func TestSeriesApply(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3}, nil)
	result := s.Apply(func(val interface{}) interface{} {
		if v, ok := val.(float64); ok {
			return v + 100
		}
		return nil
	})
	if result.String(0) != "101" {
		t.Errorf("Apply[0] = %q, want %q", result.String(0), "101")
	}
}

func TestDataFrameTransform(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"a", "b", "c"}, nil),
		NewFloat64Series("val", []float64{1, 2, 3}, nil),
		NewInt64Series("id", []int64{10, 20, 30}, nil),
	)
	result := df.Transform(func(v float64) float64 {
		return v * 2
	})
	// name should be unchanged
	if result.Col("name").String(0) != "a" {
		t.Errorf("name[0] = %q, want a", result.Col("name").String(0))
	}
	// val should be doubled
	if result.Col("val").Float(0) != 2 {
		t.Errorf("val[0] = %g, want 2", result.Col("val").Float(0))
	}
	// id should be doubled
	if result.Col("id").Float(0) != 20 {
		t.Errorf("id[0] = %g, want 20", result.Col("id").Float(0))
	}
}

func TestDataFrameApplyCols(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	result := df.ApplyCols(func(col core.Series) core.Series {
		return col.(*ArrowSeries).MapFloat(func(v float64) float64 {
			return v + 1
		})
	})
	if result.Col("x").Float(0) != 2 || result.Col("y").Float(2) != 31 {
		t.Errorf("ApplyCols: x[0]=%g, y[2]=%g", result.Col("x").Float(0), result.Col("y").Float(2))
	}
}

func TestDataFrameApplyRows(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob"}, nil),
		NewFloat64Series("score", []float64{80, 90}, nil),
	)
	result := df.ApplyRows(func(row map[string]interface{}) map[string]interface{} {
		// Add a "passed" field based on score
		score, _ := row["score"].(float64)
		row["passed"] = score >= 85
		return row
	})
	rows, cols := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
	if cols != 3 { // name, score, passed
		t.Fatalf("cols = %d, want 3", cols)
	}
	if !result.Col("passed").Bool(1) {
		t.Error("bob should have passed=true")
	}
	if result.Col("passed").Bool(0) {
		t.Error("alice should have passed=false")
	}
}

func TestApplyRowsPreservesValues(t *testing.T) {
	df := NewDataFrame(
		NewInt64Series("a", []int64{1, 2}, nil),
		NewStringSeries("b", []string{"x", "y"}, nil),
	)
	result := df.ApplyRows(func(row map[string]interface{}) map[string]interface{} {
		return row // identity
	})
	if result.Col("a").Int(0) != 1 {
		t.Errorf("a[0] = %d, want 1", result.Col("a").Int(0))
	}
	if result.Col("b").String(1) != "y" {
		t.Errorf("b[1] = %q, want y", result.Col("b").String(1))
	}
}
