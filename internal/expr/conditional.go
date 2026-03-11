package expr

import (
	"fmt"
	"math/bits"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
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

// evaluateAsScalar evaluates an expression. If it's a literal, returns a
// length-1 series to avoid broadcasting, enabling scalar fast paths.
func evaluateAsScalar(e Expr, ctx *Context) (*series.Series, error) {
	if lit, ok := e.(*litExpr); ok {
		return broadcastLiteral(lit.value, 1)
	}
	return e.Evaluate(ctx)
}

func (w *whenExpr) Evaluate(ctx *Context) (*series.Series, error) {
	cond, err := w.condition.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if cond.DataType() != dtype.Boolean {
		return nil, fmt.Errorf("golars: When condition must be boolean, got %s", cond.DataType())
	}
	thenS, err := evaluateAsScalar(w.thenVal, ctx)
	if err != nil {
		return nil, err
	}

	var otherS *series.Series
	if w.otherwiseVal != nil {
		otherS, err = evaluateAsScalar(w.otherwiseVal, ctx)
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
		return whenTyped[int64](ba, thenS, otherS, n, resultName, dt)
	case dtype.Float64:
		return whenTyped[float64](ba, thenS, otherS, n, resultName, dt)
	case dtype.Int32, dtype.Date:
		return whenTyped[int32](ba, thenS, otherS, n, resultName, dt)
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

// whenTyped handles When/Then/Otherwise for numeric types using bulk array access.
func whenTyped[T any](ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string, dt dtype.DataType) (*series.Series, error) {
	result := make([]T, n)
	condWords := ba.DataBitmap().Words()
	condHasNulls := ba.Validity() != nil

	// Extract scalar values or array slices for then/otherwise
	thenArr := thenS.Array().(*array.TypedArray[T])
	thenVals := thenArr.Values()
	thenScalar := len(thenVals) == 1
	thenHasNulls := thenArr.Validity() != nil

	var otherVals []T
	var otherScalar bool
	var otherHasNulls bool
	hasOther := otherS != nil
	if hasOther {
		otherArr := otherS.Array().(*array.TypedArray[T])
		otherVals = otherArr.Values()
		otherScalar = len(otherVals) == 1
		otherHasNulls = otherArr.Validity() != nil
	}

	anyNulls := condHasNulls || thenHasNulls || otherHasNulls || !hasOther

	// Fast path: no nulls anywhere, both scalar
	if !anyNulls && thenScalar && otherScalar {
		tv := thenVals[0]
		ov := otherVals[0]
		for wi, w := range condWords {
			base := wi * 64
			remaining := n - base
			if remaining > 64 {
				remaining = 64
			}
			// Set all to otherwise value, then overwrite true positions
			for j := 0; j < remaining; j++ {
				result[base+j] = ov
			}
			for bw := w; bw != 0; {
				j := bits.TrailingZeros64(bw)
				result[base+j] = tv
				bw &= bw - 1
			}
		}
		return series.New(name, array.NewTypedArray(result, dt, nil)), nil
	}

	// Fast path: no nulls anywhere, scalar then + array other (or vice versa)
	if !anyNulls && hasOther {
		for wi, w := range condWords {
			base := wi * 64
			remaining := n - base
			if remaining > 64 {
				remaining = 64
			}
			if thenScalar && !otherScalar {
				tv := thenVals[0]
				copy(result[base:base+remaining], otherVals[base:base+remaining])
				for bw := w; bw != 0; {
					j := bits.TrailingZeros64(bw)
					result[base+j] = tv
					bw &= bw - 1
				}
			} else if !thenScalar && otherScalar {
				ov := otherVals[0]
				copy(result[base:base+remaining], thenVals[base:base+remaining])
				inverted := ^w
				if remaining < 64 {
					inverted &= (1 << remaining) - 1
				}
				for bw := inverted; bw != 0; {
					j := bits.TrailingZeros64(bw)
					result[base+j] = ov
					bw &= bw - 1
				}
			} else {
				// Both arrays, no nulls
				copy(result[base:base+remaining], otherVals[base:base+remaining])
				for bw := w; bw != 0; {
					j := bits.TrailingZeros64(bw)
					result[base+j] = thenVals[base+j]
					bw &= bw - 1
				}
			}
		}
		return series.New(name, array.NewTypedArray(result, dt, nil)), nil
	}

	// General path with null handling
	validity := bitmap.New(n)
	var condValWords []uint64
	if condHasNulls {
		condValWords = ba.Validity().Words()
	}
	var thenValBm *bitmap.Bitmap
	if thenHasNulls {
		thenValBm = thenArr.Validity()
	}
	var otherValBm *bitmap.Bitmap
	if otherHasNulls {
		otherArr := otherS.Array().(*array.TypedArray[T])
		otherValBm = otherArr.Validity()
	}

	for wi, cw := range condWords {
		base := wi * 64
		remaining := n - base
		if remaining > 64 {
			remaining = 64
		}

		validWord := uint64(^uint64(0))
		if remaining < 64 {
			validWord = (1 << remaining) - 1
		}

		// Mask out null condition positions
		condValid := validWord
		if condHasNulls {
			condValid = condValWords[wi]
		}

		trueMask := cw & condValid
		falseMask := (^cw) & condValid
		nullCondMask := validWord &^ condValid

		// Fill from then branch (true positions)
		if trueMask != 0 {
			if thenScalar {
				tv := thenVals[0]
				for b := trueMask; b != 0; {
					j := bits.TrailingZeros64(b)
					result[base+j] = tv
					b &= b - 1
				}
			} else {
				for b := trueMask; b != 0; {
					j := bits.TrailingZeros64(b)
					result[base+j] = thenVals[base+j]
					b &= b - 1
				}
			}
			// Check then nulls
			if thenHasNulls {
				thenInvalid := trueMask &^ thenValBm.Words()[wi]
				validWord &^= thenInvalid
			}
		}

		// Fill from otherwise branch (false positions)
		if falseMask != 0 {
			if hasOther {
				if otherScalar {
					ov := otherVals[0]
					for b := falseMask; b != 0; {
						j := bits.TrailingZeros64(b)
						result[base+j] = ov
						b &= b - 1
					}
				} else {
					for b := falseMask; b != 0; {
						j := bits.TrailingZeros64(b)
						result[base+j] = otherVals[base+j]
						b &= b - 1
					}
				}
				if otherHasNulls {
					otherInvalid := falseMask &^ otherValBm.Words()[wi]
					validWord &^= otherInvalid
				}
			} else {
				// No otherwise: false positions are null
				validWord &^= falseMask
			}
		}

		// Null condition positions are null in result
		validWord &^= nullCondMask

		validity.SetWord(wi, validWord)
	}

	if validity.AllSet() {
		return series.New(name, array.NewTypedArray(result, dt, nil)), nil
	}
	return series.New(name, array.NewTypedArray(result, dt, validity)), nil
}

func whenString(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	result := make([]string, n)
	validity := bitmap.New(n)
	condWords := ba.DataBitmap().Words()
	condHasNulls := ba.Validity() != nil

	thenSA := thenS.StringArray()
	thenScalar := thenSA.Len() == 1

	var otherSA *array.StringArray
	hasOther := otherS != nil
	otherScalar := false
	if hasOther {
		otherSA = otherS.StringArray()
		otherScalar = otherSA.Len() == 1
	}

	var condValWords []uint64
	if condHasNulls {
		condValWords = ba.Validity().Words()
	}

	for wi, cw := range condWords {
		base := wi * 64
		remaining := n - base
		if remaining > 64 {
			remaining = 64
		}

		validWord := uint64(^uint64(0))
		if remaining < 64 {
			validWord = (1 << remaining) - 1
		}

		condValid := validWord
		if condHasNulls {
			condValid = condValWords[wi]
		}

		trueMask := cw & condValid
		falseMask := (^cw) & condValid
		nullCondMask := validWord &^ condValid

		for b := trueMask; b != 0; {
			j := bits.TrailingZeros64(b)
			idx := base + j
			if thenSA.IsValid(idx) || thenSA.Validity() == nil {
				if thenScalar {
					result[idx] = thenSA.Value(0)
				} else {
					result[idx] = thenSA.Value(idx)
				}
			} else {
				validWord &^= 1 << j
			}
			b &= b - 1
		}

		if hasOther {
			for b := falseMask; b != 0; {
				j := bits.TrailingZeros64(b)
				idx := base + j
				if otherSA.IsValid(idx) || otherSA.Validity() == nil {
					if otherScalar {
						result[idx] = otherSA.Value(0)
					} else {
						result[idx] = otherSA.Value(idx)
					}
				} else {
					validWord &^= 1 << j
				}
				b &= b - 1
			}
		} else {
			validWord &^= falseMask
		}

		validWord &^= nullCondMask
		validity.SetWord(wi, validWord)
	}

	if validity.AllSet() {
		return series.NewString(name, result), nil
	}
	return series.New(name, array.NewStringArray(result, validity)), nil
}

func whenBool(ba *array.BooleanArray, thenS, otherS *series.Series, n int, name string) (*series.Series, error) {
	resultBm := bitmap.NewEmpty(n)
	validity := bitmap.New(n)
	condWords := ba.DataBitmap().Words()
	condHasNulls := ba.Validity() != nil

	thenBA := thenS.BooleanArray()
	thenWords := thenBA.DataBitmap().Words()
	thenScalar := thenBA.Len() == 1
	thenHasNulls := thenBA.Validity() != nil

	var otherWords []uint64
	hasOther := otherS != nil
	otherScalar := false
	otherHasNulls := false
	if hasOther {
		otherBA := otherS.BooleanArray()
		otherWords = otherBA.DataBitmap().Words()
		otherScalar = otherBA.Len() == 1
		otherHasNulls = otherBA.Validity() != nil
	}

	anyNulls := condHasNulls || thenHasNulls || otherHasNulls || !hasOther
	resultWords := resultBm.Words()

	var condValWords []uint64
	if condHasNulls {
		condValWords = ba.Validity().Words()
	}

	for wi, cw := range condWords {
		remaining := n - wi*64
		if remaining > 64 {
			remaining = 64
		}

		wordMask := uint64(^uint64(0))
		if remaining < 64 {
			wordMask = (1 << remaining) - 1
		}

		condValid := wordMask
		if condHasNulls {
			condValid = condValWords[wi]
		}

		trueMask := cw & condValid
		falseMask := (^cw) & condValid

		// Get then values for true positions
		var thenW uint64
		if thenScalar {
			if thenWords[0]&1 != 0 {
				thenW = trueMask
			}
		} else if wi < len(thenWords) {
			thenW = thenWords[wi] & trueMask
		}

		// Get other values for false positions
		var otherW uint64
		if hasOther {
			if otherScalar {
				if otherWords[0]&1 != 0 {
					otherW = falseMask
				}
			} else if wi < len(otherWords) {
				otherW = otherWords[wi] & falseMask
			}
		}

		resultWords[wi] = thenW | otherW

		if anyNulls {
			validWord := wordMask
			validWord &^= wordMask &^ condValid // null condition
			if !hasOther {
				validWord &^= falseMask
			}
			if thenHasNulls {
				validWord &^= trueMask &^ thenBA.Validity().Words()[wi]
			}
			if otherHasNulls {
				otherBA := otherS.BooleanArray()
				validWord &^= falseMask &^ otherBA.Validity().Words()[wi]
			}
			validity.SetWord(wi, validWord)
		}
	}

	if !anyNulls || validity.AllSet() {
		return series.New(name, array.NewBooleanArrayFromBitmap(resultBm, nil)), nil
	}
	return series.New(name, array.NewBooleanArrayFromBitmap(resultBm, validity)), nil
}
