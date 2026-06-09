package arrow

import (
	"testing"
)

func TestNewCategoricalSeries(t *testing.T) {
	cs := NewCategoricalSeries("color", []string{"red", "blue", "red", "green", "blue"}, false)
	if cs.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", cs.Len())
	}
	if cs.NCategories() != 3 {
		t.Fatalf("NCategories() = %d, want 3", cs.NCategories())
	}
	if cs.String(0) != "red" || cs.String(1) != "blue" {
		t.Errorf("values: %s,%s", cs.String(0), cs.String(1))
	}
}

func TestCategoricalCodes(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "a", "c"}, false)
	if cs.Code(0) != cs.Code(2) {
		t.Error("same value should have same code")
	}
	if cs.Code(0) == cs.Code(1) {
		t.Error("different values should have different codes")
	}
}

func TestCategoricalNulls(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "", "b", ""}, false)
	if cs.NullCount() != 2 {
		t.Fatalf("NullCount() = %d, want 2", cs.NullCount())
	}
	if !cs.IsNull(1) || cs.NotNull(0) != true {
		t.Error("null mask wrong")
	}
}

func TestCatCodes(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "a"}, false)
	codes := cs.Cat().Codes()
	if codes.Len() != 3 {
		t.Fatalf("Codes.Len() = %d, want 3", codes.Len())
	}
	if codes.Int(0) != codes.Int(2) {
		t.Error("same value should have same code")
	}
}

func TestCatCategories(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "c"}, false)
	cats := cs.Cat().Categories()
	if cats.Len() != 3 {
		t.Fatalf("Categories.Len() = %d, want 3", cats.Len())
	}
}

func TestCatReorder(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "c"}, true)
	reordered := cs.Cat().Reorder([]string{"c", "b", "a"})
	if reordered.String(0) != "a" {
		t.Errorf("value[0] = %q, want a", reordered.String(0))
	}
	if reordered.Code(0) != 2 { // "a" is now at index 2
		t.Errorf("code[0] = %d, want 2", reordered.Code(0))
	}
}

func TestCatAddCategories(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b"}, false)
	newCs := cs.Cat().AddCategories([]string{"c", "d"})
	if newCs.NCategories() != 4 {
		t.Fatalf("NCategories() = %d, want 4", newCs.NCategories())
	}
}

func TestCatRemoveCategories(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "c", "a"}, false)
	newCs := cs.Cat().RemoveCategories([]string{"b"})
	if newCs.NCategories() != 2 {
		t.Fatalf("NCategories() = %d, want 2", newCs.NCategories())
	}
	if newCs.IsNull(1) != true {
		t.Error("removed value should be null")
	}
	if newCs.String(0) != "a" || newCs.String(3) != "a" {
		t.Error("non-removed values should be preserved")
	}
}

func TestCatValueCounts(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "a", "a", "b"}, false)
	counts := cs.Cat().ValueCounts()
	if counts.Len() != 2 {
		t.Fatalf("ValueCounts.Len() = %d, want 2", counts.Len())
	}
	// a=3, b=2
	if counts.Int(0) != 3 || counts.Int(1) != 2 {
		t.Errorf("counts: %d,%d", counts.Int(0), counts.Int(1))
	}
}

func TestCatIsIn(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b", "c", "a"}, false)
	r := cs.Cat().IsIn([]string{"a", "c"})
	if !r.Bool(0) || r.Bool(1) || !r.Bool(2) || !r.Bool(3) {
		t.Error("IsIn failed")
	}
}

func TestCategoricalSort(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"c", "a", "b"}, true)
	indices := cs.Sort(true)
	if cs.String(indices[0]) != "a" || cs.String(indices[2]) != "c" {
		t.Errorf("Sort asc: %s,%s", cs.String(indices[0]), cs.String(indices[2]))
	}

	indicesDesc := cs.Sort(false)
	if cs.String(indicesDesc[0]) != "c" || cs.String(indicesDesc[2]) != "a" {
		t.Errorf("Sort desc: %s,%s", cs.String(indicesDesc[0]), cs.String(indicesDesc[2]))
	}
}

func TestCategoricalStringFmt(t *testing.T) {
	cs := NewCategoricalSeries("x", []string{"a", "b"}, false)
	s := cs.StringFmt()
	if s == "" {
		t.Error("StringFmt() returned empty")
	}
}

func TestCategoricalOrdered(t *testing.T) {
	ordered := NewCategoricalSeries("x", []string{"low", "med", "high"}, true)
	if ordered.Ordered() != true {
		t.Error("should be ordered")
	}
	unordered := NewCategoricalSeries("x", []string{"a", "b"}, false)
	if unordered.Ordered() != false {
		t.Error("should be unordered")
	}
}
