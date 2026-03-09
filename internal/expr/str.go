package expr

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// strMethodExpr delegates to a Series StrAccessor method.
type strMethodExpr struct {
	exprBase
	inner  Expr
	method func(sa *series.StrAccessor) *series.Series
	desc   string
}

func (e *strMethodExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() != dtype.String {
		return nil, fmt.Errorf("golars: str operation requires String series, got %s", s.DataType())
	}
	sa := s.Str()
	if sa == nil {
		return nil, fmt.Errorf("golars: str accessor unavailable")
	}
	return e.method(sa), nil
}

func (e *strMethodExpr) String() string {
	return fmt.Sprintf("%s.str.%s", e.inner.String(), e.desc)
}

// StrNamespace provides string operations on expressions.
type StrNamespace struct {
	inner Expr
}

// Contains returns a boolean expression indicating whether each string
// contains the given substring.
func (s *StrNamespace) Contains(pattern string) Expr {
	e := &strContainsExpr{pattern: pattern, inner: s.inner}
	e.exprBase.self = e
	return e
}

// ToUpper returns an expression with all strings converted to uppercase.
func (s *StrNamespace) ToUpper() Expr {
	e := &strTransformExpr{inner: s.inner, op: strUpper}
	e.exprBase.self = e
	return e
}

// ToLower returns an expression with all strings converted to lowercase.
func (s *StrNamespace) ToLower() Expr {
	e := &strTransformExpr{inner: s.inner, op: strLower}
	e.exprBase.self = e
	return e
}

// Lengths returns an Int64 expression with the length of each string.
func (s *StrNamespace) Lengths() Expr {
	e := &strLenExpr{inner: s.inner}
	e.exprBase.self = e
	return e
}

// StartsWith returns a boolean expression indicating whether each string
// starts with the given prefix.
func (s *StrNamespace) StartsWith(prefix string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.StartsWith(prefix) },
		desc:   fmt.Sprintf("starts_with(%q)", prefix),
	}
	e.exprBase.self = e
	return e
}

// EndsWith returns a boolean expression indicating whether each string
// ends with the given suffix.
func (s *StrNamespace) EndsWith(suffix string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.EndsWith(suffix) },
		desc:   fmt.Sprintf("ends_with(%q)", suffix),
	}
	e.exprBase.self = e
	return e
}

// Replace returns an expression with all occurrences of old replaced with new.
func (s *StrNamespace) Replace(old, new string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Replace(old, new) },
		desc:   fmt.Sprintf("replace(%q, %q)", old, new),
	}
	e.exprBase.self = e
	return e
}

// Trim returns an expression with leading and trailing whitespace removed.
func (s *StrNamespace) Trim() Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Trim() },
		desc:   "trim()",
	}
	e.exprBase.self = e
	return e
}

// Split splits each string by the separator and returns the nth element.
func (s *StrNamespace) Split(sep string, index int) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Split(sep, index) },
		desc:   fmt.Sprintf("split(%q, %d)", sep, index),
	}
	e.exprBase.self = e
	return e
}

// Slice extracts a substring from each string using start and length.
func (s *StrNamespace) Slice(start, length int) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Slice(start, length) },
		desc:   fmt.Sprintf("slice(%d, %d)", start, length),
	}
	e.exprBase.self = e
	return e
}

// Pad pads each string to the given width with fillChar.
func (s *StrNamespace) Pad(width int, side string, fillChar rune) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Pad(width, side, fillChar) },
		desc:   fmt.Sprintf("pad(%d, %q, %q)", width, side, fillChar),
	}
	e.exprBase.self = e
	return e
}

// Strip removes the given characters from both ends of each string.
func (s *StrNamespace) Strip(chars string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Strip(chars) },
		desc:   fmt.Sprintf("strip(%q)", chars),
	}
	e.exprBase.self = e
	return e
}

// Extract extracts the first regex match, returning the given capture group.
func (s *StrNamespace) Extract(pattern string, groupIndex int) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Extract(pattern, groupIndex) },
		desc:   fmt.Sprintf("extract(%q, %d)", pattern, groupIndex),
	}
	e.exprBase.self = e
	return e
}

// CountMatches counts non-overlapping regex matches in each string.
func (s *StrNamespace) CountMatches(pattern string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.CountMatches(pattern) },
		desc:   fmt.Sprintf("count_matches(%q)", pattern),
	}
	e.exprBase.self = e
	return e
}

// Capitalize returns an expression with the first character uppercased and rest lowercased.
func (s *StrNamespace) Capitalize() Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.Capitalize() },
		desc:   "capitalize()",
	}
	e.exprBase.self = e
	return e
}

// ZFill pads each string on the left with '0' to the given width.
func (s *StrNamespace) ZFill(width int) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.ZFill(width) },
		desc:   fmt.Sprintf("zfill(%d)", width),
	}
	e.exprBase.self = e
	return e
}

// ToDatetime parses string values into a DateTime Series using the given Go time layout.
func (s *StrNamespace) ToDatetime(layout string) Expr {
	e := &strMethodExpr{
		inner:  s.inner,
		method: func(sa *series.StrAccessor) *series.Series { return sa.ToDatetime(layout) },
		desc:   fmt.Sprintf("to_datetime(%q)", layout),
	}
	e.exprBase.self = e
	return e
}

// strContainsExpr checks if strings contain a substring.
type strContainsExpr struct {
	exprBase
	inner   Expr
	pattern string
}

func (e *strContainsExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() != dtype.String {
		return nil, fmt.Errorf("golars: str.contains requires String series, got %s", s.DataType())
	}
	sa := s.StringArray()
	n := s.Len()
	result := make([]bool, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			continue
		}
		valid[i] = true
		result[i] = strings.Contains(sa.Value(i), e.pattern)
	}
	if s.HasNulls() {
		return series.NewBooleanWithValidity(s.Name(), result, valid), nil
	}
	return series.NewBoolean(s.Name(), result), nil
}

func (e *strContainsExpr) String() string {
	return fmt.Sprintf("%s.str.contains(%q)", e.inner.String(), e.pattern)
}

type strOp int

const (
	strUpper strOp = iota
	strLower
)

// strTransformExpr applies string transformations.
type strTransformExpr struct {
	exprBase
	inner Expr
	op    strOp
}

func (e *strTransformExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() != dtype.String {
		return nil, fmt.Errorf("golars: str operation requires String series, got %s", s.DataType())
	}
	sa := s.StringArray()
	n := s.Len()
	result := make([]string, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			continue
		}
		valid[i] = true
		v := sa.Value(i)
		switch e.op {
		case strUpper:
			result[i] = strings.ToUpper(v)
		case strLower:
			result[i] = strings.ToLower(v)
		}
	}
	if s.HasNulls() {
		return series.NewStringWithValidity(s.Name(), result, valid), nil
	}
	return series.NewString(s.Name(), result), nil
}

func (e *strTransformExpr) String() string {
	switch e.op {
	case strUpper:
		return fmt.Sprintf("%s.str.to_upper()", e.inner.String())
	case strLower:
		return fmt.Sprintf("%s.str.to_lower()", e.inner.String())
	}
	return ""
}

// strLenExpr returns the length of each string.
type strLenExpr struct {
	exprBase
	inner Expr
}

func (e *strLenExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if s.DataType() != dtype.String {
		return nil, fmt.Errorf("golars: str.lengths requires String series, got %s", s.DataType())
	}
	sa := s.StringArray()
	n := s.Len()
	result := make([]int64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if s.IsNull(i) {
			continue
		}
		valid[i] = true
		result[i] = int64(utf8.RuneCountInString(sa.Value(i)))
	}
	if s.HasNulls() {
		return series.NewInt64WithValidity(s.Name(), result, valid), nil
	}
	return series.NewInt64(s.Name(), result), nil
}

func (e *strLenExpr) String() string {
	return fmt.Sprintf("%s.str.lengths()", e.inner.String())
}
