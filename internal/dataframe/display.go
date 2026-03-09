package dataframe

import (
	"fmt"
	"strings"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// maxDisplayRows is the threshold above which the display truncates to head/tail.
const maxDisplayRows = 20

// displayHeadTail is the number of rows shown from each end when truncated.
const displayHeadTail = 5

// shortTypeName returns a compact type abbreviation used in table headers.
func shortTypeName(dt dtype.DataType) string {
	switch dt {
	case dtype.Null:
		return "null"
	case dtype.Boolean:
		return "bool"
	case dtype.Int8:
		return "i8"
	case dtype.Int16:
		return "i16"
	case dtype.Int32:
		return "i32"
	case dtype.Int64:
		return "i64"
	case dtype.UInt8:
		return "u8"
	case dtype.UInt16:
		return "u16"
	case dtype.UInt32:
		return "u32"
	case dtype.UInt64:
		return "u64"
	case dtype.Float32:
		return "f32"
	case dtype.Float64:
		return "f64"
	case dtype.String:
		return "str"
	case dtype.Date:
		return "date"
	case dtype.DateTime:
		return "datetime"
	case dtype.Time:
		return "time"
	case dtype.Duration:
		return "duration"
	case dtype.Binary:
		return "binary"
	case dtype.Decimal:
		return "decimal"
	default:
		return dt.String()
	}
}

// formatValue formats the value at row i in the given series for display.
func formatValue(s *series.Series, i int) string {
	if s.IsNull(i) {
		return "null"
	}
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
		return fmt.Sprintf("%t", v)
	default:
		return "?"
	}
}

// formatTable renders the DataFrame as a Polars-style ASCII table.
func formatTable(df *DataFrame) string {
	if df.Width() == 0 {
		return fmt.Sprintf("shape: (%d, 0)\n┌┐\n└┘", df.height)
	}

	// Determine which rows to display.
	truncated := df.height > maxDisplayRows
	var rowIndices []int
	if !truncated {
		rowIndices = make([]int, df.height)
		for i := range rowIndices {
			rowIndices[i] = i
		}
	} else {
		rowIndices = make([]int, 0, displayHeadTail*2)
		for i := 0; i < displayHeadTail; i++ {
			rowIndices = append(rowIndices, i)
		}
		rowIndices = append(rowIndices, -1) // sentinel for "..."
		for i := df.height - displayHeadTail; i < df.height; i++ {
			rowIndices = append(rowIndices, i)
		}
	}

	w := df.Width()
	// Build cell content: header row (names), separator row (---), type row, then data rows.
	names := make([]string, w)
	seps := make([]string, w)
	types := make([]string, w)
	for j := 0; j < w; j++ {
		col := df.columns[j]
		names[j] = col.Name()
		seps[j] = "---"
		types[j] = shortTypeName(col.DataType())
	}

	dataRows := make([][]string, len(rowIndices))
	for ri, idx := range rowIndices {
		row := make([]string, w)
		if idx < 0 {
			for j := range row {
				row[j] = "..."
			}
		} else {
			for j := 0; j < w; j++ {
				row[j] = formatValue(df.columns[j], idx)
			}
		}
		dataRows[ri] = row
	}

	// Compute column widths.
	colWidths := make([]int, w)
	for j := 0; j < w; j++ {
		colWidths[j] = max(len(names[j]), len(seps[j]), len(types[j]))
		for _, row := range dataRows {
			if len(row[j]) > colWidths[j] {
				colWidths[j] = len(row[j])
			}
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("shape: (%d, %d)\n", df.height, w))

	// Top border.
	b.WriteString(topBorder(colWidths))
	// Header names.
	b.WriteString(dataRow(names, colWidths))
	// Separator.
	b.WriteString(dataRow(seps, colWidths))
	// Types.
	b.WriteString(dataRow(types, colWidths))
	// Header/data separator.
	b.WriteString(midBorder(colWidths))
	// Data rows.
	for _, row := range dataRows {
		b.WriteString(dataRow(row, colWidths))
	}
	// Bottom border.
	b.WriteString(bottomBorder(colWidths))

	return b.String()
}

// topBorder draws the top line: ┌─────┬─────┐
func topBorder(widths []int) string {
	var b strings.Builder
	for i, w := range widths {
		if i == 0 {
			b.WriteString("\u250c")
		} else {
			b.WriteString("\u252c")
		}
		b.WriteString(strings.Repeat("\u2500", w+2))
	}
	b.WriteString("\u2510\n")
	return b.String()
}

// midBorder draws the header/data separator: ╞═════╪═════╡
func midBorder(widths []int) string {
	var b strings.Builder
	for i, w := range widths {
		if i == 0 {
			b.WriteString("\u255e")
		} else {
			b.WriteString("\u256a")
		}
		b.WriteString(strings.Repeat("\u2550", w+2))
	}
	b.WriteString("\u2561\n")
	return b.String()
}

// bottomBorder draws the bottom line: └─────┴─────┘
func bottomBorder(widths []int) string {
	var b strings.Builder
	for i, w := range widths {
		if i == 0 {
			b.WriteString("\u2514")
		} else {
			b.WriteString("\u2534")
		}
		b.WriteString(strings.Repeat("\u2500", w+2))
	}
	b.WriteString("\u2518")
	return b.String()
}

// dataRow draws a row of values: │ val ┆ val │
func dataRow(values []string, widths []int) string {
	var b strings.Builder
	for i, v := range values {
		if i == 0 {
			b.WriteString("\u2502 ")
		} else {
			b.WriteString("\u2506 ")
		}
		b.WriteString(v)
		b.WriteString(strings.Repeat(" ", widths[i]-len(v)))
		b.WriteString(" ")
	}
	b.WriteString("\u2502\n")
	return b.String()
}

// max returns the largest of the given integers.
func max(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
