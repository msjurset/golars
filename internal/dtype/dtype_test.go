package dtype

import "testing"

func TestDataTypeString(t *testing.T) {
	tests := []struct {
		dt   DataType
		want string
	}{
		{Null, "Null"},
		{Boolean, "Boolean"},
		{Int8, "Int8"},
		{Int16, "Int16"},
		{Int32, "Int32"},
		{Int64, "Int64"},
		{UInt8, "UInt8"},
		{UInt16, "UInt16"},
		{UInt32, "UInt32"},
		{UInt64, "UInt64"},
		{Float32, "Float32"},
		{Float64, "Float64"},
		{Decimal, "Decimal"},
		{String, "String"},
		{Binary, "Binary"},
		{Date, "Date"},
		{DateTime, "DateTime"},
		{Time, "Time"},
		{Duration, "Duration"},
		{List, "List"},
		{Array, "Array"},
		{Struct, "Struct"},
		{Categorical, "Categorical"},
		{Enum, "Enum"},
		{Unknown, "Unknown"},
		{DataType(999), "DataType(999)"},
	}
	for _, tt := range tests {
		if got := tt.dt.String(); got != tt.want {
			t.Errorf("DataType(%d).String() = %q, want %q", int(tt.dt), got, tt.want)
		}
	}
}

func TestTimeUnitString(t *testing.T) {
	tests := []struct {
		tu   TimeUnit
		want string
	}{
		{Nanoseconds, "ns"},
		{Microseconds, "us"},
		{Milliseconds, "ms"},
		{TimeUnit(99), "TimeUnit(99)"},
	}
	for _, tt := range tests {
		if got := tt.tu.String(); got != tt.want {
			t.Errorf("TimeUnit(%d).String() = %q, want %q", int(tt.tu), got, tt.want)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	numeric := []DataType{
		Int8, Int16, Int32, Int64,
		UInt8, UInt16, UInt32, UInt64,
		Float32, Float64, Decimal,
	}
	notNumeric := []DataType{
		Null, Boolean, String, Binary, Date, DateTime, Time, Duration,
		List, Array, Struct, Categorical, Enum, Unknown,
	}
	for _, dt := range numeric {
		if !IsNumeric(dt) {
			t.Errorf("IsNumeric(%s) = false, want true", dt)
		}
	}
	for _, dt := range notNumeric {
		if IsNumeric(dt) {
			t.Errorf("IsNumeric(%s) = true, want false", dt)
		}
	}
}

func TestIsInteger(t *testing.T) {
	integers := []DataType{Int8, Int16, Int32, Int64, UInt8, UInt16, UInt32, UInt64}
	notIntegers := []DataType{Float32, Float64, Decimal, Boolean, String, Null}
	for _, dt := range integers {
		if !IsInteger(dt) {
			t.Errorf("IsInteger(%s) = false, want true", dt)
		}
	}
	for _, dt := range notIntegers {
		if IsInteger(dt) {
			t.Errorf("IsInteger(%s) = true, want false", dt)
		}
	}
}

func TestIsFloat(t *testing.T) {
	if !IsFloat(Float32) {
		t.Error("IsFloat(Float32) = false, want true")
	}
	if !IsFloat(Float64) {
		t.Error("IsFloat(Float64) = false, want true")
	}
	if IsFloat(Int32) {
		t.Error("IsFloat(Int32) = true, want false")
	}
}

func TestIsTemporal(t *testing.T) {
	temporal := []DataType{Date, DateTime, Time, Duration}
	notTemporal := []DataType{Int32, Float64, String, Boolean, Null}
	for _, dt := range temporal {
		if !IsTemporal(dt) {
			t.Errorf("IsTemporal(%s) = false, want true", dt)
		}
	}
	for _, dt := range notTemporal {
		if IsTemporal(dt) {
			t.Errorf("IsTemporal(%s) = true, want false", dt)
		}
	}
}

func TestIsNested(t *testing.T) {
	nested := []DataType{List, Array, Struct}
	notNested := []DataType{Int32, Float64, String, Boolean, Date}
	for _, dt := range nested {
		if !IsNested(dt) {
			t.Errorf("IsNested(%s) = false, want true", dt)
		}
	}
	for _, dt := range notNested {
		if IsNested(dt) {
			t.Errorf("IsNested(%s) = true, want false", dt)
		}
	}
}

func TestIsSigned(t *testing.T) {
	signed := []DataType{Int8, Int16, Int32, Int64}
	for _, dt := range signed {
		if !IsSigned(dt) {
			t.Errorf("IsSigned(%s) = false, want true", dt)
		}
	}
	if IsSigned(UInt32) {
		t.Error("IsSigned(UInt32) = true, want false")
	}
}

func TestIsUnsigned(t *testing.T) {
	unsigned := []DataType{UInt8, UInt16, UInt32, UInt64}
	for _, dt := range unsigned {
		if !IsUnsigned(dt) {
			t.Errorf("IsUnsigned(%s) = false, want true", dt)
		}
	}
	if IsUnsigned(Int32) {
		t.Error("IsUnsigned(Int32) = true, want false")
	}
}

// --- CanCast tests ---

func TestCanCastIdentity(t *testing.T) {
	types := []DataType{
		Null, Boolean, Int8, Int32, Int64, UInt32, Float32, Float64,
		String, Date, DateTime, Time, Duration, List, Categorical,
	}
	for _, dt := range types {
		if !CanCast(dt, dt) {
			t.Errorf("CanCast(%s, %s) = false, want true", dt, dt)
		}
	}
}

func TestCanCastNullToAny(t *testing.T) {
	targets := []DataType{Boolean, Int32, Float64, String, Date, List}
	for _, to := range targets {
		if !CanCast(Null, to) {
			t.Errorf("CanCast(Null, %s) = false, want true", to)
		}
	}
}

func TestCanCastToString(t *testing.T) {
	sources := []DataType{Boolean, Int32, Float64, Date, DateTime, List, Null}
	for _, from := range sources {
		if !CanCast(from, String) {
			t.Errorf("CanCast(%s, String) = false, want true", from)
		}
	}
}

func TestCanCastStringParsing(t *testing.T) {
	valid := []DataType{Boolean, Int8, Int32, Int64, UInt32, Float32, Float64, Decimal, Date, DateTime, Time, Duration}
	for _, to := range valid {
		if !CanCast(String, to) {
			t.Errorf("CanCast(String, %s) = false, want true", to)
		}
	}
	invalid := []DataType{List, Array, Struct}
	for _, to := range invalid {
		if CanCast(String, to) {
			t.Errorf("CanCast(String, %s) = true, want false", to)
		}
	}
}

func TestCanCastNumericCrossTypes(t *testing.T) {
	tests := []struct {
		from, to DataType
		want     bool
	}{
		{Int8, Int64, true},
		{Int32, Float64, true},
		{UInt16, Int32, true},
		{Float32, Int8, true},
		{Int32, Decimal, true},
		{Boolean, Int32, true},
		{Int32, Boolean, true},
	}
	for _, tt := range tests {
		if got := CanCast(tt.from, tt.to); got != tt.want {
			t.Errorf("CanCast(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestCanCastTemporalInteger(t *testing.T) {
	tests := []struct {
		from, to DataType
		want     bool
	}{
		{Date, DateTime, true},
		{DateTime, Date, true},
		{DateTime, Int64, true},
		{Int64, DateTime, true},
		{Date, Int32, true},
		{Int32, Date, true},
		{Duration, Int64, true},
		{Int64, Duration, true},
		{Time, Int64, true},
		{Int64, Time, true},
	}
	for _, tt := range tests {
		if got := CanCast(tt.from, tt.to); got != tt.want {
			t.Errorf("CanCast(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestCanCastInvalid(t *testing.T) {
	tests := []struct {
		from, to DataType
	}{
		{List, Array},
		{Boolean, Date},
		{Date, Duration},
		{Binary, Int32},
	}
	for _, tt := range tests {
		if CanCast(tt.from, tt.to) {
			t.Errorf("CanCast(%s, %s) = true, want false", tt.from, tt.to)
		}
	}
}

// --- SuperType tests ---

func TestSuperTypeIdentity(t *testing.T) {
	types := []DataType{Int32, Float64, String, Date, Boolean}
	for _, dt := range types {
		got, err := SuperType(dt, dt)
		if err != nil {
			t.Errorf("SuperType(%s, %s) unexpected error: %v", dt, dt, err)
		}
		if got != dt {
			t.Errorf("SuperType(%s, %s) = %s, want %s", dt, dt, got, dt)
		}
	}
}

func TestSuperTypeNull(t *testing.T) {
	types := []DataType{Int32, Float64, String, Boolean}
	for _, dt := range types {
		got, err := SuperType(Null, dt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != dt {
			t.Errorf("SuperType(Null, %s) = %s, want %s", dt, got, dt)
		}
		got, err = SuperType(dt, Null)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != dt {
			t.Errorf("SuperType(%s, Null) = %s, want %s", dt, got, dt)
		}
	}
}

func TestSuperTypeIntegerPromotion(t *testing.T) {
	tests := []struct {
		a, b DataType
		want DataType
	}{
		{Int8, Int32, Int32},
		{Int32, Int64, Int64},
		{UInt8, UInt32, UInt32},
		{Int16, Int16, Int16},
	}
	for _, tt := range tests {
		got, err := SuperType(tt.a, tt.b)
		if err != nil {
			t.Fatalf("SuperType(%s, %s) error: %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Errorf("SuperType(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuperTypeMixedSign(t *testing.T) {
	tests := []struct {
		a, b DataType
		want DataType
	}{
		{UInt8, Int8, Int16},    // unsigned promotes to next wider signed
		{UInt16, Int16, Int32},  // same
		{UInt32, Int32, Int64},  // same
		{UInt8, Int32, Int32},   // signed is wider
		{UInt16, Int64, Int64},  // signed is wider
		{UInt64, Int64, Float64}, // no lossless int, fall back to float
	}
	for _, tt := range tests {
		got, err := SuperType(tt.a, tt.b)
		if err != nil {
			t.Fatalf("SuperType(%s, %s) error: %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Errorf("SuperType(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuperTypeFloatPromotion(t *testing.T) {
	tests := []struct {
		a, b DataType
		want DataType
	}{
		{Float32, Float32, Float32},
		{Float32, Float64, Float64},
		{Float64, Float32, Float64},
	}
	for _, tt := range tests {
		got, err := SuperType(tt.a, tt.b)
		if err != nil {
			t.Fatalf("SuperType(%s, %s) error: %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Errorf("SuperType(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuperTypeIntFloat(t *testing.T) {
	tests := []struct {
		a, b DataType
		want DataType
	}{
		{Int8, Float32, Float32},
		{Int32, Float32, Float32},
		{Int64, Float32, Float64}, // Int64 requires Float64
		{Int32, Float64, Float64},
		{UInt64, Float32, Float64},
	}
	for _, tt := range tests {
		got, err := SuperType(tt.a, tt.b)
		if err != nil {
			t.Fatalf("SuperType(%s, %s) error: %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Errorf("SuperType(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuperTypeBooleanNumeric(t *testing.T) {
	got, err := SuperType(Boolean, Int32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != Int32 {
		t.Errorf("SuperType(Boolean, Int32) = %s, want Int32", got)
	}

	got, err = SuperType(Float64, Boolean)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != Float64 {
		t.Errorf("SuperType(Float64, Boolean) = %s, want Float64", got)
	}
}

func TestSuperTypeDateDateTime(t *testing.T) {
	got, err := SuperType(Date, DateTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != DateTime {
		t.Errorf("SuperType(Date, DateTime) = %s, want DateTime", got)
	}
}

func TestSuperTypeString(t *testing.T) {
	tests := []DataType{Int32, Float64, Boolean, Date}
	for _, dt := range tests {
		got, err := SuperType(String, dt)
		if err != nil {
			t.Fatalf("SuperType(String, %s) error: %v", dt, err)
		}
		if got != String {
			t.Errorf("SuperType(String, %s) = %s, want String", dt, got)
		}
	}
}

func TestSuperTypeUnknown(t *testing.T) {
	got, err := SuperType(Unknown, Int32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != Unknown {
		t.Errorf("SuperType(Unknown, Int32) = %s, want Unknown", got)
	}
}

func TestSuperTypeNoCommon(t *testing.T) {
	_, err := SuperType(Date, Duration)
	if err == nil {
		t.Error("SuperType(Date, Duration) expected error, got nil")
	}
}

func TestSuperTypeDecimal(t *testing.T) {
	got, err := SuperType(Decimal, Int32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != Decimal {
		t.Errorf("SuperType(Decimal, Int32) = %s, want Decimal", got)
	}

	got, err = SuperType(Decimal, Float64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != Float64 {
		t.Errorf("SuperType(Decimal, Float64) = %s, want Float64", got)
	}
}

// --- Schema tests ---

func TestSchemaBasic(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
		{Name: "c", Dtype: String},
	})

	if s.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", s.Len())
	}

	f := s.Field(0)
	if f.Name != "a" || f.Dtype != Int32 {
		t.Errorf("Field(0) = %v, want {a, Int32}", f)
	}

	f, ok := s.FieldByName("b")
	if !ok || f.Dtype != Float64 {
		t.Errorf("FieldByName(b) = %v, %v", f, ok)
	}

	_, ok = s.FieldByName("z")
	if ok {
		t.Error("FieldByName(z) should return false")
	}
}

func TestSchemaNames(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "x", Dtype: Int32},
		{Name: "y", Dtype: Float64},
	})
	names := s.Names()
	if len(names) != 2 || names[0] != "x" || names[1] != "y" {
		t.Errorf("Names() = %v, want [x y]", names)
	}
}

func TestSchemaDtypes(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "x", Dtype: Int32},
		{Name: "y", Dtype: Float64},
	})
	dtypes := s.Dtypes()
	if len(dtypes) != 2 || dtypes[0] != Int32 || dtypes[1] != Float64 {
		t.Errorf("Dtypes() = %v, want [Int32 Float64]", dtypes)
	}
}

func TestSchemaIndex(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
	})
	if idx := s.Index("a"); idx != 0 {
		t.Errorf("Index(a) = %d, want 0", idx)
	}
	if idx := s.Index("b"); idx != 1 {
		t.Errorf("Index(b) = %d, want 1", idx)
	}
	if idx := s.Index("z"); idx != -1 {
		t.Errorf("Index(z) = %d, want -1", idx)
	}
}

func TestSchemaContains(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
	})
	if !s.Contains("a") {
		t.Error("Contains(a) = false, want true")
	}
	if s.Contains("z") {
		t.Error("Contains(z) = true, want false")
	}
}

func TestSchemaEqual(t *testing.T) {
	s1 := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
	})
	s2 := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
	})
	s3 := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: String},
	})
	s4 := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
	})

	if !s1.Equal(s2) {
		t.Error("s1.Equal(s2) = false, want true")
	}
	if s1.Equal(s3) {
		t.Error("s1.Equal(s3) = true, want false")
	}
	if s1.Equal(s4) {
		t.Error("s1.Equal(s4) = true, want false")
	}
}

func TestSchemaString(t *testing.T) {
	s := NewSchema([]Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
	})
	got := s.String()
	want := "Schema{a: Int32, b: Float64}"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestSchemaImmutability(t *testing.T) {
	fields := []Field{
		{Name: "a", Dtype: Int32},
		{Name: "b", Dtype: Float64},
	}
	s := NewSchema(fields)

	// Mutating the original slice should not affect the schema.
	fields[0].Name = "modified"
	if s.Field(0).Name != "a" {
		t.Error("schema was mutated via original fields slice")
	}

	// Mutating Names() output should not affect the schema.
	names := s.Names()
	names[0] = "modified"
	if s.Field(0).Name != "a" {
		t.Error("schema was mutated via Names() slice")
	}
}
