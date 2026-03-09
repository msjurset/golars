package golars

import (
	"database/sql"
	"io"

	csvio "github.com/msjurset/golars/internal/io/csv"
	dbio "github.com/msjurset/golars/internal/io/database"
	excelio "github.com/msjurset/golars/internal/io/excel"
	jsonio "github.com/msjurset/golars/internal/io/json"
	parquetio "github.com/msjurset/golars/internal/io/parquet"
)

// CSV read options re-exported as top-level functions.

// ReadCSVOption is a functional option for CSV reading.
type ReadCSVOption = csvio.ReadOption

// ReadCSV reads a CSV file into a DataFrame.
func ReadCSV(path string, opts ...ReadCSVOption) (*DataFrame, error) {
	cols, err := csvio.ReadFile(path, opts...)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// ReadCSVFromReader reads CSV data from an io.Reader into a DataFrame.
func ReadCSVFromReader(r io.Reader, opts ...ReadCSVOption) (*DataFrame, error) {
	cols, err := csvio.Read(r, opts...)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// WriteCSV writes a DataFrame to a CSV file.
func WriteCSV(df *DataFrame, w io.Writer, opts ...csvio.WriteOption) error {
	return csvio.Write(w, df.Columns(), opts...)
}

// WriteCSVFile writes a DataFrame to a CSV file at the given path.
func WriteCSVFile(df *DataFrame, path string, opts ...csvio.WriteOption) error {
	return csvio.WriteFile(path, df.Columns(), opts...)
}

// CSV option re-exports for convenience.
var (
	// WithSeparator sets the CSV field separator character.
	WithSeparator = csvio.WithSeparator
	// WithNullValues sets the strings treated as null in CSV reading.
	WithNullValues = csvio.WithNullValues
	// WithCSVColumns restricts CSV reading to the named columns.
	WithCSVColumns = csvio.WithColumns
	// WithInferSchemaLength sets how many rows to scan for CSV type inference.
	WithInferSchemaLength = csvio.WithInferSchemaLength
	// WithSkipRows skips the first n rows of a CSV file.
	WithSkipRows = csvio.WithSkipRows
	// WithNRows limits CSV reading to at most n data rows.
	WithNRows = csvio.WithNRows
	// WithHasHeader controls whether the first CSV row is a header.
	WithHasHeader = csvio.WithHasHeader
)

// JSON I/O

// ReadJSON reads a JSON array-of-objects file into a DataFrame.
func ReadJSON(path string) (*DataFrame, error) {
	cols, err := jsonio.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// ReadJSONFromReader reads JSON data from an io.Reader into a DataFrame.
func ReadJSONFromReader(r io.Reader) (*DataFrame, error) {
	cols, err := jsonio.Read(r)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// WriteJSON writes a DataFrame as JSON to an io.Writer.
func WriteJSON(df *DataFrame, w io.Writer) error {
	return jsonio.Write(w, df.Columns())
}

// WriteJSONFile writes a DataFrame as JSON to a file.
func WriteJSONFile(df *DataFrame, path string) error {
	return jsonio.WriteFile(path, df.Columns())
}

// ReadNDJSON reads a newline-delimited JSON file into a DataFrame.
func ReadNDJSON(path string) (*DataFrame, error) {
	cols, err := jsonio.ReadNDJSONFile(path)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// ReadNDJSONFromReader reads NDJSON from an io.Reader into a DataFrame.
func ReadNDJSONFromReader(r io.Reader) (*DataFrame, error) {
	cols, err := jsonio.ReadNDJSON(r)
	if err != nil {
		return nil, err
	}
	return NewDataFrame(cols...)
}

// WriteNDJSON writes a DataFrame as NDJSON to an io.Writer.
func WriteNDJSON(df *DataFrame, w io.Writer) error {
	return jsonio.WriteNDJSON(w, df.Columns())
}

// WriteNDJSONFile writes a DataFrame as NDJSON to a file.
func WriteNDJSONFile(df *DataFrame, path string) error {
	return jsonio.WriteNDJSONFile(path, df.Columns())
}

// Database I/O

// ReadSQL executes a SQL query against a database and returns the result as a DataFrame.
// The caller must import the appropriate database driver (e.g., _ "github.com/mattn/go-sqlite3").
func ReadSQL(db *sql.DB, query string, args ...any) (*DataFrame, error) {
	return dbio.ReadSQL(db, query, args...)
}

// ReadSQLRows converts *sql.Rows to a DataFrame.
func ReadSQLRows(rows *sql.Rows) (*DataFrame, error) {
	return dbio.ReadSQLRows(rows)
}

// Parquet I/O

// ReadParquet reads a Parquet file into a DataFrame.
func ReadParquet(path string) (*DataFrame, error) {
	return parquetio.ReadFile(path)
}

// ReadParquetFromReader reads Parquet data from an io.ReadSeeker into a DataFrame.
func ReadParquetFromReader(r io.ReadSeeker) (*DataFrame, error) {
	return parquetio.Read(r)
}

// WriteParquet writes a DataFrame as Parquet to an io.Writer.
func WriteParquet(df *DataFrame, w io.Writer) error {
	return parquetio.Write(w, df)
}

// WriteParquetFile writes a DataFrame to a Parquet file at the given path.
func WriteParquetFile(df *DataFrame, path string) error {
	return parquetio.WriteFile(path, df)
}

// Excel I/O

// ReadExcel reads an Excel .xlsx file into a DataFrame.
func ReadExcel(path string) (*DataFrame, error) {
	return excelio.ReadFile(path)
}

// ReadExcelFromReader reads an Excel file from an io.ReaderAt with the given size.
func ReadExcelFromReader(r io.ReaderAt, size int64) (*DataFrame, error) {
	return excelio.Read(r, size)
}

// WriteExcel writes a DataFrame as an Excel .xlsx file to an io.Writer.
func WriteExcel(df *DataFrame, w io.Writer) error {
	return excelio.Write(w, df)
}

// WriteExcelFile writes a DataFrame to an Excel .xlsx file at the given path.
func WriteExcelFile(df *DataFrame, path string) error {
	return excelio.WriteFile(path, df)
}
