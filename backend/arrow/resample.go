package arrow

import (
	"fmt"
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

type resampleBucket struct {
	label time.Time
	vals  []float64
}

type dfBucket struct {
	label   time.Time
	indices []int
}

// resampleSeries resamples a timestamp-indexed series to the given frequency.
func resampleSeries(s *ArrowSeries, rule core.ResampleRule) *ArrowDataFrame {
	if s.Len() == 0 {
		return NewDataFrame()
	}

	// Collect all timestamps and values
	type entry struct {
		ts    time.Time
		value float64
		valid bool
	}

	entries := make([]entry, s.Len())
	for i := 0; i < s.Len(); i++ {
		entries[i] = entry{
			ts:    time.Unix(0, s.Int(i)),
			value: s.Float(i),
			valid: s.NotNull(i),
		}
	}

	// Group by time bucket
	buckets := make([]resampleBucket, 0)
	bucketMap := make(map[int64]int) // unix nano -> bucket index

	for _, e := range entries {
		if !e.valid {
			continue
		}
		bucketTime := truncateTime(e.ts, rule.Frequency)
		key := bucketTime.UnixNano()
		if idx, ok := bucketMap[key]; ok {
			buckets[idx].vals = append(buckets[idx].vals, e.value)
		} else {
			bucketMap[key] = len(buckets)
			buckets = append(buckets, resampleBucket{
				label: bucketTime,
				vals:  []float64{e.value},
			})
		}
	}

	alloc := memory.NewGoAllocator()
	timeBldr := array.NewInt64Builder(alloc)
	valBldr := array.NewFloat64Builder(alloc)
	timeBldr.Resize(len(buckets))
	valBldr.Resize(len(buckets))

	for _, b := range buckets {
		timeBldr.Append(b.label.UnixNano())
		valBldr.Append(applyResampleFunc(rule.Func, b.vals))
	}

	idx := core.NewDateTimeIndex(bucketTimes(buckets))
	timeSeries := NewArrowSeries(s.Name()+"_time", timeBldr.NewArray(), idx)
	valSeries := NewArrowSeries(s.Name()+"_"+rule.Func.String(), valBldr.NewArray(), idx)
	timeBldr.Release()
	valBldr.Release()

	return NewDataFrame(timeSeries, valSeries)
}

func bucketTimes(buckets []resampleBucket) []time.Time {
	times := make([]time.Time, len(buckets))
	for i, b := range buckets {
		times[i] = b.label
	}
	return times
}

func truncateTime(t time.Time, d time.Duration) time.Time {
	return t.Truncate(d)
}

func applyResampleFunc(fn core.ResampleFunc, vals []float64) float64 {
	n := len(vals)
	if n == 0 {
		return 0
	}
	switch fn {
	case core.ResampleSum:
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s
	case core.ResampleMean:
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(n)
	case core.ResampleMin:
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	case core.ResampleMax:
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	case core.ResampleFirst:
		return vals[0]
	case core.ResampleLast:
		return vals[n-1]
	case core.ResampleCount:
		return float64(n)
	default:
		return 0
	}
}

func dfBucketTimes(buckets []dfBucket) []time.Time {
	times := make([]time.Time, len(buckets))
	for i, b := range buckets {
		times[i] = b.label
	}
	return times
}

// ResampleDataFrame resamples a DataFrame by a timestamp column.
func ResampleDataFrame(df *ArrowDataFrame, timeCol string, rule core.ResampleRule) *ArrowDataFrame {
	s := df.Col(timeCol).(*ArrowSeries)
	if s.Len() == 0 {
		return NewDataFrame()
	}

	// Build time buckets
	buckets := make([]dfBucket, 0)
	bucketMap := make(map[int64]int)

	rows, _ := df.Shape()
	for i := 0; i < rows; i++ {
		if s.IsNull(i) {
			continue
		}
		t := time.Unix(0, s.Int(i))
		bucketTime := truncateTime(t, rule.Frequency)
		key := bucketTime.UnixNano()
		if idx, ok := bucketMap[key]; ok {
			buckets[idx].indices = append(buckets[idx].indices, i)
		} else {
			bucketMap[key] = len(buckets)
			buckets = append(buckets, dfBucket{
				label:   bucketTime,
				indices: []int{i},
			})
		}
	}

	alloc := memory.NewGoAllocator()
	colNames := df.Columns()
	resultSeries := make([]*ArrowSeries, len(colNames))

	for j, name := range colNames {
		col := df.Col(name).(*ArrowSeries)
		if name == timeCol {
			// Time column: use bucket labels
			bldr := array.NewInt64Builder(alloc)
			bldr.Resize(len(buckets))
			for _, b := range buckets {
				bldr.Append(b.label.UnixNano())
			}
			idx := core.NewDateTimeIndex(dfBucketTimes(buckets))
			resultSeries[j] = NewArrowSeries(name, bldr.NewArray(), idx)
			bldr.Release()
		} else if col.Dtype().IsNumeric() {
			// Numeric column: aggregate
			bldr := array.NewFloat64Builder(alloc)
			bldr.Resize(len(buckets))
			for _, b := range buckets {
				vals := make([]float64, 0, len(b.indices))
				for _, idx := range b.indices {
					if col.NotNull(idx) {
						vals = append(vals, col.Float(idx))
					}
				}
				bldr.Append(applyResampleFunc(rule.Func, vals))
			}
			idx := core.NewDateTimeIndex(dfBucketTimes(buckets))
			resultSeries[j] = NewArrowSeries(name, bldr.NewArray(), idx)
			bldr.Release()
		} else {
			// Non-numeric: take first value
			bldr := array.NewStringBuilder(alloc)
			bldr.Resize(len(buckets))
			for _, b := range buckets {
				if len(b.indices) > 0 && col.NotNull(b.indices[0]) {
					bldr.Append(col.String(b.indices[0]))
				} else {
					bldr.AppendNull()
				}
			}
			idx := core.NewDateTimeIndex(dfBucketTimes(buckets))
			resultSeries[j] = NewArrowSeries(name, bldr.NewArray(), idx)
			bldr.Release()
		}
	}

	return NewDataFrame(resultSeries...)
}

// TzLocalize assigns a timezone to a naive timestamp series.
func TzLocalize(s *ArrowSeries, loc *time.Location) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			t := time.Unix(0, s.Int(i))
			localized := t.In(loc)
			bldr.Append(localized.UnixNano())
		}
	}

	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// TzConvert converts a timestamp series from one timezone to another.
func TzConvert(s *ArrowSeries, loc *time.Location) core.Series {
	return TzLocalize(s, loc)
}

// BetweenTime returns rows where the time is between start and end (inclusive).
func BetweenTime(s *ArrowSeries, start, end time.Time) core.Series {
	indices := make([]int, 0)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		t := time.Unix(0, s.Int(i))
		if !t.Before(start) && !t.After(end) {
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

// AtTime returns rows where the time matches the given hour:minute.
func AtTime(s *ArrowSeries, hour, minute int) core.Series {
	indices := make([]int, 0)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		t := time.Unix(0, s.Int(i))
		if t.Hour() == hour && t.Minute() == minute {
			indices = append(indices, i)
		}
	}
	return s.Take(indices)
}

// NewTimestampSeries creates a timestamp series from time.Time values.
func NewTimestampSeries(name string, times []time.Time, index core.Index) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(len(times))
	for _, t := range times {
		bldr.Append(t.UnixNano())
	}
	return NewArrowSeries(name, bldr.NewArray(), index)
}

// DateRange creates a sequence of timestamps.
func DateRange(name string, start time.Time, periods int, freq time.Duration) *ArrowSeries {
	times := make([]time.Time, periods)
	for i := 0; i < periods; i++ {
		times[i] = start.Add(time.Duration(i) * freq)
	}
	return NewTimestampSeries(name, times, core.NewDateTimeIndex(times))
}

// DateRangeEnd creates a sequence of timestamps from start to end.
func DateRangeEnd(name string, start, end time.Time, freq time.Duration) *ArrowSeries {
	var times []time.Time
	for t := start; !t.After(end); t = t.Add(freq) {
		times = append(times, t)
	}
	return NewTimestampSeries(name, times, core.NewDateTimeIndex(times))
}

// Describe returns a string representation for debugging.
func (s *ArrowSeries) Describe() string {
	return fmt.Sprintf("ArrowSeries[%s] (%s) len=%d", s.Name(), s.Dtype(), s.Len())
}
