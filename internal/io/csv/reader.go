// Package csv provides CSV reading and writing for Golars DataFrames.
package csv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

	// Read entire input at once to avoid per-line Scanner overhead.
	rawBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("golars: csv: %w", err)
	}

	// Count lines for pre-allocation of column slices.
	lineCount := bytes.Count(rawBytes, []byte{'\n'})
	if len(rawBytes) > 0 && rawBytes[len(rawBytes)-1] != '\n' {
		lineCount++
	}

	// Convert to string once; substring slicing shares backing memory.
	content := string(rawBytes)
	rawBytes = nil // allow GC of the byte slice

	pos := 0

	// nextLine extracts the next line from content starting at pos,
	// advancing pos past the newline. Returns the line (without \r\n)
	// and false if no more data.
	nextLine := func() (string, bool) {
		if pos >= len(content) {
			return "", false
		}
		nlIdx := strings.IndexByte(content[pos:], '\n')
		var line string
		if nlIdx < 0 {
			line = content[pos:]
			pos = len(content)
		} else {
			line = content[pos : pos+nlIdx]
			pos += nlIdx + 1
		}
		// Strip trailing \r
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		return line, true
	}

	// Skip initial rows
	for i := 0; i < options.SkipRows; i++ {
		if _, ok := nextLine(); !ok {
			return nil, fmt.Errorf("golars: csv: not enough rows to skip")
		}
	}

	// Read header
	var colNames []string
	if options.HasHeader {
		headerLine, ok := nextLine()
		if !ok {
			return nil, fmt.Errorf("golars: csv: empty CSV")
		}
		colNames = parseLine(headerLine, options.Separator, options.Quote)
	}

	// Skip rows after header
	for i := 0; i < options.SkipRowsAfterHdr; i++ {
		if _, ok := nextLine(); !ok {
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

	// Estimate data rows for pre-allocation.
	estRows := lineCount - options.SkipRows - options.SkipRowsAfterHdr
	if options.HasHeader {
		estRows--
	}
	if estRows < 0 {
		estRows = 0
	}
	if options.NRows > 0 && estRows > options.NRows {
		estRows = options.NRows
	}

	numCols := len(colNames) // may be 0 if no header
	var colData [][]string
	rowCount := 0

	// For the no-header case, we need to determine numCols from the first data line
	// before we can dispatch to parallel parsing.
	if numCols == 0 && pos < len(content) {
		// Peek at first data line to determine column count
		peekStart := pos
		nlIdx := strings.IndexByte(content[pos:], '\n')
		var firstLine string
		if nlIdx < 0 {
			firstLine = content[pos:]
		} else {
			firstLine = content[pos : pos+nlIdx]
		}
		if len(firstLine) > 0 && firstLine[len(firstLine)-1] == '\r' {
			firstLine = firstLine[:len(firstLine)-1]
		}
		if len(firstLine) > 0 {
			fields := parseLine(firstLine, options.Separator, options.Quote)
			numCols = len(fields)
		}
		// Reset pos so the first line gets included in parallel parsing
		pos = peekStart
	}

	if numCols > 0 {
		// Parallel data line parsing
		lineStarts := buildDataLineStarts(content, pos)
		nDataLines := len(lineStarts)
		if options.NRows > 0 && nDataLines > options.NRows {
			nDataLines = options.NRows
			lineStarts = lineStarts[:nDataLines]
		}

		if nDataLines > 0 {
			colData = parallelParseDataLines(content, lineStarts, numCols, colIndices,
				options.Separator, options.Quote, options.CommentChar)
			if len(colData) > 0 {
				rowCount = len(colData[0])
			}
		} else {
			colData = make([][]string, numCols)
		}
	} else {
		// Truly empty file — sequential fallback
		colData = make([][]string, 0)
		for {
			if options.NRows > 0 && rowCount >= options.NRows {
				break
			}
			line, ok := nextLine()
			if !ok {
				break
			}
			if options.Ctx != nil && rowCount%10000 == 0 {
				if err := options.Ctx.Err(); err != nil {
					return nil, err
				}
			}
			if len(line) == 0 {
				continue
			}
			if options.CommentChar != 0 {
				ch, _ := utf8.DecodeRuneInString(line)
				if ch == options.CommentChar {
					continue
				}
			}
			fields := parseLine(line, options.Separator, options.Quote)

			if numCols == 0 {
				numCols = len(fields)
				colData = make([][]string, numCols)
				for i := range colData {
					colData[i] = make([]string, 0, estRows)
				}
			}

			for i := 0; i < numCols; i++ {
				if i < len(fields) {
					colData[i] = append(colData[i], fields[i])
				} else {
					colData[i] = append(colData[i], "")
				}
			}
			rowCount++
		}
	}

	// Generate column names if no header
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

	// Infer types and build Series in parallel
	result := make([]*series.Series, numCols)
	errs := make([]error, numCols)

	var wg sync.WaitGroup
	for i := 0; i < numCols; i++ {
		wg.Add(1)
		go func(col int) {
			defer wg.Done()
			name := colNames[col]
			var data []string
			if col < len(colData) {
				data = colData[col]
			}

			dt := dtype.String
			if options.Dtypes != nil {
				if forced, ok := options.Dtypes[name]; ok {
					dt = forced
				} else {
					dt = inferType(data, nullSet, options.InferSchemaLen)
				}
			} else {
				dt = inferType(data, nullSet, options.InferSchemaLen)
			}
			s, err := buildSeries(name, data, dt, nullSet)
			if err != nil {
				errs[col] = fmt.Errorf("golars: csv: column %q: %w", name, err)
				return
			}
			result[col] = s
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// buildDataLineStarts returns the byte offset of the start of each line in
// content beginning at fromPos.
func buildDataLineStarts(content string, fromPos int) []int {
	remaining := content[fromPos:]
	n := strings.Count(remaining, "\n")
	if len(remaining) > 0 && remaining[len(remaining)-1] != '\n' {
		n++
	}
	starts := make([]int, 0, n)
	pos := fromPos
	for pos < len(content) {
		starts = append(starts, pos)
		nlIdx := strings.IndexByte(content[pos:], '\n')
		if nlIdx < 0 {
			break
		}
		pos += nlIdx + 1
	}
	return starts
}

// extractDataLine returns the content of the line at starts[idx], stripping
// trailing \r\n.
func extractDataLine(content string, starts []int, idx int) string {
	begin := starts[idx]
	var end int
	if idx+1 < len(starts) {
		end = starts[idx+1]
	} else {
		end = len(content)
	}
	if end > begin && content[end-1] == '\n' {
		end--
	}
	if end > begin && content[end-1] == '\r' {
		end--
	}
	return content[begin:end]
}

// parallelParseDataLines distributes CSV line parsing across multiple goroutines.
// Each goroutine parses its share of lines and returns per-column string slices.
// The results are merged in order. Empty lines are skipped.
func parallelParseDataLines(
	content string,
	lineStarts []int,
	numCols int,
	colIndices []int,
	sep, quote rune,
	commentChar rune,
) [][]string {
	nLines := len(lineStarts)
	nWorkers := runtime.GOMAXPROCS(0)
	if nWorkers > nLines {
		nWorkers = nLines
	}
	if nWorkers <= 1 {
		// Sequential fallback
		result := make([][]string, numCols)
		for i := range result {
			result[i] = make([]string, 0, nLines)
		}
		for li := 0; li < nLines; li++ {
			line := extractDataLine(content, lineStarts, li)
			if len(line) == 0 {
				continue
			}
			if commentChar != 0 {
				ch, _ := utf8.DecodeRuneInString(line)
				if ch == commentChar {
					continue
				}
			}
			fields := parseLine(line, sep, quote)
			if colIndices != nil {
				for ci, idx := range colIndices {
					if idx < len(fields) {
						result[ci] = append(result[ci], fields[idx])
					} else {
						result[ci] = append(result[ci], "")
					}
				}
			} else {
				for i := 0; i < numCols; i++ {
					if i < len(fields) {
						result[i] = append(result[i], fields[i])
					} else {
						result[i] = append(result[i], "")
					}
				}
			}
		}
		return result
	}

	chunkSize := (nLines + nWorkers - 1) / nWorkers
	type chunkResult [][]string
	chunks := make([]chunkResult, nWorkers)

	var wg sync.WaitGroup
	for w := 0; w < nWorkers; w++ {
		lineStart := w * chunkSize
		lineEnd := lineStart + chunkSize
		if lineEnd > nLines {
			lineEnd = nLines
		}
		if lineStart >= lineEnd {
			break
		}
		wg.Add(1)
		go func(workerID, ls, le int) {
			defer wg.Done()
			n := le - ls
			chunk := make([][]string, numCols)
			for i := range chunk {
				chunk[i] = make([]string, 0, n)
			}
			for li := ls; li < le; li++ {
				line := extractDataLine(content, lineStarts, li)
				if len(line) == 0 {
					continue
				}
				if commentChar != 0 {
					ch, _ := utf8.DecodeRuneInString(line)
					if ch == commentChar {
						continue
					}
				}
				fields := parseLine(line, sep, quote)
				if colIndices != nil {
					for ci, idx := range colIndices {
						if idx < len(fields) {
							chunk[ci] = append(chunk[ci], fields[idx])
						} else {
							chunk[ci] = append(chunk[ci], "")
						}
					}
				} else {
					for i := 0; i < numCols; i++ {
						if i < len(fields) {
							chunk[i] = append(chunk[i], fields[i])
						} else {
							chunk[i] = append(chunk[i], "")
						}
					}
				}
			}
			chunks[workerID] = chunk
		}(w, lineStart, lineEnd)
	}
	wg.Wait()

	// Merge: count total rows and concatenate per-column slices
	total := 0
	for _, ch := range chunks {
		if ch != nil && len(ch) > 0 {
			total += len(ch[0])
		}
	}
	result := make([][]string, numCols)
	for ci := range result {
		result[ci] = make([]string, 0, total)
		for _, ch := range chunks {
			if ch != nil && ci < len(ch) {
				result[ci] = append(result[ci], ch[ci]...)
			}
		}
	}
	return result
}

// parseLine splits a CSV line into fields respecting quotes.
// It dispatches to an optimized ASCII byte-level parser when both separator
// and quote character are in the ASCII range (the common case).
func parseLine(line string, sep, quote rune) []string {
	if sep < utf8.RuneSelf && quote < utf8.RuneSelf {
		return parseLineASCII(line, byte(sep), byte(quote))
	}
	return parseLineRune(line, sep, quote)
}

// parseLineASCII is the fast path for ASCII separator and quote characters.
// It works at the byte level, avoiding []rune conversion, and uses substring
// slicing (zero-allocation) for unquoted fields.
func parseLineASCII(line string, sep, quote byte) []string {
	// Pre-count fields by counting unquoted separators for a single allocation.
	n := 1
	inQ := false
	for i := 0; i < len(line); i++ {
		if line[i] == quote {
			inQ = !inQ
		} else if line[i] == sep && !inQ {
			n++
		}
	}

	fields := make([]string, 0, n)
	start := 0
	inQ = false
	hasQuote := false

	for i := 0; i < len(line); i++ {
		b := line[i]
		if inQ {
			if b == quote {
				if i+1 < len(line) && line[i+1] == quote {
					hasQuote = true
					i++ // skip escaped quote
				} else {
					inQ = false
				}
			}
		} else {
			if b == quote {
				inQ = true
				hasQuote = true
			} else if b == sep {
				fields = append(fields, extractField(line[start:i], quote, hasQuote))
				start = i + 1
				hasQuote = false
			}
		}
	}
	fields = append(fields, extractField(line[start:], quote, hasQuote))
	return fields
}

// extractField processes a raw field substring. If the field has no quotes it
// is returned as-is (a zero-allocation substring of the original line). Quoted
// fields have surrounding quotes stripped and escaped (doubled) quotes resolved.
func extractField(s string, quote byte, hasQuote bool) string {
	if !hasQuote {
		return s // zero-allocation substring
	}
	// Strip surrounding quotes
	if len(s) >= 2 && s[0] == quote && s[len(s)-1] == quote {
		s = s[1 : len(s)-1]
	}
	// Only allocate if there are escaped quotes inside
	if strings.IndexByte(s, quote) < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == quote && i+1 < len(s) && s[i+1] == quote {
			b.WriteByte(quote)
			i++
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// parseLineRune is the slow path for non-ASCII separator or quote characters.
// It converts the line to runes and parses field by field.
func parseLineRune(line string, sep, quote rune) []string {
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
		v := trimSpaceFast(values[i])
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
		// Early termination: if only String is possible, stop scanning.
		if !canInt && !canFloat && !canBool {
			return dtype.String
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

// trimSpaceFast is a fast-path TrimSpace that avoids calling strings.TrimSpace
// when the string has no leading/trailing whitespace (the common case for CSV).
func trimSpaceFast(s string) string {
	if len(s) == 0 {
		return s
	}
	// Fast check: if first and last bytes are not ASCII space characters, return as-is.
	if s[0] > ' ' && s[len(s)-1] > ' ' {
		return s
	}
	return strings.TrimSpace(s)
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
			v = trimSpaceFast(v)
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
			v = trimSpaceFast(v)
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
			v = trimSpaceFast(v)
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
