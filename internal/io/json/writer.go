package json

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// WriteFile writes Series columns as a JSON array-of-objects to a file.
func WriteFile(path string, columns []*series.Series) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: json: %w", err)
	}
	defer f.Close()
	return Write(f, columns)
}

// Write writes Series columns as a JSON array of objects to an io.Writer.
func Write(w io.Writer, columns []*series.Series) error {
	if len(columns) == 0 {
		_, err := w.Write([]byte("[]\n"))
		return err
	}

	height := columns[0].Len()
	rows := make([]map[string]any, height)
	for i := 0; i < height; i++ {
		row := make(map[string]any, len(columns))
		for _, c := range columns {
			if c.IsNull(i) {
				row[c.Name()] = nil
				continue
			}
			switch c.DataType() {
			case dtype.Int64:
				v, _ := c.GetInt64(i)
				row[c.Name()] = v
			case dtype.Float64:
				v, _ := c.GetFloat64(i)
				row[c.Name()] = v
			case dtype.String:
				v, _ := c.GetString(i)
				row[c.Name()] = v
			case dtype.Boolean:
				v, _ := c.GetBool(i)
				row[c.Name()] = v
			default:
				row[c.Name()] = nil
			}
		}
		rows[i] = row
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// WriteNDJSONFile writes Series columns as NDJSON to a file.
func WriteNDJSONFile(path string, columns []*series.Series) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: ndjson: %w", err)
	}
	defer f.Close()
	return WriteNDJSON(f, columns)
}

// WriteNDJSON writes Series columns as newline-delimited JSON to an io.Writer.
func WriteNDJSON(w io.Writer, columns []*series.Series) error {
	if len(columns) == 0 {
		return nil
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	height := columns[0].Len()
	enc := json.NewEncoder(bw)

	for i := 0; i < height; i++ {
		row := make(map[string]any, len(columns))
		for _, c := range columns {
			if c.IsNull(i) {
				row[c.Name()] = nil
				continue
			}
			switch c.DataType() {
			case dtype.Int64:
				v, _ := c.GetInt64(i)
				row[c.Name()] = v
			case dtype.Float64:
				v, _ := c.GetFloat64(i)
				row[c.Name()] = v
			case dtype.String:
				v, _ := c.GetString(i)
				row[c.Name()] = v
			case dtype.Boolean:
				v, _ := c.GetBool(i)
				row[c.Name()] = v
			default:
				row[c.Name()] = nil
			}
		}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("golars: ndjson: %w", err)
		}
	}
	return bw.Flush()
}
