package arrow

import (
	"testing"
	"time"

	"github.com/lekeeith/godas/core"
)

func TestGroupByDynamic(t *testing.T) {
	// Create hourly data over 2 days
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	times := make([]time.Time, 48)
	vals := make([]float64, 48)
	for i := 0; i < 48; i++ {
		times[i] = start.Add(time.Duration(i) * time.Hour)
		vals[i] = float64(i + 1)
	}

	df := NewDataFrame(
		NewTimestampSeries("ts", times, nil),
		NewFloat64Series("val", vals, nil),
	)

	// Group by 12-hour windows, sum
	result := df.GroupByDynamic("ts", 12*time.Hour, 0).
		Agg(map[string]core.AggFunc{"val": core.AggSum})

	rows, _ := result.Shape()
	if rows != 4 { // 48h / 12h = 4 groups
		t.Fatalf("rows = %d, want 4", rows)
	}
}

func TestGroupByDynamicWithBy(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df := NewDataFrame(
		NewTimestampSeries("ts", []time.Time{
			start, start.Add(time.Hour),
			start, start.Add(time.Hour),
		}, nil),
		NewStringSeries("cat", []string{"A", "A", "B", "B"}, nil),
		NewFloat64Series("val", []float64{10, 20, 30, 40}, nil),
	)

	result := df.GroupByDynamic("ts", 2*time.Hour, 0).
		By("cat").
		Agg(map[string]core.AggFunc{"val": core.AggSum})

	rows, _ := result.Shape()
	if rows != 2 { // A and B in same window
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestGroupByDynamicInfo(t *testing.T) {
	df := NewDataFrame(
		NewTimestampSeries("ts", []time.Time{time.Now()}, nil),
	)
	dg := df.GroupByDynamic("ts", time.Hour, 0)
	s := dg.Info()
	if s == "" {
		t.Error("Info() returned empty")
	}
}

func TestGroupByDynamicEmpty(t *testing.T) {
	df := NewDataFrame(
		NewTimestampSeries("ts", []time.Time{}, nil),
		NewFloat64Series("val", []float64{}, nil),
	)
	result := df.GroupByDynamic("ts", time.Hour, 0).
		Agg(map[string]core.AggFunc{"val": core.AggSum})
	rows, _ := result.Shape()
	if rows != 0 {
		t.Fatalf("rows = %d, want 0", rows)
	}
}

func TestGroupByDynamicClosed(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	df := NewDataFrame(
		NewTimestampSeries("ts", []time.Time{
			start,
			start.Add(2 * time.Hour),
			start.Add(4 * time.Hour),
		}, nil),
		NewFloat64Series("val", []float64{1, 2, 3}, nil),
	)

	result := df.GroupByDynamic("ts", 3*time.Hour, 0).
		Closed("right").
		Agg(map[string]core.AggFunc{"val": core.AggSum})

	rows, _ := result.Shape()
	if rows == 0 {
		t.Fatal("expected at least 1 group")
	}
}
