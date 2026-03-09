package expr

import (
	"fmt"

	"github.com/msjurset/golars/internal/series"
)

// DtNamespace provides temporal operations on expressions.
type DtNamespace struct {
	inner Expr
}

func (d *DtNamespace) Year() Expr       { return d.dtExpr("year") }
func (d *DtNamespace) Month() Expr      { return d.dtExpr("month") }
func (d *DtNamespace) Day() Expr        { return d.dtExpr("day") }
func (d *DtNamespace) Hour() Expr       { return d.dtExpr("hour") }
func (d *DtNamespace) Minute() Expr     { return d.dtExpr("minute") }
func (d *DtNamespace) Second() Expr     { return d.dtExpr("second") }
func (d *DtNamespace) Nanosecond() Expr { return d.dtExpr("nanosecond") }
func (d *DtNamespace) Weekday() Expr    { return d.dtExpr("weekday") }
func (d *DtNamespace) IsoWeek() Expr    { return d.dtExpr("iso_week") }
func (d *DtNamespace) DayOfYear() Expr  { return d.dtExpr("day_of_year") }
func (d *DtNamespace) Quarter() Expr    { return d.dtExpr("quarter") }

func (d *DtNamespace) dtExpr(op string) Expr {
	e := &dtComponentExpr{inner: d.inner, op: op}
	e.exprBase.self = e
	return e
}

// Truncate returns an expression that truncates temporal values.
func (d *DtNamespace) Truncate(unit string) Expr {
	e := &dtTruncateExpr{inner: d.inner, unit: unit}
	e.exprBase.self = e
	return e
}

// Strftime returns an expression that formats temporal values.
func (d *DtNamespace) Strftime(format string) Expr {
	e := &dtStrftimeExpr{inner: d.inner, format: format}
	e.exprBase.self = e
	return e
}

// OffsetBy returns an expression that offsets temporal values.
func (d *DtNamespace) OffsetBy(duration string) Expr {
	e := &dtOffsetExpr{inner: d.inner, duration: duration}
	e.exprBase.self = e
	return e
}

// Epoch returns an expression that converts temporal values to epoch in the given unit.
func (d *DtNamespace) Epoch(unit string) Expr {
	e := &dtEpochExpr{inner: d.inner, unit: unit}
	e.exprBase.self = e
	return e
}

// TotalSeconds returns an expression that converts Duration values to total seconds.
func (d *DtNamespace) TotalSeconds() Expr {
	e := &dtTotalSecondsExpr{inner: d.inner}
	e.exprBase.self = e
	return e
}

// dtComponentExpr extracts a temporal component.
type dtComponentExpr struct {
	exprBase
	inner Expr
	op    string
}

func (e *dtComponentExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.%s requires a temporal series, got %s", e.op, s.DataType())
	}
	switch e.op {
	case "year":
		return dt.Year(), nil
	case "month":
		return dt.Month(), nil
	case "day":
		return dt.Day(), nil
	case "hour":
		return dt.Hour(), nil
	case "minute":
		return dt.Minute(), nil
	case "second":
		return dt.Second(), nil
	case "nanosecond":
		return dt.Nanosecond(), nil
	case "weekday":
		return dt.Weekday(), nil
	case "iso_week":
		return dt.IsoWeek(), nil
	case "day_of_year":
		return dt.DayOfYear(), nil
	case "quarter":
		return dt.Quarter(), nil
	default:
		return nil, fmt.Errorf("golars: unknown dt operation %q", e.op)
	}
}

func (e *dtComponentExpr) String() string {
	return fmt.Sprintf("%s.dt.%s()", e.inner.String(), e.op)
}

// dtTruncateExpr truncates temporal values.
type dtTruncateExpr struct {
	exprBase
	inner Expr
	unit  string
}

func (e *dtTruncateExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.truncate requires a temporal series, got %s", s.DataType())
	}
	return dt.Truncate(e.unit), nil
}

func (e *dtTruncateExpr) String() string {
	return fmt.Sprintf("%s.dt.truncate(%q)", e.inner.String(), e.unit)
}

// dtStrftimeExpr formats temporal values.
type dtStrftimeExpr struct {
	exprBase
	inner  Expr
	format string
}

func (e *dtStrftimeExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.strftime requires a temporal series, got %s", s.DataType())
	}
	return dt.Strftime(e.format), nil
}

func (e *dtStrftimeExpr) String() string {
	return fmt.Sprintf("%s.dt.strftime(%q)", e.inner.String(), e.format)
}

// dtOffsetExpr offsets temporal values.
type dtOffsetExpr struct {
	exprBase
	inner    Expr
	duration string
}

func (e *dtOffsetExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.offset_by requires a temporal series, got %s", s.DataType())
	}
	return dt.OffsetBy(e.duration), nil
}

func (e *dtOffsetExpr) String() string {
	return fmt.Sprintf("%s.dt.offset_by(%q)", e.inner.String(), e.duration)
}

// dtEpochExpr converts temporal values to epoch.
type dtEpochExpr struct {
	exprBase
	inner Expr
	unit  string
}

func (e *dtEpochExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.epoch requires a temporal series, got %s", s.DataType())
	}
	return dt.Epoch(e.unit), nil
}

func (e *dtEpochExpr) String() string {
	return fmt.Sprintf("%s.dt.epoch(%q)", e.inner.String(), e.unit)
}

// dtTotalSecondsExpr converts Duration values to total seconds.
type dtTotalSecondsExpr struct {
	exprBase
	inner Expr
}

func (e *dtTotalSecondsExpr) Evaluate(ctx *Context) (*series.Series, error) {
	s, err := e.inner.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	dt := s.Dt()
	if dt == nil {
		return nil, fmt.Errorf("golars: dt.total_seconds requires a temporal series, got %s", s.DataType())
	}
	return dt.TotalSeconds(), nil
}

func (e *dtTotalSecondsExpr) String() string {
	return fmt.Sprintf("%s.dt.total_seconds()", e.inner.String())
}
