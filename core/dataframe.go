package core

// DataFrame is a two-dimensional labeled data structure.
type DataFrame interface {
	// Shape returns (rows, cols).
	Shape() (int, int)
	// Len returns the number of rows.
	Len() int
	// Columns returns the column names.
	Columns() []string
	// Index returns the row index.
	Index() Index
	// Dtypes returns the data type of each column.
	Dtypes() []DType

	// Column access
	// Col returns a Series for the given column name.
	Col(name string) Series
	// SelectCols returns a new DataFrame with only the given columns.
	SelectCols(names []string) DataFrame
	// DropCols returns a new DataFrame without the given columns.
	DropCols(names []string) DataFrame

	// Row access
	// Head returns the first n rows.
	Head(n int) DataFrame
	// Tail returns the last n rows.
	Tail(n int) DataFrame
	// Slice returns rows from start to end (exclusive).
	Slice(start, end int) DataFrame
	// Filter returns rows where mask is true.
	Filter(mask []bool) DataFrame
	// Take returns rows at the given positions.
	Take(indices []int) DataFrame

	// Metadata
	// Info returns a summary of the DataFrame (types, non-null counts, memory).
	Info() string
	// Describe returns descriptive statistics for numeric columns.
	Describe() DataFrame

	// Mutation
	// WithColumn returns a new DataFrame with the column added or replaced.
	WithColumn(name string, s Series) DataFrame
	// DropNA returns a new DataFrame with rows containing any null removed.
	DropNA() DataFrame
	// FillNA returns a new DataFrame with nulls filled by the given value.
	FillNA(value interface{}) DataFrame
	// Rename returns a new DataFrame with columns renamed per the mapping.
	Rename(mapping map[string]string) DataFrame
	// SetIndex returns a new DataFrame with the given column as the index.
	SetIndex(name string) DataFrame
	// ResetIndex returns a new DataFrame with the index as a column.
	ResetIndex() DataFrame

	// Sorting
	// SortBy returns a new DataFrame sorted by the given columns.
	SortBy(names []string, ascending []bool) DataFrame

	// Merge / Join
	// Join merges this DataFrame with another on their indices.
	Join(other DataFrame, how JoinType) DataFrame
	// MergeOn merges this DataFrame with another on the given column(s).
	MergeOn(other DataFrame, on []string, how JoinType) DataFrame

	// GroupBy
	// GroupByGroups returns group indices for the given columns.
	GroupByGroups(names []string) map[string][]int
	// Agg applies aggregation functions per group.
	Agg(groupCols []string, aggs map[string]AggFunc) DataFrame

	// I/O
	// ToCSV writes the DataFrame to a CSV string.
	ToCSV() string

	// Display
	// Fmt returns a formatted table string with default row count (top 5 + bottom 5).
	Fmt() string
	// Display returns a formatted table string showing top rows and bottom rows.
	Display(top, bottom int) string
}
