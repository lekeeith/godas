package arrow

import (
	"testing"
)

func TestAsofJoin(t *testing.T) {
	// Trades: timestamp, price
	trades := NewDataFrame(
		NewFloat64Series("ts", []float64{1, 3, 5, 7}, nil),
		NewFloat64Series("trade_price", []float64{100, 101, 102, 103}, nil),
	)
	// Quotes: timestamp, bid
	quotes := NewDataFrame(
		NewFloat64Series("ts", []float64{0, 2, 4, 6, 8}, nil),
		NewFloat64Series("bid", []float64{99, 100, 101, 102, 103}, nil),
	)
	result := trades.AsofJoin(quotes, "ts", nil)
	rows, _ := result.Shape()
	if rows != 4 {
		t.Fatalf("rows = %d, want 4", rows)
	}
	// trade at ts=1 should get quote at ts=0 (bid=99)
	if result.Col("bid").(*ArrowSeries).Float(0) != 99 {
		t.Errorf("bid[0] = %g, want 99", result.Col("bid").(*ArrowSeries).Float(0))
	}
	// trade at ts=3 should get quote at ts=2 (bid=100)
	if result.Col("bid").(*ArrowSeries).Float(1) != 100 {
		t.Errorf("bid[1] = %g, want 100", result.Col("bid").(*ArrowSeries).Float(1))
	}
}

func TestSemiJoin(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("key", []string{"a", "b", "c", "d"}, nil),
		NewInt64Series("val", []int64{1, 2, 3, 4}, nil),
	)
	right := NewDataFrame(
		NewStringSeries("key", []string{"b", "d", "e"}, nil),
	)
	result := left.SemiJoin(right, []string{"key"})
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
	if result.Col("key").String(0) != "b" || result.Col("key").String(1) != "d" {
		t.Errorf("keys: %s,%s", result.Col("key").String(0), result.Col("key").String(1))
	}
}

func TestAntiJoin(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("key", []string{"a", "b", "c", "d"}, nil),
		NewInt64Series("val", []int64{1, 2, 3, 4}, nil),
	)
	right := NewDataFrame(
		NewStringSeries("key", []string{"b", "d", "e"}, nil),
	)
	result := left.AntiJoin(right, []string{"key"})
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
	if result.Col("key").String(0) != "a" || result.Col("key").String(1) != "c" {
		t.Errorf("keys: %s,%s", result.Col("key").String(0), result.Col("key").String(1))
	}
}

func TestCrossJoin(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("a", []string{"x", "y"}, nil),
	)
	right := NewDataFrame(
		NewInt64Series("b", []int64{1, 2, 3}, nil),
	)
	result := left.CrossJoin(right)
	rows, cols := result.Shape()
	if rows != 6 {
		t.Fatalf("rows = %d, want 6", rows)
	}
	if cols != 2 {
		t.Fatalf("cols = %d, want 2", cols)
	}
	// x1, x2, x3, y1, y2, y3
	if result.Col("a").String(0) != "x" || result.Col("b").Int(0) != 1 {
		t.Errorf("first: %s,%d", result.Col("a").String(0), result.Col("b").Int(0))
	}
	if result.Col("a").String(3) != "y" || result.Col("b").Int(3) != 1 {
		t.Errorf("fourth: %s,%d", result.Col("a").String(3), result.Col("b").Int(3))
	}
}

func TestSemiJoinNoMatch(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("key", []string{"a", "b"}, nil),
	)
	right := NewDataFrame(
		NewStringSeries("key", []string{"x", "y"}, nil),
	)
	result := left.SemiJoin(right, []string{"key"})
	if result.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", result.Len())
	}
}

func TestAntiJoinAllMatch(t *testing.T) {
	left := NewDataFrame(
		NewStringSeries("key", []string{"a", "b"}, nil),
	)
	right := NewDataFrame(
		NewStringSeries("key", []string{"a", "b", "c"}, nil),
	)
	result := left.AntiJoin(right, []string{"key"})
	if result.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", result.Len())
	}
}

func TestAsofJoinEmpty(t *testing.T) {
	left := NewDataFrame(
		NewFloat64Series("ts", []float64{1, 2}, nil),
	)
	right := NewDataFrame(
		NewFloat64Series("ts", []float64{}, nil),
		NewFloat64Series("val", []float64{}, nil),
	)
	result := left.AsofJoin(right, "ts", nil)
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
	// All right values should be null
	if result.Col("val").NullCount() != 2 {
		t.Errorf("nulls = %d, want 2", result.Col("val").NullCount())
	}
}
