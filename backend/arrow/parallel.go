package arrow

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// ParallelConfig controls parallelism behavior.
type ParallelConfig struct {
	Workers int // 0 = runtime.NumCPU()
}

var defaultParallel = ParallelConfig{Workers: 0}

func (c ParallelConfig) numWorkers() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return runtime.NumCPU()
}

// ParallelTransform applies a function to each numeric column in parallel.
func (df *ArrowDataFrame) ParallelTransform(fn MapFloatFunc, cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.numWorkers())

	for i, name := range cols {
		col := df.Col(name).(*ArrowSeries)
		if col.Dtype().IsNumeric() {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int, s *ArrowSeries) {
				defer wg.Done()
				defer func() { <-sem }()
				series[idx] = s.MapFloat(fn).(*ArrowSeries)
			}(i, col)
		} else {
			series[i] = col
		}
	}
	wg.Wait()
	return NewDataFrame(series...)
}

// ParallelApplyCols applies a function to each column in parallel.
func (df *ArrowDataFrame) ParallelApplyCols(fn ColApplyFunc, cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	cols := df.Columns()
	series := make([]*ArrowSeries, len(cols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.numWorkers())

	for i, name := range cols {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, colName string) {
			defer wg.Done()
			defer func() { <-sem }()
			result := fn(df.Col(colName))
			series[idx] = result.(*ArrowSeries)
		}(i, name)
	}
	wg.Wait()
	return NewDataFrame(series...)
}

// ParallelAgg performs groupby aggregation in parallel per column.
func (df *ArrowDataFrame) ParallelAgg(groupCols []string, aggs map[string]core.AggFunc, cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	groups := df.GroupByGroups(groupCols)
	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	numGroups := len(groupKeys)

	alloc := memory.NewGoAllocator()

	// Build group key columns (sequential, small)
	var resultSeries []*ArrowSeries
	for _, gc := range groupCols {
		s := df.Col(gc).(*ArrowSeries)
		bldr := newBuilder(s.Dtype(), alloc)
		bldr.Resize(numGroups)
		for _, k := range groupKeys {
			firstIdx := groups[k][0]
			copyValue(bldr, s, firstIdx)
		}
		resultSeries = append(resultSeries, NewArrowSeries(gc, bldr.NewArray(), nil))
	}

	// Aggregation columns in parallel
	aggCols := make([]string, 0, len(aggs))
	for colName := range aggs {
		aggCols = append(aggCols, colName)
	}
	aggResults := make([]*ArrowSeries, len(aggCols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.numWorkers())

	for i, colName := range aggCols {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, name string, fn core.AggFunc) {
			defer wg.Done()
			defer func() { <-sem }()
			s := df.Col(name).(*ArrowSeries)
			bldr := array.NewFloat64Builder(alloc)
			bldr.Resize(numGroups)
			for _, k := range groupKeys {
				vals := make([]float64, 0, len(groups[k]))
				for _, rowIdx := range groups[k] {
					if s.NotNull(rowIdx) {
						vals = append(vals, s.Float(rowIdx))
					}
				}
				bldr.Append(applyAgg(fn, vals))
			}
			aggResults[idx] = NewArrowSeries(name+"_"+fn.String(), bldr.NewArray(), nil)
		}(i, colName, aggs[colName])
	}
	wg.Wait()

	resultSeries = append(resultSeries, aggResults...)
	return NewDataFrame(resultSeries...)
}

// ParallelFilter filters rows in parallel chunks.
func (df *ArrowDataFrame) ParallelFilter(mask []bool, cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	rows, _ := df.Shape()
	workers := c.numWorkers()
	if workers > rows {
		workers = rows
	}
	if workers == 0 {
		workers = 1
	}
	chunkSize := (rows + workers - 1) / workers

	type chunkResult struct {
		indices []int
	}
	results := make([]chunkResult, workers)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			start := worker * chunkSize
			end := start + chunkSize
			if end > rows {
				end = rows
			}
			idx := make([]int, 0)
			for i := start; i < end; i++ {
				if mask[i] {
					idx = append(idx, i)
				}
			}
			results[worker] = chunkResult{indices: idx}
		}(w)
	}
	wg.Wait()

	allIndices := make([]int, 0)
	for _, r := range results {
		allIndices = append(allIndices, r.indices...)
	}
	return df.Take(allIndices)
}

// ParallelInfo returns CPU count info.
func ParallelInfo() string {
	return fmt.Sprintf("CPUs: %d", runtime.NumCPU())
}

// ParallelDropNA drops rows with any null in parallel column checks.
func (df *ArrowDataFrame) ParallelDropNA(cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	rows, _ := df.Shape()
	numCols := len(df.columns)
	workers := c.numWorkers()
	if workers > numCols {
		workers = numCols
	}
	if workers <= 1 {
		return df.DropNA()
	}

	// Each column builds its own null mask in parallel
	nullMasks := make([][]bool, numCols)
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for j, col := range df.columns {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s *ArrowSeries) {
			defer wg.Done()
			defer func() { <-sem }()
			mask := make([]bool, rows)
			for i := 0; i < rows; i++ {
				mask[i] = s.IsNull(i)
			}
			nullMasks[idx] = mask
		}(j, col)
	}
	wg.Wait()

	// Combine masks: keep row if NO column has null
	keep := make([]bool, rows)
	for i := 0; i < rows; i++ {
		keep[i] = true
		for j := 0; j < numCols; j++ {
			if nullMasks[j][i] {
				keep[i] = false
				break
			}
		}
	}
	return df.Filter(keep)
}

// ParallelFillNA fills nulls in parallel per column.
func (df *ArrowDataFrame) ParallelFillNA(value interface{}, cfg ...ParallelConfig) core.DataFrame {
	c := defaultParallel
	if len(cfg) > 0 {
		c = cfg[0]
	}

	numCols := len(df.columns)
	workers := c.numWorkers()
	if workers <= 1 || numCols <= 1 {
		return df.FillNA(value)
	}

	series := make([]*ArrowSeries, numCols)
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	alloc := memory.NewGoAllocator()

	for j, col := range df.columns {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s *ArrowSeries) {
			defer wg.Done()
			defer func() { <-sem }()
			bldr := array.NewBuilder(alloc, s.arr.DataType())
			bldr.Resize(s.Len())
			for i := 0; i < s.Len(); i++ {
				if s.IsNull(i) {
					appendValue(bldr, s.Dtype(), value)
				} else {
					copyValue(bldr, s, i)
				}
			}
			series[idx] = NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
			bldr.Release()
		}(j, col)
	}
	wg.Wait()

	return NewDataFrame(series...)
}

// autoParallelThresholds defines when to auto-enable parallelism.
const (
	autoParallelMinCols = 4    // Agg/FillNA/DropNA: parallelize when >= this many columns
	autoParallelMinRows = 10000 // Filter: parallelize when >= this many rows
)
