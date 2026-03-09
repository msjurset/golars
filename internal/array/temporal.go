package array

import (
	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dtype"
)

// DateArray stores date values as int32 (days since Unix epoch 1970-01-01).
type DateArray = TypedArray[int32]

// NewDateArray creates a new DateArray.
func NewDateArray(data []int32, validity *bitmap.Bitmap) *DateArray {
	return NewTypedArray(data, dtype.Date, validity)
}

// DateTimeArray stores datetime values as int64 (microseconds since Unix epoch by default).
type DateTimeArray = TypedArray[int64]

// NewDateTimeArray creates a new DateTimeArray with the given time unit.
func NewDateTimeArray(data []int64, validity *bitmap.Bitmap) *DateTimeArray {
	return NewTypedArray(data, dtype.DateTime, validity)
}

// TimeArray stores time-of-day values as int64 (nanoseconds since midnight).
type TimeArray = TypedArray[int64]

// NewTimeArray creates a new TimeArray.
func NewTimeArray(data []int64, validity *bitmap.Bitmap) *TimeArray {
	return NewTypedArray(data, dtype.Time, validity)
}

// DurationArray stores duration values as int64 (microseconds by default).
type DurationArray = TypedArray[int64]

// NewDurationArray creates a new DurationArray.
func NewDurationArray(data []int64, validity *bitmap.Bitmap) *DurationArray {
	return NewTypedArray(data, dtype.Duration, validity)
}
