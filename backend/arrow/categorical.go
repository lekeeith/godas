package arrow

import (
	"fmt"
	"sort"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// CategoricalSeries represents a Series with categorical (dictionary-encoded) values.
// Categories are stored once, and values are stored as integer codes.
type CategoricalSeries struct {
	name     string
	codes    []int32        // index into categories (-1 for null)
	cats     []string       // unique category values
	ordered  bool           // whether categories have a natural order
	catMap   map[string]int32 // category value -> code
	nullMask []bool         // true = null
	index    core.Index
}

// NewCategoricalSeries creates a CategoricalSeries from string values.
func NewCategoricalSeries(name string, values []string, ordered bool) *CategoricalSeries {
	catMap := make(map[string]int32)
	cats := make([]string, 0)
	codes := make([]int32, len(values))
	nullMask := make([]bool, len(values))

	// Collect unique values
	uniqueSet := make(map[string]bool)
	for _, v := range values {
		if v != "" {
			uniqueSet[v] = true
		}
	}

	// Build category list (sorted if ordered)
	if ordered {
		for v := range uniqueSet {
			cats = append(cats, v)
		}
		sort.Strings(cats)
	} else {
		// Preserve insertion order
		seen := make(map[string]bool)
		for _, v := range values {
			if v != "" && !seen[v] {
				seen[v] = true
				cats = append(cats, v)
			}
		}
	}

	for i, c := range cats {
		catMap[c] = int32(i)
	}

	for i, v := range values {
		if v == "" {
			codes[i] = -1
			nullMask[i] = true
			continue
		}
		codes[i] = catMap[v]
	}

	return &CategoricalSeries{
		name:     name,
		codes:    codes,
		cats:     cats,
		ordered:  ordered,
		catMap:   catMap,
		nullMask: nullMask,
	}
}

func (cs *CategoricalSeries) Name() string    { return cs.name }
func (cs *CategoricalSeries) Len() int        { return len(cs.codes) }
func (cs *CategoricalSeries) NullCount() int  {
	n := 0
	for _, nm := range cs.nullMask {
		if nm {
			n++
		}
	}
	return n
}
func (cs *CategoricalSeries) IsNull(i int) bool  { return cs.nullMask[i] }
func (cs *CategoricalSeries) NotNull(i int) bool  { return !cs.nullMask[i] }
func (cs *CategoricalSeries) Dtype() core.DType   { return core.STRING }
func (cs *CategoricalSeries) Index() core.Index    { return cs.index }
func (cs *CategoricalSeries) Ordered() bool        { return cs.ordered }

// String returns the string value at position i.
func (cs *CategoricalSeries) String(i int) string {
	if cs.IsNull(i) {
		return ""
	}
	return cs.cats[cs.codes[i]]
}

// Code returns the integer code at position i.
func (cs *CategoricalSeries) Code(i int) int32 {
	return cs.codes[i]
}

// Categories returns the unique category values.
func (cs *CategoricalSeries) Categories() []string {
	result := make([]string, len(cs.cats))
	copy(result, cs.cats)
	return result
}

// NCategories returns the number of unique categories.
func (cs *CategoricalSeries) NCategories() int {
	return len(cs.cats)
}

// CatAccessor provides categorical operations.
type CatAccessor struct {
	cs *CategoricalSeries
}

// Cat returns a CatAccessor for categorical operations.
func (cs *CategoricalSeries) Cat() *CatAccessor {
	return &CatAccessor{cs: cs}
}

// Codes returns the integer codes as a Series.
func (ca *CatAccessor) Codes() *ArrowSeries {
	alloc := allocNew()
	bldr := array.NewInt32Builder(alloc)
	bldr.Resize(ca.cs.Len())
	for i := 0; i < ca.cs.Len(); i++ {
		if ca.cs.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(ca.cs.codes[i])
		}
	}
	return NewArrowSeries(ca.cs.name+"_codes", bldr.NewArray(), ca.cs.index)
}

// Categories returns the category values as a Series.
func (ca *CatAccessor) Categories() *ArrowSeries {
	alloc := allocNew()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(len(ca.cs.cats))
	for _, c := range ca.cs.cats {
		bldr.Append(c)
	}
	return NewArrowSeries(ca.cs.name+"_categories", bldr.NewArray(), nil)
}

// Reorder reorders the categories (for ordered categoricals).
func (ca *CatAccessor) Reorder(newOrder []string) *CategoricalSeries {
	newCats := make([]string, len(newOrder))
	copy(newCats, newOrder)
	newCatMap := make(map[string]int32, len(newCats))
	for i, c := range newCats {
		newCatMap[c] = int32(i)
	}

	newCodes := make([]int32, ca.cs.Len())
	for i := 0; i < ca.cs.Len(); i++ {
		if ca.cs.IsNull(i) {
			newCodes[i] = -1
		} else {
			oldCat := ca.cs.cats[ca.cs.codes[i]]
			if code, ok := newCatMap[oldCat]; ok {
				newCodes[i] = code
			} else {
				newCodes[i] = -1
			}
		}
	}

	return &CategoricalSeries{
		name:     ca.cs.name,
		codes:    newCodes,
		cats:     newCats,
		ordered:  ca.cs.ordered,
		catMap:   newCatMap,
		nullMask: ca.cs.nullMask,
		index:    ca.cs.index,
	}
}

// AddCategories adds new categories.
func (ca *CatAccessor) AddCategories(newCats []string) *CategoricalSeries {
	cats := make([]string, len(ca.cs.cats))
	copy(cats, ca.cs.cats)
	catMap := make(map[string]int32, len(ca.cs.catMap))
	for k, v := range ca.cs.catMap {
		catMap[k] = v
	}
	for _, c := range newCats {
		if _, exists := catMap[c]; !exists {
			catMap[c] = int32(len(cats))
			cats = append(cats, c)
		}
	}
	return &CategoricalSeries{
		name:     ca.cs.name,
		codes:    ca.cs.codes,
		cats:     cats,
		ordered:  ca.cs.ordered,
		catMap:   catMap,
		nullMask: ca.cs.nullMask,
		index:    ca.cs.index,
	}
}

// RemoveCategories removes categories and sets affected values to null.
func (ca *CatAccessor) RemoveCategories(removeCats []string) *CategoricalSeries {
	removeSet := make(map[string]bool, len(removeCats))
	for _, c := range removeCats {
		removeSet[c] = true
	}

	// Build new category list
	newCats := make([]string, 0)
	newCatMap := make(map[string]int32)
	for _, c := range ca.cs.cats {
		if !removeSet[c] {
			newCatMap[c] = int32(len(newCats))
			newCats = append(newCats, c)
		}
	}

	// Remap codes
	newCodes := make([]int32, ca.cs.Len())
	newMask := make([]bool, ca.cs.Len())
	for i := 0; i < ca.cs.Len(); i++ {
		if ca.cs.IsNull(i) {
			newCodes[i] = -1
			newMask[i] = true
			continue
		}
		oldCat := ca.cs.cats[ca.cs.codes[i]]
		if removeSet[oldCat] {
			newCodes[i] = -1
			newMask[i] = true
		} else {
			newCodes[i] = newCatMap[oldCat]
			newMask[i] = false
		}
	}

	return &CategoricalSeries{
		name:     ca.cs.name,
		codes:    newCodes,
		cats:     newCats,
		ordered:  ca.cs.ordered,
		catMap:   newCatMap,
		nullMask: newMask,
		index:    ca.cs.index,
	}
}

// ValueCounts counts occurrences of each category.
func (ca *CatAccessor) ValueCounts() *ArrowSeries {
	counts := make([]int64, len(ca.cs.cats))
	for i := 0; i < ca.cs.Len(); i++ {
		if ca.cs.NotNull(i) {
			counts[ca.cs.codes[i]]++
		}
	}
	alloc := allocNew()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(len(counts))
	for _, c := range counts {
		bldr.Append(c)
	}
	return NewArrowSeries(ca.cs.name+"_counts", bldr.NewArray(), ca.cs.index)
}

// IsIn checks if each value is in the given set.
func (ca *CatAccessor) IsIn(values []string) *ArrowSeries {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	alloc := allocNew()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(ca.cs.Len())
	for i := 0; i < ca.cs.Len(); i++ {
		if ca.cs.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(set[ca.cs.cats[ca.cs.codes[i]]])
		}
	}
	return NewArrowSeries(ca.cs.name+"_in", bldr.NewArray(), ca.cs.index)
}

// String representation
func (cs *CategoricalSeries) StringFmt() string {
	s := fmt.Sprintf("CategoricalSeries[%s] (ordered=%v) len=%d, categories=%d\n",
		cs.name, cs.ordered, cs.Len(), cs.NCategories())
	for i := 0; i < cs.Len(); i++ {
		label := fmt.Sprintf("%d", i)
		if cs.index != nil {
			label = cs.index.Get(i)
		}
		if cs.IsNull(i) {
			s += fmt.Sprintf("  %s: null\n", label)
		} else {
			s += fmt.Sprintf("  %s: %s (code=%d)\n", label, cs.String(i), cs.codes[i])
		}
	}
	return s
}

// Sort sorts the series and returns sorted indices.
func (cs *CategoricalSeries) Sort(ascending bool) []int {
	indices := make([]int, cs.Len())
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		if cs.IsNull(indices[a]) {
			return true
		}
		if cs.IsNull(indices[b]) {
			return false
		}
		ca, cb := cs.codes[indices[a]], cs.codes[indices[b]]
		if ascending {
			return ca < cb
		}
		return ca > cb
	})
	return indices
}

func allocNew() memory.Allocator {
	a := memory.NewGoAllocator()
	return a
}
