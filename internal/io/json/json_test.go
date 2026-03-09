package json

import (
	"bytes"
	"strings"
	"testing"

	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

func TestReadJSON(t *testing.T) {
	input := `[{"name":"Alice","age":25},{"name":"Bob","age":30}]`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}

	// Find the age column
	var ageCols, nameCols *series.Series
	for _, c := range cols {
		if c.Name() == "age" {
			ageCols = c
		}
		if c.Name() == "name" {
			nameCols = c
		}
	}
	if ageCols == nil || nameCols == nil {
		t.Fatal("expected age and name columns")
	}

	if ageCols.DataType() != dtype.Int64 {
		t.Errorf("expected Int64 for age, got %s", ageCols.DataType())
	}
	v, ok := ageCols.GetInt64(0)
	if !ok || v != 25 {
		t.Errorf("expected 25, got %d", v)
	}

	if nameCols.DataType() != dtype.String {
		t.Errorf("expected String for name, got %s", nameCols.DataType())
	}
	sv, ok := nameCols.GetString(1)
	if !ok || sv != "Bob" {
		t.Errorf("expected Bob, got %q", sv)
	}
}

func TestReadJSONWithNulls(t *testing.T) {
	input := `[{"x":1,"y":"hello"},{"x":null,"y":"world"},{"x":3}]`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	var xCol, yCol *series.Series
	for _, c := range cols {
		if c.Name() == "x" {
			xCol = c
		}
		if c.Name() == "y" {
			yCol = c
		}
	}

	if xCol == nil || yCol == nil {
		t.Fatal("expected x and y columns")
	}

	if !xCol.IsNull(1) {
		t.Error("expected x[1] to be null")
	}
	if !yCol.IsNull(2) {
		t.Error("expected y[2] to be null (missing key)")
	}
}

func TestReadNDJSON(t *testing.T) {
	input := `{"name":"Alice","score":88.5}
{"name":"Bob","score":92.3}
{"name":"Charlie","score":76.1}
`
	cols, err := ReadNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
	if cols[0].Len() != 3 {
		t.Errorf("expected 3 rows, got %d", cols[0].Len())
	}
}

func TestWriteJSON(t *testing.T) {
	cols := []*series.Series{
		series.NewString("name", []string{"Alice", "Bob"}),
		series.NewInt64("age", []int64{25, 30}),
	}

	var buf bytes.Buffer
	err := Write(&buf, cols)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Alice") {
		t.Errorf("expected Alice in output:\n%s", output)
	}

	// Round-trip
	cols2, err := Read(strings.NewReader(output))
	if err != nil {
		t.Fatalf("round-trip read failed: %v", err)
	}
	if len(cols2) != 2 {
		t.Errorf("round-trip: expected 2 columns, got %d", len(cols2))
	}
}

func TestWriteNDJSON(t *testing.T) {
	cols := []*series.Series{
		series.NewString("x", []string{"a", "b"}),
		series.NewFloat64("y", []float64{1.5, 2.5}),
	}

	var buf bytes.Buffer
	err := WriteNDJSON(&buf, cols)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 2 lines
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Round-trip
	cols2, err := ReadNDJSON(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if cols2[0].Len() != 2 {
		t.Errorf("round-trip: expected 2 rows, got %d", cols2[0].Len())
	}
}

func TestReadJSONBooleans(t *testing.T) {
	input := `[{"flag":true},{"flag":false},{"flag":true}]`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].DataType() != dtype.Boolean {
		t.Errorf("expected Boolean, got %s", cols[0].DataType())
	}
	v, ok := cols[0].GetBool(0)
	if !ok || !v {
		t.Error("expected true")
	}
}

func TestReadJSONFloats(t *testing.T) {
	input := `[{"val":1.5},{"val":2.7}]`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if cols[0].DataType() != dtype.Float64 {
		t.Errorf("expected Float64, got %s", cols[0].DataType())
	}
}

func TestEmptyJSON(t *testing.T) {
	input := `[]`
	cols, err := Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if cols != nil {
		t.Errorf("expected nil columns for empty array, got %d", len(cols))
	}
}
