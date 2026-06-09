package arrow

import (
	"testing"

	"github.com/godans/godans/core"
)

func TestNewInt64Series(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20, 30, 40}, nil)
	if s.Name() != "x" {
		t.Fatalf("Name() = %q, want %q", s.Name(), "x")
	}
	if s.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", s.Len())
	}
	if s.Dtype() != core.INT64 {
		t.Fatalf("Dtype() = %v, want INT64", s.Dtype())
	}
	if s.Int(0) != 10 || s.Int(3) != 40 {
		t.Errorf("values: got %d,%d want 10,40", s.Int(0), s.Int(3))
	}
	if s.Float(1) != 20.0 {
		t.Errorf("Float(1) = %g, want 20", s.Float(1))
	}
	if s.NullCount() != 0 {
		t.Errorf("NullCount() = %d, want 0", s.NullCount())
	}
}

func TestNewFloat64Series(t *testing.T) {
	s := NewFloat64Series("f", []float64{1.1, 2.2, 3.3}, nil)
	if s.Dtype() != core.FLOAT64 {
		t.Fatalf("Dtype() = %v, want FLOAT64", s.Dtype())
	}
	if s.Float(0) != 1.1 {
		t.Errorf("Float(0) = %g, want 1.1", s.Float(0))
	}
	if s.Int(1) != 2 {
		t.Errorf("Int(1) = %d, want 2", s.Int(1))
	}
}

func TestNewStringSeries(t *testing.T) {
	s := NewStringSeries("names", []string{"alice", "bob", "charlie"}, nil)
	if s.Dtype() != core.STRING {
		t.Fatalf("Dtype() = %v, want STRING", s.Dtype())
	}
	if s.String(0) != "alice" {
		t.Errorf("String(0) = %q, want %q", s.String(0), "alice")
	}
}

func TestNewBoolSeries(t *testing.T) {
	s := NewBoolSeries("b", []bool{true, false, true}, nil)
	if s.Dtype() != core.BOOL {
		t.Fatalf("Dtype() = %v, want BOOL", s.Dtype())
	}
	if !s.Bool(0) || s.Bool(1) || !s.Bool(2) {
		t.Errorf("bool values mismatch")
	}
}

func TestSeriesWithNulls(t *testing.T) {
	s := NewInt64SeriesWithNulls("n", []int64{1, 2, 3, 4}, []bool{true, false, true, false}, nil)
	if s.NullCount() != 2 {
		t.Fatalf("NullCount() = %d, want 2", s.NullCount())
	}
	if !s.IsNull(1) {
		t.Error("expected index 1 to be null")
	}
	if s.IsNull(0) {
		t.Error("expected index 0 to be valid")
	}
	if s.Int(1) != 0 {
		t.Errorf("null Int() = %d, want 0", s.Int(1))
	}
}

func TestSeriesHeadTail(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 3, 4, 5}, nil)

	h := s.Head(3)
	if h.Len() != 3 {
		t.Fatalf("Head.Len() = %d, want 3", h.Len())
	}
	if h.Int(0) != 1 || h.Int(2) != 3 {
		t.Errorf("Head values wrong")
	}

	ta := s.Tail(2)
	if ta.Len() != 2 {
		t.Fatalf("Tail.Len() = %d, want 2", ta.Len())
	}
	if ta.Int(0) != 4 || ta.Int(1) != 5 {
		t.Errorf("Tail values: got %d,%d want 4,5", ta.Int(0), ta.Int(1))
	}
}

func TestSeriesSlice(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20, 30, 40, 50}, nil)
	sub := s.Slice(1, 4)
	if sub.Len() != 3 {
		t.Fatalf("Slice.Len() = %d, want 3", sub.Len())
	}
	if sub.Int(0) != 20 || sub.Int(2) != 40 {
		t.Errorf("Slice values wrong")
	}
}

func TestSeriesFilter(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 3, 4, 5}, nil)
	mask := []bool{false, true, false, true, false}
	f := s.Filter(mask)
	if f.Len() != 2 {
		t.Fatalf("Filter.Len() = %d, want 2", f.Len())
	}
	if f.Int(0) != 2 || f.Int(1) != 4 {
		t.Errorf("Filter values: got %d,%d want 2,4", f.Int(0), f.Int(1))
	}
}

func TestSeriesTake(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20, 30, 40, 50}, nil)
	r := s.Take([]int{0, 3, 4})
	if r.Len() != 3 {
		t.Fatalf("Take.Len() = %d, want 3", r.Len())
	}
	if r.Int(0) != 10 || r.Int(1) != 40 || r.Int(2) != 50 {
		t.Errorf("Take values: got %d,%d,%d", r.Int(0), r.Int(1), r.Int(2))
	}
}

func TestSeriesCopy(t *testing.T) {
	s := NewInt64Series("x", []int64{1, 2, 3}, nil)
	cp := s.Copy()
	if cp.Len() != s.Len() || cp.Int(0) != s.Int(0) {
		t.Error("Copy mismatch")
	}
}

func TestSeriesSetName(t *testing.T) {
	s := NewInt64Series("old", []int64{1}, nil)
	r := s.SetName("new")
	if r.Name() != "new" {
		t.Errorf("Name() = %q, want %q", r.Name(), "new")
	}
}

func TestSeriesToSlice(t *testing.T) {
	s := NewInt64Series("x", []int64{10, 20}, nil)
	sl := s.ToSlice()
	if len(sl) != 2 {
		t.Fatalf("len = %d, want 2", len(sl))
	}
	if sl[0] != int64(10) || sl[1] != int64(20) {
		t.Errorf("ToSlice = %v", sl)
	}
}

func TestSeriesCustomIndex(t *testing.T) {
	idx := core.NewStringIndex([]string{"a", "b", "c"})
	s := NewInt64Series("x", []int64{10, 20, 30}, idx)
	if s.Index().Get(0) != "a" {
		t.Errorf("Index.Get(0) = %q, want %q", s.Index().Get(0), "a")
	}
}

func TestSeriesBuilderAppend(t *testing.T) {
	b := NewSeriesBuilder("built", core.INT64, nil)
	b.AppendInt(1)
	b.AppendInt(2)
	b.AppendNull()
	b.AppendInt(4)
	s := b.Build()
	if s.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", s.Len())
	}
	if s.NullCount() != 1 {
		t.Errorf("NullCount() = %d, want 1", s.NullCount())
	}
	if s.Int(0) != 1 || s.Int(3) != 4 {
		t.Errorf("values wrong")
	}
}

func TestDTypeConversion(t *testing.T) {
	tests := []struct {
		core core.DType
	}{
		{core.BOOL}, {core.INT8}, {core.INT16}, {core.INT32}, {core.INT64},
		{core.UINT8}, {core.UINT16}, {core.UINT32}, {core.UINT64},
		{core.FLOAT32}, {core.FLOAT64}, {core.STRING},
	}
	for _, tt := range tests {
		arrowDt := DTypeToArrow(tt.core)
		got := ArrowToDType(arrowDt)
		if got != tt.core {
			t.Errorf("roundtrip %v: got %v", tt.core, got)
		}
	}
}
