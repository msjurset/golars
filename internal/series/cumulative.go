package series

import (
	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/dtype"
)

// CumSum returns a new Series with the cumulative sum.
// Only supports Int64 and Float64 types. Other types return nil.
func (s *Series) CumSum() *Series {
	switch s.dtype {
	case dtype.Int64:
		ta := s.arr.(*array.TypedArray[int64])
		return New(s.name, array.CumSum(ta))
	case dtype.Float64:
		ta := s.arr.(*array.TypedArray[float64])
		return New(s.name, array.CumSum(ta))
	default:
		return nil
	}
}

// CumProd returns a new Series with the cumulative product.
// Only supports Int64 and Float64 types. Other types return nil.
func (s *Series) CumProd() *Series {
	switch s.dtype {
	case dtype.Int64:
		ta := s.arr.(*array.TypedArray[int64])
		return New(s.name, array.CumProd(ta))
	case dtype.Float64:
		ta := s.arr.(*array.TypedArray[float64])
		return New(s.name, array.CumProd(ta))
	default:
		return nil
	}
}

// CumMin returns a new Series with the cumulative minimum.
// Only supports Int64 and Float64 types. Other types return nil.
func (s *Series) CumMin() *Series {
	switch s.dtype {
	case dtype.Int64:
		ta := s.arr.(*array.TypedArray[int64])
		return New(s.name, array.CumMin(ta))
	case dtype.Float64:
		ta := s.arr.(*array.TypedArray[float64])
		return New(s.name, array.CumMin(ta))
	default:
		return nil
	}
}

// CumMax returns a new Series with the cumulative maximum.
// Only supports Int64 and Float64 types. Other types return nil.
func (s *Series) CumMax() *Series {
	switch s.dtype {
	case dtype.Int64:
		ta := s.arr.(*array.TypedArray[int64])
		return New(s.name, array.CumMax(ta))
	case dtype.Float64:
		ta := s.arr.(*array.TypedArray[float64])
		return New(s.name, array.CumMax(ta))
	default:
		return nil
	}
}
