package arrow

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// --- clip ---

// Clip limits values to the range [lo, hi].
func (s *ArrowSeries) Clip(lo, hi float64) core.Series {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			}
			bldr.Append(v)
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// --- convert_dtypes ---

// ConvertDtypes attempts to convert each column to the most appropriate dtype.
// Strings that look like ints become int64, strings that look like floats become float64, etc.
func (df *ArrowDataFrame) ConvertDtypes() core.DataFrame {
	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	for i, name := range cols {
		series[i] = convertColumnDtype(df.Col(name).(*ArrowSeries))
	}
	return NewDataFrame(series...)
}

func convertColumnDtype(s *ArrowSeries) *ArrowSeries {
	if s.Dtype() != core.STRING {
		return s
	}
	// Try int first
	allInt := true
	allFloat := true
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		v := strings.TrimSpace(s.String(i))
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			allInt = false
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			allFloat = false
		}
	}
	alloc := memory.NewGoAllocator()
	if allInt {
		bldr := array.NewInt64Builder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				v, _ := strconv.ParseInt(strings.TrimSpace(s.String(i)), 10, 64)
				bldr.Append(v)
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
	}
	if allFloat {
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.IsNull(i) {
				bldr.AppendNull()
			} else {
				v, _ := strconv.ParseFloat(strings.TrimSpace(s.String(i)), 64)
				bldr.Append(v)
			}
		}
		return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
	}
	return s
}

// --- query ---

// Query filters rows using a simple expression string.
// Supported: "col > val", "col < val", "col == val", "col != val", "col >= val", "col <= val"
func (df *ArrowDataFrame) Query(expr string) core.DataFrame {
	expr = strings.TrimSpace(expr)

	// Parse: col op val
	var colName, op, valStr string
	for _, o := range []string{">=", "<=", "!=", "==", ">", "<"} {
		idx := strings.Index(expr, o)
		if idx >= 0 {
			colName = strings.TrimSpace(expr[:idx])
			op = o
			valStr = strings.TrimSpace(expr[idx+len(o):])
			break
		}
	}
	if colName == "" || op == "" {
		return df
	}

	// Remove quotes from value
	valStr = strings.Trim(valStr, `"'`)

	s := df.Col(colName).(*ArrowSeries)
	mask := make([]bool, df.Len())

	for i := 0; i < df.Len(); i++ {
		if s.IsNull(i) {
			mask[i] = false
			continue
		}

		var cmp int
		if s.Dtype() == core.STRING {
			cmp = strings.Compare(s.String(i), valStr)
		} else {
			fv, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				mask[i] = false
				continue
			}
			sv := s.Float(i)
			if sv < fv {
				cmp = -1
			} else if sv > fv {
				cmp = 1
			} else {
				cmp = 0
			}
		}

		switch op {
		case "==":
			mask[i] = cmp == 0
		case "!=":
			mask[i] = cmp != 0
		case ">":
			mask[i] = cmp > 0
		case "<":
			mask[i] = cmp < 0
		case ">=":
			mask[i] = cmp >= 0
		case "<=":
			mask[i] = cmp <= 0
		}
	}

	return df.Filter(mask)
}

// --- pivot / pivot_table ---

// Pivot reshapes a DataFrame from long to wide format.
// index: column to use as new index, columns: column whose values become new columns,
// values: column to fill in the pivoted table.
func (df *ArrowDataFrame) Pivot(index, columns, values string) core.DataFrame {
	// Collect unique column values
	colSeries := df.Col(columns).(*ArrowSeries)
	valSeries := df.Col(values).(*ArrowSeries)
	idxSeries := df.Col(index).(*ArrowSeries)

	// Build unique index values and column values in order
	idxOrder := make([]string, 0)
	idxSeen := make(map[string]bool)
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)

	for i := 0; i < df.Len(); i++ {
		iv := idxSeries.String(i)
		if !idxSeen[iv] {
			idxSeen[iv] = true
			idxOrder = append(idxOrder, iv)
		}
		cv := colSeries.String(i)
		if cv != "" && !colSeen[cv] {
			colSeen[cv] = true
			colOrder = append(colOrder, cv)
		}
	}

	// Build lookup: (indexVal, colVal) -> value
	type cellKey struct {
		row, col string
	}
	cellMap := make(map[cellKey]float64)
	cellValid := make(map[cellKey]bool)
	for i := 0; i < df.Len(); i++ {
		rv := idxSeries.String(i)
		cv := colSeries.String(i)
		if valSeries.NotNull(i) {
			cellMap[cellKey{rv, cv}] = valSeries.Float(i)
			cellValid[cellKey{rv, cv}] = true
		}
	}

	alloc := memory.NewGoAllocator()
	// Build index column
	idxBldr := array.NewStringBuilder(alloc)
	idxBldr.Resize(len(idxOrder))
	for _, v := range idxOrder {
		idxBldr.Append(v)
	}
	resultSeries := []*ArrowSeries{NewArrowSeries(index, idxBldr.NewArray(), nil)}

	// Build value columns
	for _, cv := range colOrder {
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(len(idxOrder))
		for _, rv := range idxOrder {
			key := cellKey{rv, cv}
			if cellValid[key] {
				bldr.Append(cellMap[key])
			} else {
				bldr.AppendNull()
			}
		}
		resultSeries = append(resultSeries, NewArrowSeries(cv, bldr.NewArray(), nil))
	}

	return NewDataFrame(resultSeries...)
}

// PivotTable creates a spreadsheet-style pivot table with aggregation.
func (df *ArrowDataFrame) PivotTable(index, columns, values string, aggFn core.AggFunc) core.DataFrame {
	// Similar to Pivot but aggregates duplicates
	colSeries := df.Col(columns).(*ArrowSeries)
	valSeries := df.Col(values).(*ArrowSeries)
	idxSeries := df.Col(index).(*ArrowSeries)

	idxOrder := make([]string, 0)
	idxSeen := make(map[string]bool)
	colOrder := make([]string, 0)
	colSeen := make(map[string]bool)

	for i := 0; i < df.Len(); i++ {
		iv := idxSeries.String(i)
		if !idxSeen[iv] {
			idxSeen[iv] = true
			idxOrder = append(idxOrder, iv)
		}
		cv := colSeries.String(i)
		if !colSeen[cv] {
			colSeen[cv] = true
			colOrder = append(colOrder, cv)
		}
	}

	type cellKey struct {
		row, col string
	}
	cellVals := make(map[cellKey][]float64)
	for i := 0; i < df.Len(); i++ {
		if valSeries.IsNull(i) {
			continue
		}
		rv := idxSeries.String(i)
		cv := colSeries.String(i)
		cellVals[cellKey{rv, cv}] = append(cellVals[cellKey{rv, cv}], valSeries.Float(i))
	}

	alloc := memory.NewGoAllocator()
	idxBldr := array.NewStringBuilder(alloc)
	idxBldr.Resize(len(idxOrder))
	for _, v := range idxOrder {
		idxBldr.Append(v)
	}
	resultSeries := []*ArrowSeries{NewArrowSeries(index, idxBldr.NewArray(), nil)}

	for _, cv := range colOrder {
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(len(idxOrder))
		for _, rv := range idxOrder {
			vals := cellVals[cellKey{rv, cv}]
			if len(vals) == 0 {
				bldr.AppendNull()
			} else {
				bldr.Append(applyAgg(aggFn, vals))
			}
		}
		resultSeries = append(resultSeries, NewArrowSeries(cv, bldr.NewArray(), nil))
	}

	return NewDataFrame(resultSeries...)
}

// --- melt ---

// Melt unpivots a DataFrame from wide to long format.
// idVars: columns to use as identifier variables.
// valueVars: columns to unpivot (if empty, all non-id columns).
func (df *ArrowDataFrame) Melt(idVars []string, valueVars []string) core.DataFrame {
	if len(valueVars) == 0 {
		// Use all non-id columns
		idSet := make(map[string]bool)
		for _, v := range idVars {
			idSet[v] = true
		}
		for _, name := range df.Columns() {
			if !idSet[name] {
				valueVars = append(valueVars, name)
			}
		}
	}

	rows, _ := df.Shape()
	nMelt := len(valueVars)
	totalRows := rows * nMelt

	alloc := memory.NewGoAllocator()

	// Build id columns (repeated nMelt times)
	idSeries := make([]*ArrowSeries, len(idVars))
	for j, name := range idVars {
		src := df.Col(name).(*ArrowSeries)
		bldr := array.NewBuilder(alloc, src.arr.DataType())
		bldr.Resize(totalRows)
		for rep := 0; rep < nMelt; rep++ {
			for i := 0; i < rows; i++ {
				if src.IsNull(i) {
					bldr.AppendNull()
				} else {
					copyValue(bldr, src, i)
				}
			}
		}
		idSeries[j] = NewArrowSeries(name, bldr.NewArray(), nil)
	}

	// Build variable column
	varBldr := array.NewStringBuilder(alloc)
	varBldr.Resize(totalRows)
	for _, vname := range valueVars {
		for i := 0; i < rows; i++ {
			varBldr.Append(vname)
		}
	}
	varSeries := NewArrowSeries("variable", varBldr.NewArray(), nil)

	// Build value column
	valBldr := array.NewFloat64Builder(alloc)
	valBldr.Resize(totalRows)
	for _, vname := range valueVars {
		src := df.Col(vname).(*ArrowSeries)
		for i := 0; i < rows; i++ {
			if src.IsNull(i) {
				valBldr.AppendNull()
			} else {
				valBldr.Append(src.Float(i))
			}
		}
	}
	valSeries := NewArrowSeries("value", valBldr.NewArray(), nil)

	allSeries := append(idSeries, varSeries, valSeries)
	return NewDataFrame(allSeries...)
}

// --- stack / unstack ---

// Stack pivots the columns into the index (wide to long, like melt but index-based).
func (df *ArrowDataFrame) Stack() core.DataFrame {
	return df.Melt(nil, nil)
}

// Unstack pivots the innermost index level into columns.
func (df *ArrowDataFrame) Unstack(level int) core.DataFrame {
	// Simplified: unstack column "level" from index
	return df // TODO: full MultiIndex support
}

// --- compare ---

// Compare compares two DataFrames element-wise, returning a DataFrame showing differences.
func (df *ArrowDataFrame) Compare(other core.DataFrame, keepEqual bool) core.DataFrame {
	o := other.(*ArrowDataFrame)
	rows, _ := df.Shape()
	cols := df.Columns()
	alloc := memory.NewGoAllocator()

	var resultSeries []*ArrowSeries
	for _, name := range cols {
		s1 := df.Col(name).(*ArrowSeries)
		s2, err := tryCol(o, name)
		if err != nil {
			continue
		}

		bldr := array.NewStringBuilder(alloc)
		bldr.Resize(rows)
		for i := 0; i < rows; i++ {
			if s1.IsNull(i) && s2.IsNull(i) {
				if keepEqual {
					bldr.Append("<nil>")
				} else {
					bldr.AppendNull()
				}
			} else if s1.IsNull(i) || s2.IsNull(i) {
				bldr.Append(fmt.Sprintf("%v→%v", nullStr(s1, i), nullStr(s2, i)))
			} else {
				v1 := valueStr(s1, i)
				v2 := valueStr(s2, i)
				if v1 == v2 {
					if keepEqual {
						bldr.Append(v1)
					} else {
						bldr.AppendNull()
					}
				} else {
					bldr.Append(v1 + "→" + v2)
				}
			}
		}
		resultSeries = append(resultSeries, NewArrowSeries(name, bldr.NewArray(), df.Index()))
	}
	return NewDataFrame(resultSeries...)
}

func tryCol(df *ArrowDataFrame, name string) (*ArrowSeries, error) {
	defer func() { recover() }()
	return df.Col(name).(*ArrowSeries), nil
}

func nullStr(s *ArrowSeries, i int) string {
	if s.IsNull(i) {
		return "<nil>"
	}
	return valueStr(s, i)
}

func valueStr(s *ArrowSeries, i int) string {
	switch s.Dtype() {
	case core.BOOL:
		return fmt.Sprintf("%v", s.Bool(i))
	case core.FLOAT32, core.FLOAT64:
		return fmt.Sprintf("%g", s.Float(i))
	case core.STRING:
		return s.String(i)
	default:
		return fmt.Sprintf("%d", s.Int(i))
	}
}

// --- asfreq ---

// AsFreq converts a time series to the specified frequency.
// Fills missing values with the given fillValue.
func (s *ArrowSeries) AsFreq(freq int64, fillValue float64) core.Series {
	if s.Len() == 0 {
		return s
	}
	// freq is in nanoseconds
	start := s.Int(0)
	end := s.Int(s.Len() - 1)

	// Build lookup
	valMap := make(map[int64]float64)
	validMap := make(map[int64]bool)
	for i := 0; i < s.Len(); i++ {
		ts := s.Int(i)
		if s.NotNull(i) {
			valMap[ts] = s.Float(i)
			validMap[ts] = true
		}
	}

	alloc := memory.NewGoAllocator()
	timeBldr := array.NewInt64Builder(alloc)
	valBldr := array.NewFloat64Builder(alloc)

	for ts := start; ts <= end; ts += freq {
		timeBldr.Append(ts)
		if validMap[ts] {
			valBldr.Append(valMap[ts])
		} else {
			if math.IsNaN(fillValue) {
				valBldr.AppendNull()
			} else {
				valBldr.Append(fillValue)
			}
		}
	}

	return NewArrowSeries(s.Name(), valBldr.NewArray(), nil)
}

// --- mode ---

// Mode returns the most frequent value(s).
func (s *ArrowSeries) Mode() core.Series {
	counts := make(map[interface{}]int)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		counts[key]++
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	modes := make([]interface{}, 0)
	seen := make(map[interface{}]bool)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var key interface{}
		switch s.Dtype() {
		case core.BOOL:
			key = s.Bool(i)
		case core.STRING:
			key = s.String(i)
		case core.FLOAT32, core.FLOAT64:
			key = s.Float(i)
		default:
			key = s.Int(i)
		}
		if counts[key] == maxCount && !seen[key] {
			seen[key] = true
			modes = append(modes, key)
		}
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewBuilder(alloc, s.arr.DataType())
	bldr.Resize(len(modes))
	for _, v := range modes {
		switch s.Dtype() {
		case core.BOOL:
			bldr.(*array.BooleanBuilder).Append(v.(bool))
		case core.STRING:
			bldr.(*array.StringBuilder).Append(v.(string))
		case core.FLOAT64:
			bldr.(*array.Float64Builder).Append(v.(float64))
		default:
			bldr.(*array.Int64Builder).Append(v.(int64))
		}
	}
	return NewArrowSeries(s.Name()+"_mode", bldr.NewArray(), nil)
}

// --- skew / kurt ---

// Skew returns the skewness of the series.
func (s *ArrowSeries) Skew() float64 {
	vals := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			vals = append(vals, s.Float(i))
		}
	}
	n := float64(len(vals))
	if n < 3 {
		return math.NaN()
	}
	mean := sumF(vals) / n
	var ss, sss float64
	for _, v := range vals {
		d := v - mean
		ss += d * d
		sss += d * d * d
	}
	std := math.Sqrt(ss / n)
	if std == 0 {
		return math.NaN()
	}
	return (sss / n) / (std * std * std)
}

// Kurt returns the excess kurtosis of the series.
func (s *ArrowSeries) Kurt() float64 {
	vals := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			vals = append(vals, s.Float(i))
		}
	}
	n := float64(len(vals))
	if n < 4 {
		return math.NaN()
	}
	mean := sumF(vals) / n
	var ss, s4 float64
	for _, v := range vals {
		d := v - mean
		ss += d * d
		s4 += d * d * d * d
	}
	std := math.Sqrt(ss / n)
	if std == 0 {
		return math.NaN()
	}
	return (s4/n)/(std*std*std*std) - 3
}

// --- memory_usage ---

// MemoryUsage returns the approximate memory usage of the DataFrame in bytes.
func (df *ArrowDataFrame) MemoryUsage() map[string]int64 {
	result := make(map[string]int64)
	for _, name := range df.Columns() {
		s := df.Col(name).(*ArrowSeries)
		var bytes int64
		switch s.Dtype() {
		case core.BOOL:
			bytes = int64(s.Len()) // 1 bit per value, simplified to 1 byte
		case core.INT8:
			bytes = int64(s.Len())
		case core.INT16, core.UINT16:
			bytes = int64(s.Len()) * 2
		case core.INT32, core.UINT32, core.FLOAT32:
			bytes = int64(s.Len()) * 4
		case core.INT64, core.UINT64, core.FLOAT64:
			bytes = int64(s.Len()) * 8
		case core.STRING:
			// Estimate: 16 bytes per string header + average content
			bytes = int64(s.Len()) * 32
		default:
			bytes = int64(s.Len()) * 8
		}
		// Add null bitmap overhead
		if s.NullCount() > 0 {
			bytes += int64(s.Len()+7) / 8
		}
		result[name] = bytes
	}
	return result
}

// --- pipe ---

// Pipe applies a function that takes a DataFrame and returns a DataFrame.
func (df *ArrowDataFrame) Pipe(fn func(*ArrowDataFrame) *ArrowDataFrame) *ArrowDataFrame {
	return fn(df)
}

// --- get_dummies ---

// GetDummies converts categorical variable(s) into dummy/indicator variables.
func GetDummies(df *ArrowDataFrame, columns []string) core.DataFrame {
	result := df
	for _, colName := range columns {
		s := df.Col(colName).(*ArrowSeries)
		unique := s.Unique().(*ArrowSeries)
		alloc := memory.NewGoAllocator()

		for i := 0; i < unique.Len(); i++ {
			val := unique.String(i)
			bldr := array.NewInt64Builder(alloc)
			bldr.Resize(s.Len())
			for j := 0; j < s.Len(); j++ {
				if s.IsNull(j) {
					bldr.Append(0)
				} else if s.String(j) == val {
					bldr.Append(1)
				} else {
					bldr.Append(0)
				}
			}
			dummyName := colName + "_" + val
			result = result.WithColumn(dummyName, NewArrowSeries(dummyName, bldr.NewArray(), nil)).(*ArrowDataFrame)
		}
	}
	return result
}

// --- cut / qcut ---

// Cut bins values into discrete intervals.
// bins: number of equal-width bins, or explicit edges.
func Cut(s *ArrowSeries, bins int) core.Series {
	// Find min/max
	minVal, maxVal := math.Inf(1), math.Inf(-1)
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			v := s.Float(i)
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}

	width := (maxVal - minVal) / float64(bins)
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			idx := int((v - minVal) / width)
			if idx >= bins {
				idx = bins - 1
			}
			lo := minVal + float64(idx)*width
			hi := minVal + float64(idx+1)*width
			bldr.Append(fmt.Sprintf("(%g,%g]", lo, hi))
		}
	}
	return NewArrowSeries(s.Name()+"_binned", bldr.NewArray(), s.Index())
}

// QCut bins values into quantile-based intervals.
func QCut(s *ArrowSeries, q int) core.Series {
	vals := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			vals = append(vals, s.Float(i))
		}
	}
	sort.Float64s(vals)

	edges := make([]float64, q+1)
	for i := 0; i <= q; i++ {
		p := float64(i) / float64(q)
		edges[i] = percentileSorted(vals, p)
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			for j := 0; j < q; j++ {
				if v <= edges[j+1] {
					bldr.Append(fmt.Sprintf("[%g,%g]", edges[j], edges[j+1]))
					break
				}
			}
		}
	}
	return NewArrowSeries(s.Name()+"_qbin", bldr.NewArray(), s.Index())
}

// --- explode ---

// Explode transforms each element of a list-like column into a row.
// The column values are expected to be comma-separated strings.
func (s *ArrowSeries) Explode(sep string) core.Series {
	vals := make([]string, 0)
	idxVals := make([]string, 0)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			vals = append(vals, "")
			idxVals = append(idxVals, s.Index().Get(i))
		} else {
			parts := strings.Split(s.String(i), sep)
			for _, p := range parts {
				vals = append(vals, strings.TrimSpace(p))
				idxVals = append(idxVals, s.Index().Get(i))
			}
		}
	}

	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(len(vals))
	for _, v := range vals {
		if v == "" {
			bldr.AppendNull()
		} else {
			bldr.Append(v)
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), core.NewStringIndex(idxVals))
}

// --- factorize ---

// Factorize encodes object values as an enumerated type (integer codes).
// Returns (codes, uniques).
func Factorize(s *ArrowSeries) (core.Series, core.Series) {
	alloc := memory.NewGoAllocator()
	codeBldr := array.NewInt64Builder(alloc)
	codeBldr.Resize(s.Len())

	valMap := make(map[string]int64)
	order := make([]string, 0)
	nextCode := int64(0)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			codeBldr.AppendNull()
			continue
		}
		key := valueStr(s, i)
		code, exists := valMap[key]
		if !exists {
			code = nextCode
			valMap[key] = nextCode
			order = append(order, key)
			nextCode++
		}
		codeBldr.Append(code)
	}

	// Build uniques
	uniBldr := array.NewStringBuilder(alloc)
	uniBldr.Resize(len(order))
	for _, v := range order {
		uniBldr.Append(v)
	}

	return NewArrowSeries(s.Name()+"_codes", codeBldr.NewArray(), s.Index()),
		NewArrowSeries(s.Name()+"_uniques", uniBldr.NewArray(), nil)
}
