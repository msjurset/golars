package parquet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/msjurseth/golars/internal/array"
	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/dtype"
	"github.com/msjurseth/golars/internal/series"
)

// WriteFile writes a DataFrame to a Parquet file at the given path.
func WriteFile(path string, df *dataframe.DataFrame) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: parquet: creating file: %w", err)
	}
	defer f.Close()

	if err := Write(f, df); err != nil {
		return err
	}
	return f.Close()
}

// Write writes a DataFrame as Parquet to an io.Writer.
func Write(w io.Writer, df *dataframe.DataFrame) error {
	// Write magic.
	if _, err := w.Write(magic[:]); err != nil {
		return fmt.Errorf("golars: parquet: writing magic: %w", err)
	}

	columns := df.Columns()
	numRows := int64(df.Height())
	numCols := len(columns)

	// Encode each column's data page and track offsets.
	type colPage struct {
		headerBytes []byte
		dataBytes   []byte
		offset      int64
	}

	var pages []colPage
	offset := int64(4) // after the initial magic

	for _, col := range columns {
		pageData, err := encodeColumnData(col)
		if err != nil {
			return err
		}

		ph := &pageHeader{
			Type:              PageDataPage,
			UncompressedSize:  int32(len(pageData)),
			CompressedSize:    int32(len(pageData)),
			HasDataPageHeader: true,
			DataPageNumValues: int32(col.Len()),
			DataPageEncoding:  EncodingPlain,
			DataPageDefEnc:    EncodingPlain,
			DataPageRepEnc:    EncodingPlain,
		}
		headerData := encodePageHeader(ph)

		pages = append(pages, colPage{
			headerBytes: headerData,
			dataBytes:   pageData,
			offset:      offset,
		})
		offset += int64(len(headerData)) + int64(len(pageData))
	}

	// Write all column data pages.
	for _, p := range pages {
		if _, err := w.Write(p.headerBytes); err != nil {
			return fmt.Errorf("golars: parquet: writing page header: %w", err)
		}
		if _, err := w.Write(p.dataBytes); err != nil {
			return fmt.Errorf("golars: parquet: writing page data: %w", err)
		}
	}

	// Build file metadata.
	schema := buildSchema(columns)
	chunks := make([]columnChunk, numCols)
	var totalByteSize int64

	for i, p := range pages {
		totalSize := int64(len(p.headerBytes)) + int64(len(p.dataBytes))
		totalByteSize += totalSize
		chunks[i] = columnChunk{
			FileOffset: p.offset,
			MetaData: columnMetaData{
				Type:                  golarsTypeToParquet(columns[i].DataType()),
				PathInSchema:          []string{columns[i].Name()},
				Codec:                 CodecUncompressed,
				NumValues:             int64(columns[i].Len()),
				TotalUncompressedSize: totalSize,
				TotalCompressedSize:   totalSize,
				DataPageOffset:        p.offset,
			},
		}
	}

	md := &fileMetaData{
		Version: 1,
		Schema:  schema,
		NumRows: numRows,
		RowGroups: []rowGroup{
			{
				Columns:       chunks,
				TotalByteSize: totalByteSize,
				NumRows:       numRows,
			},
		},
	}

	metaBytes := encodeFileMetaData(md)

	// Write file metadata.
	if _, err := w.Write(metaBytes); err != nil {
		return fmt.Errorf("golars: parquet: writing metadata: %w", err)
	}

	// Write metadata length as 4-byte LE int32.
	if err := writeLE32(w, int32(len(metaBytes))); err != nil {
		return fmt.Errorf("golars: parquet: writing metadata length: %w", err)
	}

	// Write trailing magic.
	if _, err := w.Write(magic[:]); err != nil {
		return fmt.Errorf("golars: parquet: writing trailing magic: %w", err)
	}

	return nil
}

// buildSchema creates the schema elements for the file metadata.
// The first element is the root with num_children equal to the column count.
func buildSchema(columns []*series.Series) []schemaElement {
	elements := make([]schemaElement, 0, len(columns)+1)

	// Root schema element.
	elements = append(elements, schemaElement{
		Name:           "schema",
		NumChildren:    int32(len(columns)),
		HasNumChildren: true,
	})

	for _, col := range columns {
		ptype := golarsTypeToParquet(col.DataType())
		rep := int32(RepOptional) // all columns are optional to support nulls
		elements = append(elements, schemaElement{
			Name:           col.Name(),
			Type:           ptype,
			HasType:        true,
			RepetitionType: rep,
			HasRepetition:  true,
		})
	}

	return elements
}

// golarsTypeToParquet maps golars data types to Parquet physical types.
func golarsTypeToParquet(dt dtype.DataType) int32 {
	switch dt {
	case dtype.Boolean:
		return TypeBoolean
	case dtype.Int8, dtype.Int16, dtype.Int32, dtype.UInt8, dtype.UInt16:
		return TypeInt32
	case dtype.Int64, dtype.UInt32, dtype.UInt64:
		return TypeInt64
	case dtype.Float32:
		return TypeFloat
	case dtype.Float64:
		return TypeDouble
	case dtype.String:
		return TypeByteArray
	default:
		return TypeByteArray
	}
}

// encodeColumnData encodes a column's data page payload (definition levels + values).
func encodeColumnData(col *series.Series) ([]byte, error) {
	var buf bytes.Buffer
	n := col.Len()

	// Write definition levels: 1 byte per value (0=null, 1=present).
	for i := 0; i < n; i++ {
		if col.IsNull(i) {
			buf.WriteByte(0)
		} else {
			buf.WriteByte(1)
		}
	}

	// Write data values in PLAIN encoding.
	switch col.DataType() {
	case dtype.Boolean:
		ba := col.BooleanArray()
		if ba == nil {
			return nil, fmt.Errorf("golars: parquet: expected boolean array for column %q", col.Name())
		}
		for i := 0; i < n; i++ {
			if col.IsValid(i) {
				if ba.Value(i) {
					buf.WriteByte(1)
				} else {
					buf.WriteByte(0)
				}
			}
		}

	case dtype.Int32:
		if ta, ok := col.Array().(*array.TypedArray[int32]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, ta.Value(i)); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int32: %w", err)
					}
				}
			}
		}

	case dtype.Int8:
		if ta, ok := col.Array().(*array.TypedArray[int8]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int32(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int32: %w", err)
					}
				}
			}
		}

	case dtype.Int16:
		if ta, ok := col.Array().(*array.TypedArray[int16]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int32(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int32: %w", err)
					}
				}
			}
		}

	case dtype.UInt8:
		if ta, ok := col.Array().(*array.TypedArray[uint8]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int32(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int32: %w", err)
					}
				}
			}
		}

	case dtype.UInt16:
		if ta, ok := col.Array().(*array.TypedArray[uint16]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int32(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int32: %w", err)
					}
				}
			}
		}

	case dtype.Int64:
		if ta, ok := col.Array().(*array.TypedArray[int64]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, ta.Value(i)); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int64: %w", err)
					}
				}
			}
		}

	case dtype.UInt32:
		if ta, ok := col.Array().(*array.TypedArray[uint32]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int64(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int64: %w", err)
					}
				}
			}
		}

	case dtype.UInt64:
		if ta, ok := col.Array().(*array.TypedArray[uint64]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, int64(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding int64: %w", err)
					}
				}
			}
		}

	case dtype.Float32:
		if ta, ok := col.Array().(*array.TypedArray[float32]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, math.Float32bits(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding float32: %w", err)
					}
				}
			}
		}

	case dtype.Float64:
		if ta, ok := col.Array().(*array.TypedArray[float64]); ok {
			for i := 0; i < n; i++ {
				if col.IsValid(i) {
					if err := binary.Write(&buf, binary.LittleEndian, math.Float64bits(ta.Value(i))); err != nil {
						return nil, fmt.Errorf("golars: parquet: encoding float64: %w", err)
					}
				}
			}
		}

	case dtype.String:
		for i := 0; i < n; i++ {
			if col.IsValid(i) {
				v, _ := col.GetString(i)
				b := []byte(v)
				if err := binary.Write(&buf, binary.LittleEndian, int32(len(b))); err != nil {
					return nil, fmt.Errorf("golars: parquet: encoding string length: %w", err)
				}
				buf.Write(b)
			}
		}

	default:
		return nil, fmt.Errorf("golars: parquet: unsupported data type %v for column %q", col.DataType(), col.Name())
	}

	return buf.Bytes(), nil
}

