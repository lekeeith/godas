package arrow

import (
	"math"
	"testing"

	"github.com/lekeeith/godas/core"
)

func TestAdd(t *testing.T) {
	a := NewFloat64Series("x", []float64{1, 2, 3}, nil)
	b := NewFloat64Series("y", []float64{10, 20, 30}, nil)
	result := a.Add(b)
	expected := []float64{11, 22, 33}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Add[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestAddScalar(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3}, nil)
	result := s.AddScalar(100)
	if result.Float(0) != 101 || result.Float(2) != 103 {
		t.Errorf("AddScalar: %g, %g", result.Float(0), result.Float(2))
	}
}

func TestSub(t *testing.T) {
	a := NewFloat64Series("x", []float64{10, 20, 30}, nil)
	b := NewFloat64Series("y", []float64{1, 2, 3}, nil)
	result := a.Sub(b)
	expected := []float64{9, 18, 27}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Sub[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestMul(t *testing.T) {
	a := NewFloat64Series("x", []float64{2, 3, 4}, nil)
	b := NewFloat64Series("y", []float64{5, 6, 7}, nil)
	result := a.Mul(b)
	expected := []float64{10, 18, 28}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Mul[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestDiv(t *testing.T) {
	a := NewFloat64Series("x", []float64{10, 20, 30}, nil)
	b := NewFloat64Series("y", []float64{2, 5, 10}, nil)
	result := a.Div(b)
	expected := []float64{5, 4, 3}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Div[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestDivByZero(t *testing.T) {
	a := NewFloat64Series("x", []float64{10, 20}, nil)
	b := NewFloat64Series("y", []float64{0, 5}, nil)
	result := a.Div(b)
	if !math.IsNaN(result.Float(0)) {
		t.Error("DivByZero should return NaN")
	}
	if result.Float(1) != 4 {
		t.Errorf("Div[1] = %g, want 4", result.Float(1))
	}
}

func TestMod(t *testing.T) {
	a := NewFloat64Series("x", []float64{10, 7, 15}, nil)
	b := NewFloat64Series("y", []float64{3, 2, 4}, nil)
	result := a.Mod(b)
	expected := []float64{1, 1, 3}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Mod[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestNeg(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, -2, 3}, nil)
	result := s.Neg()
	expected := []float64{-1, 2, -3}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Neg[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestAbs(t *testing.T) {
	s := NewFloat64Series("x", []float64{-1, 2, -3, 0}, nil)
	result := s.Abs()
	expected := []float64{1, 2, 3, 0}
	for i, want := range expected {
		if result.Float(i) != want {
			t.Errorf("Abs[%d] = %g, want %g", i, result.Float(i), want)
		}
	}
}

func TestArithmeticWithNulls(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.FLOAT64, nil)
	bldr.AppendFloat(1)
	bldr.AppendNull()
	bldr.AppendFloat(3)
	a := bldr.Build()
	b := NewFloat64Series("y", []float64{10, 20, 30}, nil)
	result := a.Add(b)
	if !result.IsNull(1) {
		t.Error("null + value should be null")
	}
	if result.Float(0) != 11 {
		t.Errorf("Add[0] = %g, want 11", result.Float(0))
	}
}

func TestIntArithmetic(t *testing.T) {
	a := NewInt64Series("x", []int64{10, 20, 30}, nil)
	b := NewInt64Series("y", []int64{3, 4, 5}, nil)
	sum := a.Add(b)
	if sum.Float(0) != 13 || sum.Float(2) != 35 {
		t.Errorf("IntAdd: %g, %g", sum.Float(0), sum.Float(2))
	}
	div := a.Div(b)
	if div.Float(0) != 10.0/3.0 {
		t.Errorf("IntDiv: %g", div.Float(0))
	}
}

func TestComparisonOps(t *testing.T) {
	a := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	b := NewFloat64Series("y", []float64{5, 4, 3, 2, 1}, nil)

	lt := a.Lt(b)
	if !lt.Bool(0) || lt.Bool(4) {
		t.Error("Lt failed")
	}

	gt := a.Gt(b)
	if gt.Bool(0) || !gt.Bool(4) {
		t.Error("Gt failed")
	}

	eq := a.Eq(b)
	if eq.Bool(0) || !eq.Bool(2) {
		t.Error("Eq failed")
	}
}

func TestComparisonScalar(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	gt3 := s.GtScalar(3)
	if gt3.Bool(0) || gt3.Bool(1) || gt3.Bool(2) || !gt3.Bool(3) || !gt3.Bool(4) {
		t.Error("GtScalar(3) failed")
	}

	le2 := s.LeScalar(2)
	if !le2.Bool(0) || !le2.Bool(1) || le2.Bool(2) {
		t.Error("LeScalar(2) failed")
	}
}

func TestLogicOps(t *testing.T) {
	a := NewBoolSeries("a", []bool{true, true, false, false}, nil)
	b := NewBoolSeries("b", []bool{true, false, true, false}, nil)

	and := a.And(b)
	if !and.Bool(0) || and.Bool(1) || and.Bool(2) || and.Bool(3) {
		t.Error("And failed")
	}

	or := a.Or(b)
	if !or.Bool(0) || !or.Bool(1) || !or.Bool(2) || or.Bool(3) {
		t.Error("Or failed")
	}

	not := a.Not()
	if not.Bool(0) || !not.Bool(2) {
		t.Error("Not failed")
	}
}
