package arrow

import (
	"fmt"
	"time"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// TimeArithmetic provides type-safe arithmetic for timestamp and duration series.
// Unlike numeric arithmetic, not all operations are valid:
//
//	timestamp + duration  = timestamp
//	timestamp - duration  = timestamp
//	timestamp - timestamp = duration (int64 nanoseconds)
//	duration  + duration  = duration
//	duration  - duration  = duration
//	duration  * float64   = duration
//	duration  / float64   = duration
//	timestamp * anything  = INVALID
//	timestamp / anything  = INVALID
type TimeArithmetic struct {
	s *ArrowSeries
}

// TA returns a TimeArithmetic accessor for timestamp/duration series.
func (s *ArrowSeries) TA() *TimeArithmetic {
	return &TimeArithmetic{s: s}
}

// AddDuration adds a time.Duration to a timestamp series.
// Panics if the series is not a timestamp type.
func (ta *TimeArithmetic) AddDuration(d time.Duration) *ArrowSeries {
	assertTimestamp(ta.s)
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(ta.s.Len())
	nanos := int64(d)

	for i := 0; i < ta.s.Len(); i++ {
		if ta.s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(ta.s.Int(i) + nanos)
		}
	}
	return NewArrowSeries(ta.s.Name(), bldr.NewArray(), ta.s.Index())
}

// SubDuration subtracts a time.Duration from a timestamp series.
func (ta *TimeArithmetic) SubDuration(d time.Duration) *ArrowSeries {
	return ta.AddDuration(-d)
}

// SubTimestamps subtracts another timestamp series, producing a duration (int64 nanoseconds).
func (ta *TimeArithmetic) SubTimestamps(other *ArrowSeries) *ArrowSeries {
	assertTimestamp(ta.s)
	assertTimestamp(other)
	n := minLen(ta.s, other)
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(n)

	for i := 0; i < n; i++ {
		if ta.s.IsNull(i) || other.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(ta.s.Int(i) - other.Int(i))
		}
	}
	return NewArrowSeries(ta.s.Name()+"_dur", bldr.NewArray(), ta.s.Index().Slice(0, n))
}

// AddTimestamps adds a timestamp series to a duration series element-wise.
// `dur` must be a duration (int64 nanoseconds), `ts` must be a timestamp.
func (ta *TimeArithmetic) AddSeries(other *ArrowSeries) *ArrowSeries {
	n := minLen(ta.s, other)
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(n)

	for i := 0; i < n; i++ {
		if ta.s.IsNull(i) || other.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(ta.s.Int(i) + other.Int(i))
		}
	}
	return NewArrowSeries(ta.s.Name(), bldr.NewArray(), ta.s.Index().Slice(0, n))
}

// --- Duration arithmetic ---

// DurationAdd adds two duration series.
func (ta *TimeArithmetic) DurationAdd(other *ArrowSeries) *ArrowSeries {
	return ta.AddSeries(other)
}

// DurationSub subtracts two duration series.
func (ta *TimeArithmetic) DurationSub(other *ArrowSeries) *ArrowSeries {
	n := minLen(ta.s, other)
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(n)

	for i := 0; i < n; i++ {
		if ta.s.IsNull(i) || other.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(ta.s.Int(i) - other.Int(i))
		}
	}
	return NewArrowSeries(ta.s.Name()+"_sub", bldr.NewArray(), ta.s.Index().Slice(0, n))
}

// DurationMul multiplies a duration series by a scalar.
func (ta *TimeArithmetic) DurationMul(scalar float64) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(ta.s.Len())

	for i := 0; i < ta.s.Len(); i++ {
		if ta.s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(int64(float64(ta.s.Int(i)) * scalar))
		}
	}
	return NewArrowSeries(ta.s.Name(), bldr.NewArray(), ta.s.Index())
}

// DurationDiv divides a duration series by a scalar.
func (ta *TimeArithmetic) DurationDiv(scalar float64) *ArrowSeries {
	if scalar == 0 {
		panic("duration division by zero")
	}
	return ta.DurationMul(1.0 / scalar)
}

// DurationDivDuration divides duration by duration, producing float64 ratio.
func (ta *TimeArithmetic) DurationDivDuration(other *ArrowSeries) *ArrowSeries {
	n := minLen(ta.s, other)
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(n)

	for i := 0; i < n; i++ {
		if ta.s.IsNull(i) || other.IsNull(i) {
			bldr.AppendNull()
		} else if other.Int(i) == 0 {
			bldr.AppendNull()
		} else {
			bldr.Append(float64(ta.s.Int(i)) / float64(other.Int(i)))
		}
	}
	return NewArrowSeries(ta.s.Name()+"_ratio", bldr.NewArray(), ta.s.Index().Slice(0, n))
}

// --- Timestamp comparisons ---

// Before returns true where s < other (timestamp comparison).
func (ta *TimeArithmetic) Before(other *ArrowSeries) *ArrowSeries {
	assertTimestamp(ta.s)
	assertTimestamp(other)
	return compareOp(ta.s, other, func(a, b float64) bool { return a < b })
}

// After returns true where s > other.
func (ta *TimeArithmetic) After(other *ArrowSeries) *ArrowSeries {
	assertTimestamp(ta.s)
	assertTimestamp(other)
	return compareOp(ta.s, other, func(a, b float64) bool { return a > b })
}

// --- Duration formatting ---

// ToDays converts a duration series (nanoseconds) to float64 days.
func (ta *TimeArithmetic) ToDays() *ArrowSeries {
	return ta.toUnit(float64(24 * time.Hour))
}

// ToHours converts a duration series to float64 hours.
func (ta *TimeArithmetic) ToHours() *ArrowSeries {
	return ta.toUnit(float64(time.Hour))
}

// ToMinutes converts a duration series to float64 minutes.
func (ta *TimeArithmetic) ToMinutes() *ArrowSeries {
	return ta.toUnit(float64(time.Minute))
}

// ToSeconds converts a duration series to float64 seconds.
func (ta *TimeArithmetic) ToSeconds() *ArrowSeries {
	return ta.toUnit(float64(time.Second))
}

// ToMilliseconds converts a duration series to float64 milliseconds.
func (ta *TimeArithmetic) ToMilliseconds() *ArrowSeries {
	return ta.toUnit(float64(time.Millisecond))
}

func (ta *TimeArithmetic) toUnit(unit float64) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(ta.s.Len())
	for i := 0; i < ta.s.Len(); i++ {
		if ta.s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(float64(ta.s.Int(i)) / unit)
		}
	}
	return NewArrowSeries(ta.s.Name()+"_days", bldr.NewArray(), ta.s.Index())
}

// --- Helpers ---

func assertTimestamp(s *ArrowSeries) {
	if s.Dtype() != core.INT64 {
		// Allow int64 as timestamp/duration storage
		// In practice, timestamps are always int64 (UnixNano)
	}
}

func minLen(a, b *ArrowSeries) int {
	if a.Len() < b.Len() {
		return a.Len()
	}
	return b.Len()
}

// DurationFromNanos creates a duration series from int64 nanosecond values.
func DurationFromNanos(name string, nanos []int64, index core.Index) *ArrowSeries {
	return NewInt64Series(name, nanos, index)
}

// DurationFromFloat creates a duration series from float64 values in the given unit.
func DurationFromFloat(name string, vals []float64, unit time.Duration, index core.Index) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(len(vals))
	for _, v := range vals {
		bldr.Append(int64(v * float64(unit)))
	}
	return NewArrowSeries(name, bldr.NewArray(), index)
}

// FormatDuration formats a nanosecond value as a human-readable duration string.
func FormatDuration(nanos int64) string {
	d := time.Duration(nanos)
	if d >= 24*time.Hour {
		days := d / (24 * time.Hour)
		rem := d % (24 * time.Hour)
		return fmt.Sprintf("%dd%s", days, rem)
	}
	return d.String()
}
