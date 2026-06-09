package arrow

import (
	"testing"
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

func makeTimestampSeries() *ArrowSeries {
	times := []time.Time{
		time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		time.Date(2024, 3, 20, 14, 45, 30, 0, time.UTC),
		time.Date(2024, 6, 5, 8, 0, 0, 0, time.UTC),
		time.Date(2024, 9, 10, 16, 15, 45, 0, time.UTC),
		time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
	}
	return NewTimestampSeries("ts", times, nil)
}

func TestDTYear(t *testing.T) {
	s := makeTimestampSeries()
	year := s.DT().Year()
	for i := 0; i < s.Len(); i++ {
		if year.Int(i) != 2024 {
			t.Errorf("Year[%d] = %d, want 2024", i, year.Int(i))
		}
	}
}

func TestDTMonth(t *testing.T) {
	s := makeTimestampSeries()
	month := s.DT().Month()
	expected := []int64{1, 3, 6, 9, 12}
	for i, want := range expected {
		if month.Int(i) != want {
			t.Errorf("Month[%d] = %d, want %d", i, month.Int(i), want)
		}
	}
}

func TestDTDay(t *testing.T) {
	s := makeTimestampSeries()
	day := s.DT().Day()
	expected := []int64{15, 20, 5, 10, 25}
	for i, want := range expected {
		if day.Int(i) != want {
			t.Errorf("Day[%d] = %d, want %d", i, day.Int(i), want)
		}
	}
}

func TestDTHourMinuteSecond(t *testing.T) {
	s := makeTimestampSeries()
	hour := s.DT().Hour()
	minute := s.DT().Minute()
	second := s.DT().Second()

	if hour.Int(0) != 10 || minute.Int(0) != 30 || second.Int(0) != 0 {
		t.Errorf("row 0: %02d:%02d:%02d", hour.Int(0), minute.Int(0), second.Int(0))
	}
	if hour.Int(1) != 14 || minute.Int(1) != 45 || second.Int(1) != 30 {
		t.Errorf("row 1: %02d:%02d:%02d", hour.Int(1), minute.Int(1), second.Int(1))
	}
}

func TestDTDayOfWeek(t *testing.T) {
	s := makeTimestampSeries()
	dow := s.DT().DayOfWeek()
	// 2024-01-15 is Monday (1)
	if dow.Int(0) != 1 {
		t.Errorf("DayOfWeek[0] = %d, want 1 (Monday)", dow.Int(0))
	}
}

func TestDTDayOfYear(t *testing.T) {
	s := makeTimestampSeries()
	doy := s.DT().DayOfYear()
	if doy.Int(0) != 15 {
		t.Errorf("DayOfYear[0] = %d, want 15", doy.Int(0))
	}
}

func TestDTQuarter(t *testing.T) {
	s := makeTimestampSeries()
	q := s.DT().Quarter()
	expected := []int64{1, 1, 2, 3, 4}
	for i, want := range expected {
		if q.Int(i) != want {
			t.Errorf("Quarter[%d] = %d, want %d", i, q.Int(i), want)
		}
	}
}

func TestDTDate(t *testing.T) {
	s := makeTimestampSeries()
	date := s.DT().Date()
	if date.String(0) != "2024-01-15" {
		t.Errorf("Date[0] = %q, want %q", date.String(0), "2024-01-15")
	}
}

func TestDTUnix(t *testing.T) {
	s := makeTimestampSeries()
	unix := s.DT().Unix()
	if unix.Int(0) != time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix() {
		t.Errorf("Unix[0] = %d", unix.Int(0))
	}
}

func TestDTNulls(t *testing.T) {
	times := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	s := NewTimestampSeriesWithNulls("ts", times, []bool{true, false, true}, nil)
	month := s.DT().Month()
	if !month.IsNull(1) {
		t.Error("expected null at index 1")
	}
	if month.Int(0) != 1 || month.Int(2) != 3 {
		t.Errorf("months: %d, %d", month.Int(0), month.Int(2))
	}
}

func TestShift(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20, 30, 40, 50}, nil)
	shifted := s.Shift(2)
	if shifted.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", shifted.Len())
	}
	// First 2 should be null
	if !shifted.IsNull(0) || !shifted.IsNull(1) {
		t.Error("expected null for first 2 elements")
	}
	if shifted.Int(2) != 10 {
		t.Errorf("Shift(2)[2] = %d, want 10", shifted.Int(2))
	}
	if shifted.Int(4) != 30 {
		t.Errorf("Shift(2)[4] = %d, want 30", shifted.Int(4))
	}
}

func TestShiftNegative(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20, 30, 40, 50}, nil)
	shifted := s.Shift(-1)
	// Last should be null
	if !shifted.IsNull(4) {
		t.Error("expected null for last element")
	}
	if shifted.Int(0) != 20 {
		t.Errorf("Shift(-1)[0] = %d, want 20", shifted.Int(0))
	}
}

func TestPctChange(t *testing.T) {
	s := NewFloat64Series("x", []float64{100, 110, 121, 133.1}, nil)
	pct := s.PctChange(1)
	if !pct.IsNull(0) {
		t.Error("expected null at index 0")
	}
	if pct.Float(1) != 0.1 {
		t.Errorf("PctChange[1] = %g, want 0.1", pct.Float(1))
	}
}

func TestDiff(t *testing.T) {
	s := NewFloat64Series("x", []float64{10, 15, 13, 20}, nil)
	diff := s.Diff(1)
	if !diff.IsNull(0) {
		t.Error("expected null at index 0")
	}
	if diff.Float(1) != 5.0 {
		t.Errorf("Diff[1] = %g, want 5", diff.Float(1))
	}
	if diff.Float(2) != -2.0 {
		t.Errorf("Diff[2] = %g, want -2", diff.Float(2))
	}
}

func TestCumSum(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4}, nil)
	cum := s.CumSum()
	expected := []float64{1, 3, 6, 10}
	for i, want := range expected {
		if cum.Float(i) != want {
			t.Errorf("CumSum[%d] = %g, want %g", i, cum.Float(i), want)
		}
	}
}

func TestCumMax(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5}, nil)
	cum := s.CumMax()
	expected := []float64{3, 3, 4, 4, 5}
	for i, want := range expected {
		if cum.Float(i) != want {
			t.Errorf("CumMax[%d] = %g, want %g", i, cum.Float(i), want)
		}
	}
}

func TestDateRange(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s := DateRange("dates", start, 5, 24*time.Hour)
	if s.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", s.Len())
	}
	// Check last date
	last := time.Unix(0, s.Int(4))
	if last.Day() != 5 {
		t.Errorf("last day = %d, want 5", last.Day())
	}
}

func TestResample(t *testing.T) {
	// Create hourly data
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	times := make([]time.Time, 24)
	for i := 0; i < 24; i++ {
		times[i] = start.Add(time.Duration(i) * time.Hour)
	}
	vals := make([]float64, 24)
	for i := range vals {
		vals[i] = float64(i + 1)
	}
	s := NewTimestampSeries("ts", times, nil)
	_ = s
	valSeries := NewFloat64Series("val", vals, nil)

	// Create a DataFrame with timestamp and value
	df := NewDataFrame(s, valSeries)

	// Resample to 6 hours, sum
	rule := core.NewResampleRule(6*time.Hour, core.ResampleSum)
	resampled := ResampleDataFrame(df, "ts", rule)
	rows, _ := resampled.Shape()
	if rows != 4 {
		t.Fatalf("resampled rows = %d, want 4", rows)
	}
}

// Helper
func NewTimestampSeriesWithNulls(name string, times []time.Time, valid []bool, index core.Index) *ArrowSeries {
	alloc := newAllocator()
	bldr := newInt64Builder(alloc)
	bldr.Resize(len(times))
	for i, t := range times {
		if i < len(valid) && !valid[i] {
			bldr.AppendNull()
		} else {
			bldr.Append(t.UnixNano())
		}
	}
	return NewArrowSeries(name, bldr.NewArray(), index)
}

func newAllocator() memory.Allocator {
	return memory.NewGoAllocator()
}

func newInt64Builder(alloc memory.Allocator) *array.Int64Builder {
	return array.NewInt64Builder(alloc)
}
