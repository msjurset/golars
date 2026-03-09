// Package csv provides CSV reading and writing for Golars DataFrames.
package csv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// ReadOptions configures CSV reading behavior.
type ReadOptions struct {
	Separator        rune
	Quote            rune
	HasHeader        bool
	NullValues       []string
	Columns          []string // only read these columns (nil = all)
	Dtypes           map[string]dtype.DataType
	InferSchemaLen   int // number of rows to scan for type inference
	SkipRows         int
	SkipRowsAfterHdr int
	CommentChar      rune
	TruncateLines    bool // if true, short rows are padded with nulls
	NRows            int  // max rows to read (0 = all)
	Ctx              context.Context
}

// DefaultReadOptions returns sensible defaults for CSV reading.
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		Separator:      ',',
		Quote:          '"',
		HasHeader:      true,
		NullValues:     []string{"", "null", "NULL", "NA", "N/A", "NaN", "nan"},
		InferSchemaLen: 100,
	}
}

// ReadOption is a functional option for CSV reading.
type ReadOption func(*ReadOptions)

// WithSeparator sets the field separator character.
func WithSeparator(sep rune) ReadOption {
	return func(o *ReadOptions) { o.Separator = sep }
}

// WithQuote sets the quote character.
func WithQuote(q rune) ReadOption {
	return func(o *ReadOptions) { o.Quote = q }
}

// WithHasHeader controls whether the first row is a header.
func WithHasHeader(has bool) ReadOption {
	return func(o *ReadOptions) { o.HasHeader = has }
}

// WithNullValues sets the strings treated as null.
func WithNullValues(values ...string) ReadOption {
	return func(o *ReadOptions) { o.NullValues = values }
}

// WithColumns restricts reading to the named columns.
func WithColumns(names ...string) ReadOption {
	return func(o *ReadOptions) { o.Columns = names }
}

// WithDtypes overrides inferred types for specific columns.
func WithDtypes(dtypes map[string]dtype.DataType) ReadOption {
	return func(o *ReadOptions) { o.Dtypes = dtypes }
}

// WithInferSchemaLength sets how many rows to scan for type inference.
func WithInferSchemaLength(n int) ReadOption {
	return func(o *ReadOptions) { o.InferSchemaLen = n }
}

// WithSkipRows skips the first n rows before reading.
func WithSkipRows(n int) ReadOption {
	return func(o *ReadOptions) { o.SkipRows = n }
}

// WithNRows limits reading to at most n rows of data.
func WithNRows(n int) ReadOption {
	return func(o *ReadOptions) { o.NRows = n }
}

// WithContext sets a context for cancellation during CSV reading.
func WithContext(ctx context.Context) ReadOption {
	return func(o *ReadOptions) { o.Ctx = ctx }
}

// ReadFile reads a CSV file into column data.
func ReadFile(path string, opts ...ReadOption) ([]*series.Series, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("golars: csv: %w", err)
	}
	defer f.Close()
	return Read(f, opts...)
}

// Read reads CSV data from an io.Reader.
func Read(r io.Reader, opts ...ReadOption) ([]*series.Series, error) {
	options := DefaultReadOptions()
	for _, o := range opts {
		o(&options)
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	// Skip initial rows
	for i := 0; i < options.SkipRows; i++ {
		if !scanner.Scan() {
			return nil, fmt.Errorf("golars: csv: not enough rows to skip")
		}
	}

	// Read header
	var colNames []string
	if options.HasHeader {
		if !scanner.Scan() {
			return nil, fmt.Errorf("golars: csv: empty CSV")
		}
		colNames = parseLine(scanner.Text(), options.Separator, options.Quote)
	}

	// Skip rows after header
	for i := 0; i < options.SkipRowsAfterHdr; i++ {
		if !scanner.Scan() {
			break
		}
	}

	// Determine which columns to keep
	var colIndices []int
	if options.Columns != nil && options.HasHeader {
		nameToIdx := make(map[string]int)
		for i, name := range colNames {
			nameToIdx[name] = i
		}
		for _, name := range options.Columns {
			idx, ok := nameToIdx[name]
			if !ok {
				return nil, fmt.Errorf("golars: csv: column %q not found", name)
			}
			colIndices = append(colIndices, idx)
		}
		filteredNames := make([]string, len(colIndices))
		for i, idx := range colIndices {
			filteredNames[i] = colNames[idx]
		}
		colNames = filteredNames
	}

	// Read all data rows as strings
	var rawRows [][]string
	rowCount := 0
	for scanner.Scan() {
		if options.Ctx != nil {
			if err := options.Ctx.Err(); err != nil {
				return nil, err
			}
		}
		if options.NRows > 0 && rowCount >= options.NRows {
			break
		}
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if options.CommentChar != 0 {
			r, _ := utf8.DecodeRuneInString(line)
			if r == options.CommentChar {
				continue
			}
		}
		fields := parseLine(line, options.Separator, options.Quote)

		if colIndices != nil {
			filtered := make([]string, len(colIndices))
			for i, idx := range colIndices {
				if idx < len(fields) {
					filtered[i] = fields[idx]
				}
			}
			fields = filtered
		}

		rawRows = append(rawRows, fields)
		rowCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("golars: csv: scan error: %w", err)
	}

	// Generate column names if no header
	numCols := 0
	if len(rawRows) > 0 {
		numCols = len(rawRows[0])
	}
	if !options.HasHeader {
		colNames = make([]string, numCols)
		for i := range colNames {
			colNames[i] = fmt.Sprintf("column_%d", i)
		}
	}
	if numCols == 0 {
		numCols = len(colNames)
	}

	// Build null value lookup
	nullSet := make(map[string]struct{}, len(options.NullValues))
	for _, nv := range options.NullValues {
		nullSet[nv] = struct{}{}
	}

	// Extract column data
	colData := make([][]string, numCols)
	for i := range colData {
		colData[i] = make([]string, len(rawRows))
	}
	for rowIdx, row := range rawRows {
		for colIdx := 0; colIdx < numCols; colIdx++ {
			if colIdx < len(row) {
				colData[colIdx][rowIdx] = row[colIdx]
			}
		}
	}

	// Infer types and build Series
	result := make([]*series.Series, numCols)
	for i := 0; i < numCols; i++ {
		name := colNames[i]
		dt := dtype.String
		if options.Dtypes != nil {
			if forced, ok := options.Dtypes[name]; ok {
				dt = forced
			} else {
				dt = inferType(colData[i], nullSet, options.InferSchemaLen)
			}
		} else {
			dt = inferType(colData[i], nullSet, options.InferSchemaLen)
		}
		s, err := buildSeries(name, colData[i], dt, nullSet)
		if err != nil {
			return nil, fmt.Errorf("golars: csv: column %q: %w", name, err)
		}
		result[i] = s
	}

	return result, nil
}

// parseLine splits a CSV line into fields respecting quotes.
func parseLine(line string, sep, quote rune) []string {
	var fields []string
	var field strings.Builder
	inQuotes := false

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inQuotes {
			if c == quote {
				if i+1 < len(runes) && runes[i+1] == quote {
					field.WriteRune(quote) // escaped quote
					i++
				} else {
					inQuotes = false
				}
			} else {
				field.WriteRune(c)
			}
		} else {
			if c == quote {
				inQuotes = true
			} else if c == sep {
				fields = append(fields, field.String())
				field.Reset()
			} else {
				field.WriteRune(c)
			}
		}
	}
	fields = append(fields, field.String())
	return fields
}

// inferType infers the data type from sample values.
func inferType(values []string, nullSet map[string]struct{}, maxScan int) dtype.DataType {
	if maxScan <= 0 || maxScan > len(values) {
		maxScan = len(values)
	}

	canInt := true
	canFloat := true
	canBool := true
	seenNonNull := false

	for i := 0; i < maxScan; i++ {
		v := strings.TrimSpace(values[i])
		if _, isNull := nullSet[v]; isNull {
			continue
		}
		seenNonNull = true

		if canBool {
			lower := strings.ToLower(v)
			if lower != "true" && lower != "false" && lower != "1" && lower != "0" {
				canBool = false
			}
		}
		if canInt {
			if _, err := strconv.ParseInt(v, 10, 64); err != nil {
				canInt = false
			}
		}
		if canFloat {
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				canFloat = false
			}
		}
	}

	if !seenNonNull {
		return dtype.String
	}
	if canBool {
		return dtype.Boolean
	}
	if canInt {
		return dtype.Int64
	}
	if canFloat {
		return dtype.Float64
	}
	return dtype.String
}

// buildSeries creates a Series from string values with the given type.
func buildSeries(name string, values []string, dt dtype.DataType, nullSet map[string]struct{}) (*series.Series, error) {
	n := len(values)

	switch dt {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			v = strings.TrimSpace(v)
			if _, isNull := nullSet[v]; isNull {
				hasNulls = true
				continue
			}
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as Int64: %w", v, err)
			}
			data[i] = parsed
			valid[i] = true
		}
		if hasNulls {
			return series.NewInt64WithValidity(name, data, valid), nil
		}
		return series.NewInt64(name, data), nil

	case dtype.Float64:
		data := make([]float64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			v = strings.TrimSpace(v)
			if _, isNull := nullSet[v]; isNull {
				hasNulls = true
				continue
			}
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as Float64: %w", v, err)
			}
			data[i] = parsed
			valid[i] = true
		}
		if hasNulls {
			return series.NewFloat64WithValidity(name, data, valid), nil
		}
		return series.NewFloat64(name, data), nil

	case dtype.Boolean:
		data := make([]bool, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			v = strings.TrimSpace(v)
			if _, isNull := nullSet[v]; isNull {
				hasNulls = true
				continue
			}
			lower := strings.ToLower(v)
			data[i] = lower == "true" || lower == "1"
			valid[i] = true
		}
		if hasNulls {
			return series.NewBooleanWithValidity(name, data, valid), nil
		}
		return series.NewBoolean(name, data), nil

	default: // String
		data := make([]string, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			if _, isNull := nullSet[v]; isNull {
				hasNulls = true
				continue
			}
			data[i] = v
			valid[i] = true
		}
		if hasNulls {
			return series.NewStringWithValidity(name, data, valid), nil
		}
		return series.NewString(name, data), nil
	}
}

// Ensure array and bitmap imports are used for downstream.
var _ = array.NewInt64Array
var _ = bitmap.New
