// Package database provides DataFrame I/O via Go's database/sql interface.
package database

import (
	"database/sql"
	"fmt"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/series"
)

// ReadSQL executes a SQL query against a database connection and returns
// the result as a DataFrame. The caller is responsible for importing the
// appropriate database driver.
func ReadSQL(db *sql.DB, query string, args ...any) (*dataframe.DataFrame, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("golars: database: query: %w", err)
	}
	defer rows.Close()

	return rowsToDataFrame(rows)
}

// ReadSQLRows converts *sql.Rows to a DataFrame.
func ReadSQLRows(rows *sql.Rows) (*dataframe.DataFrame, error) {
	return rowsToDataFrame(rows)
}

func rowsToDataFrame(rows *sql.Rows) (*dataframe.DataFrame, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("golars: database: columns: %w", err)
	}

	nCols := len(colNames)
	data := make([][]any, nCols)
	for i := range data {
		data[i] = make([]any, 0)
	}

	scanDest := make([]any, nCols)
	scanPtrs := make([]any, nCols)
	for i := range scanDest {
		scanPtrs[i] = &scanDest[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, fmt.Errorf("golars: database: scan: %w", err)
		}
		for i, v := range scanDest {
			data[i] = append(data[i], v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("golars: database: rows: %w", err)
	}

	if len(data[0]) == 0 {
		cols := make([]*series.Series, nCols)
		for i, name := range colNames {
			cols[i] = series.NewString(name, nil)
		}
		return dataframe.New(cols...)
	}

	cols := make([]*series.Series, nCols)
	for i, name := range colNames {
		cols[i] = anySliceToSeries(name, data[i])
	}

	return dataframe.New(cols...)
}

func anySliceToSeries(name string, values []any) *series.Series {
	if len(values) == 0 {
		return series.NewString(name, nil)
	}

	// Determine type from first non-nil value.
	var sampleType string
	for _, v := range values {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int64:
			sampleType = "int64"
		case float64:
			sampleType = "float64"
		case string:
			sampleType = "string"
		case bool:
			sampleType = "bool"
		case []byte:
			sampleType = "string"
		default:
			sampleType = "string"
		}
		break
	}

	if sampleType == "" {
		valid := make([]bool, len(values))
		return series.NewStringWithValidity(name, make([]string, len(values)), valid)
	}

	n := len(values)
	switch sampleType {
	case "int64":
		data := make([]int64, n)
		valid := make([]bool, n)
		for i, v := range values {
			if v == nil {
				continue
			}
			data[i] = v.(int64)
			valid[i] = true
		}
		return series.NewInt64WithValidity(name, data, valid)
	case "float64":
		data := make([]float64, n)
		valid := make([]bool, n)
		for i, v := range values {
			if v == nil {
				continue
			}
			data[i] = v.(float64)
			valid[i] = true
		}
		return series.NewFloat64WithValidity(name, data, valid)
	case "bool":
		data := make([]bool, n)
		valid := make([]bool, n)
		for i, v := range values {
			if v == nil {
				continue
			}
			data[i] = v.(bool)
			valid[i] = true
		}
		return series.NewBooleanWithValidity(name, data, valid)
	default:
		data := make([]string, n)
		valid := make([]bool, n)
		for i, v := range values {
			if v == nil {
				continue
			}
			switch val := v.(type) {
			case string:
				data[i] = val
			case []byte:
				data[i] = string(val)
			default:
				data[i] = fmt.Sprintf("%v", v)
			}
			valid[i] = true
		}
		return series.NewStringWithValidity(name, data, valid)
	}
}
