package series

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/msjurset/golars/internal/dtype"
)

// StrAccessor provides string operations on a String Series.
type StrAccessor struct {
	s *Series
}

// Str returns a StrAccessor for string operations. Returns nil if the Series
// is not of String type.
func (s *Series) Str() *StrAccessor {
	if s.dtype != dtype.String {
		return nil
	}
	return &StrAccessor{s: s}
}

// Contains returns a Boolean Series indicating whether each string contains substr.
func (a *StrAccessor) Contains(substr string) *Series {
	n := a.s.Len()
	data := make([]bool, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.Contains(v, substr)
			valid[i] = true
		}
	}
	return NewBooleanWithValidity(a.s.name, data, valid)
}

// StartsWith returns a Boolean Series indicating whether each string starts with prefix.
func (a *StrAccessor) StartsWith(prefix string) *Series {
	n := a.s.Len()
	data := make([]bool, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.HasPrefix(v, prefix)
			valid[i] = true
		}
	}
	return NewBooleanWithValidity(a.s.name, data, valid)
}

// EndsWith returns a Boolean Series indicating whether each string ends with suffix.
func (a *StrAccessor) EndsWith(suffix string) *Series {
	n := a.s.Len()
	data := make([]bool, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.HasSuffix(v, suffix)
			valid[i] = true
		}
	}
	return NewBooleanWithValidity(a.s.name, data, valid)
}

// ToUpper returns a new String Series with all characters uppercased.
func (a *StrAccessor) ToUpper() *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.ToUpper(v)
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// ToLower returns a new String Series with all characters lowercased.
func (a *StrAccessor) ToLower() *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.ToLower(v)
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Replace replaces all occurrences of old with new in each string.
func (a *StrAccessor) Replace(old, new string) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.ReplaceAll(v, old, new)
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Trim removes leading and trailing whitespace from each string.
func (a *StrAccessor) Trim() *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.TrimSpace(v)
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Lengths returns an Int64 Series with the length (in runes) of each string.
func (a *StrAccessor) Lengths() *Series {
	n := a.s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = int64(utf8.RuneCountInString(v))
			valid[i] = true
		}
	}
	return NewInt64WithValidity(a.s.name+"_len", data, valid)
}

// Split splits each string by the separator and returns a new String Series
// containing the nth element (0-indexed). If the index is out of range, the
// value is null.
func (a *StrAccessor) Split(sep string, index int) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			parts := strings.Split(v, sep)
			if index >= 0 && index < len(parts) {
				data[i] = parts[index]
				valid[i] = true
			}
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Slice extracts a substring from each string using start and length.
func (a *StrAccessor) Slice(start, length int) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			runes := []rune(v)
			s := start
			if s < 0 {
				s = len(runes) + s
			}
			if s < 0 {
				s = 0
			}
			e := s + length
			if e > len(runes) {
				e = len(runes)
			}
			if s < len(runes) {
				data[i] = string(runes[s:e])
				valid[i] = true
			}
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Pad pads each string to the given width with fillChar.
// side must be "left", "right", or "both".
func (a *StrAccessor) Pad(width int, side string, fillChar rune) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	fill := string(fillChar)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			runeLen := utf8.RuneCountInString(v)
			padLen := width - runeLen
			if padLen <= 0 {
				data[i] = v
			} else {
				padding := strings.Repeat(fill, padLen)
				switch side {
				case "left":
					data[i] = padding + v
				case "right":
					data[i] = v + padding
				case "both":
					left := padLen / 2
					right := padLen - left
					data[i] = strings.Repeat(fill, left) + v + strings.Repeat(fill, right)
				default:
					data[i] = v
				}
			}
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Strip removes the given characters from both ends of each string.
func (a *StrAccessor) Strip(chars string) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = strings.Trim(v, chars)
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Extract extracts the first match of a regex pattern, returning the given group.
// groupIndex 0 returns the full match; 1+ return capture groups.
func (a *StrAccessor) Extract(pattern string, groupIndex int) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return NewStringWithValidity(a.s.name, data, valid)
	}
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			matches := re.FindStringSubmatch(v)
			if groupIndex < len(matches) {
				data[i] = matches[groupIndex]
				valid[i] = true
			}
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// Capitalize returns a new String Series with the first character of each string uppercased
// and the rest lowercased.
func (a *StrAccessor) Capitalize() *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			if len(v) == 0 {
				data[i] = v
			} else {
				r, size := utf8.DecodeRuneInString(v)
				data[i] = string(unicode.ToUpper(r)) + strings.ToLower(v[size:])
			}
			valid[i] = true
		}
	}
	return NewStringWithValidity(a.s.name, data, valid)
}

// ZFill pads each string on the left with '0' to the given width.
func (a *StrAccessor) ZFill(width int) *Series {
	return a.Pad(width, "left", '0')
}

// ToDatetime parses each string into a DateTime Series using the given Go time layout.
// Returns a DateTime Series (microseconds since epoch). Unparseable values become null.
func (a *StrAccessor) ToDatetime(layout string) *Series {
	n := a.s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			t, err := time.Parse(layout, v)
			if err == nil {
				data[i] = t.UnixMicro()
				valid[i] = true
			}
		}
	}
	return NewDateTimeWithValidity(fmt.Sprintf("%s_datetime", a.s.name), data, valid)
}

// CountMatches counts the number of non-overlapping matches of a regex pattern.
func (a *StrAccessor) CountMatches(pattern string) *Series {
	n := a.s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return NewInt64WithValidity(a.s.name, data, valid)
	}
	for i := 0; i < n; i++ {
		if a.s.IsValid(i) {
			v, _ := a.s.GetString(i)
			data[i] = int64(len(re.FindAllString(v, -1)))
			valid[i] = true
		}
	}
	return NewInt64WithValidity(a.s.name, data, valid)
}
