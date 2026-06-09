package arrow

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/godans/godans/core"
)

// StringAccessor provides string operations on a Series.
type StringAccessor struct {
	s *ArrowSeries
}

// Str returns a StringAccessor for string series.
func (s *ArrowSeries) Str() *StringAccessor {
	return &StringAccessor{s: s}
}

func (sa *StringAccessor) mapString(fn func(string) string) core.Series {
	s := sa.s
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.String(i)))
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

func (sa *StringAccessor) mapBool(fn func(string) bool) core.Series {
	s := sa.s
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.String(i)))
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

func (sa *StringAccessor) mapInt(fn func(string) int64) core.Series {
	s := sa.s
	alloc := memory.NewGoAllocator()
	bldr := array.NewInt64Builder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.AppendNull()
		} else {
			bldr.Append(fn(s.String(i)))
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}

// Upper converts to uppercase.
func (sa *StringAccessor) Upper() core.Series {
	return sa.mapString(strings.ToUpper)
}

// Lower converts to lowercase.
func (sa *StringAccessor) Lower() core.Series {
	return sa.mapString(strings.ToLower)
}

// Strip removes leading/trailing whitespace.
func (sa *StringAccessor) Strip() core.Series {
	return sa.mapString(strings.TrimSpace)
}

// LStrip removes leading whitespace.
func (sa *StringAccessor) LStrip() core.Series {
	return sa.mapString(func(s string) string { return strings.TrimLeft(s, " \t\n\r") })
}

// RStrip removes trailing whitespace.
func (sa *StringAccessor) RStrip() core.Series {
	return sa.mapString(func(s string) string { return strings.TrimRight(s, " \t\n\r") })
}

// Len returns the length of each string.
func (sa *StringAccessor) Len() core.Series {
	return sa.mapInt(func(s string) int64 { return int64(len(s)) })
}

// Contains checks if pattern is contained in each string.
func (sa *StringAccessor) Contains(pattern string) core.Series {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Fallback to plain string search
		return sa.mapBool(func(s string) bool {
			return strings.Contains(s, pattern)
		})
	}
	return sa.mapBool(func(s string) bool {
		return re.MatchString(s)
	})
}

// StartsWith checks if each string starts with prefix.
func (sa *StringAccessor) StartsWith(prefix string) core.Series {
	return sa.mapBool(func(s string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

// EndsWith checks if each string ends with suffix.
func (sa *StringAccessor) EndsWith(suffix string) core.Series {
	return sa.mapBool(func(s string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

// Replace replaces occurrences of old with new.
func (sa *StringAccessor) Replace(old, new string) core.Series {
	return sa.mapString(func(s string) string {
		return strings.ReplaceAll(s, old, new)
	})
}

// ReplaceRegex replaces regex pattern with replacement.
func (sa *StringAccessor) ReplaceRegex(pattern, repl string) core.Series {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return sa.Replace(pattern, repl)
	}
	return sa.mapString(func(s string) string {
		return re.ReplaceAllString(s, repl)
	})
}

// Split splits each string by separator and returns the nth element (0-based).
func (sa *StringAccessor) Split(sep string, n int) core.Series {
	return sa.mapString(func(s string) string {
		parts := strings.Split(s, sep)
		if n < 0 || n >= len(parts) {
			return ""
		}
		return parts[n]
	})
}

// Count counts occurrences of pattern in each string.
func (sa *StringAccessor) Count(pattern string) core.Series {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return sa.mapInt(func(s string) int64 {
			return int64(strings.Count(s, pattern))
		})
	}
	return sa.mapInt(func(s string) int64 {
		return int64(len(re.FindAllString(s, -1)))
	})
}

// Pad pads each string to the given width.
// side: "left", "right".
func (sa *StringAccessor) Pad(width int, padChar string, side string) core.Series {
	return sa.mapString(func(s string) string {
		if len(s) >= width {
			return s
		}
		pad := strings.Repeat(padChar, width-len(s))
		switch side {
		case "left":
			return pad + s
		case "right":
			return s + pad
		default:
			return s
		}
	})
}

// Extract extracts the first match of regex pattern.
func (sa *StringAccessor) Extract(pattern string) core.Series {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return sa.mapString(func(s string) string { return "" })
	}
	return sa.mapString(func(s string) string {
		matches := re.FindStringSubmatch(s)
		if len(matches) > 1 {
			return matches[1]
		}
		if len(matches) > 0 {
			return matches[0]
		}
		return ""
	})
}

// ExtractAll extracts all matches of regex pattern.
func (sa *StringAccessor) ExtractAll(pattern string) core.Series {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return sa.mapString(func(s string) string { return "" })
	}
	return sa.mapString(func(s string) string {
		matches := re.FindAllString(s, -1)
		return strings.Join(matches, ",")
	})
}

// Cat concatenates all strings with the given separator.
func (sa *StringAccessor) Cat(sep string) string {
	s := sa.s
	parts := make([]string, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if s.NotNull(i) {
			parts = append(parts, s.String(i))
		}
	}
	return strings.Join(parts, sep)
}

// Title capitalizes the first letter of each word.
func (sa *StringAccessor) Title() core.Series {
	return sa.mapString(func(s string) string {
		return strings.Title(s) //nolint:staticcheck
	})
}

// SwapCase swaps case of each character.
func (sa *StringAccessor) SwapCase() core.Series {
	return sa.mapString(func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if r >= 'a' && r <= 'z' {
				b.WriteRune(r - 32)
			} else if r >= 'A' && r <= 'Z' {
				b.WriteRune(r + 32)
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	})
}

// IsNumeric checks if each string is numeric.
func (sa *StringAccessor) IsNumeric() core.Series {
	return sa.mapBool(func(s string) bool {
		_, err := strconv.ParseFloat(s, 64)
		return err == nil
	})
}

// IsAlpha checks if each string is alphabetic.
func (sa *StringAccessor) IsAlpha() core.Series {
	return sa.mapBool(func(s string) bool {
		if len(s) == 0 {
			return false
		}
		for _, r := range s {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				return false
			}
		}
		return true
	})
}

// IsDigit checks if each string is digits only.
func (sa *StringAccessor) IsDigit() core.Series {
	return sa.mapBool(func(s string) bool {
		if len(s) == 0 {
			return false
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	})
}

// IsEmpty checks if each string is empty.
func (sa *StringAccessor) IsEmpty() core.Series {
	return sa.mapBool(func(s string) bool { return len(s) == 0 })
}

// ZFill pads with zeros on the left to reach the given width.
func (sa *StringAccessor) ZFill(width int) core.Series {
	return sa.mapString(func(s string) string {
		if len(s) >= width {
			return s
		}
		return strings.Repeat("0", width-len(s)) + s
	})
}
