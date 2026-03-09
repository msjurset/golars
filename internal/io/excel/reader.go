// Package excel provides XLSX reading and writing for Golars DataFrames.
package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// XML struct types for parsing XLSX shared strings.

type xlsxSST struct {
	XMLName xml.Name `xml:"sst"`
	SI      []xlsxSI `xml:"si"`
}

type xlsxSI struct {
	T string `xml:"t"`
}

// XML struct types for parsing XLSX worksheet data.

type xlsxWorksheet struct {
	XMLName   xml.Name      `xml:"worksheet"`
	SheetData xlsxSheetData `xml:"sheetData"`
}

type xlsxSheetData struct {
	Rows []xlsxRow `xml:"row"`
}

type xlsxRow struct {
	R     int        `xml:"r,attr"`
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	R string `xml:"r,attr"` // e.g. "A1"
	T string `xml:"t,attr"` // type: "s", "n", "b", "inlineStr"
	V string `xml:"v"`      // value
}

// ReadFile reads an Excel .xlsx file and returns a DataFrame.
// By default reads the first worksheet. The first row is treated as headers.
func ReadFile(path string) (*dataframe.DataFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("golars: excel: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("golars: excel: %w", err)
	}

	return Read(f, info.Size())
}

// Read reads an Excel file from an io.ReaderAt with the given size.
func Read(r io.ReaderAt, size int64) (*dataframe.DataFrame, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("golars: excel: open zip: %w", err)
	}

	// Parse shared strings.
	sharedStrings, err := parseSharedStrings(zr)
	if err != nil {
		return nil, err
	}

	// Parse the first worksheet.
	ws, err := parseWorksheet(zr)
	if err != nil {
		return nil, err
	}

	if len(ws.Rows) == 0 {
		return dataframe.New()
	}

	// First row is the header.
	headerRow := ws.Rows[0]
	numCols := 0
	colNames := make(map[int]string)
	for _, cell := range headerRow.Cells {
		colIdx := colLetterToIndex(cellCol(cell.R))
		val := cellValue(cell, sharedStrings)
		colNames[colIdx] = val
		if colIdx+1 > numCols {
			numCols = colIdx + 1
		}
	}

	// Build ordered column name list.
	headers := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		if name, ok := colNames[i]; ok {
			headers[i] = name
		} else {
			headers[i] = fmt.Sprintf("column_%d", i)
		}
	}

	// Read data rows (everything after the first row).
	dataRows := ws.Rows[1:]
	nRows := len(dataRows)

	// Collect raw string values per column.
	colData := make([][]string, numCols)
	for i := range colData {
		colData[i] = make([]string, nRows)
	}

	for rowIdx, row := range dataRows {
		for _, cell := range row.Cells {
			colIdx := colLetterToIndex(cellCol(cell.R))
			if colIdx < numCols {
				colData[colIdx][rowIdx] = cellValue(cell, sharedStrings)
			}
		}
	}

	// Infer types and build Series for each column.
	cols := make([]*series.Series, numCols)
	for i := 0; i < numCols; i++ {
		cols[i] = buildColumn(headers[i], colData[i])
	}

	return dataframe.New(cols...)
}

// cellValue resolves the string value of a cell.
func cellValue(cell xlsxCell, sharedStrings []string) string {
	switch cell.T {
	case "s":
		// Shared string reference.
		idx, err := strconv.Atoi(cell.V)
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return cell.V
		}
		return sharedStrings[idx]
	case "b":
		return cell.V
	case "inlineStr":
		return cell.V
	default:
		// Number or untyped.
		return cell.V
	}
}

// buildColumn infers the type and creates a Series from raw string values.
func buildColumn(name string, values []string) *series.Series {
	dt := inferColumnType(values)

	switch dt {
	case dtype.Int64:
		data := make([]int64, len(values))
		for i, v := range values {
			if v == "" {
				continue
			}
			parsed, _ := strconv.ParseInt(v, 10, 64)
			data[i] = parsed
		}
		return series.NewInt64(name, data)

	case dtype.Float64:
		data := make([]float64, len(values))
		for i, v := range values {
			if v == "" {
				continue
			}
			parsed, _ := strconv.ParseFloat(v, 64)
			data[i] = parsed
		}
		return series.NewFloat64(name, data)

	default:
		return series.NewString(name, values)
	}
}

// inferColumnType checks all values to determine the best type.
func inferColumnType(values []string) dtype.DataType {
	canInt := true
	canFloat := true
	seenNonEmpty := false

	for _, v := range values {
		if v == "" {
			continue
		}
		seenNonEmpty = true

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
		if !canFloat {
			break
		}
	}

	if !seenNonEmpty {
		return dtype.String
	}
	if canInt {
		return dtype.Int64
	}
	if canFloat {
		return dtype.Float64
	}
	return dtype.String
}

// parseSharedStrings reads the shared string table from the ZIP archive.
func parseSharedStrings(zr *zip.Reader) ([]string, error) {
	f := findFile(zr, "xl/sharedStrings.xml")
	if f == nil {
		return nil, nil // No shared strings is valid.
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("golars: excel: open shared strings: %w", err)
	}
	defer rc.Close()

	var sst xlsxSST
	if err := xml.NewDecoder(rc).Decode(&sst); err != nil {
		return nil, fmt.Errorf("golars: excel: parse shared strings: %w", err)
	}

	result := make([]string, len(sst.SI))
	for i, si := range sst.SI {
		result[i] = si.T
	}
	return result, nil
}

// parseWorksheet reads the first worksheet from the ZIP archive.
func parseWorksheet(zr *zip.Reader) (*xlsxSheetData, error) {
	f := findFile(zr, "xl/worksheets/sheet1.xml")
	if f == nil {
		return nil, fmt.Errorf("golars: excel: worksheet sheet1.xml not found")
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("golars: excel: open worksheet: %w", err)
	}
	defer rc.Close()

	var ws xlsxWorksheet
	if err := xml.NewDecoder(rc).Decode(&ws); err != nil {
		return nil, fmt.Errorf("golars: excel: parse worksheet: %w", err)
	}

	return &ws.SheetData, nil
}

// findFile locates a file in the ZIP archive by name.
func findFile(zr *zip.Reader, name string) *zip.File {
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// colLetterToIndex converts "A" -> 0, "B" -> 1, ..., "Z" -> 25, "AA" -> 26.
func colLetterToIndex(col string) int {
	result := 0
	for _, c := range strings.ToUpper(col) {
		result = result*26 + int(c-'A') + 1
	}
	return result - 1
}

// cellCol extracts the column letters from a cell ref like "A1" -> "A", "AB12" -> "AB".
func cellCol(ref string) string {
	for i, c := range ref {
		if c >= '0' && c <= '9' {
			return ref[:i]
		}
	}
	return ref
}
