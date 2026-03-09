package expr

import (
	"fmt"
	"strings"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// windowExpr evaluates an expression per partition and broadcasts the result
// back to the original row order, implementing SQL-style window functions.
type windowExpr struct {
	exprBase
	inner       Expr
	partitionBy []string
}

func (w *windowExpr) Evaluate(ctx *Context) (*series.Series, error) {
	if ctx == nil || ctx.DF == nil {
		return nil, fmt.Errorf("golars: window expression requires a DataFrame context")
	}

	df := ctx.DF
	n := df.Height()

	if len(w.partitionBy) == 0 {
		return nil, fmt.Errorf("golars: window Over() requires at least one partition column")
	}

	// Get partition key columns.
	keyCols := make([]*series.Series, len(w.partitionBy))
	for i, name := range w.partitionBy {
		col, err := df.Column(name)
		if err != nil {
			return nil, fmt.Errorf("golars: window Over: column %q not found", name)
		}
		keyCols[i] = col
	}

	// Build groups: hash -> row indices, preserving insertion order.
	groups := make(map[string][]int)
	var orderedHashes []string
	for i := 0; i < n; i++ {
		h := windowHashRow(keyCols, i)
		if _, exists := groups[h]; !exists {
			orderedHashes = append(orderedHashes, h)
		}
		groups[h] = append(groups[h], i)
	}

	// Determine the result series name from a dummy evaluation is not practical;
	// we will get it from the first group evaluation.
	var resultName string
	var resultDtype dtype.DataType
	resultFloat64 := make([]float64, n)
	resultInt64 := make([]int64, n)
	resultString := make([]string, n)
	resultBool := make([]bool, n)
	resultValid := make([]bool, n)
	for i := range resultValid {
		resultValid[i] = true
	}

	first := true
	for _, hash := range orderedHashes {
		indices := groups[hash]

		// Build sub-DataFrame from selected rows.
		subDF, err := buildSubDataFrame(df, indices)
		if err != nil {
			return nil, fmt.Errorf("golars: window Over: %w", err)
		}

		subCtx := &Context{DF: subDF}
		result, err := w.inner.Evaluate(subCtx)
		if err != nil {
			return nil, err
		}

		if first {
			resultName = result.Name()
			resultDtype = result.DataType()
			first = false
		}

		isAgg := result.Len() == 1
		groupSize := len(indices)

		if !isAgg && result.Len() != groupSize {
			return nil, fmt.Errorf("golars: window Over: inner expression returned %d rows for group of %d rows", result.Len(), groupSize)
		}

		for j, origIdx := range indices {
			srcIdx := j
			if isAgg {
				srcIdx = 0
			}

			if result.IsNull(srcIdx) {
				resultValid[origIdx] = false
				continue
			}

			switch resultDtype {
			case dtype.Float64:
				v, _ := result.GetFloat64(srcIdx)
				resultFloat64[origIdx] = v
			case dtype.Int64:
				v, _ := result.GetInt64(srcIdx)
				resultInt64[origIdx] = v
			case dtype.String:
				v, _ := result.GetString(srcIdx)
				resultString[origIdx] = v
			case dtype.Boolean:
				v, _ := result.GetBool(srcIdx)
				resultBool[origIdx] = v
			default:
				return nil, fmt.Errorf("golars: window Over: unsupported result type %s", resultDtype)
			}
		}
	}

	if first {
		// No groups at all (empty DataFrame).
		return series.NewFloat64(resultName, nil), nil
	}

	hasNulls := false
	for _, v := range resultValid {
		if !v {
			hasNulls = true
			break
		}
	}

	switch resultDtype {
	case dtype.Float64:
		if hasNulls {
			return series.NewFloat64WithValidity(resultName, resultFloat64, resultValid), nil
		}
		return series.NewFloat64(resultName, resultFloat64), nil
	case dtype.Int64:
		if hasNulls {
			return series.NewInt64WithValidity(resultName, resultInt64, resultValid), nil
		}
		return series.NewInt64(resultName, resultInt64), nil
	case dtype.String:
		if hasNulls {
			return series.NewStringWithValidity(resultName, resultString, resultValid), nil
		}
		return series.NewString(resultName, resultString), nil
	case dtype.Boolean:
		if hasNulls {
			return series.NewBooleanWithValidity(resultName, resultBool, resultValid), nil
		}
		return series.NewBoolean(resultName, resultBool), nil
	default:
		return nil, fmt.Errorf("golars: window Over: unsupported result type %s", resultDtype)
	}
}

func (w *windowExpr) String() string {
	return fmt.Sprintf("%s.over(%s)", w.inner.String(), strings.Join(w.partitionBy, ", "))
}

// windowHashRow builds a string key from column values at a given row index.
func windowHashRow(cols []*series.Series, i int) string {
	if len(cols) == 1 {
		return fmt.Sprintf("%v", windowGetAny(cols[0], i))
	}
	var b strings.Builder
	for j, col := range cols {
		if j > 0 {
			b.WriteByte(0)
		}
		fmt.Fprintf(&b, "%v", windowGetAny(col, i))
	}
	return b.String()
}

// windowGetAny returns the value at index i as any.
func windowGetAny(s *series.Series, i int) any {
	if s.IsNull(i) {
		return nil
	}
	switch s.DataType() {
	case dtype.Int64:
		v, _ := s.GetInt64(i)
		return v
	case dtype.Float64:
		v, _ := s.GetFloat64(i)
		return v
	case dtype.String:
		v, _ := s.GetString(i)
		return v
	case dtype.Boolean:
		v, _ := s.GetBool(i)
		return v
	default:
		return nil
	}
}

// buildSubDataFrame creates a new DataFrame from selected row indices.
func buildSubDataFrame(df *dataframe.DataFrame, indices []int) (*dataframe.DataFrame, error) {
	cols := df.Columns()
	subCols := make([]*series.Series, len(cols))
	for i, col := range cols {
		subCols[i] = col.Take(indices)
	}
	return dataframe.New(subCols...)
}
