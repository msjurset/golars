package dataframe

import (
	"fmt"

	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/series"
)

// Filter returns a new DataFrame containing only the rows where the boolean
// mask Series is true. The mask must be a Boolean Series with the same length
// as the DataFrame.
func (df *DataFrame) Filter(mask *series.Series) (*DataFrame, error) {
	if mask == nil {
		return nil, fmt.Errorf("golars: filter mask is nil")
	}
	ba := mask.BooleanArray()
	if ba == nil {
		return nil, fmt.Errorf("golars: filter mask must be a Boolean Series")
	}
	if mask.Len() != df.height {
		return nil, fmt.Errorf("golars: filter mask length %d does not match DataFrame height %d", mask.Len(), df.height)
	}

	bm := bitmap.New(df.height)
	for i := 0; i < df.height; i++ {
		if mask.IsNull(i) || !ba.Value(i) {
			bm.Clear(i)
		}
	}
	return df.FilterMask(bm), nil
}

// FilterMask returns a new DataFrame containing only the rows where the
// corresponding bit in the bitmap is set.
func (df *DataFrame) FilterMask(mask *bitmap.Bitmap) *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.Filter(mask)
	}
	// All filtered columns have the same length and original unique names,
	// so New cannot fail here.
	height := 0
	if len(cols) > 0 {
		height = cols[0].Len()
	}
	result, _ := New(cols...)
	if result == nil {
		result = &DataFrame{height: height}
	}
	return result
}
