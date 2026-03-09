package dtype

import (
	"errors"
	"fmt"
)

// ErrNoSuperType is returned when two data types have no common supertype.
var ErrNoSuperType = errors.New("no common supertype")

// CanCast reports whether a value of type from can be cast to type to.
// This defines the set of valid explicit casts in the type system.
func CanCast(from, to DataType) bool {
	if from == to {
		return true
	}

	// Null can be cast to any type.
	if from == Null {
		return true
	}

	// Any type can be cast to String.
	if to == String {
		return true
	}

	// String can be parsed into numeric, boolean, and temporal types.
	if from == String {
		switch to {
		case Boolean, Int8, Int16, Int32, Int64,
			UInt8, UInt16, UInt32, UInt64,
			Float32, Float64, Decimal,
			Date, DateTime, Time, Duration:
			return true
		}
		return false
	}

	// Boolean can be cast to any numeric type.
	if from == Boolean && IsNumeric(to) {
		return true
	}

	// Any numeric type can be cast to Boolean.
	if IsNumeric(from) && to == Boolean {
		return true
	}

	// Numeric types can be cast between each other.
	if IsNumeric(from) && IsNumeric(to) {
		return true
	}

	// Date can be cast to DateTime and vice versa.
	if from == Date && to == DateTime {
		return true
	}
	if from == DateTime && to == Date {
		return true
	}

	// Duration can be cast to integer types and vice versa (underlying representation).
	if from == Duration && IsInteger(to) {
		return true
	}
	if IsInteger(from) && to == Duration {
		return true
	}

	// DateTime can be cast to integer types (epoch) and vice versa.
	if from == DateTime && IsInteger(to) {
		return true
	}
	if IsInteger(from) && to == DateTime {
		return true
	}

	// Date can be cast to integer types (days since epoch) and vice versa.
	if from == Date && IsInteger(to) {
		return true
	}
	if IsInteger(from) && to == Date {
		return true
	}

	// Time can be cast to integer types (nanoseconds since midnight) and vice versa.
	if from == Time && IsInteger(to) {
		return true
	}
	if IsInteger(from) && to == Time {
		return true
	}

	// Categorical can be cast to/from String.
	if from == Categorical && to == String {
		return true // already covered by to == String above
	}
	if from == String && to == Categorical {
		return true
	}

	// Enum can be cast to/from String.
	if from == Enum && to == String {
		return true // already covered
	}
	if from == String && to == Enum {
		return true
	}

	// Categorical and Enum are interchangeable.
	if (from == Categorical && to == Enum) || (from == Enum && to == Categorical) {
		return true
	}

	return false
}

// integerRank returns a numeric rank for integer types used in promotion.
// Signed and unsigned types share the same width ranking.
var integerRank = map[DataType]int{
	Int8:   1,
	UInt8:  1,
	Int16:  2,
	UInt16: 2,
	Int32:  3,
	UInt32: 3,
	Int64:  4,
	UInt64: 4,
}

// signedCounterpart maps unsigned integer types to their signed equivalents
// at the next wider width, used for mixed-sign promotion.
var signedPromotion = map[DataType]DataType{
	UInt8:  Int16,
	UInt16: Int32,
	UInt32: Int64,
}

// SuperType finds the smallest common type that can represent values of both
// types a and b without loss of information, following numeric promotion rules.
// Returns ErrNoSuperType if no common supertype exists.
func SuperType(a, b DataType) (DataType, error) {
	if a == b {
		return a, nil
	}

	// Null is absorbed by any other type.
	if a == Null {
		return b, nil
	}
	if b == Null {
		return a, nil
	}

	// Unknown combined with anything yields Unknown.
	if a == Unknown || b == Unknown {
		return Unknown, nil
	}

	// Both integers: promote to the wider type, handling sign mixing.
	if IsInteger(a) && IsInteger(b) {
		return integerSuperType(a, b), nil
	}

	// Both floats: promote to wider float.
	if IsFloat(a) && IsFloat(b) {
		if a == Float64 || b == Float64 {
			return Float64, nil
		}
		return Float32, nil
	}

	// Integer + Float: promote to float.
	if IsInteger(a) && IsFloat(b) {
		return floatForInteger(a, b), nil
	}
	if IsFloat(a) && IsInteger(b) {
		return floatForInteger(b, a), nil
	}

	// Decimal with integer or float: promote to Decimal (or Float64 for float).
	if a == Decimal && IsInteger(b) || b == Decimal && IsInteger(a) {
		return Decimal, nil
	}
	if a == Decimal && IsFloat(b) || b == Decimal && IsFloat(a) {
		return Float64, nil
	}

	// Boolean with numeric: promote to the numeric type.
	if a == Boolean && IsNumeric(b) {
		return b, nil
	}
	if IsNumeric(a) && b == Boolean {
		return a, nil
	}

	// Date + DateTime: promote to DateTime.
	if (a == Date && b == DateTime) || (a == DateTime && b == Date) {
		return DateTime, nil
	}

	// String combined with any other type: promote to String.
	if a == String || b == String {
		return String, nil
	}

	return Null, fmt.Errorf("%w: cannot find supertype of %s and %s", ErrNoSuperType, a, b)
}

// integerSuperType handles promotion between two integer types.
func integerSuperType(a, b DataType) DataType {
	aSigned := IsSigned(a)
	bSigned := IsSigned(b)
	aRank := integerRank[a]
	bRank := integerRank[b]

	// Same signedness: pick the wider one.
	if aSigned == bSigned {
		if aRank >= bRank {
			return a
		}
		return b
	}

	// Mixed signs: identify unsigned and signed.
	var unsigned, signed DataType
	var uRank, sRank int
	if !aSigned {
		unsigned, signed = a, b
		uRank, sRank = aRank, bRank
	} else {
		unsigned, signed = b, a
		uRank, sRank = bRank, aRank
	}

	// If signed is strictly wider, it can hold the unsigned values.
	if sRank > uRank {
		return signed
	}

	// Otherwise promote the unsigned to the next wider signed type.
	if promoted, ok := signedPromotion[unsigned]; ok {
		return promoted
	}

	// UInt64 mixed with any signed type: no lossless integer representation,
	// fall back to Float64.
	_ = signed
	return Float64
}

// floatForInteger returns the appropriate float type when combining an integer
// with a float. Int64/UInt64 combined with Float32 promotes to Float64 to
// minimize precision loss.
func floatForInteger(intType, floatType DataType) DataType {
	rank := integerRank[intType]
	if rank >= 4 || floatType == Float64 {
		return Float64
	}
	return floatType
}
