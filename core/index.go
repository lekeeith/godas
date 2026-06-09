package core

import (
	"fmt"
	"time"
)

// Index represents the row labels of a Series or DataFrame.
type Index interface {
	// Len returns the number of elements in the index.
	Len() int
	// Get returns the label at position i as a string.
	Get(i int) string
	// Slice returns a new Index from start to end (exclusive).
	Slice(start, end int) Index
	// Copy returns a deep copy of the index.
	Copy() Index
	// Type returns the DType of the index labels.
	Type() DType
}

// RangeIndex is a simple integer-based index like Python's range.
type RangeIndex struct {
	Start int
	Stop  int
	Step  int
}

// NewRangeIndex creates a RangeIndex from start to stop (exclusive) with step 1.
func NewRangeIndex(start, stop int) *RangeIndex {
	return &RangeIndex{Start: start, Stop: stop, Step: 1}
}

// NewDefaultIndex creates a RangeIndex [0, n).
func NewDefaultIndex(n int) *RangeIndex {
	return &RangeIndex{Start: 0, Stop: n, Step: 1}
}

func (idx *RangeIndex) Len() int {
	if idx.Step == 0 {
		return 0
	}
	n := (idx.Stop - idx.Start + idx.Step - 1) / idx.Step
	if n < 0 {
		return 0
	}
	return n
}

func (idx *RangeIndex) Get(i int) string {
	if i < 0 || i >= idx.Len() {
		panic(fmt.Sprintf("RangeIndex: index %d out of range [0:%d)", i, idx.Len()))
	}
	return fmt.Sprintf("%d", idx.Start+i*idx.Step)
}

func (idx *RangeIndex) Slice(start, end int) Index {
	if start < 0 || end > idx.Len() || start > end {
		panic(fmt.Sprintf("RangeIndex.Slice: invalid range [%d:%d] for length %d", start, end, idx.Len()))
	}
	return &RangeIndex{
		Start: idx.Start + start*idx.Step,
		Stop:  idx.Start + end*idx.Step,
		Step:  idx.Step,
	}
}

func (idx *RangeIndex) Copy() Index {
	return &RangeIndex{Start: idx.Start, Stop: idx.Stop, Step: idx.Step}
}

func (idx *RangeIndex) Type() DType {
	return INT64
}

// Int64Index is an index backed by a slice of int64 values.
type Int64Index struct {
	Values []int64
}

func NewInt64Index(values []int64) *Int64Index {
	cp := make([]int64, len(values))
	copy(cp, values)
	return &Int64Index{Values: cp}
}

func (idx *Int64Index) Len() int            { return len(idx.Values) }
func (idx *Int64Index) Get(i int) string    { return fmt.Sprintf("%d", idx.Values[i]) }
func (idx *Int64Index) Type() DType         { return INT64 }

func (idx *Int64Index) Slice(start, end int) Index {
	return &Int64Index{Values: idx.Values[start:end]}
}

func (idx *Int64Index) Copy() Index {
	cp := make([]int64, len(idx.Values))
	copy(cp, idx.Values)
	return &Int64Index{Values: cp}
}

// StringIndex is an index backed by a slice of strings.
type StringIndex struct {
	Values []string
}

func NewStringIndex(values []string) *StringIndex {
	cp := make([]string, len(values))
	copy(cp, values)
	return &StringIndex{Values: cp}
}

func (idx *StringIndex) Len() int            { return len(idx.Values) }
func (idx *StringIndex) Get(i int) string    { return idx.Values[i] }
func (idx *StringIndex) Type() DType         { return STRING }

func (idx *StringIndex) Slice(start, end int) Index {
	return &StringIndex{Values: idx.Values[start:end]}
}

func (idx *StringIndex) Copy() Index {
	cp := make([]string, len(idx.Values))
	copy(cp, idx.Values)
	return &StringIndex{Values: cp}
}

// DateTimeIndex is an index backed by a slice of time.Time values.
type DateTimeIndex struct {
	Values []time.Time
}

func NewDateTimeIndex(values []time.Time) *DateTimeIndex {
	cp := make([]time.Time, len(values))
	copy(cp, values)
	return &DateTimeIndex{Values: cp}
}

func (idx *DateTimeIndex) Len() int         { return len(idx.Values) }
func (idx *DateTimeIndex) Get(i int) string { return idx.Values[i].Format(time.RFC3339) }
func (idx *DateTimeIndex) Type() DType      { return TIMESTAMP }

func (idx *DateTimeIndex) Slice(start, end int) Index {
	return &DateTimeIndex{Values: idx.Values[start:end]}
}

func (idx *DateTimeIndex) Copy() Index {
	cp := make([]time.Time, len(idx.Values))
	copy(cp, idx.Values)
	return &DateTimeIndex{Values: cp}
}
