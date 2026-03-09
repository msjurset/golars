package dataframe

import (
	"fmt"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// Pivot reshapes the DataFrame from long to wide format. The indexCol becomes
// the row labels (first column), columnsCol values become new column names, and
// valuesCol values fill the cells. If multiple values exist for the same
// index+column combination, the first encountered value is used.
func (df *DataFrame) Pivot(indexCol, columnsCol, valuesCol string) (*DataFrame, error) {
	idxSeries, err := df.Column(indexCol)
	if err != nil {
		return nil, fmt.Errorf("golars: pivot: column %q not found", indexCol)
	}
	colSeries, err := df.Column(columnsCol)
	if err != nil {
		return nil, fmt.Errorf("golars: pivot: column %q not found", columnsCol)
	}
	valSeries, err := df.Column(valuesCol)
	if err != nil {
		return nil, fmt.Errorf("golars: pivot: column %q not found", valuesCol)
	}

	// Collect unique index values (preserving order).
	indexOrder := make([]string, 0)
	indexSet := make(map[string]int) // value -> position in indexOrder
	for i := 0; i < df.height; i++ {
		v := anyToString(getAny(idxSeries, i))
		if _, exists := indexSet[v]; !exists {
			indexSet[v] = len(indexOrder)
			indexOrder = append(indexOrder, v)
		}
	}

	// Collect unique column values (preserving order).
	colOrder := make([]string, 0)
	colSet := make(map[string]int)
	for i := 0; i < df.height; i++ {
		v := anyToString(getAny(colSeries, i))
		if _, exists := colSet[v]; !exists {
			colSet[v] = len(colOrder)
			colOrder = append(colOrder, v)
		}
	}

	nRows := len(indexOrder)
	nPivotCols := len(colOrder)
	valDt := valSeries.DataType()

	// Build a grid: [indexPos][colPos] -> (value, hasValue)
	grid := make([][]pivotCell, nRows)
	for i := range grid {
		grid[i] = make([]pivotCell, nPivotCols)
	}

	for i := 0; i < df.height; i++ {
		idxKey := anyToString(getAny(idxSeries, i))
		colKey := anyToString(getAny(colSeries, i))
		r := indexSet[idxKey]
		c := colSet[colKey]
		if !grid[r][c].set {
			grid[r][c] = pivotCell{val: getAny(valSeries, i), set: true}
		}
	}

	// Build the index column based on the original index column type.
	resultCols := make([]*series.Series, 0, 1+nPivotCols)
	resultCols = append(resultCols, buildColumnFromStrings(indexCol, idxSeries.DataType(), indexOrder))

	// Build value columns.
	for ci, colName := range colOrder {
		resultCols = append(resultCols, buildPivotValueColumn(colName, valDt, grid, ci, nRows))
	}

	return New(resultCols...)
}

// Unpivot (also known as Melt) reshapes the DataFrame from wide to long format.
// idCols are kept as identifier columns, while valueCols are melted into two
// new columns: "variable" (the original column name) and "value".
func (df *DataFrame) Unpivot(idCols []string, valueCols []string) (*DataFrame, error) {
	// Validate id columns.
	for _, name := range idCols {
		if !df.schema.Contains(name) {
			return nil, fmt.Errorf("golars: unpivot: column %q not found", name)
		}
	}

	if len(valueCols) == 0 {
		return nil, fmt.Errorf("golars: unpivot: no value columns specified")
	}

	// Validate value columns and determine common type.
	valColSeries := make([]*series.Series, len(valueCols))
	var commonDt dtype.DataType
	for i, name := range valueCols {
		col, err := df.Column(name)
		if err != nil {
			return nil, fmt.Errorf("golars: unpivot: column %q not found", name)
		}
		valColSeries[i] = col
		if i == 0 {
			commonDt = col.DataType()
		} else {
			commonDt = promoteType(commonDt, col.DataType())
		}
	}

	nResult := df.height * len(valueCols)

	// Build id columns (repeated for each value column).
	resultCols := make([]*series.Series, 0, len(idCols)+2)
	for _, idName := range idCols {
		idCol, _ := df.Column(idName)
		resultCols = append(resultCols, repeatColumn(idCol, len(valueCols)))
	}

	// Build variable column.
	varValues := make([]string, nResult)
	idx := 0
	for _, vName := range valueCols {
		for r := 0; r < df.height; r++ {
			varValues[idx] = vName
			idx++
		}
	}
	resultCols = append(resultCols, series.NewString("variable", varValues))

	// Build value column.
	resultCols = append(resultCols, buildMeltedValueColumn("value", commonDt, valColSeries, df.height))

	return New(resultCols...)
}

// Transpose flips rows and columns. All columns must have a numeric or common
// data type. Original column names become a "column" column, and new columns
// are named "column_0", "column_1", etc. Numeric types are promoted to Float64.
func (df *DataFrame) Transpose() (*DataFrame, error) {
	if df.Width() == 0 || df.height == 0 {
		colNames := series.NewString("column", []string{})
		return New(colNames)
	}

	// Determine common type; all must be promotable.
	commonDt := df.columns[0].DataType()
	for i := 1; i < len(df.columns); i++ {
		commonDt = promoteType(commonDt, df.columns[i].DataType())
	}

	// For simplicity, convert all numerics to Float64.
	if commonDt == dtype.Int64 {
		commonDt = dtype.Float64
	}

	// Validate all columns are compatible.
	for _, col := range df.columns {
		if !isPromotableTo(col.DataType(), commonDt) {
			return nil, fmt.Errorf("golars: transpose: cannot promote column %q of type %v to common type %v", col.Name(), col.DataType(), commonDt)
		}
	}

	nOldCols := len(df.columns)
	nOldRows := df.height

	// New shape: nOldCols rows, nOldRows+1 columns (including "column").
	resultCols := make([]*series.Series, 0, nOldRows+1)

	// Column name column.
	names := make([]string, nOldCols)
	for i, col := range df.columns {
		names[i] = col.Name()
	}
	resultCols = append(resultCols, series.NewString("column", names))

	// Data columns.
	for j := 0; j < nOldRows; j++ {
		colName := fmt.Sprintf("column_%d", j)
		resultCols = append(resultCols, buildTransposeColumn(colName, commonDt, df.columns, j))
	}

	return New(resultCols...)
}

// Explode takes a String column, splits each value by comma, and produces a
// new row for each split value. All other columns are repeated accordingly.
func (df *DataFrame) Explode(colName string) (*DataFrame, error) {
	col, err := df.Column(colName)
	if err != nil {
		return nil, fmt.Errorf("golars: explode: column %q not found", colName)
	}
	if col.DataType() != dtype.String {
		return nil, fmt.Errorf("golars: explode: column %q must be String type, got %v", colName, col.DataType())
	}

	// First pass: determine split values and row mapping.
	var entries []explodeEntry
	for i := 0; i < df.height; i++ {
		if col.IsNull(i) {
			entries = append(entries, explodeEntry{origRow: i, isNull: true})
			continue
		}
		v, _ := col.GetString(i)
		parts := splitByComma(v)
		for _, p := range parts {
			entries = append(entries, explodeEntry{origRow: i, value: trimSpaces(p)})
		}
	}

	nResult := len(entries)

	// Build result columns.
	resultCols := make([]*series.Series, 0, len(df.columns))
	for _, srcCol := range df.columns {
		if srcCol.Name() == colName {
			// Build the exploded string column.
			vals := make([]string, nResult)
			validity := make([]bool, nResult)
			for k, e := range entries {
				if e.isNull {
					vals[k] = ""
					validity[k] = false
				} else {
					vals[k] = e.value
					validity[k] = true
				}
			}
			resultCols = append(resultCols, series.NewStringWithValidity(colName, vals, validity))
		} else {
			resultCols = append(resultCols, expandColumn(srcCol, entries, nResult))
		}
	}

	return New(resultCols...)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func anyToString(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", v)
}

// splitByComma splits a string by comma without importing strings.
func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpaces removes leading and trailing spaces from a string.
func trimSpaces(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

func buildColumnFromStrings(name string, dt dtype.DataType, values []string) *series.Series {
	// Rebuild the index column in its original type where possible.
	switch dt {
	case dtype.String:
		return series.NewString(name, values)
	default:
		// Fall back to string for simplicity in mixed/null cases.
		return series.NewString(name, values)
	}
}

type pivotCell struct {
	val any
	set bool
}

func buildPivotValueColumn(name string, dt dtype.DataType, grid [][]pivotCell, colIdx, nRows int) *series.Series {
	switch dt {
	case dtype.Int64:
		vals := make([]int64, nRows)
		validity := make([]bool, nRows)
		for r := 0; r < nRows; r++ {
			c := grid[r][colIdx]
			if c.set && c.val != nil {
				vals[r] = c.val.(int64)
				validity[r] = true
			}
		}
		return series.NewInt64WithValidity(name, vals, validity)
	case dtype.Float64:
		vals := make([]float64, nRows)
		validity := make([]bool, nRows)
		for r := 0; r < nRows; r++ {
			c := grid[r][colIdx]
			if c.set && c.val != nil {
				vals[r] = c.val.(float64)
				validity[r] = true
			}
		}
		return series.NewFloat64WithValidity(name, vals, validity)
	case dtype.Boolean:
		vals := make([]bool, nRows)
		validity := make([]bool, nRows)
		for r := 0; r < nRows; r++ {
			c := grid[r][colIdx]
			if c.set && c.val != nil {
				vals[r] = c.val.(bool)
				validity[r] = true
			}
		}
		return series.NewBooleanWithValidity(name, vals, validity)
	default: // String
		vals := make([]string, nRows)
		validity := make([]bool, nRows)
		for r := 0; r < nRows; r++ {
			c := grid[r][colIdx]
			if c.set && c.val != nil {
				vals[r] = c.val.(string)
				validity[r] = true
			}
		}
		return series.NewStringWithValidity(name, vals, validity)
	}
}

// promoteType returns the common type when mixing two types.
// Int64 + Float64 -> Float64. Other mixes are unsupported and return the first type.
func promoteType(a, b dtype.DataType) dtype.DataType {
	if a == b {
		return a
	}
	if (a == dtype.Int64 && b == dtype.Float64) || (a == dtype.Float64 && b == dtype.Int64) {
		return dtype.Float64
	}
	return a
}

func isPromotableTo(from, to dtype.DataType) bool {
	if from == to {
		return true
	}
	if from == dtype.Int64 && to == dtype.Float64 {
		return true
	}
	return false
}

// repeatColumn repeats the column values n times (in blocks).
func repeatColumn(col *series.Series, n int) *series.Series {
	h := col.Len()
	total := h * n

	switch col.DataType() {
	case dtype.Int64:
		vals := make([]int64, total)
		validity := make([]bool, total)
		idx := 0
		for block := 0; block < n; block++ {
			for r := 0; r < h; r++ {
				if col.IsValid(r) {
					v, _ := col.GetInt64(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewInt64WithValidity(col.Name(), vals, validity)
	case dtype.Float64:
		vals := make([]float64, total)
		validity := make([]bool, total)
		idx := 0
		for block := 0; block < n; block++ {
			for r := 0; r < h; r++ {
				if col.IsValid(r) {
					v, _ := col.GetFloat64(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewFloat64WithValidity(col.Name(), vals, validity)
	case dtype.Boolean:
		vals := make([]bool, total)
		validity := make([]bool, total)
		idx := 0
		for block := 0; block < n; block++ {
			for r := 0; r < h; r++ {
				if col.IsValid(r) {
					v, _ := col.GetBool(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewBooleanWithValidity(col.Name(), vals, validity)
	default: // String
		vals := make([]string, total)
		validity := make([]bool, total)
		idx := 0
		for block := 0; block < n; block++ {
			for r := 0; r < h; r++ {
				if col.IsValid(r) {
					v, _ := col.GetString(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewStringWithValidity(col.Name(), vals, validity)
	}
}

func buildMeltedValueColumn(name string, dt dtype.DataType, valCols []*series.Series, height int) *series.Series {
	total := height * len(valCols)

	switch dt {
	case dtype.Int64:
		vals := make([]int64, total)
		validity := make([]bool, total)
		idx := 0
		for _, vc := range valCols {
			for r := 0; r < height; r++ {
				if vc.IsValid(r) {
					v, _ := vc.GetInt64(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewInt64WithValidity(name, vals, validity)
	case dtype.Float64:
		vals := make([]float64, total)
		validity := make([]bool, total)
		idx := 0
		for _, vc := range valCols {
			for r := 0; r < height; r++ {
				if vc.IsValid(r) {
					switch vc.DataType() {
					case dtype.Float64:
						v, _ := vc.GetFloat64(r)
						vals[idx] = v
					case dtype.Int64:
						v, _ := vc.GetInt64(r)
						vals[idx] = float64(v)
					}
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewFloat64WithValidity(name, vals, validity)
	case dtype.Boolean:
		vals := make([]bool, total)
		validity := make([]bool, total)
		idx := 0
		for _, vc := range valCols {
			for r := 0; r < height; r++ {
				if vc.IsValid(r) {
					v, _ := vc.GetBool(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewBooleanWithValidity(name, vals, validity)
	default: // String
		vals := make([]string, total)
		validity := make([]bool, total)
		idx := 0
		for _, vc := range valCols {
			for r := 0; r < height; r++ {
				if vc.IsValid(r) {
					v, _ := vc.GetString(r)
					vals[idx] = v
					validity[idx] = true
				}
				idx++
			}
		}
		return series.NewStringWithValidity(name, vals, validity)
	}
}

func buildTransposeColumn(name string, dt dtype.DataType, cols []*series.Series, rowIdx int) *series.Series {
	nCols := len(cols)
	switch dt {
	case dtype.Float64:
		vals := make([]float64, nCols)
		validity := make([]bool, nCols)
		for i, col := range cols {
			if col.IsValid(rowIdx) {
				switch col.DataType() {
				case dtype.Float64:
					v, _ := col.GetFloat64(rowIdx)
					vals[i] = v
				case dtype.Int64:
					v, _ := col.GetInt64(rowIdx)
					vals[i] = float64(v)
				}
				validity[i] = true
			}
		}
		return series.NewFloat64WithValidity(name, vals, validity)
	case dtype.String:
		vals := make([]string, nCols)
		validity := make([]bool, nCols)
		for i, col := range cols {
			if col.IsValid(rowIdx) {
				v, _ := col.GetString(rowIdx)
				vals[i] = v
				validity[i] = true
			}
		}
		return series.NewStringWithValidity(name, vals, validity)
	case dtype.Boolean:
		vals := make([]bool, nCols)
		validity := make([]bool, nCols)
		for i, col := range cols {
			if col.IsValid(rowIdx) {
				v, _ := col.GetBool(rowIdx)
				vals[i] = v
				validity[i] = true
			}
		}
		return series.NewBooleanWithValidity(name, vals, validity)
	default: // Int64 (shouldn't happen since we promote, but handle anyway)
		vals := make([]int64, nCols)
		validity := make([]bool, nCols)
		for i, col := range cols {
			if col.IsValid(rowIdx) {
				v, _ := col.GetInt64(rowIdx)
				vals[i] = v
				validity[i] = true
			}
		}
		return series.NewInt64WithValidity(name, vals, validity)
	}
}

type explodeEntry struct {
	origRow int
	value   string
	isNull  bool
}

func expandColumn(col *series.Series, entries []explodeEntry, nResult int) *series.Series {
	switch col.DataType() {
	case dtype.Int64:
		vals := make([]int64, nResult)
		validity := make([]bool, nResult)
		for k, e := range entries {
			r := e.origRow
			if col.IsValid(r) {
				v, _ := col.GetInt64(r)
				vals[k] = v
				validity[k] = true
			}
		}
		return series.NewInt64WithValidity(col.Name(), vals, validity)
	case dtype.Float64:
		vals := make([]float64, nResult)
		validity := make([]bool, nResult)
		for k, e := range entries {
			r := e.origRow
			if col.IsValid(r) {
				v, _ := col.GetFloat64(r)
				vals[k] = v
				validity[k] = true
			}
		}
		return series.NewFloat64WithValidity(col.Name(), vals, validity)
	case dtype.Boolean:
		vals := make([]bool, nResult)
		validity := make([]bool, nResult)
		for k, e := range entries {
			r := e.origRow
			if col.IsValid(r) {
				v, _ := col.GetBool(r)
				vals[k] = v
				validity[k] = true
			}
		}
		return series.NewBooleanWithValidity(col.Name(), vals, validity)
	default: // String
		vals := make([]string, nResult)
		validity := make([]bool, nResult)
		for k, e := range entries {
			r := e.origRow
			if col.IsValid(r) {
				v, _ := col.GetString(r)
				vals[k] = v
				validity[k] = true
			}
		}
		return series.NewStringWithValidity(col.Name(), vals, validity)
	}
}
