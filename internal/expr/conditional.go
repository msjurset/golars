package expr

import (
	"fmt"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// WhenBuilder holds the condition for a When/Then/Otherwise chain.
type WhenBuilder struct {
	condition Expr
}

// ThenBuilder holds the condition and the "then" value.
type ThenBuilder struct {
	condition Expr
	thenVal   Expr
}

// When starts a conditional expression chain.
func When(condition Expr) *WhenBuilder {
	return &WhenBuilder{condition: condition}
}

// Then sets the value to use when the condition is true.
func (w *WhenBuilder) Then(value Expr) *ThenBuilder {
	return &ThenBuilder{condition: w.condition, thenVal: value}
}

// Otherwise sets the value to use when the condition is false and returns
// the final expression.
func (t *ThenBuilder) Otherwise(value Expr) Expr {
	e := &whenExpr{
		condition:    t.condition,
		thenVal:      t.thenVal,
		otherwiseVal: value,
	}
	e.exprBase.self = e
	return e
}

// Alias is a convenience that wraps the ThenBuilder result (with nil Otherwise)
// in an alias.
func (t *ThenBuilder) Alias(name string) Expr {
	e := &whenExpr{
		condition:    t.condition,
		thenVal:      t.thenVal,
		otherwiseVal: nil,
	}
	e.exprBase.self = e
	return e.exprBase.Alias(name)
}

// whenExpr implements the When/Then/Otherwise conditional logic.
type whenExpr struct {
	exprBase
	condition    Expr
	thenVal      Expr
	otherwiseVal Expr
}

func (w *whenExpr) Evaluate(ctx *Context) (*series.Series, error) {
	cond, err := w.condition.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if cond.DataType() != dtype.Boolean {
		return nil, fmt.Errorf("golars: When condition must be boolean, got %s", cond.DataType())
	}
	thenS, err := w.thenVal.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	var otherS *series.Series
	if w.otherwiseVal != nil {
		otherS, err = w.otherwiseVal.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
	}

	ba := cond.BooleanArray()
	if ba == nil {
		return nil, fmt.Errorf("golars: expected boolean array for When condition")
	}
	n := ba.Len()

	// If both sides are the same type, pick element-wise
	if otherS != nil && thenS.DataType() != otherS.DataType() {
		// Try promoting to float64 for mixed numeric
		if dtype.IsNumeric(thenS.DataType()) && dtype.IsNumeric(otherS.DataType()) {
			thenS, err = promoteToFloat64(thenS)
			if err != nil {
				return nil, err
			}
			otherS, err = promoteToFloat64(otherS)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("golars: When/Then/Otherwise type mismatch: %s vs %s", thenS.DataType(), otherS.DataType())
		}
	}

	resultName := thenS.Name()
	dt := thenS.DataType()

	switch dt {
	case dtype.Int64:
		return whenInt64(ba, thenS, otherS, n, resultName)
	case dtype.Float64:
		return whenFloat64(ba, thenS, otherS, n, resultName)
	case dtype.String:
		return whenString(ba, thenS, otherS, n, resultName)
	case dtype.Boolean:
		return whenBool(ba, thenS, otherS, n, resultName)
	default:
		return nil, fmt.Errorf("golars: When/Then/Otherwise not supported for type %s", dt)
	}
}

func (w *whenExpr) String() string {
	if w.otherwiseVal != nil {
		return fmt.Sprintf("when(%s).then(%s).otherwise(%s)", w.condition.String(), w.thenVal.String(), w.otherwiseVal.String())
	}
	return fmt.Sprintf("when(%s).then(%s)", w.condition.String(), w.thenVal.String())
}

func whenInt64(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	result := make([]int64, n)
	validity := bitmap.New(n)
	for i := 0; i < n; i++ {
		if ba.IsNull(i) {
			validity.Clear(i)
			continue
		}
		if ba.Value(i) {
			v, ok := thenS.GetInt64(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else if otherS != nil {
			v, ok := otherS.GetInt64(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else {
			validity.Clear(i)
		}
	}
	if validity.AllSet() {
		return series.NewInt64(name, result), nil
	}
	return series.New(name, array.NewInt64Array(result, validity)), nil
}

func whenFloat64(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	result := make([]float64, n)
	validity := bitmap.New(n)
	for i := 0; i < n; i++ {
		if ba.IsNull(i) {
			validity.Clear(i)
			continue
		}
		if ba.Value(i) {
			v, ok := thenS.GetFloat64(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else if otherS != nil {
			v, ok := otherS.GetFloat64(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else {
			validity.Clear(i)
		}
	}
	if validity.AllSet() {
		return series.NewFloat64(name, result), nil
	}
	return series.New(name, array.NewFloat64Array(result, validity)), nil
}

func whenString(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	result := make([]string, n)
	validity := bitmap.New(n)
	for i := 0; i < n; i++ {
		if ba.IsNull(i) {
			validity.Clear(i)
			continue
		}
		if ba.Value(i) {
			v, ok := thenS.GetString(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else if otherS != nil {
			v, ok := otherS.GetString(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else {
			validity.Clear(i)
		}
	}
	if validity.AllSet() {
		return series.NewString(name, result), nil
	}
	return series.New(name, array.NewStringArray(result, validity)), nil
}

func whenBool(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	result := make([]bool, n)
	validity := bitmap.New(n)
	for i := 0; i < n; i++ {
		if ba.IsNull(i) {
			validity.Clear(i)
			continue
		}
		if ba.Value(i) {
			v, ok := thenS.GetBool(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else if otherS != nil {
			v, ok := otherS.GetBool(i)
			if !ok {
				validity.Clear(i)
			} else {
				result[i] = v
			}
		} else {
			validity.Clear(i)
		}
	}
	if validity.AllSet() {
		return series.NewBoolean(name, result), nil
	}
	return series.New(name, array.NewBooleanArray(result, validity)), nil
}
