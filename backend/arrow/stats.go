package arrow

import (
	"math"
	"sort"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

// NLargest returns the n largest values.
func (s *ArrowSeries) NLargest(n int) core.Series {
	if n > s.Len() {
		n = s.Len()
	}
	indices := s.sortedIndices(false) // descending
	return s.Take(indices[:n])
}

// NSmallest returns the n smallest values.
func (s *ArrowSeries) NSmallest(n int) core.Series {
	if n > s.Len() {
		n = s.Len()
	}
	indices := s.sortedIndices(true) // ascending
	return s.Take(indices[:n])
}

// sortedIndices returns indices sorted by value.
func (s *ArrowSeries) sortedIndices(ascending bool) []int {
	type pair struct {
		idx int
		val float64
	}
	pairs := make([]pair, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			pairs = append(pairs, pair{i, s.Float(i)})
		}
	}
	sort.Slice(pairs, func(a, b int) bool {
		if ascending {
			return pairs[a].val < pairs[b].val
		}
		return pairs[a].val > pairs[b].val
	})
	indices := make([]int, len(pairs))
	for i, p := range pairs {
		indices[i] = p.idx
	}
	return indices
}

// Rank returns the rank of each value.
// method: "average", "min", "max", "first", "dense".
func (s *ArrowSeries) Rank(method string) core.Series {
	type pair struct {
		idx int
		val float64
	}
	pairs := make([]pair, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			pairs[i] = pair{i, s.Float(i)}
		} else {
			pairs[i] = pair{i, math.NaN()}
		}
	}

	// Sort by value
	sort.Slice(pairs, func(a, b int) bool {
		return pairs[a].val < pairs[b].val
	})

	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	ranks := make([]float64, s.Len())

	switch method {
	case "dense":
		rank := 0.0
		for i := 0; i < len(pairs); i++ {
			if math.IsNaN(pairs[i].val) {
				ranks[pairs[i].idx] = math.NaN()
				continue
			}
			if i > 0 && pairs[i].val != pairs[i-1].val {
				rank++
			}
			ranks[pairs[i].idx] = rank
		}

	case "min":
		// Same rank for ties (minimum)
		for i := 0; i < len(pairs); i++ {
			if math.IsNaN(pairs[i].val) {
				ranks[pairs[i].idx] = math.NaN()
				continue
			}
			j := i
			for j > 0 && pairs[j-1].val == pairs[i].val {
				j--
			}
			ranks[pairs[i].idx] = float64(j + 1)
		}

	case "max":
		for i := 0; i < len(pairs); i++ {
			if math.IsNaN(pairs[i].val) {
				ranks[pairs[i].idx] = math.NaN()
				continue
			}
			j := i
			for j < len(pairs)-1 && pairs[j+1].val == pairs[i].val {
				j++
			}
			ranks[pairs[i].idx] = float64(j + 1)
		}

	case "first":
		for i := 0; i < len(pairs); i++ {
			ranks[pairs[i].idx] = float64(i + 1)
		}

	default: // "average"
		for i := 0; i < len(pairs); i++ {
			if math.IsNaN(pairs[i].val) {
				ranks[pairs[i].idx] = math.NaN()
				continue
			}
			// Find range of ties
			start := i
			for i < len(pairs)-1 && pairs[i+1].val == pairs[i].val {
				i++
			}
			avg := float64(start+i+2) / 2.0
			for j := start; j <= i; j++ {
				ranks[pairs[j].idx] = avg
			}
		}
	}

	for _, r := range ranks {
		if math.IsNaN(r) {
			bldr.AppendNull()
		} else {
			bldr.Append(r)
		}
	}
	return NewArrowSeries(s.Name()+"_rank", bldr.NewArray(), s.Index())
}

// Quantile returns the value at the given quantile (0.0 to 1.0).
func (s *ArrowSeries) Quantile(p float64) float64 {
	vals := make([]float64, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			vals = append(vals, s.Float(i))
		}
	}
	if len(vals) == 0 {
		return math.NaN()
	}
	sort.Float64s(vals)
	return percentileSorted(vals, p)
}

func percentileSorted(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0]
	}
	pos := p * float64(n-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

// --- Corr / Cov ---

// Corr computes the Pearson correlation between two series.
func Corr(a, b *ArrowSeries) float64 {
	return corrImpl(a, b)
}

// Cov computes the covariance between two series.
func Cov(a, b *ArrowSeries) float64 {
	n := 0
	sumA, sumB, sumAB := 0.0, 0.0, 0.0
	minLen := a.Len()
	if b.Len() < minLen {
		minLen = b.Len()
	}
	for i := 0; i < minLen; i++ {
		if a.NotNull(i) && b.NotNull(i) {
			va, vb := a.Float(i), b.Float(i)
			sumA += va
			sumB += vb
			sumAB += va * vb
			n++
		}
	}
	if n < 2 {
		return math.NaN()
	}
	meanA := sumA / float64(n)
	meanB := sumB / float64(n)
	return (sumAB/float64(n)) - meanA*meanB
}

func corrImpl(a, b *ArrowSeries) float64 {
	cov := Cov(a, b)
	if math.IsNaN(cov) {
		return math.NaN()
	}
	stdA := stdDev(a)
	stdB := stdDev(b)
	if stdA == 0 || stdB == 0 {
		return math.NaN()
	}
	return cov / (stdA * stdB)
}

func stdDev(s *ArrowSeries) float64 {
	sum := 0.0
	sumSq := 0.0
	n := 0
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			v := s.Float(i)
			sum += v
			sumSq += v * v
			n++
		}
	}
	if n < 2 {
		return 0
	}
	mean := sum / float64(n)
	variance := sumSq/float64(n) - mean*mean
	if variance < 0 {
		return 0
	}
	return math.Sqrt(variance)
}

// CorrMatrix computes the correlation matrix for a DataFrame's numeric columns.
func CorrMatrix(df *ArrowDataFrame) *ArrowDataFrame {
	cols := df.Columns()
	numCols := make([]string, 0)
	for _, name := range cols {
		if df.Col(name).Dtype().IsNumeric() {
			numCols = append(numCols, name)
		}
	}
	n := len(numCols)
	if n == 0 {
		return NewDataFrame()
	}

	alloc := memory.NewGoAllocator()
	series := make([]*ArrowSeries, n)
	for j, colName := range numCols {
		bldr := array.NewFloat64Builder(alloc)
		bldr.Resize(n)
		for _, rowName := range numCols {
			if colName == rowName {
				bldr.Append(1.0)
			} else {
				bldr.Append(Corr(
					df.Col(colName).(*ArrowSeries),
					df.Col(rowName).(*ArrowSeries),
				))
			}
		}
		series[j] = NewArrowSeries(colName, bldr.NewArray(), core.NewStringIndex(numCols))
	}
	return NewDataFrame(series...)
}
