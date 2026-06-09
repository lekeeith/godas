package core

import "time"

// DateTimeAccessor provides datetime field access on a Series.
// Access via series.DT() when the series dtype is TIMESTAMP.
type DateTimeAccessor interface {
	// Year returns a Series of year values.
	Year() Series
	// Month returns a Series of month values (1-12).
	Month() Series
	// Day returns a Series of day-of-month values (1-31).
	Day() Series
	// Hour returns a Series of hour values (0-23).
	Hour() Series
	// Minute returns a Series of minute values (0-59).
	Minute() Series
	// Second returns a Series of second values (0-59).
	Second() Series
	// DayOfWeek returns a Series of day-of-week values (0=Sunday, 6=Saturday).
	DayOfWeek() Series
	// DayOfYear returns a Series of day-of-year values (1-366).
	DayOfYear() Series
	// Quarter returns a Series of quarter values (1-4).
	Quarter() Series
	// Week returns a Series of ISO week numbers.
	Week() Series
	// Date returns a Series of date strings (YYYY-MM-DD).
	Date() Series
	// Time returns a Series of time strings (HH:MM:SS).
	Time() Series
	// Unix returns a Series of Unix timestamps (seconds since epoch).
	Unix() Series
	// Floor rounds down to the given duration.
	Floor(d time.Duration) Series
	// Ceil rounds up to the given duration.
	Ceil(d time.Duration) Series
	// Round rounds to the nearest given duration.
	Round(d time.Duration) Series
}

// ResampleFunc specifies how to aggregate during resampling.
type ResampleFunc int

const (
	ResampleSum ResampleFunc = iota
	ResampleMean
	ResampleMin
	ResampleMax
	ResampleFirst
	ResampleLast
	ResampleCount
)

func (f ResampleFunc) String() string {
	switch f {
	case ResampleSum:
		return "sum"
	case ResampleMean:
		return "mean"
	case ResampleMin:
		return "min"
	case ResampleMax:
		return "max"
	case ResampleFirst:
		return "first"
	case ResampleLast:
		return "last"
	case ResampleCount:
		return "count"
	default:
		return "unknown"
	}
}

// ResampleRule defines a resampling rule.
type ResampleRule struct {
	Frequency time.Duration
	Func      ResampleFunc
	Closed    string // "left" or "right"
	Label     string // "left" or "right"
}

// NewResampleRule creates a ResampleRule with sensible defaults.
func NewResampleRule(freq time.Duration, fn ResampleFunc) ResampleRule {
	return ResampleRule{
		Frequency: freq,
		Func:      fn,
		Closed:    "left",
		Label:     "left",
	}
}
