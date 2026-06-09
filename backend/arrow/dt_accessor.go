package arrow

import (
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

// ArrowDateTimeAccessor implements core.DateTimeAccessor for Arrow-backed series.
type ArrowDateTimeAccessor struct {
	series *ArrowSeries
}

// NewDateTimeAccessor creates a DT accessor for the given series.
func NewDateTimeAccessor(s *ArrowSeries) *ArrowDateTimeAccessor {
	return &ArrowDateTimeAccessor{series: s}
}

func (dt *ArrowDateTimeAccessor) Year() core.Series      { return dt.extractField(func(t time.Time) int64 { return int64(t.Year()) }) }
func (dt *ArrowDateTimeAccessor) Month() core.Series     { return dt.extractField(func(t time.Time) int64 { return int64(t.Month()) }) }
func (dt *ArrowDateTimeAccessor) Day() core.Series       { return dt.extractField(func(t time.Time) int64 { return int64(t.Day()) }) }
func (dt *ArrowDateTimeAccessor) Hour() core.Series      { return dt.extractField(func(t time.Time) int64 { return int64(t.Hour()) }) }
func (dt *ArrowDateTimeAccessor) Minute() core.Series    { return dt.extractField(func(t time.Time) int64 { return int64(t.Minute()) }) }
func (dt *ArrowDateTimeAccessor) Second() core.Series    { return dt.extractField(func(t time.Time) int64 { return int64(t.Second()) }) }
func (dt *ArrowDateTimeAccessor) DayOfWeek() core.Series { return dt.extractField(func(t time.Time) int64 { return int64(t.Weekday()) }) }
func (dt *ArrowDateTimeAccessor) DayOfYear() core.Series { return dt.extractField(func(t time.Time) int64 { return int64(t.YearDay()) }) }
func (dt *ArrowDateTimeAccessor) Quarter() core.Series   { return dt.extractField(func(t time.Time) int64 { return int64((int(t.Month())-1)/3 + 1) }) }
func (dt *ArrowDateTimeAccessor) Week() core.Series      { return dt.extractField(func(t time.Time) int64 { _, w := t.ISOWeek(); return int64(w) }) }
func (dt *ArrowDateTimeAccessor) Unix() core.Series      { return dt.extractField(func(t time.Time) int64 { return t.Unix() }) }

func (dt *ArrowDateTimeAccessor) Date() core.Series {
	return dt.extractString(func(t time.Time) string { return t.Format("2006-01-02") })
}

func (dt *ArrowDateTimeAccessor) Time() core.Series {
	return dt.extractString(func(t time.Time) string { return t.Format("15:04:05") })
}

func (dt *ArrowDateTimeAccessor) Floor(d time.Duration) core.Series {
	return dt.transformTime(func(t time.Time) time.Time { return t.Truncate(d) })
}

func (dt *ArrowDateTimeAccessor) Ceil(d time.Duration) core.Series {
	return dt.transformTime(func(t time.Time) time.Time {
		floored := t.Truncate(d)
		if floored.Before(t) {
			return floored.Add(d)
		}
		return floored
	})
}

func (dt *ArrowDateTimeAccessor) Round(d time.Duration) core.Series {
	return dt.transformTime(func(t time.Time) time.Time { return t.Round(d) })
}

// extractField extracts a time field as int64 series.
func (dt *ArrowDateTimeAccessor) extractField(fn func(time.Time) int64) core.Series {
	s := dt.series
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			t := s.getTime(i)
			bldr.Append(fn(t))
		}
	}

	return NewArrowSeries(dt.series.Name()+"_dt", bldr.NewArray(), s.Index())
}

// extractString extracts a time field as string series.
func (dt *ArrowDateTimeAccessor) extractString(fn func(time.Time) string) core.Series {
	s := dt.series
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			t := s.getTime(i)
			bldr.Append(fn(t))
		}
	}

	return NewArrowSeries(dt.series.Name()+"_dt", bldr.NewArray(), s.Index())
}

// transformTime applies a time transformation and returns a timestamp series.
func (dt *ArrowDateTimeAccessor) transformTime(fn func(time.Time) time.Time) core.Series {
	s := dt.series
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			t := s.getTime(i)
			bldr.Append(fn(t).UnixNano())
		}
	}

	return NewArrowSeries(dt.series.Name()+"_dt", bldr.NewArray(), s.Index())
}

// getTime extracts a time.Time from the series at position i.
func (s *ArrowSeries) getTime(i int) time.Time {
	// The value is stored as int64 (UnixNano), always in UTC
	return time.Unix(0, s.Int(i)).UTC()
}

// DT returns the DateTimeAccessor for timestamp series.
func (s *ArrowSeries) DT() core.DateTimeAccessor {
	return NewDateTimeAccessor(s)
}

// Resample resamples a timestamp-indexed series to the given frequency.
func (s *ArrowSeries) Resample(rule core.ResampleRule) *ArrowDataFrame {
	return resampleSeries(s, rule)
}

// Shift shifts the series by n periods. Positive n shifts down (lag), negative shifts up (lead).
func (s *ArrowSeries) Shift(n int) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		srcIdx := i - n
		if srcIdx < 0 || srcIdx >= s.Len() {
			bldr.AppendNull()
		} else if s.IsNull(srcIdx) {
			bldr.AppendNull()
		} else {
			copyValue(bldr, s, srcIdx)
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// PctChange computes the percentage change between the current and prior element.
func (s *ArrowSeries) PctChange(periods int) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		prevIdx := i - periods
		if prevIdx < 0 || prevIdx >= s.Len() {
			bldr.AppendNull()
		} else if s.IsNull(i) || s.IsNull(prevIdx) {
			bldr.AppendNull()
		} else {
			prev := s.Float(prevIdx)
			curr := s.Float(i)
			if prev == 0 {
				bldr.AppendNull()
			} else {
				bldr.Append((curr - prev) / prev)
			}
		}
	}

	return NewArrowSeries(s.Name()+"_pct", bldr.NewArray(), s.Index())
}

// Diff computes the difference between the current and prior element.
func (s *ArrowSeries) Diff(periods int) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		prevIdx := i - periods
		if prevIdx < 0 || prevIdx >= s.Len() {
			bldr.AppendNull()
		} else if s.IsNull(i) || s.IsNull(prevIdx) {
			bldr.AppendNull()
		} else {
			bldr.Append(s.Float(i) - s.Float(prevIdx))
		}
	}

	return NewArrowSeries(s.Name()+"_diff", bldr.NewArray(), s.Index())
}

// CumSum returns the cumulative sum.
func (s *ArrowSeries) CumSum() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	sum := 0.0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			sum += s.Float(i)
			bldr.Append(sum)
		}
	}
	return NewArrowSeries(s.Name()+"_cumsum", bldr.NewArray(), s.Index())
}

// CumProd returns the cumulative product.
func (s *ArrowSeries) CumProd() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	prod := 1.0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			prod *= s.Float(i)
			bldr.Append(prod)
		}
	}
	return NewArrowSeries(s.Name()+"_cumprod", bldr.NewArray(), s.Index())
}

// CumMax returns the cumulative maximum.
func (s *ArrowSeries) CumMax() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	max := 0.0
	first := true
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			if first || v > max {
				max = v
				first = false
			}
			bldr.Append(max)
		}
	}
	return NewArrowSeries(s.Name()+"_cummax", bldr.NewArray(), s.Index())
}

// CumMin returns the cumulative minimum.
func (s *ArrowSeries) CumMin() core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	min := 0.0
	first := true
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			if first || v < min {
				min = v
				first = false
			}
			bldr.Append(min)
		}
	}
	return NewArrowSeries(s.Name()+"_cummin", bldr.NewArray(), s.Index())
}
