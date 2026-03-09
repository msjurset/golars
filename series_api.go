package golars

import "github.com/msjurseth/golars/internal/series"

// NewInt8Series creates a new Series of int8 values.
func NewInt8Series(name string, data []int8) *Series {
	return series.NewInt8(name, data)
}

// NewInt16Series creates a new Series of int16 values.
func NewInt16Series(name string, data []int16) *Series {
	return series.NewInt16(name, data)
}

// NewInt32Series creates a new Series of int32 values.
func NewInt32Series(name string, data []int32) *Series {
	return series.NewInt32(name, data)
}

// NewInt64Series creates a new Series of int64 values.
func NewInt64Series(name string, data []int64) *Series {
	return series.NewInt64(name, data)
}

// NewUInt8Series creates a new Series of uint8 values.
func NewUInt8Series(name string, data []uint8) *Series {
	return series.NewUInt8(name, data)
}

// NewUInt16Series creates a new Series of uint16 values.
func NewUInt16Series(name string, data []uint16) *Series {
	return series.NewUInt16(name, data)
}

// NewUInt32Series creates a new Series of uint32 values.
func NewUInt32Series(name string, data []uint32) *Series {
	return series.NewUInt32(name, data)
}

// NewUInt64Series creates a new Series of uint64 values.
func NewUInt64Series(name string, data []uint64) *Series {
	return series.NewUInt64(name, data)
}

// NewFloat32Series creates a new Series of float32 values.
func NewFloat32Series(name string, data []float32) *Series {
	return series.NewFloat32(name, data)
}

// NewFloat64Series creates a new Series of float64 values.
func NewFloat64Series(name string, data []float64) *Series {
	return series.NewFloat64(name, data)
}

// NewBooleanSeries creates a new Series of boolean values.
func NewBooleanSeries(name string, data []bool) *Series {
	return series.NewBoolean(name, data)
}

// NewStringSeries creates a new Series of string values.
func NewStringSeries(name string, data []string) *Series {
	return series.NewString(name, data)
}

// NewInt64SeriesWithValidity creates an Int64 Series with explicit null tracking.
// Positions where valid[i] is false are marked as null.
func NewInt64SeriesWithValidity(name string, data []int64, valid []bool) *Series {
	return series.NewInt64WithValidity(name, data, valid)
}

// NewFloat64SeriesWithValidity creates a Float64 Series with explicit null tracking.
func NewFloat64SeriesWithValidity(name string, data []float64, valid []bool) *Series {
	return series.NewFloat64WithValidity(name, data, valid)
}

// NewStringSeriesWithValidity creates a String Series with explicit null tracking.
func NewStringSeriesWithValidity(name string, data []string, valid []bool) *Series {
	return series.NewStringWithValidity(name, data, valid)
}

// NewBooleanSeriesWithValidity creates a Boolean Series with explicit null tracking.
func NewBooleanSeriesWithValidity(name string, data []bool, valid []bool) *Series {
	return series.NewBooleanWithValidity(name, data, valid)
}
