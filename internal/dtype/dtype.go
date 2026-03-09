// Package dtype defines the type system for the Golars DataFrame library.
// It provides data type enumerations, type descriptors for parameterized types,
// and helper functions for type classification.
package dtype

import "fmt"

// DataType represents a logical data type in the type system.
type DataType int

const (
	// Null represents a null/missing value type.
	Null DataType = iota
	// Boolean represents a true/false value.
	Boolean
	// Int8 represents an 8-bit signed integer.
	Int8
	// Int16 represents a 16-bit signed integer.
	Int16
	// Int32 represents a 32-bit signed integer.
	Int32
	// Int64 represents a 64-bit signed integer.
	Int64
	// UInt8 represents an 8-bit unsigned integer.
	UInt8
	// UInt16 represents a 16-bit unsigned integer.
	UInt16
	// UInt32 represents a 32-bit unsigned integer.
	UInt32
	// UInt64 represents a 64-bit unsigned integer.
	UInt64
	// Float32 represents a 32-bit floating point number.
	Float32
	// Float64 represents a 64-bit floating point number.
	Float64
	// Decimal represents a fixed-point decimal with configurable precision and scale.
	Decimal
	// String represents a UTF-8 encoded string.
	String
	// Binary represents arbitrary byte data.
	Binary
	// Date represents a calendar date (days since epoch).
	Date
	// DateTime represents a date and time with a specific time unit and optional timezone.
	DateTime
	// Time represents a time of day.
	Time
	// Duration represents a time duration with a specific time unit.
	Duration
	// List represents a variable-length list of values with a single inner type.
	List
	// Array represents a fixed-size array of values with a single inner type.
	Array
	// Struct represents a composite type with named fields.
	Struct
	// Categorical represents a dictionary-encoded categorical type.
	Categorical
	// Enum represents an enumeration type with a fixed set of string values.
	Enum
	// Unknown represents an unresolved or unknown data type.
	Unknown
)

// dataTypeNames maps each DataType to its string representation.
var dataTypeNames = [...]string{
	Null:        "Null",
	Boolean:     "Boolean",
	Int8:        "Int8",
	Int16:       "Int16",
	Int32:       "Int32",
	Int64:       "Int64",
	UInt8:       "UInt8",
	UInt16:      "UInt16",
	UInt32:      "UInt32",
	UInt64:      "UInt64",
	Float32:     "Float32",
	Float64:     "Float64",
	Decimal:     "Decimal",
	String:      "String",
	Binary:      "Binary",
	Date:        "Date",
	DateTime:    "DateTime",
	Time:        "Time",
	Duration:    "Duration",
	List:        "List",
	Array:       "Array",
	Struct:      "Struct",
	Categorical: "Categorical",
	Enum:        "Enum",
	Unknown:     "Unknown",
}

// String returns the human-readable name of the data type.
func (d DataType) String() string {
	if d >= 0 && int(d) < len(dataTypeNames) {
		return dataTypeNames[d]
	}
	return fmt.Sprintf("DataType(%d)", int(d))
}

// TimeUnit represents the resolution of temporal types.
type TimeUnit int

const (
	// Nanoseconds represents nanosecond precision.
	Nanoseconds TimeUnit = iota
	// Microseconds represents microsecond precision.
	Microseconds
	// Milliseconds represents millisecond precision.
	Milliseconds
)

// timeUnitNames maps each TimeUnit to its string representation.
var timeUnitNames = [...]string{
	Nanoseconds:  "ns",
	Microseconds: "us",
	Milliseconds: "ms",
}

// String returns the abbreviated string representation of the time unit.
func (tu TimeUnit) String() string {
	if tu >= 0 && int(tu) < len(timeUnitNames) {
		return timeUnitNames[tu]
	}
	return fmt.Sprintf("TimeUnit(%d)", int(tu))
}

// DataTypeDescriptor holds metadata for parameterized data types.
// For non-parameterized types, a nil descriptor is sufficient.
type DataTypeDescriptor struct {
	// Precision is the total number of digits for Decimal types.
	Precision int
	// Scale is the number of digits after the decimal point for Decimal types.
	Scale int
	// Unit is the time resolution for DateTime and Duration types.
	Unit TimeUnit
	// TimeZone is the optional IANA timezone string for DateTime types.
	TimeZone string
	// InnerType is the element type for List and Array types.
	InnerType DataType
	// InnerDescriptor is the descriptor for the inner type, if it is parameterized.
	InnerDescriptor *DataTypeDescriptor
	// ArraySize is the fixed size for Array types.
	ArraySize int
	// Fields holds the ordered fields for Struct types.
	Fields []StructField
	// Categories holds the allowed values for Enum types.
	Categories []string
}

// StructField represents a single field within a Struct data type.
type StructField struct {
	// Name is the field name.
	Name string
	// Dtype is the field's data type.
	Dtype DataType
	// Descriptor is the optional descriptor for parameterized field types.
	Descriptor *DataTypeDescriptor
}

// IsNumeric reports whether the data type is a numeric type
// (any integer, unsigned integer, float, or decimal).
func IsNumeric(d DataType) bool {
	return IsInteger(d) || IsFloat(d) || d == Decimal
}

// IsInteger reports whether the data type is a signed or unsigned integer type.
func IsInteger(d DataType) bool {
	switch d {
	case Int8, Int16, Int32, Int64, UInt8, UInt16, UInt32, UInt64:
		return true
	}
	return false
}

// IsFloat reports whether the data type is a floating point type.
func IsFloat(d DataType) bool {
	return d == Float32 || d == Float64
}

// IsTemporal reports whether the data type is a date, time, or duration type.
func IsTemporal(d DataType) bool {
	switch d {
	case Date, DateTime, Time, Duration:
		return true
	}
	return false
}

// IsNested reports whether the data type is a composite or container type
// (List, Array, or Struct).
func IsNested(d DataType) bool {
	switch d {
	case List, Array, Struct:
		return true
	}
	return false
}

// IsSigned reports whether the data type is a signed integer type.
func IsSigned(d DataType) bool {
	switch d {
	case Int8, Int16, Int32, Int64:
		return true
	}
	return false
}

// IsUnsigned reports whether the data type is an unsigned integer type.
func IsUnsigned(d DataType) bool {
	switch d {
	case UInt8, UInt16, UInt32, UInt64:
		return true
	}
	return false
}
