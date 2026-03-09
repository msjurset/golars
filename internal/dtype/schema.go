package dtype

import (
	"fmt"
	"strings"
)

// Field represents a named column with a data type and optional descriptor.
type Field struct {
	// Name is the column name.
	Name string
	// Dtype is the column's data type.
	Dtype DataType
	// Descriptor holds metadata for parameterized types. May be nil.
	Descriptor *DataTypeDescriptor
}

// String returns a human-readable representation of the field.
func (f Field) String() string {
	if f.Descriptor != nil {
		return fmt.Sprintf("%s: %s(%v)", f.Name, f.Dtype, f.Descriptor)
	}
	return fmt.Sprintf("%s: %s", f.Name, f.Dtype)
}

// Equal reports whether two fields have the same name and type.
func (f Field) Equal(other Field) bool {
	return f.Name == other.Name && f.Dtype == other.Dtype
}

// Schema represents an ordered, immutable collection of named fields
// that defines the structure of a DataFrame.
type Schema struct {
	fields []Field
	index  map[string]int
}

// NewSchema creates a new Schema from the given fields.
// Field order is preserved. Duplicate field names are not allowed;
// if duplicates are found the last occurrence wins for name-based lookup.
func NewSchema(fields []Field) *Schema {
	copied := make([]Field, len(fields))
	copy(copied, fields)

	idx := make(map[string]int, len(copied))
	for i, f := range copied {
		idx[f.Name] = i
	}

	return &Schema{
		fields: copied,
		index:  idx,
	}
}

// Len returns the number of fields in the schema.
func (s *Schema) Len() int {
	return len(s.fields)
}

// Field returns the field at position i.
// It panics if i is out of range.
func (s *Schema) Field(i int) Field {
	return s.fields[i]
}

// FieldByName returns the field with the given name and true if found,
// or a zero-value Field and false if not found.
func (s *Schema) FieldByName(name string) (Field, bool) {
	i, ok := s.index[name]
	if !ok {
		return Field{}, false
	}
	return s.fields[i], true
}

// Names returns the ordered list of field names.
// The returned slice is a copy; modifying it does not affect the schema.
func (s *Schema) Names() []string {
	names := make([]string, len(s.fields))
	for i, f := range s.fields {
		names[i] = f.Name
	}
	return names
}

// Dtypes returns the ordered list of data types.
// The returned slice is a copy; modifying it does not affect the schema.
func (s *Schema) Dtypes() []DataType {
	dtypes := make([]DataType, len(s.fields))
	for i, f := range s.fields {
		dtypes[i] = f.Dtype
	}
	return dtypes
}

// Index returns the zero-based position of the named field, or -1 if not found.
func (s *Schema) Index(name string) int {
	i, ok := s.index[name]
	if !ok {
		return -1
	}
	return i
}

// Contains reports whether the schema contains a field with the given name.
func (s *Schema) Contains(name string) bool {
	_, ok := s.index[name]
	return ok
}

// Equal reports whether two schemas have the same fields in the same order.
func (s *Schema) Equal(other *Schema) bool {
	if s.Len() != other.Len() {
		return false
	}
	for i, f := range s.fields {
		if !f.Equal(other.fields[i]) {
			return false
		}
	}
	return true
}

// String returns a human-readable representation of the schema.
func (s *Schema) String() string {
	parts := make([]string, len(s.fields))
	for i, f := range s.fields {
		parts[i] = f.String()
	}
	return fmt.Sprintf("Schema{%s}", strings.Join(parts, ", "))
}
