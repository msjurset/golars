package csv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/msjurset/golars/internal/dtype"
)

func TestReadBasic(t *testing.T) {
	input := "name,age,score\nAlice,25,88.5\nBob,30,92.3\nCharlie,35,76.1\n"
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}

	// name column
	if cols[0].Name() != "name" {
		t.Errorf("expected column name 'name', got %q", cols[0].Name())
	}
	if cols[0].DataType() != dtype.String {
		t.Errorf("expected String type, got %v", cols[0].DataType())
	}
	if cols[0].Len() != 3 {
		t.Errorf("expected 3 rows, got %d", cols[0].Len())
	}
	v, ok := cols[0].GetString(0)
	if !ok || v != "Alice" {
		t.Errorf("expected Alice, got %q (%v)", v, ok)
	}

	// age column - should be inferred as Int64
	if cols[1].DataType() != dtype.Int64 {
		t.Errorf("expected Int64 type for age, got %v", cols[1].DataType())
	}
	iv, ok := cols[1].GetInt64(1)
	if !ok || iv != 30 {
		t.Errorf("expected 30, got %d (%v)", iv, ok)
	}

	// score column - should be inferred as Float64
	if cols[2].DataType() != dtype.Float64 {
		t.Errorf("expected Float64 type for score, got %v", cols[2].DataType())
	}
	fv, ok := cols[2].GetFloat64(0)
	if !ok || fv != 88.5 {
		t.Errorf("expected 88.5, got %g (%v)", fv, ok)
	}
}

func TestReadWithNulls(t *testing.T) {
	input := "x,y\n1,hello\n,world\n3,\n"
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	// x column has a null
	if !cols[0].HasNulls() {
		t.Error("expected x column to have nulls")
	}
	if !cols[0].IsNull(1) {
		t.Error("expected x[1] to be null")
	}
	v, ok := cols[0].GetInt64(0)
	if !ok || v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	// y column has a null
	if !cols[1].HasNulls() {
		t.Error("expected y column to have nulls")
	}
	if !cols[1].IsNull(2) {
		t.Error("expected y[2] to be null")
	}
}

func TestReadCustomSeparator(t *testing.T) {
	input := "a\tb\n1\t2\n3\t4\n"
	cols, err := Read(strings.NewReader(input), WithSeparator('\t'))
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
	v, ok := cols[0].GetInt64(0)
	if !ok || v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
}

func TestReadQuotedFields(t *testing.T) {
	input := `name,desc
"Alice","has a, comma"
"Bob","says ""hello"""
`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	v, _ := cols[1].GetString(0)
	if v != "has a, comma" {
		t.Errorf("expected 'has a, comma', got %q", v)
	}
	v, _ = cols[1].GetString(1)
	if v != `says "hello"` {
		t.Errorf("expected 'says \"hello\"', got %q", v)
	}
}

func TestReadSelectColumns(t *testing.T) {
	input := "a,b,c\n1,2,3\n4,5,6\n"
	cols, err := Read(strings.NewReader(input), WithColumns("a", "c"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
	if cols[0].Name() != "a" || cols[1].Name() != "c" {
		t.Errorf("expected columns a,c got %s,%s", cols[0].Name(), cols[1].Name())
	}
}

func TestReadNRows(t *testing.T) {
	input := "x\n1\n2\n3\n4\n5\n"
	cols, err := Read(strings.NewReader(input), WithNRows(3))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].Len() != 3 {
		t.Errorf("expected 3 rows, got %d", cols[0].Len())
	}
}

func TestReadSkipRows(t *testing.T) {
	input := "skip this\nalso skip\nx\n1\n2\n"
	cols, err := Read(strings.NewReader(input), WithSkipRows(2))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].Name() != "x" {
		t.Errorf("expected column name 'x', got %q", cols[0].Name())
	}
	if cols[0].Len() != 2 {
		t.Errorf("expected 2 rows, got %d", cols[0].Len())
	}
}

func TestReadNoHeader(t *testing.T) {
	input := "1,hello\n2,world\n"
	cols, err := Read(strings.NewReader(input), WithHasHeader(false))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].Name() != "column_0" {
		t.Errorf("expected column_0, got %q", cols[0].Name())
	}
	if cols[1].Name() != "column_1" {
		t.Errorf("expected column_1, got %q", cols[1].Name())
	}
}

func TestReadForceDtype(t *testing.T) {
	input := "id,val\n1,2\n3,4\n"
	cols2, err := Read(strings.NewReader(input), WithDtypes(map[string]dtype.DataType{
		"id": dtype.String,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cols2[0].DataType() != dtype.String {
		t.Errorf("expected String type for id, got %v", cols2[0].DataType())
	}
}

func TestReadBoolInference(t *testing.T) {
	input := "flag\ntrue\nfalse\ntrue\n"
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].DataType() != dtype.Boolean {
		t.Errorf("expected Boolean, got %v", cols[0].DataType())
	}
	v, ok := cols[0].GetBool(0)
	if !ok || !v {
		t.Errorf("expected true, got %v", v)
	}
}

func TestWriteBasic(t *testing.T) {
	input := "name,age,score\nAlice,25,88.5\nBob,30,92.3\n"
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = Write(&buf, cols)
	if err != nil {
		t.Fatal(err)
	}

	// Read back
	cols2, err := Read(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}

	if len(cols2) != 3 {
		t.Fatalf("round-trip: expected 3 cols, got %d", len(cols2))
	}
	if cols2[0].Len() != 2 {
		t.Errorf("round-trip: expected 2 rows, got %d", cols2[0].Len())
	}
	v, _ := cols2[0].GetString(0)
	if v != "Alice" {
		t.Errorf("round-trip: expected Alice, got %q", v)
	}
}

func TestWriteWithNulls(t *testing.T) {
	input := "x,y\n1,hello\nNA,world\n3,NA\n"
	cols, err := Read(strings.NewReader(input), WithNullValues("NA"))
	if err != nil {
		t.Fatal(err)
	}
	if !cols[0].HasNulls() {
		t.Fatal("expected x column to have nulls")
	}

	var buf bytes.Buffer
	err = Write(&buf, cols, WriteNullValue("NULL"))
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "NULL") {
		t.Errorf("expected null value NULL in output, got:\n%s", output)
	}
}

func TestWriteQuoting(t *testing.T) {
	input := "text\nhello world\n\"has,comma\"\n"
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = Write(&buf, cols)
	if err != nil {
		t.Fatal(err)
	}

	// Should quote the field with comma
	if !strings.Contains(buf.String(), `"has,comma"`) {
		t.Errorf("expected quoted field in output, got:\n%s", buf.String())
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"integers", "a\n1\n2\n3\n"},
		{"floats", "a\n1.5\n2.5\n3.5\n"},
		{"strings", "a\nhello\nworld\n"},
		{"booleans", "a\ntrue\nfalse\ntrue\n"},
		{"mixed", "name,age\nAlice,25\nBob,30\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, err := Read(strings.NewReader(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			err = Write(&buf, cols)
			if err != nil {
				t.Fatal(err)
			}
			cols2, err := Read(strings.NewReader(buf.String()))
			if err != nil {
				t.Fatal(err)
			}
			if len(cols2) != len(cols) {
				t.Fatalf("column count mismatch: %d vs %d", len(cols), len(cols2))
			}
			for i := range cols {
				if cols[i].Len() != cols2[i].Len() {
					t.Errorf("column %d length mismatch: %d vs %d", i, cols[i].Len(), cols2[i].Len())
				}
			}
		})
	}
}
