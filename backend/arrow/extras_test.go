package arrow

import (
	"math"
	"strings"
	"testing"

	"github.com/lekeeith/godas/core"
)

// --- clip ---

func TestClip(t *testing.T) {
	s := NewFloat64Series("x", []float64{-5, 0, 3, 7, 12}, nil)
	r := s.Clip(0, 10).(*ArrowSeries)
	expected := []float64{0, 0, 3, 7, 10}
	for i, want := range expected {
		if r.Float(i) != want {
			t.Errorf("Clip[%d] = %g, want %g", i, r.Float(i), want)
		}
	}
}

// --- convert_dtypes ---

func TestConvertDtypes(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("a", []string{"1", "2", "3"}, nil),
		NewStringSeries("b", []string{"1.5", "2.7", "3.14"}, nil),
		NewStringSeries("c", []string{"hello", "world", "foo"}, nil),
	)
	result := df.ConvertDtypes()
	if result.Col("a").Dtype() != core.INT64 {
		t.Errorf("a dtype = %s, want int64", result.Col("a").Dtype())
	}
	if result.Col("b").Dtype() != core.FLOAT64 {
		t.Errorf("b dtype = %s, want float64", result.Col("b").Dtype())
	}
	if result.Col("c").Dtype() != core.STRING {
		t.Errorf("c dtype = %s, want string", result.Col("c").Dtype())
	}
}

// --- query ---

func TestQuery(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil),
		NewFloat64Series("age", []float64{25, 30, 35}, nil),
	)
	r := df.Query("age > 28")
	if r.Len() != 2 {
		t.Fatalf("Query.Len() = %d, want 2", r.Len())
	}
	if r.Col("name").String(0) != "bob" {
		t.Errorf("Query[0] = %q", r.Col("name").String(0))
	}
}

func TestQueryString(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil),
	)
	r := df.Query("name == bob")
	if r.Len() != 1 {
		t.Fatalf("Query.Len() = %d, want 1", r.Len())
	}
}

// --- pivot ---

func TestPivot(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("date", []string{"Jan", "Jan", "Feb", "Feb"}, nil),
		NewStringSeries("type", []string{"A", "B", "A", "B"}, nil),
		NewFloat64Series("val", []float64{10, 20, 30, 40}, nil),
	)
	r := df.Pivot("date", "type", "val")
	if r.Len() != 2 {
		t.Fatalf("Pivot.Len() = %d, want 2", r.Len())
	}
	_, cols := r.Shape()
	if cols != 3 { // date, A, B
		t.Fatalf("Pivot.Cols = %d, want 3", cols)
	}
}

// --- melt ---

func TestMelt(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob"}, nil),
		NewFloat64Series("math", []float64{90, 80}, nil),
		NewFloat64Series("eng", []float64{85, 75}, nil),
	)
	r := df.Melt([]string{"name"}, []string{"math", "eng"})
	rows, _ := r.Shape()
	if rows != 4 { // 2 names * 2 subjects
		t.Fatalf("Melt.Len() = %d, want 4", rows)
	}
}

// --- compare ---

func TestCompare(t *testing.T) {
	df1 := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	)
	df2 := NewDataFrame(
		NewFloat64Series("x", []float64{1, 5, 3}, nil),
	)
	r := df1.Compare(df2, false)
	rows, _ := r.Shape()
	// Only row 1 differs
	if rows != 3 {
		t.Fatalf("Compare rows = %d, want 3", rows)
	}
	// Row 1 should show "2→5"
	if r.Col("x").String(1) != "2→5" {
		t.Errorf("Compare[1] = %q, want %q", r.Col("x").String(1), "2→5")
	}
}

// --- mode ---

func TestMode(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 2, 3, 3, 3}, nil)
	m := s.Mode()
	if m.Len() != 1 {
		t.Fatalf("Mode.Len() = %d, want 1", m.Len())
	}
	if m.Int(0) != 3 {
		t.Errorf("Mode = %d, want 3", m.Int(0))
	}
}

func TestModeMultiple(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 1, 2, 2, 3}, nil)
	m := s.Mode()
	if m.Len() != 2 {
		t.Fatalf("Mode.Len() = %d, want 2", m.Len())
	}
}

// --- skew / kurt ---

func TestSkew(t *testing.T) {
	// Symmetric distribution should have skew ~0
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	sk := s.Skew()
	if math.Abs(sk) > 0.001 {
		t.Errorf("Skew = %g, want ~0", sk)
	}
}

func TestKurt(t *testing.T) {
	// Uniform distribution should have kurt < 0 (platykurtic)
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil)
	k := s.Kurt()
	if k > 0 {
		t.Errorf("Kurt = %g, want < 0 for uniform", k)
	}
}

// --- memory_usage ---

func TestMemoryUsage(t *testing.T) {
	df := NewDataFrame(
		NewInt64Series("a", []int64{1, 2, 3}, nil),
		NewFloat64Series("b", []float64{1.1, 2.2, 3.3}, nil),
	)
	usage := df.MemoryUsage()
	if usage["a"] != 24 { // 3 * 8 bytes
		t.Errorf("a memory = %d, want 24", usage["a"])
	}
	if usage["b"] != 24 {
		t.Errorf("b memory = %d, want 24", usage["b"])
	}
}

// --- pipe ---

func TestPipe(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("x", []float64{1, 2, 3}, nil),
	)
	r := df.Pipe(func(d *ArrowDataFrame) *ArrowDataFrame {
		return d.WithColumn("y", NewFloat64Series("y", []float64{10, 20, 30}, nil)).(*ArrowDataFrame)
	})
	_, cols := r.Shape()
	if cols != 2 {
		t.Fatalf("Pipe cols = %d, want 2", cols)
	}
}

// --- get_dummies ---

func TestGetDummies(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("color", []string{"red", "blue", "red", "green"}, nil),
	)
	r := GetDummies(df, []string{"color"})
	_, cols := r.Shape()
	if cols != 4 { // original + 3 dummies
		t.Fatalf("GetDummies cols = %d, want 4", cols)
	}
}

// --- cut ---

func TestCut(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 5, 10, 15, 20}, nil)
	r := Cut(s, 3)
	if r.Len() != 5 {
		t.Fatalf("Cut.Len() = %d, want 5", r.Len())
	}
	// First value should be in the first bin
	if !strings.HasPrefix(r.String(0), "(1,") {
		t.Errorf("Cut[0] = %q, should start with (1,", r.String(0))
	}
}

// --- qcut ---

func TestQCut(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil)
	r := QCut(s, 4)
	if r.Len() != 10 {
		t.Fatalf("QCut.Len() = %d, want 10", r.Len())
	}
}

// --- explode ---

func TestExplode(t *testing.T) {
	s := NewStringSeries("x", []string{"a,b", "c", "d,e,f"}, nil)
	r := s.Explode(",")
	if r.Len() != 6 {
		t.Fatalf("Explode.Len() = %d, want 6", r.Len())
	}
	if r.String(0) != "a" || r.String(1) != "b" || r.String(3) != "d" {
		t.Errorf("Explode values: %v", r.ToSlice())
	}
}

// --- factorize ---

func TestFactorize(t *testing.T) {
	s := NewStringSeries("x", []string{"b", "a", "c", "a", "b"}, nil)
	codes, uniques := Factorize(s)
	if codes.Len() != 5 {
		t.Fatalf("codes.Len() = %d, want 5", codes.Len())
	}
	if uniques.Len() != 3 {
		t.Fatalf("uniques.Len() = %d, want 3", uniques.Len())
	}
	// b=0, a=1, c=2
	if codes.Int(0) != 0 || codes.Int(1) != 1 || codes.Int(2) != 2 {
		t.Errorf("codes = %v", codes.ToSlice())
	}
}

// --- groupby transform/filter/apply ---

func TestGroupByTransform(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90}, nil),
	)
	// Transform: normalize within group
	result := df.GroupByTransform([]string{"dept"}, "salary", func(vals []float64) []float64 {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		mean := sum / float64(len(vals))
		r := make([]float64, len(vals))
		for i, v := range vals {
			r[i] = v / mean
		}
		return r
	})
	rows, _ := result.Shape()
	if rows != 4 {
		t.Fatalf("GroupByTransform rows = %d, want 4", rows)
	}
}

func TestGroupByFilter(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90}, nil),
	)
	// Keep groups where mean salary > 85
	result := df.GroupByFilter([]string{"dept"}, func(gdf core.DataFrame) bool {
		sum := 0.0
		s := gdf.Col("salary").(*ArrowSeries)
		for i := 0; i < s.Len(); i++ {
			if s.NotNull(i) {
				sum += s.Float(i)
			}
		}
		return sum/float64(s.Len()) > 85
	})
	rows, _ := result.Shape()
	if rows != 2 { // only eng group
		t.Fatalf("GroupByFilter rows = %d, want 2", rows)
	}
}

func TestGroupByApply(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("dept", []string{"eng", "eng", "sales"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80}, nil),
	)
	result := df.GroupByApply([]string{"dept"}, func(gdf core.DataFrame) core.DataFrame {
		s := gdf.Col("salary").(*ArrowSeries)
		sum := 0.0
		for i := 0; i < s.Len(); i++ {
			if s.NotNull(i) {
				sum += s.Float(i)
			}
		}
		return NewDataFrame(
			NewStringSeries("dept", []string{gdf.Col("dept").String(0)}, nil),
			NewFloat64Series("total", []float64{sum}, nil),
		)
	})
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("GroupByApply rows = %d, want 2", rows)
	}
}
