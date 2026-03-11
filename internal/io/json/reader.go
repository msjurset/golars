// Package json provides JSON and NDJSON reading and writing for Golars DataFrames.
package json

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/msjurset/golars/internal/series"
)

// ReadOptions configures JSON reading behavior.
type ReadOptions struct {
	InferSchemaLen int
}

// DefaultReadOptions returns default JSON read options.
func DefaultReadOptions() ReadOptions {
	return ReadOptions{
		InferSchemaLen: 100,
	}
}

// ReadOption is a functional option for JSON reading.
type ReadOption func(*ReadOptions)

// ReadFile reads a JSON array-of-objects file into Series columns.
func ReadFile(path string, opts ...ReadOption) ([]*series.Series, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("golars: json: %w", err)
	}
	defer f.Close()
	return Read(f, opts...)
}

// column type constants for the builder.
const (
	colNull = iota
	colBool
	colInt64
	colFloat64
	colString
)

// columnBuilder accumulates values for a single column during streaming parse.
type columnBuilder struct {
	name   string
	ctype  int
	ints   []int64
	floats []float64
	bools  []bool
	strs   []string
	valid  []bool
	nulls  bool
}

func newColumnBuilder(name string, capacity int) *columnBuilder {
	return &columnBuilder{
		name:  name,
		ctype: colNull,
		valid: make([]bool, 0, capacity),
	}
}

// appendNull adds a null value.
func (cb *columnBuilder) appendNull() {
	cb.nulls = true
	cb.valid = append(cb.valid, false)
	switch cb.ctype {
	case colInt64:
		cb.ints = append(cb.ints, 0)
	case colFloat64:
		cb.floats = append(cb.floats, 0)
	case colBool:
		cb.bools = append(cb.bools, false)
	case colString:
		cb.strs = append(cb.strs, "")
	}
}

// appendRawValue parses a raw JSON value slice and appends to the column.
func (cb *columnBuilder) appendRawValue(raw []byte) {
	if len(raw) == 0 {
		cb.appendNull()
		return
	}

	switch raw[0] {
	case '"':
		cb.appendString(unquoteJSON(raw))

	case 't':
		cb.appendBool(true)

	case 'f':
		cb.appendBool(false)

	case 'n':
		cb.appendNull()

	case '[', '{':
		cb.appendString(string(raw))

	default:
		// Number - try int64 first via fast byte-level parse.
		if i, ok := parseInt64Bytes(raw); ok {
			cb.appendInt64(i)
		} else if f, ok := parseFloat64Bytes(raw); ok {
			cb.appendFloat64(f)
		} else {
			cb.appendString(string(raw))
		}
	}
}

// parseInt64Bytes parses an integer from bytes without allocating a string.
func parseInt64Bytes(b []byte) (int64, bool) {
	if len(b) == 0 {
		return 0, false
	}
	neg := false
	i := 0
	if b[0] == '-' {
		neg = true
		i = 1
	} else if b[0] == '+' {
		i = 1
	}
	if i >= len(b) {
		return 0, false
	}
	var n int64
	for ; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false // contains '.', 'e', 'E', etc.
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}

// parseFloat64Bytes parses a float from bytes using strconv.
func parseFloat64Bytes(b []byte) (float64, bool) {
	// strconv.ParseFloat accepts string; the compiler may optimize this.
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// unquoteJSON removes surrounding quotes and handles escape sequences.
func unquoteJSON(raw []byte) string {
	n := len(raw)
	if n < 2 || raw[0] != '"' || raw[n-1] != '"' {
		return string(raw)
	}
	inner := raw[1 : n-1]
	// Fast path: no backslash means no escapes.
	for _, b := range inner {
		if b == '\\' {
			// Slow path: use strconv.Unquote.
			if s, err := strconv.Unquote(string(raw)); err == nil {
				return s
			}
			return string(inner)
		}
	}
	return string(inner)
}

func (cb *columnBuilder) appendInt64(i int64) {
	switch cb.ctype {
	case colNull:
		cb.ctype = colInt64
		n := len(cb.valid)
		cb.ints = make([]int64, n, cap(cb.valid))
		cb.ints = append(cb.ints, i)
	case colInt64:
		cb.ints = append(cb.ints, i)
	case colFloat64:
		cb.floats = append(cb.floats, float64(i))
		cb.valid = append(cb.valid, true)
		return
	case colString:
		cb.strs = append(cb.strs, strconv.FormatInt(i, 10))
		cb.valid = append(cb.valid, true)
		return
	case colBool:
		cb.widenToString()
		cb.strs = append(cb.strs, strconv.FormatInt(i, 10))
		cb.valid = append(cb.valid, true)
		return
	}
	cb.valid = append(cb.valid, true)
}

func (cb *columnBuilder) appendFloat64(f float64) {
	switch cb.ctype {
	case colNull:
		cb.ctype = colFloat64
		n := len(cb.valid)
		cb.floats = make([]float64, n, cap(cb.valid))
		cb.floats = append(cb.floats, f)
	case colInt64:
		cb.widenToFloat64()
		cb.floats = append(cb.floats, f)
	case colFloat64:
		cb.floats = append(cb.floats, f)
	case colString:
		cb.strs = append(cb.strs, strconv.FormatFloat(f, 'f', -1, 64))
		cb.valid = append(cb.valid, true)
		return
	case colBool:
		cb.widenToString()
		cb.strs = append(cb.strs, strconv.FormatFloat(f, 'f', -1, 64))
		cb.valid = append(cb.valid, true)
		return
	}
	cb.valid = append(cb.valid, true)
}

func (cb *columnBuilder) appendBool(b bool) {
	switch cb.ctype {
	case colNull:
		cb.ctype = colBool
		n := len(cb.valid)
		cb.bools = make([]bool, n, cap(cb.valid))
		cb.bools = append(cb.bools, b)
	case colBool:
		cb.bools = append(cb.bools, b)
	case colString:
		if b {
			cb.strs = append(cb.strs, "true")
		} else {
			cb.strs = append(cb.strs, "false")
		}
	default:
		cb.widenToString()
		if b {
			cb.strs = append(cb.strs, "true")
		} else {
			cb.strs = append(cb.strs, "false")
		}
	}
	cb.valid = append(cb.valid, true)
}

func (cb *columnBuilder) appendString(s string) {
	if cb.ctype != colString {
		cb.widenToString()
	}
	cb.strs = append(cb.strs, s)
	cb.valid = append(cb.valid, true)
}

// widenToFloat64 converts an int64 column to float64.
func (cb *columnBuilder) widenToFloat64() {
	if cb.ctype == colInt64 {
		cb.floats = make([]float64, len(cb.ints), cap(cb.ints)+1)
		for i, v := range cb.ints {
			cb.floats[i] = float64(v)
		}
		cb.ints = nil
	}
	cb.ctype = colFloat64
}

// widenToString converts any column type to string.
func (cb *columnBuilder) widenToString() {
	n := len(cb.valid)
	cb.strs = make([]string, n, cap(cb.valid))
	switch cb.ctype {
	case colInt64:
		for i, v := range cb.ints {
			if cb.valid[i] {
				cb.strs[i] = strconv.FormatInt(v, 10)
			}
		}
		cb.ints = nil
	case colFloat64:
		for i, v := range cb.floats {
			if cb.valid[i] {
				cb.strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
			}
		}
		cb.floats = nil
	case colBool:
		for i, v := range cb.bools {
			if cb.valid[i] {
				if v {
					cb.strs[i] = "true"
				} else {
					cb.strs[i] = "false"
				}
			}
		}
		cb.bools = nil
	case colNull:
		// All nulls so far, strs already zeroed.
	}
	cb.ctype = colString
}

// backfillNulls fills null slots for columns that were discovered after the first row.
func (cb *columnBuilder) backfillNulls(count int) {
	cb.valid = make([]bool, count, count+8)
	cb.nulls = count > 0
}

// toSeries converts the accumulated data into a Series.
func (cb *columnBuilder) toSeries() *series.Series {
	switch cb.ctype {
	case colInt64:
		if cb.nulls {
			return series.NewInt64WithValidity(cb.name, cb.ints, cb.valid)
		}
		return series.NewInt64(cb.name, cb.ints)
	case colFloat64:
		if cb.nulls {
			return series.NewFloat64WithValidity(cb.name, cb.floats, cb.valid)
		}
		return series.NewFloat64(cb.name, cb.floats)
	case colBool:
		if cb.nulls {
			return series.NewBooleanWithValidity(cb.name, cb.bools, cb.valid)
		}
		return series.NewBoolean(cb.name, cb.bools)
	case colString:
		if cb.nulls {
			return series.NewStringWithValidity(cb.name, cb.strs, cb.valid)
		}
		return series.NewString(cb.name, cb.strs)
	default: // colNull - all values were null
		strs := make([]string, len(cb.valid))
		valid := make([]bool, len(cb.valid))
		return series.NewStringWithValidity(cb.name, strs, valid)
	}
}

// -----------------------------------------------------------------------
// Byte-level JSON scanner
// -----------------------------------------------------------------------

// skipWhitespace advances pos past any JSON whitespace.
func skipWhitespace(data []byte, pos int) int {
	for pos < len(data) {
		switch data[pos] {
		case ' ', '\t', '\n', '\r':
			pos++
		default:
			return pos
		}
	}
	return pos
}

// scanString scans a JSON string starting at data[pos] (which must be '"').
// Returns the slice including quotes and the position after the closing quote.
func scanString(data []byte, pos int) ([]byte, int) {
	start := pos
	pos++ // skip opening quote
	for pos < len(data) {
		b := data[pos]
		if b == '\\' {
			pos += 2 // skip escaped character
			continue
		}
		if b == '"' {
			pos++ // skip closing quote
			return data[start:pos], pos
		}
		pos++
	}
	return data[start:pos], pos
}

// scanValue scans any JSON value starting at data[pos].
// Returns the raw bytes of the value and the position after it.
func scanValue(data []byte, pos int) ([]byte, int) {
	if pos >= len(data) {
		return nil, pos
	}

	switch data[pos] {
	case '"':
		return scanString(data, pos)

	case '{':
		return scanBraced(data, pos, '{', '}')

	case '[':
		return scanBraced(data, pos, '[', ']')

	default:
		// Number, bool, null - scan until delimiter.
		start := pos
		for pos < len(data) {
			switch data[pos] {
			case ',', '}', ']', ' ', '\t', '\n', '\r':
				return data[start:pos], pos
			}
			pos++
		}
		return data[start:pos], pos
	}
}

// scanBraced scans a JSON object or array, handling nesting and strings.
func scanBraced(data []byte, pos int, open, close byte) ([]byte, int) {
	start := pos
	depth := 1
	pos++ // skip opening brace/bracket
	for pos < len(data) && depth > 0 {
		b := data[pos]
		switch b {
		case '"':
			_, pos = scanString(data, pos)
			continue
		case open:
			depth++
		case close:
			depth--
		}
		pos++
	}
	return data[start:pos], pos
}

// rawKV represents a key-value pair extracted from a JSON object.
// keyRaw is the raw bytes including quotes (sub-slice of source, no alloc).
type rawKV struct {
	keyRaw []byte // raw key bytes including quotes, e.g. "name"
	value  []byte // raw JSON value bytes, sub-slice of source
}

// scanObjectRaw extracts key-value pairs from a JSON object at data[pos].
// Keys are kept as raw byte slices (no string allocation).
// Reuses the provided kvs slice.
func scanObjectRaw(data []byte, pos int, kvs []rawKV) ([]rawKV, int) {
	kvs = kvs[:0]
	pos++ // skip '{'

	pos = skipWhitespace(data, pos)
	if pos < len(data) && data[pos] == '}' {
		return kvs, pos + 1
	}

	for pos < len(data) {
		pos = skipWhitespace(data, pos)
		if pos >= len(data) || data[pos] != '"' {
			break
		}

		// Scan key (keep raw bytes).
		keyRaw, nextPos := scanString(data, pos)
		pos = nextPos

		// Skip ':'.
		pos = skipWhitespace(data, pos)
		if pos < len(data) && data[pos] == ':' {
			pos++
		}
		pos = skipWhitespace(data, pos)

		// Scan value.
		valRaw, nextPos2 := scanValue(data, pos)
		pos = nextPos2

		kvs = append(kvs, rawKV{keyRaw: keyRaw, value: valRaw})

		// Skip ',' or break at '}'.
		pos = skipWhitespace(data, pos)
		if pos < len(data) && data[pos] == ',' {
			pos++
		} else {
			break
		}
	}

	// Skip closing '}'.
	pos = skipWhitespace(data, pos)
	if pos < len(data) && data[pos] == '}' {
		pos++
	}

	return kvs, pos
}

// keyInner returns the inner bytes of a quoted key (without quotes).
// Assumes keyRaw starts and ends with '"'.
func keyInner(keyRaw []byte) []byte {
	if len(keyRaw) >= 2 {
		return keyRaw[1 : len(keyRaw)-1]
	}
	return keyRaw
}

// lookupKeyIndex finds the column index for a raw key.
// First tries byte comparison against known column key bytes (no alloc),
// falls back to string conversion + map lookup for new columns.
func lookupKeyIndex(keyRaw []byte, colKeyBytes [][]byte, colIndex map[string]int) (int, bool) {
	inner := keyInner(keyRaw)
	for i, kb := range colKeyBytes {
		if bytes.Equal(inner, kb) {
			return i, true
		}
	}
	// Key not found in known columns - need string conversion.
	key := unquoteJSON(keyRaw)
	idx, ok := colIndex[key]
	return idx, ok
}

// Read reads JSON data (array of objects) from an io.Reader.
// Uses a byte-level scanner to avoid per-row map allocations.
func Read(r io.Reader, opts ...ReadOption) ([]*series.Series, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("golars: json: read error: %w", err)
	}

	pos := skipWhitespace(data, 0)
	if pos >= len(data) || data[pos] != '[' {
		return nil, fmt.Errorf("golars: json: expected '[' at position %d", pos)
	}
	pos++ // skip '['

	var colNames []string
	var colKeyBytes [][]byte // inner key bytes for each column (no quotes)
	colIndex := make(map[string]int)
	var builders []*columnBuilder
	rowCount := 0

	// Reusable key-value buffer.
	kvBuf := make([]rawKV, 0, 16)

	// Bitmap to track which columns got a value in the current row.
	var seen []bool

	for {
		pos = skipWhitespace(data, pos)
		if pos >= len(data) || data[pos] == ']' {
			break
		}
		if data[pos] == ',' {
			pos++
			pos = skipWhitespace(data, pos)
			if pos >= len(data) || data[pos] == ']' {
				break
			}
		}

		if data[pos] != '{' {
			return nil, fmt.Errorf("golars: json: expected '{' at position %d", pos)
		}

		kvBuf, pos = scanObjectRaw(data, pos, kvBuf)

		if rowCount == 0 {
			// First row: discover columns in sorted order.
			type keyPair struct {
				name string
				raw  []byte
			}
			pairs := make([]keyPair, len(kvBuf))
			for i := range kvBuf {
				name := unquoteJSON(kvBuf[i].keyRaw)
				pairs[i] = keyPair{name: name, raw: append([]byte(nil), keyInner(kvBuf[i].keyRaw)...)}
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })

			colNames = make([]string, len(pairs))
			colKeyBytes = make([][]byte, len(pairs))
			builders = make([]*columnBuilder, len(pairs))
			for i, p := range pairs {
				colNames[i] = p.name
				colKeyBytes[i] = p.raw
				colIndex[p.name] = i
				builders[i] = newColumnBuilder(p.name, 256)
			}
			seen = make([]bool, len(pairs))
		}

		// Check for new columns.
		for i := range kvBuf {
			if _, found := lookupKeyIndex(kvBuf[i].keyRaw, colKeyBytes, colIndex); !found {
				name := unquoteJSON(kvBuf[i].keyRaw)
				idx := len(colNames)
				colIndex[name] = idx
				colNames = append(colNames, name)
				colKeyBytes = append(colKeyBytes, append([]byte(nil), keyInner(kvBuf[i].keyRaw)...))
				cb := newColumnBuilder(name, rowCount+256)
				cb.backfillNulls(rowCount)
				builders = append(builders, cb)
				seen = append(seen, false)
			}
		}

		// Reset seen bitmap.
		for i := range seen {
			seen[i] = false
		}

		// Append values from this row.
		for i := range kvBuf {
			idx, _ := lookupKeyIndex(kvBuf[i].keyRaw, colKeyBytes, colIndex)
			seen[idx] = true
			builders[idx].appendRawValue(kvBuf[i].value)
		}

		// Append nulls for missing columns.
		for i := range builders {
			if !seen[i] {
				builders[i].appendNull()
			}
		}

		rowCount++
	}

	if rowCount == 0 {
		return nil, nil
	}

	result := make([]*series.Series, len(builders))
	for i, cb := range builders {
		result[i] = cb.toSeries()
	}
	return result, nil
}

// ReadNDJSONFile reads a newline-delimited JSON file into Series columns.
func ReadNDJSONFile(path string, opts ...ReadOption) ([]*series.Series, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("golars: json: %w", err)
	}
	defer f.Close()
	return ReadNDJSON(f, opts...)
}

// ReadNDJSON reads newline-delimited JSON from an io.Reader.
func ReadNDJSON(r io.Reader, opts ...ReadOption) ([]*series.Series, error) {
	allData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("golars: ndjson: read error: %w", err)
	}

	var colNames []string
	var colKeyBytes [][]byte
	colIndex := make(map[string]int)
	var builders []*columnBuilder
	rowCount := 0
	kvBuf := make([]rawKV, 0, 16)
	var seen []bool

	// Scan line by line through the byte data.
	pos := 0
	for pos < len(allData) {
		// Find end of line.
		lineEnd := pos
		for lineEnd < len(allData) && allData[lineEnd] != '\n' {
			lineEnd++
		}
		line := allData[pos:lineEnd]
		if lineEnd < len(allData) {
			pos = lineEnd + 1 // skip '\n'
		} else {
			pos = lineEnd
		}

		// Trim \r if present.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) == 0 {
			continue
		}

		lpos := skipWhitespace(line, 0)
		if lpos >= len(line) || line[lpos] != '{' {
			return nil, fmt.Errorf("golars: ndjson: expected '{' at position %d", lpos)
		}

		kvBuf, _ = scanObjectRaw(line, lpos, kvBuf)

		if rowCount == 0 {
			type keyPair struct {
				name string
				raw  []byte
			}
			pairs := make([]keyPair, len(kvBuf))
			for i := range kvBuf {
				name := unquoteJSON(kvBuf[i].keyRaw)
				pairs[i] = keyPair{name: name, raw: append([]byte(nil), keyInner(kvBuf[i].keyRaw)...)}
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })

			colNames = make([]string, len(pairs))
			colKeyBytes = make([][]byte, len(pairs))
			builders = make([]*columnBuilder, len(pairs))
			for i, p := range pairs {
				colNames[i] = p.name
				colKeyBytes[i] = p.raw
				colIndex[p.name] = i
				builders[i] = newColumnBuilder(p.name, 256)
			}
			seen = make([]bool, len(pairs))
		}

		for i := range kvBuf {
			if _, found := lookupKeyIndex(kvBuf[i].keyRaw, colKeyBytes, colIndex); !found {
				name := unquoteJSON(kvBuf[i].keyRaw)
				idx := len(colNames)
				colIndex[name] = idx
				colNames = append(colNames, name)
				colKeyBytes = append(colKeyBytes, append([]byte(nil), keyInner(kvBuf[i].keyRaw)...))
				cb := newColumnBuilder(name, rowCount+256)
				cb.backfillNulls(rowCount)
				builders = append(builders, cb)
				seen = append(seen, false)
			}
		}

		for i := range seen {
			seen[i] = false
		}
		for i := range kvBuf {
			idx, _ := lookupKeyIndex(kvBuf[i].keyRaw, colKeyBytes, colIndex)
			seen[idx] = true
			builders[idx].appendRawValue(kvBuf[i].value)
		}
		for i := range builders {
			if !seen[i] {
				builders[i].appendNull()
			}
		}

		rowCount++
	}

	if rowCount == 0 {
		return nil, nil
	}

	result := make([]*series.Series, len(builders))
	for i, cb := range builders {
		result[i] = cb.toSeries()
	}
	return result, nil
}
