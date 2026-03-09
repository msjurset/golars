package excel

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/dtype"
)

// WriteFile writes a DataFrame to an Excel .xlsx file.
func WriteFile(path string, df *dataframe.DataFrame) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: excel: %w", err)
	}
	defer f.Close()

	if err := Write(f, df); err != nil {
		return err
	}
	return nil
}

// Write writes a DataFrame as XLSX to an io.Writer.
func Write(w io.Writer, df *dataframe.DataFrame) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Collect all unique strings (headers + string data values).
	stringIndex := make(map[string]int)
	var sharedStrings []string

	addString := func(s string) int {
		if idx, ok := stringIndex[s]; ok {
			return idx
		}
		idx := len(sharedStrings)
		sharedStrings = append(sharedStrings, s)
		stringIndex[s] = idx
		return idx
	}

	// Pre-register all column headers.
	cols := df.Columns()
	for _, col := range cols {
		addString(col.Name())
	}

	// Pre-register all string data values.
	for _, col := range cols {
		if col.DataType() == dtype.String {
			for i := 0; i < col.Len(); i++ {
				if v, ok := col.GetString(i); ok {
					addString(v)
				}
			}
		}
	}

	// Write [Content_Types].xml
	if err := writeZipFile(zw, "[Content_Types].xml", contentTypesXML); err != nil {
		return err
	}

	// Write _rels/.rels
	if err := writeZipFile(zw, "_rels/.rels", relsXML); err != nil {
		return err
	}

	// Write xl/_rels/workbook.xml.rels
	if err := writeZipFile(zw, "xl/_rels/workbook.xml.rels", workbookRelsXML); err != nil {
		return err
	}

	// Write xl/workbook.xml
	if err := writeZipFile(zw, "xl/workbook.xml", workbookXML); err != nil {
		return err
	}

	// Write xl/sharedStrings.xml
	if err := writeSharedStrings(zw, sharedStrings); err != nil {
		return err
	}

	// Write xl/worksheets/sheet1.xml
	if err := writeWorksheet(zw, df, stringIndex); err != nil {
		return err
	}

	return nil
}

// writeZipFile writes a single file into the ZIP archive.
func writeZipFile(zw *zip.Writer, name, content string) error {
	fw, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("golars: excel: create %s: %w", name, err)
	}
	if _, err := io.WriteString(fw, content); err != nil {
		return fmt.Errorf("golars: excel: write %s: %w", name, err)
	}
	return nil
}

// writeSharedStrings writes the xl/sharedStrings.xml file.
func writeSharedStrings(zw *zip.Writer, strings []string) error {
	fw, err := zw.Create("xl/sharedStrings.xml")
	if err != nil {
		return fmt.Errorf("golars: excel: create sharedStrings.xml: %w", err)
	}

	var b stringsBuilder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(fmt.Sprintf(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`,
		len(strings), len(strings)))
	for _, s := range strings {
		b.WriteString("<si><t>")
		b.WriteString(escapeXML(s))
		b.WriteString("</t></si>")
	}
	b.WriteString("</sst>")

	if _, err := io.WriteString(fw, b.String()); err != nil {
		return fmt.Errorf("golars: excel: write sharedStrings.xml: %w", err)
	}
	return nil
}

// writeWorksheet writes the xl/worksheets/sheet1.xml file.
func writeWorksheet(zw *zip.Writer, df *dataframe.DataFrame, stringIndex map[string]int) error {
	fw, err := zw.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return fmt.Errorf("golars: excel: create sheet1.xml: %w", err)
	}

	cols := df.Columns()
	var b stringsBuilder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString("<sheetData>")

	// Row 1: headers
	b.WriteString(`<row r="1">`)
	for colIdx, col := range cols {
		ref := indexToColLetter(colIdx) + "1"
		sIdx := stringIndex[col.Name()]
		b.WriteString(fmt.Sprintf(`<c r="%s" t="s"><v>%d</v></c>`, ref, sIdx))
	}
	b.WriteString("</row>")

	// Data rows
	for rowIdx := 0; rowIdx < df.Height(); rowIdx++ {
		rowNum := rowIdx + 2 // 1-based, after header
		b.WriteString(fmt.Sprintf(`<row r="%d">`, rowNum))

		for colIdx, col := range cols {
			ref := indexToColLetter(colIdx) + strconv.Itoa(rowNum)

			if col.IsNull(rowIdx) {
				continue // skip null cells
			}

			switch col.DataType() {
			case dtype.Int64:
				v, _ := col.GetInt64(rowIdx)
				b.WriteString(fmt.Sprintf(`<c r="%s"><v>%d</v></c>`, ref, v))
			case dtype.Int32, dtype.Int16, dtype.Int8:
				v, _ := col.GetInt64(rowIdx)
				b.WriteString(fmt.Sprintf(`<c r="%s"><v>%d</v></c>`, ref, v))
			case dtype.Float64, dtype.Float32:
				v, _ := col.GetFloat64(rowIdx)
				b.WriteString(fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, ref, strconv.FormatFloat(v, 'f', -1, 64)))
			case dtype.Boolean:
				v, _ := col.GetBool(rowIdx)
				bv := "0"
				if v {
					bv = "1"
				}
				b.WriteString(fmt.Sprintf(`<c r="%s" t="b"><v>%s</v></c>`, ref, bv))
			case dtype.String:
				v, _ := col.GetString(rowIdx)
				sIdx := stringIndex[v]
				b.WriteString(fmt.Sprintf(`<c r="%s" t="s"><v>%d</v></c>`, ref, sIdx))
			default:
				// Fall back to string representation.
				v, _ := col.GetString(rowIdx)
				if v != "" {
					sIdx := stringIndex[v]
					b.WriteString(fmt.Sprintf(`<c r="%s" t="s"><v>%d</v></c>`, ref, sIdx))
				}
			}
		}

		b.WriteString("</row>")
	}

	b.WriteString("</sheetData>")
	b.WriteString("</worksheet>")

	if _, err := io.WriteString(fw, b.String()); err != nil {
		return fmt.Errorf("golars: excel: write sheet1.xml: %w", err)
	}
	return nil
}

// indexToColLetter converts 0 -> "A", 1 -> "B", ..., 25 -> "Z", 26 -> "AA".
func indexToColLetter(idx int) string {
	var result []byte
	for idx >= 0 {
		result = append([]byte{byte('A' + idx%26)}, result...)
		idx = idx/26 - 1
	}
	return string(result)
}

// escapeXML escapes special XML characters in a string.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// stringsBuilder is a simple wrapper around strings.Builder for convenience.
type stringsBuilder struct {
	strings.Builder
}

// Static XML content for required XLSX files.

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbookRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`

const workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sheet1" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`
