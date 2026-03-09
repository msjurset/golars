package csv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

// WriteOptions configures CSV writing behavior.
type WriteOptions struct {
	Separator rune
	Quote     rune
	NullValue string
	HasHeader bool
	QuoteAll  bool // quote all fields, not just those needing it
}

// DefaultWriteOptions returns sensible defaults for CSV writing.
func DefaultWriteOptions() WriteOptions {
	return WriteOptions{
		Separator: ',',
		Quote:     '"',
		NullValue: "",
		HasHeader: true,
	}
}

// WriteOption is a functional option for CSV writing.
type WriteOption func(*WriteOptions)

// WriteSeparator sets the field separator for writing.
func WriteSeparator(sep rune) WriteOption {
	return func(o *WriteOptions) { o.Separator = sep }
}

// WriteNullValue sets the string representation of null values.
func WriteNullValue(v string) WriteOption {
	return func(o *WriteOptions) { o.NullValue = v }
}

// WriteHasHeader controls whether the header row is written.
func WriteHasHeader(has bool) WriteOption {
	return func(o *WriteOptions) { o.HasHeader = has }
}

// WriteFile writes Series columns to a CSV file.
func WriteFile(path string, columns []*series.Series, opts ...WriteOption) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: csv: %w", err)
	}
	defer f.Close()
	return Write(f, columns, opts...)
}

// Write writes Series columns as CSV to an io.Writer.
func Write(w io.Writer, columns []*series.Series, opts ...WriteOption) error {
	options := DefaultWriteOptions()
	for _, o := range opts {
		o(&options)
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	sep := string(options.Separator)

	if len(columns) == 0 {
		return nil
	}

	height := columns[0].Len()

	// Write header
	if options.HasHeader {
		names := make([]string, len(columns))
		for i, c := range columns {
			names[i] = quoteField(c.Name(), options.Separator, options.Quote, options.QuoteAll)
		}
		_, err := bw.WriteString(strings.Join(names, sep) + "\n")
		if err != nil {
			return fmt.Errorf("golars: csv: write header: %w", err)
		}
	}

	// Write data rows
	for row := 0; row < height; row++ {
		for col, c := range columns {
			if col > 0 {
				bw.WriteString(sep)
			}
			if c.IsNull(row) {
				bw.WriteString(options.NullValue)
				continue
			}
			val := formatSeriesValue(c, row)
			bw.WriteString(quoteField(val, options.Separator, options.Quote, options.QuoteAll))
		}
		bw.WriteString("\n")
	}

	return bw.Flush()
}

// formatSeriesValue formats a single value from a Series for CSV output.
func formatSeriesValue(s *series.Series, i int) string {
	switch s.DataType() {
	case dtype.Int64:
		v, _ := s.GetInt64(i)
		return fmt.Sprintf("%d", v)
	case dtype.Float64:
		v, _ := s.GetFloat64(i)
		return fmt.Sprintf("%g", v)
	case dtype.String:
		v, _ := s.GetString(i)
		return v
	case dtype.Boolean:
		v, _ := s.GetBool(i)
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// quoteField quotes a field if it contains the separator, quote character,
// or newline.
func quoteField(s string, sep, quote rune, quoteAll bool) string {
	needsQuote := quoteAll
	if !needsQuote {
		for _, r := range s {
			if r == sep || r == quote || r == '\n' || r == '\r' {
				needsQuote = true
				break
			}
		}
	}
	if !needsQuote {
		return s
	}
	q := string(quote)
	escaped := strings.ReplaceAll(s, q, q+q)
	return q + escaped + q
}
