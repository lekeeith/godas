package arrow

import (
	"testing"

	"github.com/lekeeith/godas/core"
)

func TestExprCol(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	result := Col("x").Eval(df).(*ArrowSeries)
	if result.Float(0) != 1 || result.Float(2) != 3 {
		t.Errorf("Col: %g,%g", result.Float(0), result.Float(2))
	}
}

func TestExprAdd(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	result := Col("x").Add(Col("y")).Eval(df).(*ArrowSeries)
	if result.Float(0) != 11 || result.Float(2) != 33 {
		t.Errorf("Add: %g,%g", result.Float(0), result.Float(2))
	}
}

func TestExprSubMulDiv(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{10, 20, 30}, nil),
		NewFloat64Series("y", []float64{2, 4, 5}, nil),
	)
	sub := Col("x").Sub(Col("y")).Eval(df).(*ArrowSeries)
	mul := Col("x").Mul(Col("y")).Eval(df).(*ArrowSeries)
	div := Col("x").Div(Col("y")).Eval(df).(*ArrowSeries)

	if sub.Float(0) != 8 || sub.Float(2) != 25 {
		t.Errorf("Sub: %g,%g", sub.Float(0), sub.Float(2))
	}
	if mul.Float(0) != 20 || mul.Float(1) != 80 {
		t.Errorf("Mul: %g,%g", mul.Float(0), mul.Float(1))
	}
	if div.Float(0) != 5 || div.Float(2) != 6 {
		t.Errorf("Div: %g,%g", div.Float(0), div.Float(2))
	}
}

func TestExprComparison(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 5, 3}, nil),
	)
	gt := Col("x").Gt(Lit(3.0)).Eval(df).(*ArrowSeries)
	if gt.Bool(0) || !gt.Bool(1) || gt.Bool(2) {
		t.Error("Gt failed")
	}

	eq := Col("x").Eq(Lit(5.0)).Eval(df).(*ArrowSeries)
	if eq.Bool(0) || !eq.Bool(1) || eq.Bool(2) {
		t.Error("Eq failed")
	}
}

func TestExprLogic(t *testing.T) {
	df := NewDataFrame(
		NewBoolSeries("a", []bool{true, true, false}, nil),
		NewBoolSeries("b", []bool{true, false, false}, nil),
	)
	and := Col("a").And(Col("b")).Eval(df).(*ArrowSeries)
	if !and.Bool(0) || and.Bool(1) || and.Bool(2) {
		t.Error("And failed")
	}

	or := Col("a").Or(Col("b")).Eval(df).(*ArrowSeries)
	if !or.Bool(0) || !or.Bool(1) || or.Bool(2) {
		t.Error("Or failed")
	}

	not := Col("a").Not().Eval(df).(*ArrowSeries)
	if not.Bool(0) || !not.Bool(2) {
		t.Error("Not failed")
	}
}

func TestExprAgg(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil),
	)
	sum := Col("x").Sum().Eval(df).(*ArrowSeries)
	if sum.Float(0) != 15 {
		t.Errorf("Sum = %g, want 15", sum.Float(0))
	}

	mean := Col("x").Mean().Eval(df).(*ArrowSeries)
	if mean.Float(0) != 3 {
		t.Errorf("Mean = %g, want 3", mean.Float(0))
	}
}

func TestExprApply(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 4, 9}, nil),
	)
	result := Col("x").Apply(func(v float64) float64 { return v * 2 }).Eval(df).(*ArrowSeries)
	if result.Float(0) != 2 || result.Float(2) != 18 {
		t.Errorf("Apply: %g,%g", result.Float(0), result.Float(2))
	}
}

func TestExprChained(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	// (x + y) * 2
	expr := Col("x").Add(Col("y")).Mul(Lit(2.0))
	result := expr.Eval(df).(*ArrowSeries)
	if result.Float(0) != 22 || result.Float(2) != 66 {
		t.Errorf("Chained: %g,%g", result.Float(0), result.Float(2))
	}
}

func TestExprAlias(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	)
	expr := Col("x").Add(Lit(10.0)).Alias("x_plus_10")
	result := expr.Eval(df).(*ArrowSeries)
	if result.Float(0) != 11 {
		t.Errorf("result = %g, want 11", result.Float(0))
	}
}

func TestExprString(t *testing.T) {
	expr := Col("x").Add(Col("y")).Mul(Lit(2.0))
	s := expr.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestExprIsNull(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.FLOAT64, nil)
	bldr.AppendFloat(1)
	bldr.AppendNull()
	bldr.AppendFloat(3)
	s := bldr.Build()
	df := NewDataFrame(s)

	isNull := Col("x").IsNull().Eval(df).(*ArrowSeries)
	if isNull.Bool(0) || !isNull.Bool(1) || isNull.Bool(2) {
		t.Error("IsNull failed")
	}
}

func TestExprFillNA(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.FLOAT64, nil)
	bldr.AppendFloat(1)
	bldr.AppendNull()
	bldr.AppendFloat(3)
	s := bldr.Build()
	df := NewDataFrame(s)

	result := Col("x").FillNA(99.0).Eval(df).(*ArrowSeries)
	if result.Float(0) != 1 || result.Float(1) != 99 || result.Float(2) != 3 {
		t.Errorf("FillNA: %g,%g,%g", result.Float(0), result.Float(1), result.Float(2))
	}
}

func TestExprNegAbs(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{-3, 0, 5}, nil),
	)
	neg := Col("x").Neg().Eval(df).(*ArrowSeries)
	if neg.Float(0) != 3 || neg.Float(2) != -5 {
		t.Errorf("Neg: %g,%g", neg.Float(0), neg.Float(2))
	}

	abs := Col("x").Abs().Eval(df).(*ArrowSeries)
	if abs.Float(0) != 3 || abs.Float(2) != 5 {
		t.Errorf("Abs: %g,%g", abs.Float(0), abs.Float(2))
	}
}

func TestExprClip(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{-5, 3, 12}, nil),
	)
	result := Col("x").Clip(0, 10).Eval(df).(*ArrowSeries)
	if result.Float(0) != 0 || result.Float(1) != 3 || result.Float(2) != 10 {
		t.Errorf("Clip: %g,%g,%g", result.Float(0), result.Float(1), result.Float(2))
	}
}

func TestExprStrUpper(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob"}, nil),
	)
	result := Col("name").StrUpper().Eval(df).(*ArrowSeries)
	if result.String(0) != "ALICE" || result.String(1) != "BOB" {
		t.Errorf("StrUpper: %s,%s", result.String(0), result.String(1))
	}
}

func TestExprStrContains(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "alicia"}, nil),
	)
	result := Col("name").StrContains("ali").Eval(df).(*ArrowSeries)
	if !result.Bool(0) || result.Bool(1) || !result.Bool(2) {
		t.Error("StrContains failed")
	}
}
