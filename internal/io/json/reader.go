// Package json provides JSON and NDJSON reading and writing for Golars DataFrames.
package json

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/msjurset/golars/internal/dtype"
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

// Read reads JSON data (array of objects) from an io.Reader.
func Read(r io.Reader, opts ...ReadOption) ([]*series.Series, error) {
	var rows []map[string]any
	dec := json.NewDecoder(r)
	if err := dec.Decode(&rows); err != nil {
		return nil, fmt.Errorf("golars: json: decode error: %w", err)
	}
	return rowsToSeries(rows)
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
	var rows []map[string]any
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("golars: ndjson: line parse error: %w", err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("golars: ndjson: scan error: %w", err)
	}
	return rowsToSeries(rows)
}

// rowsToSeries converts a slice of maps into Series columns.
func rowsToSeries(rows []map[string]any) ([]*series.Series, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	// Collect all column names preserving first-seen order.
	seen := make(map[string]struct{})
	var colNames []string
	for _, row := range rows {
		// Sort keys for deterministic order within each row.
		keys := make([]string, 0, len(row))
		for k := range row {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, exists := seen[k]; !exists {
				seen[k] = struct{}{}
				colNames = append(colNames, k)
			}
		}
	}

	// Infer types
	colTypes := make(map[string]dtype.DataType)
	for _, name := range colNames {
		colTypes[name] = inferJSONType(rows, name)
	}

	// Build Series
	result := make([]*series.Series, len(colNames))
	for i, name := range colNames {
		dt := colTypes[name]
		s, err := buildJSONSeries(name, rows, dt)
		if err != nil {
			return nil, err
		}
		result[i] = s
	}
	return result, nil
}

// inferJSONType infers the data type from JSON values.
func inferJSONType(rows []map[string]any, key string) dtype.DataType {
	hasFloat := false
	hasInt := false
	hasBool := false
	hasString := false

	for _, row := range rows {
		v, ok := row[key]
		if !ok || v == nil {
			continue
		}
		switch v.(type) {
		case float64:
			// JSON numbers are float64 by default. Check if it's actually an int.
			f := v.(float64)
			if f == float64(int64(f)) && f >= -1<<53 && f <= 1<<53 {
				hasInt = true
			} else {
				hasFloat = true
			}
		case bool:
			hasBool = true
		case string:
			hasString = true
		default:
			hasString = true
		}
	}

	if hasString {
		return dtype.String
	}
	if hasBool && !hasFloat && !hasInt {
		return dtype.Boolean
	}
	if hasFloat {
		return dtype.Float64
	}
	if hasInt {
		return dtype.Int64
	}
	return dtype.String
}

// buildJSONSeries creates a Series from JSON row data.
func buildJSONSeries(name string, rows []map[string]any, dt dtype.DataType) (*series.Series, error) {
	n := len(rows)

	switch dt {
	case dtype.Int64:
		data := make([]int64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, row := range rows {
			v, ok := row[name]
			if !ok || v == nil {
				hasNulls = true
				continue
			}
			if f, ok := v.(float64); ok {
				data[i] = int64(f)
				valid[i] = true
			} else {
				hasNulls = true
			}
		}
		if hasNulls {
			return series.NewInt64WithValidity(name, data, valid), nil
		}
		return series.NewInt64(name, data), nil

	case dtype.Float64:
		data := make([]float64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, row := range rows {
			v, ok := row[name]
			if !ok || v == nil {
				hasNulls = true
				continue
			}
			if f, ok := v.(float64); ok {
				data[i] = f
				valid[i] = true
			} else {
				hasNulls = true
			}
		}
		if hasNulls {
			return series.NewFloat64WithValidity(name, data, valid), nil
		}
		return series.NewFloat64(name, data), nil

	case dtype.Boolean:
		data := make([]bool, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, row := range rows {
			v, ok := row[name]
			if !ok || v == nil {
				hasNulls = true
				continue
			}
			if b, ok := v.(bool); ok {
				data[i] = b
				valid[i] = true
			} else {
				hasNulls = true
			}
		}
		if hasNulls {
			return series.NewBooleanWithValidity(name, data, valid), nil
		}
		return series.NewBoolean(name, data), nil

	default: // String
		data := make([]string, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, row := range rows {
			v, ok := row[name]
			if !ok || v == nil {
				hasNulls = true
				continue
			}
			if s, ok := v.(string); ok {
				data[i] = s
				valid[i] = true
			} else {
				data[i] = fmt.Sprintf("%v", v)
				valid[i] = true
			}
		}
		if hasNulls {
			return series.NewStringWithValidity(name, data, valid), nil
		}
		return series.NewString(name, data), nil
	}
}
