package arrow

import (
	"testing"
	"time"

	"github.com/godans/godans/core"
)

func TestTAAddDuration(t *testing.T) {
	times := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	s := NewTimestampSeries("ts", times, nil)
	result := s.TA().AddDuration(24 * time.Hour)

	got := time.Unix(0, result.Int(0)).UTC()
	if got.Day() != 2 {
		t.Errorf("AddDuration: got day %d, want 2", got.Day())
	}
	got1 := time.Unix(0, result.Int(1)).UTC()
	if got1.Day() != 3 {
		t.Errorf("AddDuration: got day %d, want 3", got1.Day())
	}
}

func TestTASubDuration(t *testing.T) {
	times := []time.Time{
		time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}
	s := NewTimestampSeries("ts", times, nil)
	result := s.TA().SubDuration(2 * time.Hour)

	got := time.Unix(0, result.Int(0)).UTC()
	if got.Hour() != 10 {
		t.Errorf("SubDuration: got hour %d, want 10", got.Hour())
	}
}

func TestTASubTimestamps(t *testing.T) {
	t1 := []time.Time{
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	t2 := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	s1 := NewTimestampSeries("a", t1, nil)
	s2 := NewTimestampSeries("b", t2, nil)
	dur := s1.TA().SubTimestamps(s2)

	// Should be 1 day and 2 days in nanoseconds
	day := int64(24 * time.Hour)
	if dur.Int(0) != day {
		t.Errorf("SubTimestamps[0] = %d, want %d", dur.Int(0), day)
	}
	if dur.Int(1) != 2*day {
		t.Errorf("SubTimestamps[1] = %d, want %d", dur.Int(1), 2*day)
	}
}

func TestTAAddSeries(t *testing.T) {
	// timestamp + duration
	times := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	ts := NewTimestampSeries("ts", times, nil)
	dur := DurationFromNanos("dur", []int64{int64(48 * time.Hour)}, nil)

	result := ts.TA().AddSeries(dur)
	got := time.Unix(0, result.Int(0)).UTC()
	if got.Day() != 3 {
		t.Errorf("AddSeries: got day %d, want 3", got.Day())
	}
}

func TestTADurationAdd(t *testing.T) {
	d1 := DurationFromNanos("a", []int64{int64(time.Hour), int64(2 * time.Hour)}, nil)
	d2 := DurationFromNanos("b", []int64{int64(30 * time.Minute), int64(time.Hour)}, nil)
	result := d1.TA().DurationAdd(d2)

	// 1h + 30m = 90m
	if result.Int(0) != int64(90*time.Minute) {
		t.Errorf("DurationAdd[0] = %v, want %v", time.Duration(result.Int(0)), 90*time.Minute)
	}
	// 2h + 1h = 3h
	if result.Int(1) != int64(3*time.Hour) {
		t.Errorf("DurationAdd[1] = %v, want %v", time.Duration(result.Int(1)), 3*time.Hour)
	}
}

func TestTADurationSub(t *testing.T) {
	d1 := DurationFromNanos("a", []int64{int64(3 * time.Hour)}, nil)
	d2 := DurationFromNanos("b", []int64{int64(time.Hour)}, nil)
	result := d1.TA().DurationSub(d2)

	if result.Int(0) != int64(2*time.Hour) {
		t.Errorf("DurationSub = %v, want 2h", time.Duration(result.Int(0)))
	}
}

func TestTADurationMul(t *testing.T) {
	d := DurationFromNanos("d", []int64{int64(time.Hour)}, nil)
	result := d.TA().DurationMul(2.5)

	if result.Int(0) != int64(2*time.Hour+30*time.Minute) {
		t.Errorf("DurationMul = %v, want 2h30m", time.Duration(result.Int(0)))
	}
}

func TestTADurationDiv(t *testing.T) {
	d := DurationFromNanos("d", []int64{int64(3 * time.Hour)}, nil)
	result := d.TA().DurationDiv(1.5)

	if result.Int(0) != int64(2*time.Hour) {
		t.Errorf("DurationDiv = %v, want 2h", time.Duration(result.Int(0)))
	}
}

func TestTADurationDivDuration(t *testing.T) {
	d1 := DurationFromNanos("a", []int64{int64(3 * time.Hour)}, nil)
	d2 := DurationFromNanos("b", []int64{int64(time.Hour)}, nil)
	ratio := d1.TA().DurationDivDuration(d2)

	if ratio.Float(0) != 3.0 {
		t.Errorf("DurationDivDuration = %g, want 3.0", ratio.Float(0))
	}
}

func TestTABeforeAfter(t *testing.T) {
	t1 := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
	}
	t2 := []time.Time{
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	s1 := NewTimestampSeries("a", t1, nil)
	s2 := NewTimestampSeries("b", t2, nil)

	before := s1.TA().Before(s2)
	if !before.Bool(0) {
		t.Error("Before[0] should be true (Jan < Jun)")
	}
	if before.Bool(1) {
		t.Error("Before[1] should be false (Jun == Jun)")
	}
	if before.Bool(2) {
		t.Error("Before[2] should be false (Dec > Jun)")
	}

	after := s1.TA().After(s2)
	if after.Bool(0) {
		t.Error("After[0] should be false")
	}
	if !after.Bool(2) {
		t.Error("After[2] should be true")
	}
}

func TestTAToDaysHoursMinutes(t *testing.T) {
	d := DurationFromNanos("d", []int64{
		int64(25 * time.Hour),
		int64(90 * time.Minute),
	}, nil)

	days := d.TA().ToDays()
	if days.Float(0) != 25.0/24.0*24.0 {
		// 25 hours = 1.041666... days
		if days.Float(0) < 1.04 || days.Float(0) > 1.05 {
			t.Errorf("ToDays = %g", days.Float(0))
		}
	}

	hours := d.TA().ToHours()
	if hours.Float(0) != 25.0 {
		t.Errorf("ToHours = %g, want 25", hours.Float(0))
	}
	if hours.Float(1) != 1.5 {
		t.Errorf("ToHours = %g, want 1.5", hours.Float(1))
	}

	mins := d.TA().ToMinutes()
	if mins.Float(0) != 1500.0 {
		t.Errorf("ToMinutes = %g, want 1500", mins.Float(0))
	}
}

func TestTANullPropagation(t *testing.T) {
	bldr := NewSeriesBuilder("ts", core.INT64, nil)
	bldr.AppendInt(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	bldr.AppendNull()
	bldr.AppendInt(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC).UnixNano())
	s := bldr.Build()

	result := s.TA().AddDuration(24 * time.Hour)
	if !result.IsNull(1) {
		t.Error("null + duration should be null")
	}
}

func TestDurationFromFloat(t *testing.T) {
	vals := []float64{1.5, 2.0, 0.5}
	d := DurationFromFloat("d", vals, time.Hour, nil)
	if d.Int(0) != int64(90*time.Minute) {
		t.Errorf("DurationFromFloat[0] = %v, want 90m", time.Duration(d.Int(0)))
	}
}

func TestFormatDuration(t *testing.T) {
	d := 25 * time.Hour
	s := FormatDuration(int64(d))
	if s != "1d1h0m0s" {
		t.Errorf("FormatDuration = %q, want %q", s, "1d1h0m0s")
	}
}

func TestTADivByZero(t *testing.T) {
	d := DurationFromNanos("d", []int64{int64(time.Hour)}, nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on division by zero")
		}
	}()
	d.TA().DurationDiv(0)
}
