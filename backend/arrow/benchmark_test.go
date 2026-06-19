package arrow

import (
	"fmt"
	"math/rand"
	"testing"
)

// benchmarkFilterDF creates a DataFrame with n rows and 3 filterable string columns + 10 float columns.
func benchmarkFilterDF(n int) *ArrowDataFrame {
	rng := rand.New(rand.NewSource(42))

	citycodes := []string{"320100", "320200", "320300", "320400", "320500", "320600", "320700", "320800", "320900", "321000"}
	fres := []string{"hourly", "daily", "monthly"}
	polls := []string{"co", "nox", "pm25", "pm10", "so2"}

	cityData := make([]string, n)
	freData := make([]string, n)
	pollData := make([]string, n)
	for i := 0; i < n; i++ {
		cityData[i] = citycodes[rng.Intn(len(citycodes))]
		freData[i] = fres[rng.Intn(len(fres))]
		pollData[i] = polls[rng.Intn(len(polls))]
	}

	city := NewStringSeries("citycode", cityData, nil)
	fre := NewStringSeries("fre", freData, nil)
	poll := NewStringSeries("poll", pollData, nil)

	cols := []*ArrowSeries{city, fre, poll}
	for j := 0; j < 10; j++ {
		floats := make([]float64, n)
		for i := 0; i < n; i++ {
			floats[i] = rng.Float64() * 1000
		}
		cols = append(cols, NewFloat64Series(fmt.Sprintf("val%d", j), floats, nil))
	}
	return NewDataFrame(cols...)
}

// BenchmarkFilter benchmarks the optimized Filter (SIMD + contiguous + single scan).
func BenchmarkFilter(b *testing.B) {
	for _, size := range []int{10000, 100000, 500000} {
		df := benchmarkFilterDF(size)
		b.Run(fmt.Sprintf("rows=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				city := df.Col("citycode")
				fre := df.Col("fre")
				poll := df.Col("poll")
				mask := make([]bool, df.Len())
				for j := 0; j < df.Len(); j++ {
					mask[j] = city.String(j) == "320600" &&
						fre.String(j) == "hourly" &&
						poll.String(j) == "co"
				}
				df.Filter(mask)
			}
		})
	}
}

// BenchmarkTakeContiguous benchmarks the contiguous zero-copy Take path.
func BenchmarkTakeContiguous(b *testing.B) {
	for _, size := range []int{10000, 100000, 500000} {
		df := benchmarkFilterDF(size)
		b.Run(fmt.Sprintf("rows=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				df.Slice(0, size/2) // contiguous → zero-copy
			}
		})
	}
}

// BenchmarkTakeScattered benchmarks the non-contiguous Take path.
func BenchmarkTakeScattered(b *testing.B) {
	for _, size := range []int{10000, 100000, 500000} {
		df := benchmarkFilterDF(size)
		indices := make([]int, 0, size/10)
		for j := 0; j < size; j += 10 {
			indices = append(indices, j)
		}
		b.Run(fmt.Sprintf("rows=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				df.Take(indices) // scattered → builder path
			}
		})
	}
}
