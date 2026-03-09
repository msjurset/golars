package expr

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

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
