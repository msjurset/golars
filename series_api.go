package golars

import (
	"time"

	"github.com/msjurset/golars/internal/series"
)

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

// NewDateSeries creates a new Series of Date values (days since Unix epoch as int32).
func NewDateSeries(name string, data []int32) *Series { return series.NewDate(name, data) }

// NewDateTimeSeries creates a new Series of DateTime values (microseconds since epoch as int64).
func NewDateTimeSeries(name string, data []int64) *Series { return series.NewDateTime(name, data) }

// NewTimeSeries creates a new Series of Time values (nanoseconds since midnight as int64).
func NewTimeSeries(name string, data []int64) *Series { return series.NewTime(name, data) }

// NewDurationSeries creates a new Series of Duration values (microseconds as int64).
func NewDurationSeries(name string, data []int64) *Series { return series.NewDuration(name, data) }

// NewDateSeriesFromTime creates a Date Series from a slice of time.Time values.
func NewDateSeriesFromTime(name string, times []time.Time) *Series {
	data := make([]int32, len(times))
	for i, t := range times {
		data[i] = int32(t.Unix() / 86400)
	}
	return series.NewDate(name, data)
}

// NewDateTimeSeriesFromTime creates a DateTime Series from a slice of time.Time values.
func NewDateTimeSeriesFromTime(name string, times []time.Time) *Series {
	data := make([]int64, len(times))
	for i, t := range times {
		data[i] = t.UnixMicro()
	}
	return series.NewDateTime(name, data)
}
