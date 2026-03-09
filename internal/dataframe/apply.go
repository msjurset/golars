package dataframe

import (
	"fmt"

	"github.com/msjurseth/golars/internal/series"
)

// MapRows applies a function to each row of the DataFrame and returns a new
// Series with the results. The function receives a map of column name to value
// for each row.
func (df *DataFrame) MapRows(name string, fn func(row map[string]any) any) *series.Series {
	n := df.height
	results := make([]any, n)
	for i := 0; i < n; i++ {
		row := df.Row(i)
		results[i] = fn(row)
	}
	return anySliceToSeries(name, results)
}

// MapBatches applies a function to the entire set of columns and returns
// a new Series. This is more efficient than MapRows for operations that
// can work on whole columns at once.
func (df *DataFrame) MapBatches(fn func(columns []*series.Series) *series.Series) (*series.Series, error) {
	result := fn(df.columns)
	if result == nil {
		return nil, fmt.Errorf("golars: map_batches: function returned nil")
	}
	if result.Len() != df.height {
		return nil, fmt.Errorf("golars: map_batches: result length %d does not match DataFrame height %d", result.Len(), df.height)
	}
	return result, nil
}

func anySliceToSeries(name string, values []any) *series.Series {
	if len(values) == 0 {
		return series.NewFloat64(name, nil)
	}

	// Determine type from first non-nil value
	var sampleType string
	for _, v := range values {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int64, int:
			sampleType = "int64"
		case float64:
			sampleType = "float64"
		case string:
			sampleType = "string"
		case bool:
			sampleType = "bool"
		default:
			sampleType = "string"
		}
		break
	}

	if sampleType == "" {
		// All nil
		valid := make([]bool, len(values))
		return series.NewFloat64WithValidity(name, make([]float64, len(values)), valid)
	}

	n := len(values)
	switch sampleType {
	case "int64":
		data := make([]int64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			if v == nil {
				hasNulls = true
				continue
			}
			switch val := v.(type) {
			case int64:
				data[i] = val
			case int:
				data[i] = int64(val)
			}
			valid[i] = true
		}
		if hasNulls {
			return series.NewInt64WithValidity(name, data, valid)
		}
		return series.NewInt64(name, data)

	case "float64":
		data := make([]float64, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			if v == nil {
				hasNulls = true
				continue
			}
			data[i] = v.(float64)
			valid[i] = true
		}
		if hasNulls {
			return series.NewFloat64WithValidity(name, data, valid)
		}
		return series.NewFloat64(name, data)

	case "string":
		data := make([]string, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			if v == nil {
				hasNulls = true
				continue
			}
			data[i] = fmt.Sprintf("%v", v)
			valid[i] = true
		}
		if hasNulls {
			return series.NewStringWithValidity(name, data, valid)
		}
		return series.NewString(name, data)

	case "bool":
		data := make([]bool, n)
		valid := make([]bool, n)
		hasNulls := false
		for i, v := range values {
			if v == nil {
				hasNulls = true
				continue
			}
			data[i] = v.(bool)
			valid[i] = true
		}
		if hasNulls {
			return series.NewBooleanWithValidity(name, data, valid)
		}
		return series.NewBoolean(name, data)

	default:
		data := make([]string, n)
		for i, v := range values {
			data[i] = fmt.Sprintf("%v", v)
		}
		return series.NewString(name, data)
	}
}
