package core

import (
	"testing"
	"time"
)

func TestRangeIndex(t *testing.T) {
	idx := NewRangeIndex(0, 5)
	if idx.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", idx.Len())
	}
	for i := 0; i < 5; i++ {
		want := string(rune('0' + i))
		if got := idx.Get(i); got != want {
			t.Errorf("Get(%d) = %q, want %q", i, got, want)
		}
	}

	// Slice
	sub := idx.Slice(1, 4)
	if sub.Len() != 3 {
		t.Fatalf("Slice.Len() = %d, want 3", sub.Len())
	}
	if sub.Get(0) != "1" {
		t.Errorf("Slice.Get(0) = %q, want %q", sub.Get(0), "1")
	}

	// Copy
	cp := idx.Copy()
	if cp.Len() != idx.Len() {
		t.Errorf("Copy.Len() = %d, want %d", cp.Len(), idx.Len())
	}

	if idx.Type() != INT64 {
		t.Errorf("Type() = %v, want INT64", idx.Type())
	}
}

func TestDefaultIndex(t *testing.T) {
	idx := NewDefaultIndex(3)
	if idx.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", idx.Len())
	}
	if idx.Get(0) != "0" || idx.Get(2) != "2" {
		t.Errorf("unexpected values: %s, %s", idx.Get(0), idx.Get(2))
	}
}

func TestInt64Index(t *testing.T) {
	idx := NewInt64Index([]int64{10, 20, 30})
	if idx.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", idx.Len())
	}
	if idx.Get(1) != "20" {
		t.Errorf("Get(1) = %q, want %q", idx.Get(1), "20")
	}

	sub := idx.Slice(0, 2)
	if sub.Len() != 2 {
		t.Fatalf("Slice.Len() = %d, want 2", sub.Len())
	}

	cp := idx.Copy()
	if cp.Get(2) != "30" {
		t.Errorf("Copy.Get(2) = %q, want %q", cp.Get(2), "30")
	}
}

func TestStringIndex(t *testing.T) {
	idx := NewStringIndex([]string{"a", "b", "c"})
	if idx.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", idx.Len())
	}
	if idx.Get(0) != "a" {
		t.Errorf("Get(0) = %q, want %q", idx.Get(0), "a")
	}
	if idx.Type() != STRING {
		t.Errorf("Type() = %v, want STRING", idx.Type())
	}
}

func TestDateTimeIndex(t *testing.T) {
	ts := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
	}
	idx := NewDateTimeIndex(ts)
	if idx.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", idx.Len())
	}
	if idx.Type() != TIMESTAMP {
		t.Errorf("Type() = %v, want TIMESTAMP", idx.Type())
	}
}
