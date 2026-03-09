package dataframe

import (
	"fmt"
	"strings"

	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// Describe returns a summary statistics DataFrame for all numeric columns.
// The result has rows for count, mean, std, min, and max, with a "statistic"
// string column as the first column.
func (df *DataFrame) Describe() *DataFrame {
	stats := []string{"count", "mean", "std", "min", "max"}

	// Build the statistic label column.
	statCol := series.NewString("statistic", stats)

	cols := []*series.Series{statCol}
	for _, c := range df.columns {
		if !dtype.IsNumeric(c.DataType()) {
			continue
		}

		values := make([]float64, len(stats))
		values[0] = float64(c.Count())
		if m, ok := c.Mean(); ok {
			values[1] = m
		}
		if s, ok := c.Std(); ok {
			values[2] = s
		}
		if m, ok := c.Min(); ok {
			values[3] = m
		}
		if m, ok := c.Max(); ok {
			values[4] = m
		}
		cols = append(cols, series.NewFloat64(c.Name(), values))
	}

	// New cannot fail here: columns are freshly created with matching lengths
	// and unique names, so the error is unreachable.
	result, _ := New(cols...)
	return result
}

// Glimpse returns a compact overview of the DataFrame showing column names,
// types, and the first few values of each column.
func (df *DataFrame) Glimpse() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Rows: %d\n", df.height))
	b.WriteString(fmt.Sprintf("Columns: %d\n", df.Width()))

	maxNameLen := 0
	maxTypeLen := 0
	for _, c := range df.columns {
		if len(c.Name()) > maxNameLen {
			maxNameLen = len(c.Name())
		}
		tn := shortTypeName(c.DataType())
		if len(tn) > maxTypeLen {
			maxTypeLen = len(tn)
		}
	}

	previewCount := 5
	if df.height < previewCount {
		previewCount = df.height
	}

	for _, c := range df.columns {
		name := c.Name()
		tn := shortTypeName(c.DataType())
		padding := strings.Repeat(" ", maxNameLen-len(name))
		typePadding := strings.Repeat(" ", maxTypeLen-len(tn))

		vals := make([]string, previewCount)
		for i := 0; i < previewCount; i++ {
			vals[i] = formatValue(c, i)
		}
		preview := strings.Join(vals, ", ")
		if df.height > previewCount {
			preview += ", ..."
		}

		b.WriteString(fmt.Sprintf("$ %s%s <%s>%s %s\n", name, padding, tn, typePadding, preview))
	}
	return b.String()
}
