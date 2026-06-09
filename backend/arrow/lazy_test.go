package arrow

import (
	"testing"

	"github.com/lekeeith/godas/core"
)

func TestLazySelect(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil),
		NewFloat64Series("age", []float64{25, 30, 35}, nil),
		NewFloat64Series("score", []float64{88, 92, 76}, nil),
	)
	result := df.Lazy().
		Select(Col("name"), Col("score")).
		Collect()
	_, cols := result.Shape()
	if cols != 2 {
		t.Fatalf("cols = %d, want 2", cols)
	}
}

func TestLazyFilter(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil),
		NewFloat64Series("age", []float64{25, 30, 35}, nil),
	)
	result := df.Lazy().
		Filter(Col("age").Gt(Lit(28.0))).
		Collect()
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
	if result.Col("name").String(0) != "bob" {
		t.Errorf("name[0] = %q", result.Col("name").String(0))
	}
}

func TestLazyWithColumn(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	)
	result := df.Lazy().
		WithColumn("x2", Col("x").Mul(Col("x"))).
		Collect()
	if result.Col("x2").(*ArrowSeries).Float(0) != 1 || result.Col("x2").(*ArrowSeries).Float(2) != 9 {
		t.Errorf("x2: %g,%g", result.Col("x2").(*ArrowSeries).Float(0), result.Col("x2").(*ArrowSeries).Float(2))
	}
}

func TestLazySort(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"c", "a", "b"}, nil),
		NewFloat64Series("val", []float64{30, 10, 20}, nil),
	)
	result := df.Lazy().
		Sort([]string{"val"}, []bool{true}).
		Collect()
	if result.Col("name").String(0) != "a" {
		t.Errorf("name[0] = %q, want a", result.Col("name").String(0))
	}
}

func TestLazyLimit(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil),
	)
	result := df.Lazy().Limit(3).Collect()
	if result.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", result.Len())
	}
}

func TestLazyGroupBy(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90}, nil),
	)
	result := df.Lazy().
		GroupBy("dept").
		Agg(map[string]core.AggFunc{"salary": core.AggMean}).
		Collect()
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestLazyChained(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie", "dave"}, nil),
		NewFloat64Series("age", []float64{25, 30, 35, 28}, nil),
		NewFloat64Series("score", []float64{88, 92, 76, 95}, nil),
	)
	result := df.Lazy().
		Filter(Col("age").Ge(Lit(28.0))).
		WithColumn("grade", Col("score").Div(Lit(10.0))).
		Sort([]string{"score"}, []bool{false}).
		Limit(2).
		Collect()
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
	// Highest score first
	if result.Col("name").String(0) != "dave" {
		t.Errorf("name[0] = %q, want dave", result.Col("name").String(0))
	}
}

func TestLazyExprEval(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
		NewFloat64Series("y", []float64{10, 20, 30}, nil),
	)
	// (x + y) * 2 - 1
	expr := Col("x").Add(Col("y")).Mul(Lit(2.0)).Sub(Lit(1.0))
	result := df.Lazy().
		WithColumn("z", expr).
		Collect()
	z := result.Col("z").(*ArrowSeries)
	if z.Float(0) != 21 || z.Float(1) != 43 || z.Float(2) != 65 {
		t.Errorf("z: %g,%g,%g", z.Float(0), z.Float(1), z.Float(2))
	}
}

func TestLazyDescribe(t *testing.T) {
	lf := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	).Lazy().
		Filter(Col("x").Gt(Lit(1.0))).
		Select(Col("x"))
	s := lf.Describe()
	if s == "" {
		t.Error("Describe() returned empty")
	}
}

func TestLazyEmpty(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	)
	result := df.Lazy().Collect()
	if result.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", result.Len())
	}
}
