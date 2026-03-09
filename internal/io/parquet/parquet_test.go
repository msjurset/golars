package parquet

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/series"
)

func TestParquetRoundTrip(t *testing.T) {
	df, err := dataframe.New(
		series.NewString("name", []string{"Alice", "Bob", "Charlie"}),
		series.NewInt64("age", []int64{25, 30, 35}),
		series.NewFloat64("score", []float64{88.5, 92.3, 76.1}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if df2.Height() != 3 {
		t.Errorf("expected height 3, got %d", df2.Height())
	}
	if df2.Width() != 3 {
		t.Errorf("expected width 3, got %d", df2.Width())
	}

	// Verify column names.
	cols := df2.Columns()
	expectedNames := []string{"name", "age", "score"}
	for i, name := range expectedNames {
		if cols[i].Name() != name {
			t.Errorf("column %d: expected name %q, got %q", i, name, cols[i].Name())
		}
	}

	// Verify string values.
	for i, expected := range []string{"Alice", "Bob", "Charlie"} {
		v, ok := cols[0].GetString(i)
		if !ok {
			t.Errorf("name[%d]: expected valid value", i)
		}
		if v != expected {
			t.Errorf("name[%d]: expected %q, got %q", i, expected, v)
		}
	}

	// Verify int64 values.
	for i, expected := range []int64{25, 30, 35} {
		v, ok := cols[1].GetInt64(i)
		if !ok {
			t.Errorf("age[%d]: expected valid value", i)
		}
		if v != expected {
			t.Errorf("age[%d]: expected %d, got %d", i, expected, v)
		}
	}

	// Verify float64 values.
	for i, expected := range []float64{88.5, 92.3, 76.1} {
		v, ok := cols[2].GetFloat64(i)
		if !ok {
			t.Errorf("score[%d]: expected valid value", i)
		}
		if v != expected {
			t.Errorf("score[%d]: expected %g, got %g", i, expected, v)
		}
	}
}

func TestParquetWithNulls(t *testing.T) {
	df, err := dataframe.New(
		series.NewInt64WithValidity("x", []int64{1, 0, 3}, []bool{true, false, true}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "nulls.parquet")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	col := df2.Columns()[0]
	if col.Name() != "x" {
		t.Errorf("expected name 'x', got %q", col.Name())
	}
	if col.Len() != 3 {
		t.Fatalf("expected len 3, got %d", col.Len())
	}

	// Check value at index 0.
	v0, ok := col.GetInt64(0)
	if !ok || v0 != 1 {
		t.Errorf("x[0]: expected 1, got %d (valid=%v)", v0, ok)
	}

	// Check null at index 1.
	if !col.IsNull(1) {
		t.Errorf("x[1]: expected null")
	}

	// Check value at index 2.
	v2, ok := col.GetInt64(2)
	if !ok || v2 != 3 {
		t.Errorf("x[2]: expected 3, got %d (valid=%v)", v2, ok)
	}
}

func TestParquetBoolean(t *testing.T) {
	df, err := dataframe.New(
		series.NewBoolean("flag", []bool{true, false, true}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bool.parquet")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	col := df2.Columns()[0]
	if col.Name() != "flag" {
		t.Errorf("expected name 'flag', got %q", col.Name())
	}

	expected := []bool{true, false, true}
	for i, exp := range expected {
		v, ok := col.GetBool(i)
		if !ok {
			t.Errorf("flag[%d]: expected valid value", i)
		}
		if v != exp {
			t.Errorf("flag[%d]: expected %v, got %v", i, exp, v)
		}
	}
}

func TestParquetBooleanWithNulls(t *testing.T) {
	df, err := dataframe.New(
		series.NewBooleanWithValidity("flag", []bool{true, false, true}, []bool{true, false, true}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bool_nulls.parquet")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	col := df2.Columns()[0]
	if !col.IsNull(1) {
		t.Errorf("flag[1]: expected null")
	}

	v0, ok := col.GetBool(0)
	if !ok || v0 != true {
		t.Errorf("flag[0]: expected true, got %v (valid=%v)", v0, ok)
	}

	v2, ok := col.GetBool(2)
	if !ok || v2 != true {
		t.Errorf("flag[2]: expected true, got %v (valid=%v)", v2, ok)
	}
}

func TestParquetStringWithNulls(t *testing.T) {
	df, err := dataframe.New(
		series.NewStringWithValidity("s", []string{"hello", "", "world"}, []bool{true, false, true}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "str_nulls.parquet")

	if err := WriteFile(path, df); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	df2, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	col := df2.Columns()[0]
	v0, ok := col.GetString(0)
	if !ok || v0 != "hello" {
		t.Errorf("s[0]: expected 'hello', got %q (valid=%v)", v0, ok)
	}
	if !col.IsNull(1) {
		t.Errorf("s[1]: expected null")
	}
	v2, ok := col.GetString(2)
	if !ok || v2 != "world" {
		t.Errorf("s[2]: expected 'world', got %q (valid=%v)", v2, ok)
	}
}

func TestParquetWriteRead(t *testing.T) {
	// Test using Write/Read (buffer-based, not file-based).
	df, err := dataframe.New(
		series.NewInt64("id", []int64{10, 20}),
		series.NewString("label", []string{"a", "b"}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	var buf bytes.Buffer
	if err := Write(&buf, df); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())
	df2, err := Read(reader)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if df2.Height() != 2 || df2.Width() != 2 {
		t.Errorf("expected 2x2, got %dx%d", df2.Height(), df2.Width())
	}
}

func TestThriftRoundTrip(t *testing.T) {
	// Test that encoding and decoding file metadata produces the same result.
	md := &fileMetaData{
		Version: 1,
		Schema: []schemaElement{
			{Name: "schema", NumChildren: 1, HasNumChildren: true},
			{Name: "col", Type: TypeInt64, HasType: true, RepetitionType: RepOptional, HasRepetition: true},
		},
		NumRows: 5,
		RowGroups: []rowGroup{
			{
				Columns: []columnChunk{
					{
						FileOffset: 4,
						MetaData: columnMetaData{
							Type:                  TypeInt64,
							PathInSchema:          []string{"col"},
							Codec:                 CodecUncompressed,
							NumValues:             5,
							TotalUncompressedSize: 100,
							TotalCompressedSize:   100,
							DataPageOffset:        4,
						},
					},
				},
				TotalByteSize: 100,
				NumRows:       5,
			},
		},
	}

	encoded := encodeFileMetaData(md)
	decoded, err := decodeFileMetaData(encoded)
	if err != nil {
		t.Fatalf("decoding metadata: %v", err)
	}

	if decoded.Version != md.Version {
		t.Errorf("version: expected %d, got %d", md.Version, decoded.Version)
	}
	if decoded.NumRows != md.NumRows {
		t.Errorf("num_rows: expected %d, got %d", md.NumRows, decoded.NumRows)
	}
	if len(decoded.Schema) != len(md.Schema) {
		t.Fatalf("schema length: expected %d, got %d", len(md.Schema), len(decoded.Schema))
	}
	if decoded.Schema[0].Name != "schema" {
		t.Errorf("root schema name: expected 'schema', got %q", decoded.Schema[0].Name)
	}
	if decoded.Schema[1].Name != "col" {
		t.Errorf("col schema name: expected 'col', got %q", decoded.Schema[1].Name)
	}
	if len(decoded.RowGroups) != 1 {
		t.Fatalf("row groups: expected 1, got %d", len(decoded.RowGroups))
	}
	if decoded.RowGroups[0].NumRows != 5 {
		t.Errorf("row group num_rows: expected 5, got %d", decoded.RowGroups[0].NumRows)
	}
}

func TestParquetEmptyDataFrame(t *testing.T) {
	df, err := dataframe.New(
		series.NewInt64("x", []int64{}),
	)
	if err != nil {
		t.Fatalf("creating dataframe: %v", err)
	}

	var buf bytes.Buffer
	if err := Write(&buf, df); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())
	df2, err := Read(reader)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if df2.Height() != 0 {
		t.Errorf("expected height 0, got %d", df2.Height())
	}
}
