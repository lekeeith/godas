package arrow

import (
	"math"
	"sort"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// Rolling provides rolling window operations on a Series.
type Rolling struct {
	s      *ArrowSeries
	window int
	minP   int // minimum periods (non-null count required)
}

// Rolling returns a Rolling accessor with the given window size.
func (s *ArrowSeries) Rolling(window int) *Rolling {
	return &Rolling{s: s, window: window, minP: window}
}

// MinPeriods sets the minimum number of observations required to produce a value.
func (r *Rolling) MinPeriods(minP int) *Rolling {
	return &Rolling{s: r.s, window: r.window, minP: minP}
}

// Mean computes the rolling mean.
func (r *Rolling) Mean() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		if len(vals) == 0 {
			return math.NaN()
		}
		return sumF(vals) / float64(len(vals))
	})
}

// Sum computes the rolling sum.
func (r *Rolling) Sum() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		return sumF(vals)
	})
}

// Std computes the rolling standard deviation (population).
func (r *Rolling) Std() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		n := len(vals)
		if n < 2 {
			return math.NaN()
		}
		mean := sumF(vals) / float64(n)
		ss := 0.0
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		return math.Sqrt(ss / float64(n))
	})
}

// Min computes the rolling minimum.
func (r *Rolling) Min() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		if len(vals) == 0 {
			return math.NaN()
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	})
}

// Max computes the rolling maximum.
func (r *Rolling) Max() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		if len(vals) == 0 {
			return math.NaN()
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	})
}

// Count computes the rolling count of non-null values.
func (r *Rolling) Count() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		return float64(len(vals))
	})
}

// Median computes the rolling median.
func (r *Rolling) Median() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		if len(vals) == 0 {
			return math.NaN()
		}
		sorted := make([]float64, len(vals))
		copy(sorted, vals)
		sort.Float64s(sorted)
		return percentileSorted(sorted, 0.5)
	})
}

// Var computes the rolling variance (population).
func (r *Rolling) Var() core.Series {
	return r.applyWindow(func(vals []float64) float64 {
		n := len(vals)
		if n < 2 {
			return math.NaN()
		}
		mean := sumF(vals) / float64(n)
		ss := 0.0
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		return ss / float64(n)
	})
}

// Apply applies a custom function to each window.
func (r *Rolling) Apply(fn func([]float64) float64) core.Series {
	return r.applyWindow(fn)
}

func (r *Rolling) applyWindow(fn func([]float64) float64) core.Series {
	s := r.s
	w := r.window
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		start := i - w + 1
		if start < 0 {
			start = 0
		}
		vals := make([]float64, 0, w)
		for j := start; j <= i; j++ {
			if s.NotNull(j) {
				vals = append(vals, s.Float(j))
			}
		}
		if len(vals) < r.minP {
			bldr.AppendNull()
		} else {
			result := fn(vals)
			if math.IsNaN(result) {
				bldr.AppendNull()
			} else {
				bldr.Append(result)
			}
		}
	}

	return NewArrowSeries(s.Name()+"_rolling", bldr.NewArray(), s.Index())
}

// --- Expanding ---

// Expanding provides expanding window operations.
type Expanding struct {
	s    *ArrowSeries
	minP int
}

// Expanding returns an Expanding accessor.
func (s *ArrowSeries) Expanding() *Expanding {
	return &Expanding{s: s, minP: 1}
}

// MinPeriods sets the minimum periods.
func (e *Expanding) MinPeriods(minP int) *Expanding {
	return &Expanding{s: e.s, minP: minP}
}

// Mean computes the expanding mean.
func (e *Expanding) Mean() core.Series {
	return e.applyExpanding(func(vals []float64) float64 {
		return sumF(vals) / float64(len(vals))
	})
}

// Sum computes the expanding sum.
func (e *Expanding) Sum() core.Series {
	return e.applyExpanding(func(vals []float64) float64 {
		return sumF(vals)
	})
}

// Min computes the expanding minimum.
func (e *Expanding) Min() core.Series {
	return e.applyExpanding(func(vals []float64) float64 {
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m
	})
}

// Max computes the expanding maximum.
func (e *Expanding) Max() core.Series {
	return e.applyExpanding(func(vals []float64) float64 {
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m
	})
}

// Std computes the expanding standard deviation.
func (e *Expanding) Std() core.Series {
	return e.applyExpanding(func(vals []float64) float64 {
		n := len(vals)
		if n < 2 {
			return math.NaN()
		}
		mean := sumF(vals) / float64(n)
		ss := 0.0
		for _, v := range vals {
			d := v - mean
			ss += d * d
		}
		return math.Sqrt(ss / float64(n))
	})
}

func (e *Expanding) applyExpanding(fn func([]float64) float64) core.Series {
	s := e.s
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	for i := 0; i < s.Len(); i++ {
		vals := make([]float64, 0, i+1)
		for j := 0; j <= i; j++ {
			if s.NotNull(j) {
				vals = append(vals, s.Float(j))
			}
		}
		if len(vals) < e.minP {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(vals))
		}
	}

	return NewArrowSeries(s.Name()+"_expanding", bldr.NewArray(), s.Index())
}

// --- EWM (Exponentially Weighted Moving) ---

// EWM provides exponentially weighted moving operations.
type EWM struct {
	s     *ArrowSeries
	alpha float64
	span  int
}

// EWM returns an EWM accessor. Either alpha or span should be set.
func (s *ArrowSeries) EWMAlpha(alpha float64) *EWM {
	return &EWM{s: s, alpha: alpha}
}

// EWMSpan returns an EWM accessor with span parameter.
func (s *ArrowSeries) EWMSpan(span int) *EWM {
	alpha := 2.0 / float64(span+1)
	return &EWM{s: s, alpha: alpha, span: span}
}

// Mean computes the exponentially weighted moving average.
func (e *EWM) Mean() core.Series {
	s := e.s
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())

	alpha := e.alpha
	if alpha == 0 && e.span > 0 {
		alpha = 2.0 / float64(e.span+1)
	}

	ewm := 0.0
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			v := s.Float(i)
			if i == 0 || (i > 0 && s.IsNull(i-1)) {
				ewm = v
			} else {
				ewm = alpha*v + (1-alpha)*ewm
			}
			bldr.Append(ewm)
		}
	}

	return NewArrowSeries(s.Name()+"_ewm", bldr.NewArray(), s.Index())
}

// --- GroupBy rolling ---

// GroupByRolling applies rolling operations per group.
func (df *ArrowDataFrame) GroupByRolling(groupCol string, valueCol string, window int, fn core.AggFunc) core.DataFrame {
	groups := df.GroupByGroups([]string{groupCol})
	alloc := memory.NewGoAllocator()

	// Build result
	resultVals := make([]float64, df.Len())
	resultValid := make([]bool, df.Len())
	valSeries := df.Col(valueCol).(*ArrowSeries)
	rolling := valSeries.Rolling(window)

	// For each group, apply rolling
	for _, indices := range groups {
		// Extract group values
		groupVals := make([]float64, len(indices))
		for i, idx := range indices {
			if valSeries.NotNull(idx) {
				groupVals[i] = valSeries.Float(idx)
			} else {
				groupVals[i] = math.NaN()
			}
		}

		// Compute rolling result for group
		for i, idx := range indices {
			start := i - window + 1
			if start < 0 {
				start = 0
			}
			vals := make([]float64, 0)
			for j := start; j <= i; j++ {
				if !math.IsNaN(groupVals[j]) {
					vals = append(vals, groupVals[j])
				}
			}
			if len(vals) > 0 {
				resultVals[idx] = applyAgg(fn, vals)
				resultValid[idx] = true
			}
		}
	}

	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(df.Len())
	for i := 0; i < df.Len(); i++ {
		if resultValid[i] {
			bldr.Append(resultVals[i])
		} else {
			bldr.AppendNull()
		}
	}

	_ = rolling
	return NewDataFrame(
		df.Col(groupCol).(*ArrowSeries),
		NewArrowSeries(valueCol+"_rolling", bldr.NewArray(), df.Index()),
	)
}
