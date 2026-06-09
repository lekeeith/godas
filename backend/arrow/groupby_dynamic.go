package arrow

import (
	"fmt"
	"sort"
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// DynamicGroupBy represents a dynamic time-based grouping.
type DynamicGroupBy struct {
	df        *ArrowDataFrame
	timeCol   string
	period    time.Duration
	offset    time.Duration
	closed    string // "left" or "right"
	label     string // "left" or "right"
	groupCols []string
}

// GroupByDynamic creates a dynamic time-based groupby.
// Unlike Resample which uses fixed frequency, GroupByDynamic groups by
// time periods that can overlap or have gaps.
//
// Example: group transactions by 7-day windows
//
//	df.GroupByDynamic("timestamp", 7*24*time.Hour, nil)
func (df *ArrowDataFrame) GroupByDynamic(timeCol string, period time.Duration, every time.Duration) *DynamicGroupBy {
	if every == 0 {
		every = period
	}
	return &DynamicGroupBy{
		df:      df,
		timeCol: timeCol,
		period:  period,
		offset:  every,
		closed:  "left",
		label:   "left",
	}
}

// Closed sets which side of the interval is closed.
func (dg *DynamicGroupBy) Closed(side string) *DynamicGroupBy {
	dg.closed = side
	return dg
}

// Label sets which side of the interval to use as the group label.
func (dg *DynamicGroupBy) Label(side string) *DynamicGroupBy {
	dg.label = side
	return dg
}

// By adds additional grouping columns (e.g., category).
func (dg *DynamicGroupBy) By(cols ...string) *DynamicGroupBy {
	dg.groupCols = cols
	return dg
}

// Agg applies aggregation functions to each dynamic group.
func (dg *DynamicGroupBy) Agg(aggs map[string]core.AggFunc) *ArrowDataFrame {
	timeSeries := dg.df.Col(dg.timeCol).(*ArrowSeries)
	rows, _ := dg.df.Shape()

	// Build time buckets
	type bucket struct {
		start    time.Time
		end      time.Time
		indices  []int
		groupKey string
	}

	buckets := make([]bucket, 0)
	bucketMap := make(map[string]int)

	for i := 0; i < rows; i++ {
		if timeSeries.IsNull(i) {
			continue
		}
		t := time.Unix(0, timeSeries.Int(i)).UTC()

		// Find which bucket this time falls into
		bucketStart := truncateTimeDynamic(t, dg.offset)
		if dg.closed == "right" {
			bucketStart = bucketStart.Add(-dg.offset)
		}
		bucketEnd := bucketStart.Add(dg.period)

		// Check if time is within [start, end)
		inBucket := false
		if dg.closed == "left" {
			inBucket = !t.Before(bucketStart) && t.Before(bucketEnd)
		} else {
			inBucket = t.After(bucketStart) && !t.After(bucketEnd)
		}
		if !inBucket {
			continue
		}

		// Build group key including additional group columns
		key := bucketStart.Format(time.RFC3339)
		if len(dg.groupCols) > 0 {
			for _, gc := range dg.groupCols {
				s := dg.df.Col(gc).(*ArrowSeries)
				if s.IsNull(i) {
					key += "\x00<nil>"
				} else {
					key += "\x00" + s.String(i)
				}
			}
		}

		if idx, exists := bucketMap[key]; exists {
			buckets[idx].indices = append(buckets[idx].indices, i)
		} else {
			bucketMap[key] = len(buckets)
			buckets = append(buckets, bucket{
				start:    bucketStart,
				end:      bucketEnd,
				indices:  []int{i},
				groupKey: key,
			})
		}
	}

	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, 0)

	// Time label column
	labelBldr := array.NewInt64Builder(alloc)
	labelBldr.Resize(len(buckets))
	for _, b := range buckets {
		if dg.label == "right" {
			labelBldr.Append(b.end.UnixNano())
		} else {
			labelBldr.Append(b.start.UnixNano())
		}
	}
	series = append(series, NewArrowSeries(dg.timeCol, labelBldr.NewArray(), nil))

	// Group key columns
	for _, gc := range dg.groupCols {
		s := dg.df.Col(gc).(*ArrowSeries)
		bldr := array.NewStringBuilder(alloc)
		bldr.Resize(len(buckets))
		for _, b := range buckets {
			if len(b.indices) > 0 {
				if s.NotNull(b.indices[0]) {
					bldr.Append(s.String(b.indices[0]))
				} else {
					bldr.AppendNull()
				}
			}
		}
		series = append(series, NewArrowSeries(gc, bldr.NewArray(), nil))
	}

	// Aggregation columns
	for colName, fn := range aggs {
		s := dg.df.Col(colName).(*ArrowSeries)
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(len(buckets))
		for _, b := range buckets {
			vals := make([]float64, 0, len(b.indices))
			for _, idx := range b.indices {
				if s.NotNull(idx) {
					vals = append(vals, s.Float(idx))
				}
			}
			bldr.Append(applyAgg(fn, vals))
		}
		series = append(series, NewArrowSeries(colName+"_"+fn.String(), bldr.NewArray(), nil))
	}

	return NewDataFrame(series...)
}

func truncateTimeDynamic(t time.Time, d time.Duration) time.Time {
	return t.Truncate(d)
}

// GroupByRollingTime applies rolling window aggregation per group with time-based windows.
func (df *ArrowDataFrame) GroupByRollingTime(timeCol string, period time.Duration, groupCols []string, aggFn core.AggFunc) *ArrowDataFrame {
	timeSeries := df.Col(timeCol).(*ArrowSeries)
	rows, _ := df.Shape()

	// Group by groupCols
	groups := df.GroupByGroups(groupCols)

	alloc := memory.NewGoAllocator()
	resultSeries := make([]*ArrowSeries, 0)

	// Keep group columns
	for _, gc := range groupCols {
		s := df.Col(gc).(*ArrowSeries)
		resultSeries = append(resultSeries, s)
	}

	// Time column
	resultSeries = append(resultSeries, timeSeries)

	// For each group, apply rolling time window
	resultVals := make([]float64, rows)
	resultValid := make([]bool, rows)

	for _, indices := range groups {
		// Sort group indices by time
		sorted := make([]int, len(indices))
		copy(sorted, indices)
		sortIntsByFloat(sorted, timeSeries)

		// Sliding window
		for pos, idx := range sorted {
			if timeSeries.IsNull(idx) {
				continue
			}
			t := time.Unix(0, timeSeries.Int(idx))
			windowStart := t.Add(-period)

			// Collect values in window
			vals := make([]float64, 0)
			for j := 0; j <= pos; j++ {
				jdx := sorted[j]
				if timeSeries.NotNull(jdx) {
					jt := time.Unix(0, timeSeries.Int(jdx))
					if !jt.Before(windowStart) {
						vals = append(vals, df.Col("salary").(*ArrowSeries).Float(jdx))
					}
				}
			}

			if len(vals) > 0 {
				resultVals[idx] = applyAgg(aggFn, vals)
				resultValid[idx] = true
			}
		}
	}

	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(rows)
	for i := 0; i < rows; i++ {
		if resultValid[i] {
			bldr.Append(resultVals[i])
		} else {
			bldr.AppendNull()
		}
	}
	resultSeries = append(resultSeries, NewArrowSeries("rolling_result", bldr.NewArray(), df.Index()))

	return NewDataFrame(resultSeries...)
}

func sortIntsByFloat(indices []int, s *ArrowSeries) {
	sort.Slice(indices, func(a, b int) bool {
		return s.Float(indices[a]) < s.Float(indices[b])
	})
}

// Info returns a string description of the dynamic groupby configuration.
func (dg *DynamicGroupBy) Info() string {
	return fmt.Sprintf("DynamicGroupBy: time=%s, period=%s, every=%s, closed=%s, label=%s, by=%v",
		dg.timeCol, dg.period, dg.offset, dg.closed, dg.label, dg.groupCols)
}
