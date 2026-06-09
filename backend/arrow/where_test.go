package arrow

import (
	"testing"
)

func TestWhere(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	cond := NewBoolSeries("c", []bool{true, true, false, false, true}, nil)
	r := s.Where(cond, 0.0).(*ArrowSeries)
	expected := []float64{1, 2, 0, 0, 5}
	for i, want := range expected {
		if r.Float(i) != want {
			t.Errorf("Where[%d] = %g, want %g", i, r.Float(i), want)
		}
	}
}

func TestMask(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	cond := NewBoolSeries("c", []bool{true, true, false, false, true}, nil)
	r := s.Mask(cond, 0.0).(*ArrowSeries)
	expected := []float64{0, 0, 3, 4, 0}
	for i, want := range expected {
		if r.Float(i) != want {
			t.Errorf("Mask[%d] = %g, want %g", i, r.Float(i), want)
		}
	}
}

func TestWhereNull(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3}, nil)
	cond := NewBoolSeries("c", []bool{true, false, true}, nil)
	r := s.Where(cond, nil).(*ArrowSeries)
	if r.IsNull(1) != true {
		t.Error("Where with nil should produce null")
	}
	if r.Float(0) != 1 || r.Float(2) != 3 {
		t.Error("Where kept values wrong")
	}
}

func TestDataFrameWhere(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("a", []float64{1, 2, 3}, nil),
		NewFloat64Series("b", []float64{10, 20, 30}, nil),
	)
	cond := NewBoolSeries("c", []bool{true, false, true}, nil)
	r := df.Where(cond, 0.0)
	if r.Col("a").(*ArrowSeries).Float(1) != 0 {
		t.Error("DataFrame.Where failed for a[1]")
	}
	if r.Col("b").(*ArrowSeries).Float(0) != 10 {
		t.Error("DataFrame.Where failed for b[0]")
	}
}

func TestDataFrameMask(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("a", []float64{1, 2, 3}, nil),
	)
	cond := NewBoolSeries("c", []bool{false, true, false}, nil)
	r := df.Mask(cond, 99.0)
	if r.Col("a").(*ArrowSeries).Float(0) != 1 {
		t.Error("DataFrame.Mask should keep a[0]")
	}
	if r.Col("a").(*ArrowSeries).Float(1) != 99 {
		t.Error("DataFrame.Mask should replace a[1]")
	}
}
