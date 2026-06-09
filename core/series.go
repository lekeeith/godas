package core

// Series is a one-dimensional labeled array.
type Series interface {
	// Name returns the series name.
	Name() string
	// Len returns the number of elements.
	Len() int
	// Dtype returns the data type.
	Dtype() DType
	// Index returns the row index.
	Index() Index
	// NullCount returns the number of null values.
	NullCount() int
	// IsNull returns true if the value at position i is null.
	IsNull(i int) bool
	// NotNull returns true if the value at position i is not null.
	NotNull(i int) bool

	// Element access (return 0/false/"" if null)
	// Bool returns the bool value at position i.
	Bool(i int) bool
	// Int returns the int64 value at position i.
	Int(i int) int64
	// Float returns the float64 value at position i.
	Float(i int) float64
	// String returns the string value at position i.
	String(i int) string

	// Selection
	// Head returns the first n elements.
	Head(n int) Series
	// Tail returns the last n elements.
	Tail(n int) Series
	// Slice returns elements from start to end (exclusive).
	Slice(start, end int) Series
	// Filter returns elements where mask is true.
	Filter(mask []bool) Series
	// Take returns elements at the given positions.
	Take(indices []int) Series

	// Conversion
	// ToSlice returns a new Go slice of the series data (as interface{}).
	ToSlice() []interface{}
	// Copy returns a deep copy of the series.
	Copy() Series
	// SetName returns a new series with the given name.
	SetName(name string) Series
}
