package golars

import "github.com/msjurseth/golars/internal/expr"

// Expr represents a composable expression that can be evaluated against a DataFrame.
type Expr = expr.Expr

// ExprContext holds the evaluation context for expressions.
type ExprContext = expr.Context

// WhenBuilder is the intermediate builder for When/Then/Otherwise chains.
type WhenBuilder = expr.WhenBuilder

// ThenBuilder is the intermediate builder after When().Then().
type ThenBuilder = expr.ThenBuilder

// StrNamespace provides string operations on expressions.
type StrNamespace = expr.StrNamespace

// Col creates a column reference expression.
func Col(name string) Expr { return expr.Col(name) }

// Lit creates a literal value expression. Supports int, int64, float64, string, bool.
func Lit(value any) Expr { return expr.Lit(value) }

// Cols creates multiple column reference expressions.
func Cols(names ...string) []Expr { return expr.Cols(names...) }

// AllCols returns an expression representing all columns.
func AllCols() Expr { return expr.AllCols() }

// When starts a conditional expression chain.
func When(condition Expr) *WhenBuilder { return expr.When(condition) }
