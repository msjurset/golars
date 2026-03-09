package series

import (
	"strconv"
	"strings"
	"time"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
)

// DtAccessor provides temporal operations on Date/DateTime/Time/Duration Series.
type DtAccessor struct {
	s *Series
}

// Dt returns a DtAccessor for temporal operations. Returns nil if the Series
// is not of a temporal type (Date, DateTime, Time, Duration).
func (s *Series) Dt() *DtAccessor {
	switch s.dtype {
	case dtype.Date, dtype.DateTime, dtype.Time, dtype.Duration:
		return &DtAccessor{s: s}
	}
	return nil
}

// dateToTime converts a Date value (days since epoch) to time.Time.
func dateToTime(days int32) time.Time {
	return time.Unix(int64(days)*86400, 0).UTC()
}

// dateTimeToTime converts a DateTime value (microseconds since epoch) to time.Time.
func dateTimeToTime(us int64) time.Time {
	sec := us / 1_000_000
	nsec := (us % 1_000_000) * 1000
	return time.Unix(sec, nsec).UTC()
}

// Year returns an Int32 Series with the year component.
func (a *DtAccessor) Year() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Year()) })
}

// Month returns an Int32 Series with the month component (1-12).
func (a *DtAccessor) Month() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Month()) })
}

// Day returns an Int32 Series with the day of month component.
func (a *DtAccessor) Day() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Day()) })
}

// Hour returns an Int32 Series with the hour component.
func (a *DtAccessor) Hour() *Series {
	if a.s.dtype == dtype.Time {
		return a.extractTimeComponent(func(ns int64) int32 { return int32(ns / 3_600_000_000_000) })
	}
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Hour()) })
}

// Minute returns an Int32 Series with the minute component.
func (a *DtAccessor) Minute() *Series {
	if a.s.dtype == dtype.Time {
		return a.extractTimeComponent(func(ns int64) int32 { return int32((ns / 60_000_000_000) % 60) })
	}
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Minute()) })
}

// Second returns an Int32 Series with the second component.
func (a *DtAccessor) Second() *Series {
	if a.s.dtype == dtype.Time {
		return a.extractTimeComponent(func(ns int64) int32 { return int32((ns / 1_000_000_000) % 60) })
	}
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Second()) })
}

// Nanosecond returns an Int32 Series with the nanosecond component.
func (a *DtAccessor) Nanosecond() *Series {
	if a.s.dtype == dtype.Time {
		return a.extractTimeComponent(func(ns int64) int32 { return int32(ns % 1_000_000_000) })
	}
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Nanosecond()) })
}

// Weekday returns an Int32 Series with the weekday (0=Sunday, 6=Saturday).
func (a *DtAccessor) Weekday() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32(t.Weekday()) })
}

// IsoWeek returns an Int32 Series with the ISO week number.
func (a *DtAccessor) IsoWeek() *Series {
	return a.extractComponent(func(t time.Time) int32 { _, w := t.ISOWeek(); return int32(w) })
}

// DayOfYear returns an Int32 Series with the day of year (1-366).
func (a *DtAccessor) DayOfYear() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32(t.YearDay()) })
}

// Quarter returns an Int32 Series with the quarter (1-4).
func (a *DtAccessor) Quarter() *Series {
	return a.extractComponent(func(t time.Time) int32 { return int32((t.Month()-1)/3 + 1) })
}

// extractComponent applies a function to each temporal value.
func (a *DtAccessor) extractComponent(fn func(time.Time) int32) *Series {
	n := a.s.Len()
	data := make([]int32, n)
	valid := make([]bool, n)

	for i := 0; i < n; i++ {
		if a.s.IsNull(i) {
			continue
		}
		valid[i] = true
		t := a.toTime(i)
		data[i] = fn(t)
	}

	if a.s.HasNulls() {
		bm := bitmap.New(n)
		for i, ok := range valid {
			if !ok {
				bm.Clear(i)
			}
		}
		return New(a.s.name, array.NewInt32Array(data, bm))
	}
	return NewInt32(a.s.name, data)
}

// extractTimeComponent applies a function to Time values (nanoseconds since midnight).
func (a *DtAccessor) extractTimeComponent(fn func(int64) int32) *Series {
	n := a.s.Len()
	data := make([]int32, n)
	valid := make([]bool, n)

	ta, ok := a.s.arr.(*array.TypedArray[int64])
	if !ok {
		return NewInt32(a.s.name, data)
	}

	for i := 0; i < n; i++ {
		if a.s.IsNull(i) {
			continue
		}
		valid[i] = true
		data[i] = fn(ta.Value(i))
	}

	if a.s.HasNulls() {
		bm := bitmap.New(n)
		for i, ok := range valid {
			if !ok {
				bm.Clear(i)
			}
		}
		return New(a.s.name, array.NewInt32Array(data, bm))
	}
	return NewInt32(a.s.name, data)
}

// toTime converts the value at index i to time.Time.
func (a *DtAccessor) toTime(i int) time.Time {
	switch a.s.dtype {
	case dtype.Date:
		if ta, ok := a.s.arr.(*array.TypedArray[int32]); ok {
			return dateToTime(ta.Value(i))
		}
	case dtype.DateTime:
		if ta, ok := a.s.arr.(*array.TypedArray[int64]); ok {
			return dateTimeToTime(ta.Value(i))
		}
	}
	return time.Time{}
}

// Truncate truncates temporal values to the given unit.
// Supported units: "1h", "1d", "1mo", "1y"
func (a *DtAccessor) Truncate(unit string) *Series {
	n := a.s.Len()

	switch a.s.dtype {
	case dtype.Date:
		ta, ok := a.s.arr.(*array.TypedArray[int32])
		if !ok {
			return a.s
		}
		data := make([]int32, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.s.IsNull(i) {
				continue
			}
			valid[i] = true
			t := dateToTime(ta.Value(i))
			t = truncateTime(t, unit)
			data[i] = int32(t.Unix() / 86400)
		}
		if a.s.HasNulls() {
			return NewDateWithValidity(a.s.name, data, valid)
		}
		return NewDate(a.s.name, data)
	case dtype.DateTime:
		ta, ok := a.s.arr.(*array.TypedArray[int64])
		if !ok {
			return a.s
		}
		data := make([]int64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.s.IsNull(i) {
				continue
			}
			valid[i] = true
			t := dateTimeToTime(ta.Value(i))
			t = truncateTime(t, unit)
			data[i] = t.UnixMicro()
		}
		if a.s.HasNulls() {
			return NewDateTimeWithValidity(a.s.name, data, valid)
		}
		return NewDateTime(a.s.name, data)
	}
	return a.s
}

// truncateTime truncates a time.Time to the given unit.
func truncateTime(t time.Time, unit string) time.Time {
	switch unit {
	case "1h":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case "1d":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "1mo":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "1y":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	}
	return t
}

// Strftime formats temporal values using a Go time format string.
func (a *DtAccessor) Strftime(format string) *Series {
	n := a.s.Len()
	data := make([]string, n)
	valid := make([]bool, n)

	for i := 0; i < n; i++ {
		if a.s.IsNull(i) {
			continue
		}
		valid[i] = true
		t := a.toTime(i)
		data[i] = t.Format(format)
	}

	if a.s.HasNulls() {
		return NewStringWithValidity(a.s.name, data, valid)
	}
	return NewString(a.s.name, data)
}

// OffsetBy offsets temporal values by a duration string.
// Supported formats: "1d", "2mo", "-1y", "3h", etc.
func (a *DtAccessor) OffsetBy(duration string) *Series {
	n := a.s.Len()
	amount, unit := parseDuration(duration)

	switch a.s.dtype {
	case dtype.Date:
		ta, ok := a.s.arr.(*array.TypedArray[int32])
		if !ok {
			return a.s
		}
		data := make([]int32, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.s.IsNull(i) {
				continue
			}
			valid[i] = true
			t := dateToTime(ta.Value(i))
			t = offsetTime(t, amount, unit)
			data[i] = int32(t.Unix() / 86400)
		}
		if a.s.HasNulls() {
			return NewDateWithValidity(a.s.name, data, valid)
		}
		return NewDate(a.s.name, data)
	case dtype.DateTime:
		ta, ok := a.s.arr.(*array.TypedArray[int64])
		if !ok {
			return a.s
		}
		data := make([]int64, n)
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			if a.s.IsNull(i) {
				continue
			}
			valid[i] = true
			t := dateTimeToTime(ta.Value(i))
			t = offsetTime(t, amount, unit)
			data[i] = t.UnixMicro()
		}
		if a.s.HasNulls() {
			return NewDateTimeWithValidity(a.s.name, data, valid)
		}
		return NewDateTime(a.s.name, data)
	}
	return a.s
}

// Epoch returns an Int64 Series with the epoch value in the given time unit.
// Supported units: "s" (seconds), "ms" (milliseconds), "us" (microseconds), "ns" (nanoseconds).
func (a *DtAccessor) Epoch(unit string) *Series {
	n := a.s.Len()
	data := make([]int64, n)
	valid := make([]bool, n)

	for i := 0; i < n; i++ {
		if a.s.IsNull(i) {
			continue
		}
		valid[i] = true
		t := a.toTime(i)
		switch unit {
		case "s":
			data[i] = t.Unix()
		case "ms":
			data[i] = t.UnixMilli()
		case "us":
			data[i] = t.UnixMicro()
		case "ns":
			data[i] = t.UnixNano()
		default:
			data[i] = t.Unix()
		}
	}

	if a.s.HasNulls() {
		return NewInt64WithValidity(a.s.name, data, valid)
	}
	return NewInt64(a.s.name, data)
}

// TotalSeconds returns a Float64 Series with the total duration in seconds.
// Only applicable to Duration series (microseconds).
func (a *DtAccessor) TotalSeconds() *Series {
	if a.s.dtype != dtype.Duration {
		return a.s
	}
	ta, ok := a.s.arr.(*array.TypedArray[int64])
	if !ok {
		return a.s
	}
	n := a.s.Len()
	data := make([]float64, n)
	valid := make([]bool, n)
	for i := 0; i < n; i++ {
		if a.s.IsNull(i) {
			continue
		}
		valid[i] = true
		data[i] = float64(ta.Value(i)) / 1_000_000.0
	}
	if a.s.HasNulls() {
		return NewFloat64WithValidity(a.s.name, data, valid)
	}
	return NewFloat64(a.s.name, data)
}

// parseDuration parses a duration string like "1d", "-2mo", "3y".
func parseDuration(s string) (int, string) {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	// Find where digits end
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}

	numStr := s[:i]
	unit := s[i:]

	amount, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, ""
	}
	if neg {
		amount = -amount
	}
	return amount, unit
}

// offsetTime adds the given offset to a time.
func offsetTime(t time.Time, amount int, unit string) time.Time {
	switch unit {
	case "d":
		return t.AddDate(0, 0, amount)
	case "mo":
		return t.AddDate(0, amount, 0)
	case "y":
		return t.AddDate(amount, 0, 0)
	case "h":
		return t.Add(time.Duration(amount) * time.Hour)
	case "m":
		return t.Add(time.Duration(amount) * time.Minute)
	case "s":
		return t.Add(time.Duration(amount) * time.Second)
	}
	return t
}
