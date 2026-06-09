package arrow

import (
	"testing"

	"github.com/lekeeith/godas/core"
)

// --- isin ---

func TestIsIn(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 3, 4, 5}, nil)
	result := s.IsIn([]interface{}{int64(2), int64(4)}).(*ArrowSeries)
	if result.Bool(0) || !result.Bool(1) || result.Bool(2) || !result.Bool(3) || result.Bool(4) {
		t.Error("IsIn failed")
	}
}

// --- value_counts ---

func TestValueCounts(t *testing.T) {
	s := NewStringSeries("fruit", []string{"apple", "banana", "apple", "cherry", "banana", "apple"}, nil)
	vc := s.ValueCounts()
	if vc.Values.Len() != 3 {
		t.Fatalf("unique values = %d, want 3", vc.Values.Len())
	}
	// apple should have count 3
	for i := 0; i < vc.Values.Len(); i++ {
		if vc.Values.String(i) == "apple" && vc.Counts.Int(i) != 3 {
			t.Errorf("apple count = %d, want 3", vc.Counts.Int(i))
		}
		if vc.Values.String(i) == "banana" && vc.Counts.Int(i) != 2 {
			t.Errorf("banana count = %d, want 2", vc.Counts.Int(i))
		}
	}
}

func TestNUnique(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 2, 3, 3, 3}, nil)
	if s.NUnique() != 3 {
		t.Errorf("NUnique = %d, want 3", s.NUnique())
	}
}

func TestUnique(t *testing.T) {
	s := NewInt64Series("x", []int64{3, 1, 2, 1, 3}, nil)
	u := s.Unique()
	if u.Len() != 3 {
		t.Fatalf("Unique.Len() = %d, want 3", u.Len())
	}
}

// --- duplicated ---

func TestDuplicated(t *testing.T) {
	s := NewStringSeries("x", []string{"a", "b", "a", "c", "b"}, nil)
	dup := s.Duplicated("first").(*ArrowSeries)
	// first occurrence should be false
	if dup.Bool(0) || dup.Bool(1) || dup.Bool(3) {
		t.Error("first occurrences should not be duplicated")
	}
	// second occurrence should be true
	if !dup.Bool(2) || !dup.Bool(4) {
		t.Error("second occurrences should be duplicated")
	}
}

func TestDropDuplicates(t *testing.T) {
	s := NewStringSeries("x", []string{"a", "b", "a", "c", "b"}, nil)
	result := s.DropDuplicates("first")
	if result.Len() != 3 {
		t.Fatalf("DropDuplicates.Len() = %d, want 3", result.Len())
	}
}

func TestDataFrameDuplicated(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("a", []string{"x", "y", "x", "z"}, nil),
		NewInt64Series("b", []int64{1, 2, 1, 3}, nil),
	)
	dup := df.Duplicated([]string{"a", "b"}, "first").(*ArrowSeries)
	if dup.Bool(0) || !dup.Bool(2) {
		t.Error("DataFrame Duplicated failed")
	}
}

func TestDataFrameDropDuplicates(t *testing.T) {
	df := NewDataFrame(
		NewStringSeries("a", []string{"x", "y", "x", "z"}, nil),
		NewInt64Series("b", []int64{1, 2, 1, 3}, nil),
	)
	result := df.DropDuplicates([]string{"a", "b"}, "first")
	if result.Len() != 3 {
		t.Fatalf("DropDuplicates.Len() = %d, want 3", result.Len())
	}
}

// --- concat ---

func TestConcatRows(t *testing.T) {
	df1 := NewDataFrame(
		NewStringSeries("name", []string{"a", "b"}, nil),
		NewInt64Series("val", []int64{1, 2}, nil),
	)
	df2 := NewDataFrame(
		NewStringSeries("name", []string{"c", "d"}, nil),
		NewInt64Series("val", []int64{3, 4}, nil),
	)
	result := Concat([]*ArrowDataFrame{df1, df2}, ConcatRows)
	if result.Len() != 4 {
		t.Fatalf("ConcatRows.Len() = %d, want 4", result.Len())
	}
	if result.Col("name").String(2) != "c" {
		t.Errorf("ConcatRows name[2] = %q", result.Col("name").String(2))
	}
}

func TestConcatCols(t *testing.T) {
	df1 := NewDataFrame(NewInt64Series("a", []int64{1, 2}, nil))
	df2 := NewDataFrame(NewStringSeries("b", []string{"x", "y"}, nil))
	result := Concat([]*ArrowDataFrame{df1, df2}, ConcatCols)
	if result.Len() != 2 || len(result.Columns()) != 2 {
		t.Fatalf("ConcatCols shape = (%d,%d)", result.Len(), len(result.Columns()))
	}
}

// --- interpolate/ffill/bfill ---

func TestFillForward(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.INT64, nil)
	bldr.AppendInt(1)
	bldr.AppendNull()
	bldr.AppendNull()
	bldr.AppendInt(4)
	bldr.AppendNull()
	s := bldr.Build()

	ff := s.FillForward().(*ArrowSeries)
	if ff.Int(0) != 1 || ff.Int(1) != 1 || ff.Int(2) != 1 || ff.Int(3) != 4 || ff.Int(4) != 4 {
		t.Errorf("FillForward: %v", ff.ToSlice())
	}
}

func TestFillBackward(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.INT64, nil)
	bldr.AppendNull()
	bldr.AppendInt(2)
	bldr.AppendNull()
	bldr.AppendNull()
	bldr.AppendInt(5)
	s := bldr.Build()

	fb := s.FillBackward().(*ArrowSeries)
	// backward fill: index 0←2, 2←5, 3←5
	if fb.Int(0) != 2 || fb.Int(2) != 5 || fb.Int(3) != 5 {
		t.Errorf("FillBackward: %v", fb.ToSlice())
	}
	if !fb.IsNull(1) || fb.Int(1) != 2 {
		// index 1 is valid (2), should stay 2
		if fb.Int(1) != 2 {
			t.Errorf("FillBackward[1] = %d, want 2", fb.Int(1))
		}
	}
}

func TestInterpolate(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.FLOAT64, nil)
	bldr.AppendFloat(1)
	bldr.AppendNull()
	bldr.AppendFloat(3)
	bldr.AppendNull()
	bldr.AppendFloat(5)
	s := bldr.Build()

	ip := s.Interpolate().(*ArrowSeries)
	if ip.Float(0) != 1 || ip.Float(2) != 3 || ip.Float(4) != 5 {
		t.Errorf("Interpolate basic: %v", ip.ToSlice())
	}
	// Interpolated value at index 1 should be 2
	if ip.Float(1) != 2 {
		t.Errorf("Interpolate[1] = %g, want 2", ip.Float(1))
	}
}

func TestDataFrameFillNAMethod(t *testing.T) {
	bldr := NewSeriesBuilder("x", core.INT64, nil)
	bldr.AppendInt(1)
	bldr.AppendNull()
	bldr.AppendInt(3)
	s := bldr.Build()
	df := NewDataFrame(s)

	result := df.FillNAMethod("ffill")
	if result.Col("x").(*ArrowSeries).Int(1) != 1 {
		t.Errorf("FillNAMethod ffill[1] = %d, want 1", result.Col("x").(*ArrowSeries).Int(1))
	}
}

// --- astype ---

func TestAsType(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 3}, nil)

	// int → float
	f := s.AsType(core.FLOAT64).(*ArrowSeries)
	if f.Float(0) != 1.0 || f.Dtype() != core.FLOAT64 {
		t.Error("AsType int→float failed")
	}

	// int → string
	str := s.AsType(core.STRING).(*ArrowSeries)
	if str.String(0) != "1" || str.Dtype() != core.STRING {
		t.Error("AsType int→string failed")
	}

	// float → int
	fs := NewFloat64Series("y", []float64{1.9, 2.5, 3.1}, nil)
	i := fs.AsType(core.INT64).(*ArrowSeries)
	if i.Int(0) != 1 || i.Int(2) != 3 {
		t.Error("AsType float→int failed")
	}
}

func TestToNumeric(t *testing.T) {
	s := NewStringSeries("x", []string{"1.5", "2.7", "abc", "4.0"}, nil)
	result := ToNumeric(s).(*ArrowSeries)
	if result.Float(0) != 1.5 || result.Float(1) != 2.7 {
		t.Error("ToNumeric valid values failed")
	}
	if !result.IsNull(2) {
		t.Error("ToNumeric should null for 'abc'")
	}
}

// --- .str accessor ---

func TestStrUpperLower(t *testing.T) {
	s := NewStringSeries("x", []string{"Hello", "World"}, nil)
	u := s.Str().Upper().(*ArrowSeries)
	l := s.Str().Lower().(*ArrowSeries)
	if u.String(0) != "HELLO" || l.String(1) != "world" {
		t.Error("Upper/Lower failed")
	}
}

func TestStrContains(t *testing.T) {
	s := NewStringSeries("x", []string{"hello world", "foo bar", "hello"}, nil)
	c := s.Str().Contains("hello").(*ArrowSeries)
	if !c.Bool(0) || c.Bool(1) || !c.Bool(2) {
		t.Error("Contains failed")
	}
}

func TestStrStartsWith(t *testing.T) {
	s := NewStringSeries("x", []string{"hello", "help", "world"}, nil)
	r := s.Str().StartsWith("hel").(*ArrowSeries)
	if !r.Bool(0) || !r.Bool(1) || r.Bool(2) {
		t.Error("StartsWith failed")
	}
}

func TestStrReplace(t *testing.T) {
	s := NewStringSeries("x", []string{"a-b-c", "d-e"}, nil)
	r := s.Str().Replace("-", "_").(*ArrowSeries)
	if r.String(0) != "a_b_c" {
		t.Errorf("Replace = %q", r.String(0))
	}
}

func TestStrSplit(t *testing.T) {
	s := NewStringSeries("x", []string{"a,b,c", "d,e"}, nil)
	r := s.Str().Split(",", 1).(*ArrowSeries)
	if r.String(0) != "b" || r.String(1) != "e" {
		t.Error("Split failed")
	}
}

func TestStrLen(t *testing.T) {
	s := NewStringSeries("x", []string{"abc", "de", ""}, nil)
	r := s.Str().Len().(*ArrowSeries)
	if r.Int(0) != 3 || r.Int(1) != 2 || r.Int(2) != 0 {
		t.Error("Len failed")
	}
}

func TestStrStrip(t *testing.T) {
	s := NewStringSeries("x", []string{"  hello  ", " world "}, nil)
	r := s.Str().Strip().(*ArrowSeries)
	if r.String(0) != "hello" || r.String(1) != "world" {
		t.Error("Strip failed")
	}
}

func TestStrIsDigit(t *testing.T) {
	s := NewStringSeries("x", []string{"123", "abc", "12a"}, nil)
	r := s.Str().IsDigit().(*ArrowSeries)
	if !r.Bool(0) || r.Bool(1) || r.Bool(2) {
		t.Error("IsDigit failed")
	}
}

// --- nlargest/nsmallest ---

func TestNLargest(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5, 9, 2}, nil)
	r := s.NLargest(3)
	if r.Len() != 3 {
		t.Fatalf("NLargest.Len() = %d, want 3", r.Len())
	}
	// Should be 9, 5, 4
	if r.Float(0) != 9 || r.Float(1) != 5 || r.Float(2) != 4 {
		t.Errorf("NLargest: %g,%g,%g", r.Float(0), r.Float(1), r.Float(2))
	}
}

func TestNSmallest(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5, 9, 2}, nil)
	r := s.NSmallest(3)
	if r.Len() != 3 {
		t.Fatalf("NSmallest.Len() = %d, want 3", r.Len())
	}
	if r.Float(0) != 1 || r.Float(1) != 1 || r.Float(2) != 2 {
		t.Errorf("NSmallest: %g,%g,%g", r.Float(0), r.Float(1), r.Float(2))
	}
}

// --- quantile ---

func TestQuantile(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	if s.Quantile(0) != 1 {
		t.Errorf("Q0 = %g, want 1", s.Quantile(0))
	}
	if s.Quantile(1) != 5 {
		t.Errorf("Q1 = %g, want 5", s.Quantile(1))
	}
	if s.Quantile(0.5) != 3 {
		t.Errorf("Q0.5 = %g, want 3", s.Quantile(0.5))
	}
}

// --- rank ---

func TestRank(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5}, nil)
	r := s.Rank("average").(*ArrowSeries)
	// Values sorted: 1(1), 1(2), 3(3), 4(4), 5(5)
	// Rank of first 1: average of 1,2 = 1.5
	if r.Float(1) != 1.5 && r.Float(3) != 1.5 {
		t.Errorf("Rank average for ties: %g, %g", r.Float(1), r.Float(3))
	}
	if r.Float(0) != 3 || r.Float(2) != 4 || r.Float(4) != 5 {
		t.Errorf("Rank: %v", r.ToSlice())
	}
}

func TestRankDense(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5}, nil)
	r := s.Rank("dense").(*ArrowSeries)
	// Dense rank: 1→0, 1→0, 3→1, 4→2, 5→3
	if r.Float(1) != 0 || r.Float(3) != 0 {
		t.Errorf("Dense rank for ties should be 0, got %g,%g", r.Float(1), r.Float(3))
	}
	if r.Float(0) != 1 || r.Float(2) != 2 || r.Float(4) != 3 {
		t.Errorf("Dense rank: %v", r.ToSlice())
	}
}

// --- corr/cov ---

func TestCorr(t *testing.T) {
	a := NewFloat64Series("a", []float64{1, 2, 3, 4, 5}, nil)
	b := NewFloat64Series("b", []float64{2, 4, 6, 8, 10}, nil)
	r := Corr(a, b)
	if r < 0.9999 || r > 1.0001 {
		t.Errorf("Corr = %g, want ~1.0", r)
	}
}

func TestCov(t *testing.T) {
	a := NewFloat64Series("a", []float64{1, 2, 3, 4, 5}, nil)
	b := NewFloat64Series("b", []float64{1, 2, 3, 4, 5}, nil)
	c := Cov(a, b)
	if c != 2.0 {
		t.Errorf("Cov = %g, want 2.0", c)
	}
}

func TestCorrMatrix(t *testing.T) {
	df := NewDataFrame(
		NewFloat64Series("a", []float64{1, 2, 3, 4, 5}, nil),
		NewFloat64Series("b", []float64{2, 4, 6, 8, 10}, nil),
	)
	m := CorrMatrix(df)
	diag := m.Col("a").(*ArrowSeries).Float(0)
	if diag < 0.9999 || diag > 1.0001 {
		t.Errorf("CorrMatrix diagonal = %g, want ~1", diag)
	}
	off := m.Col("a").(*ArrowSeries).Float(1)
	if off < 0.9999 || off > 1.0001 {
		t.Errorf("CorrMatrix a,b = %g, want ~1", off)
	}
}

// --- rolling ---

func TestRollingMean(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	r := s.Rolling(3).Mean().(*ArrowSeries)
	// window=3, default minP=3: first 2 are null
	if !r.IsNull(0) || !r.IsNull(1) {
		t.Errorf("RollingMean first 2 should be null: %v", r.ToSlice()[:2])
	}
	if r.Float(2) != 2 || r.Float(3) != 3 || r.Float(4) != 4 {
		t.Errorf("RollingMean: %g,%g,%g", r.Float(2), r.Float(3), r.Float(4))
	}
}

func TestRollingMeanMinPeriods(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	r := s.Rolling(3).MinPeriods(1).Mean().(*ArrowSeries)
	if r.Float(0) != 1 || r.Float(1) != 1.5 {
		t.Errorf("RollingMean minP=1: %g,%g", r.Float(0), r.Float(1))
	}
	if r.Float(2) != 2 || r.Float(3) != 3 || r.Float(4) != 4 {
		t.Errorf("RollingMean minP=1: %g,%g,%g", r.Float(2), r.Float(3), r.Float(4))
	}
}

func TestRollingSum(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	r := s.Rolling(3).Sum().(*ArrowSeries)
	if r.Float(2) != 6 || r.Float(4) != 12 {
		t.Errorf("RollingSum: %g,%g", r.Float(2), r.Float(4))
	}
}

func TestRollingMin(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5}, nil)
	r := s.Rolling(3).Min().(*ArrowSeries)
	if r.Float(2) != 1 || r.Float(3) != 1 || r.Float(4) != 1 {
		t.Errorf("RollingMin: %g,%g,%g", r.Float(2), r.Float(3), r.Float(4))
	}
}

func TestRollingMax(t *testing.T) {
	s := NewFloat64Series("x", []float64{3, 1, 4, 1, 5}, nil)
	r := s.Rolling(3).Max().(*ArrowSeries)
	if r.Float(2) != 4 || r.Float(3) != 4 || r.Float(4) != 5 {
		t.Errorf("RollingMax: %g,%g,%g", r.Float(2), r.Float(3), r.Float(4))
	}
}

func TestExpanding(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	r := s.Expanding().Mean().(*ArrowSeries)
	if r.Float(0) != 1 || r.Float(1) != 1.5 || r.Float(4) != 3 {
		t.Errorf("ExpandingMean: %g,%g,%g", r.Float(0), r.Float(1), r.Float(4))
	}
}

func TestEWM(t *testing.T) {
	s := NewFloat64Series("x", []float64{1, 2, 3, 4, 5}, nil)
	r := s.EWMAlpha(0.5).Mean().(*ArrowSeries)
	if r.Float(0) != 1 {
		t.Errorf("EWM[0] = %g, want 1", r.Float(0))
	}
	// EWM with alpha=0.5: 0.5*2 + 0.5*1 = 1.5
	if r.Float(1) != 1.5 {
		t.Errorf("EWM[1] = %g, want 1.5", r.Float(1))
	}
}
