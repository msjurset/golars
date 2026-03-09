package parquet

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/msjurseth/golars/internal/bitmap"
	"github.com/msjurseth/golars/internal/dataframe"
	"github.com/msjurseth/golars/internal/series"
)

// ReadFile reads a Parquet file into a DataFrame.
func ReadFile(path string) (*dataframe.DataFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("golars: parquet: opening file: %w", err)
	}
	defer f.Close()
	return Read(f)
}

// Read reads Parquet data from an io.ReadSeeker into a DataFrame.
func Read(r io.ReadSeeker) (*dataframe.DataFrame, error) {
	// Verify leading magic.
	var headerMagic [4]byte
	if _, err := io.ReadFull(r, headerMagic[:]); err != nil {
		return nil, fmt.Errorf("golars: parquet: reading header magic: %w", err)
	}
	if headerMagic != magic {
		return nil, fmt.Errorf("golars: parquet: invalid header magic %q", headerMagic)
	}

	// Seek to end - 8 to read metadata length and trailing magic.
	if _, err := r.Seek(-8, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("golars: parquet: seeking to footer: %w", err)
	}

	var metaLen int32
	if err := binary.Read(r, binary.LittleEndian, &metaLen); err != nil {
		return nil, fmt.Errorf("golars: parquet: reading metadata length: %w", err)
	}

	var footerMagic [4]byte
	if _, err := io.ReadFull(r, footerMagic[:]); err != nil {
		return nil, fmt.Errorf("golars: parquet: reading footer magic: %w", err)
	}
	if footerMagic != magic {
		return nil, fmt.Errorf("golars: parquet: invalid footer magic %q", footerMagic)
	}

	// Seek to metadata start and read it.
	metaOffset := -8 - int64(metaLen)
	if _, err := r.Seek(metaOffset, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("golars: parquet: seeking to metadata: %w", err)
	}

	metaBytes := make([]byte, metaLen)
	if _, err := io.ReadFull(r, metaBytes); err != nil {
		return nil, fmt.Errorf("golars: parquet: reading metadata: %w", err)
	}

	md, err := decodeFileMetaData(metaBytes)
	if err != nil {
		return nil, fmt.Errorf("golars: parquet: decoding metadata: %w", err)
	}

	if len(md.RowGroups) == 0 {
		return dataframe.New()
	}

	// Build a map from column name to its schema element (skip the root).
	schemaMap := make(map[string]*schemaElement)
	if len(md.Schema) > 1 {
		for i := 1; i < len(md.Schema); i++ {
			se := md.Schema[i]
			schemaMap[se.Name] = &se
		}
	}

	// Read each column from the first row group.
	rg := md.RowGroups[0]
	numRows := int(rg.NumRows)
	cols := make([]*series.Series, len(rg.Columns))

	for i, cc := range rg.Columns {
		colName := ""
		if len(cc.MetaData.PathInSchema) > 0 {
			colName = cc.MetaData.PathInSchema[len(cc.MetaData.PathInSchema)-1]
		}

		// Seek to the column data page offset.
		if _, err := r.Seek(cc.MetaData.DataPageOffset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("golars: parquet: seeking to column %q: %w", colName, err)
		}

		// Read the total data for this column chunk.
		chunkSize := cc.MetaData.TotalCompressedSize
		chunkData := make([]byte, chunkSize)
		if _, err := io.ReadFull(r, chunkData); err != nil {
			return nil, fmt.Errorf("golars: parquet: reading column %q data: %w", colName, err)
		}

		// Decode page header.
		ph, headerLen, err := decodePageHeader(chunkData)
		if err != nil {
			return nil, fmt.Errorf("golars: parquet: decoding page header for %q: %w", colName, err)
		}

		pageData := chunkData[headerLen : headerLen+int(ph.CompressedSize)]

		// Determine whether column is optional.
		isOptional := true
		if se, ok := schemaMap[colName]; ok {
			isOptional = se.HasRepetition && se.RepetitionType == RepOptional
		}

		col, err := decodeColumnValues(colName, cc.MetaData.Type, pageData, numRows, isOptional)
		if err != nil {
			return nil, err
		}
		cols[i] = col
	}

	return dataframe.New(cols...)
}

// decodeColumnValues decodes a data page into a Series.
func decodeColumnValues(name string, ptype int32, data []byte, numRows int, isOptional bool) (*series.Series, error) {
	if len(data) < numRows && isOptional {
		return nil, fmt.Errorf("golars: parquet: column %q data too short for definition levels", name)
	}

	// Read definition levels.
	var defLevels []byte
	offset := 0
	if isOptional {
		defLevels = data[:numRows]
		offset = numRows
	}

	remaining := data[offset:]

	switch ptype {
	case TypeBoolean:
		return decodeBooleanColumn(name, remaining, defLevels, numRows)
	case TypeInt32:
		return decodeInt32Column(name, remaining, defLevels, numRows)
	case TypeInt64:
		return decodeInt64Column(name, remaining, defLevels, numRows)
	case TypeFloat:
		return decodeFloatColumn(name, remaining, defLevels, numRows)
	case TypeDouble:
		return decodeDoubleColumn(name, remaining, defLevels, numRows)
	case TypeByteArray:
		return decodeByteArrayColumn(name, remaining, defLevels, numRows)
	default:
		return nil, fmt.Errorf("golars: parquet: unsupported parquet type %d for column %q", ptype, name)
	}
}

func hasNulls(defLevels []byte) bool {
	for _, d := range defLevels {
		if d == 0 {
			return true
		}
	}
	return false
}

func buildValidity(defLevels []byte, n int) *bitmap.Bitmap {
	if defLevels == nil || !hasNulls(defLevels) {
		return nil
	}
	v := bitmap.New(n)
	for i, d := range defLevels {
		if d == 0 {
			v.Clear(i)
		}
	}
	return v
}

func decodeBooleanColumn(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	values := make([]bool, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("golars: parquet: boolean data underflow for column %q", name)
			}
			values[i] = data[pos] != 0
			pos++
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewBooleanWithValidity(name, values, valid), nil
	}
	return series.NewBoolean(name, values), nil
}

func decodeInt32Column(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	// Read back as Int64 for golars compatibility.
	values := make([]int64, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos+4 > len(data) {
				return nil, fmt.Errorf("golars: parquet: int32 data underflow for column %q", name)
			}
			v := int32(binary.LittleEndian.Uint32(data[pos : pos+4]))
			values[i] = int64(v)
			pos += 4
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewInt64WithValidity(name, values, valid), nil
	}
	return series.NewInt64(name, values), nil
}

func decodeInt64Column(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	values := make([]int64, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos+8 > len(data) {
				return nil, fmt.Errorf("golars: parquet: int64 data underflow for column %q", name)
			}
			values[i] = int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
			pos += 8
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewInt64WithValidity(name, values, valid), nil
	}
	return series.NewInt64(name, values), nil
}

func decodeFloatColumn(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	// Read as Float64 for golars compatibility.
	values := make([]float64, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos+4 > len(data) {
				return nil, fmt.Errorf("golars: parquet: float data underflow for column %q", name)
			}
			bits := binary.LittleEndian.Uint32(data[pos : pos+4])
			values[i] = float64(math.Float32frombits(bits))
			pos += 4
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewFloat64WithValidity(name, values, valid), nil
	}
	return series.NewFloat64(name, values), nil
}

func decodeDoubleColumn(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	values := make([]float64, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos+8 > len(data) {
				return nil, fmt.Errorf("golars: parquet: double data underflow for column %q", name)
			}
			bits := binary.LittleEndian.Uint64(data[pos : pos+8])
			values[i] = math.Float64frombits(bits)
			pos += 8
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewFloat64WithValidity(name, values, valid), nil
	}
	return series.NewFloat64(name, values), nil
}

func decodeByteArrayColumn(name string, data []byte, defLevels []byte, n int) (*series.Series, error) {
	values := make([]string, n)
	pos := 0
	for i := 0; i < n; i++ {
		if defLevels == nil || defLevels[i] != 0 {
			if pos+4 > len(data) {
				return nil, fmt.Errorf("golars: parquet: byte_array length underflow for column %q", name)
			}
			strLen := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
			pos += 4
			if pos+strLen > len(data) {
				return nil, fmt.Errorf("golars: parquet: byte_array data underflow for column %q", name)
			}
			values[i] = string(data[pos : pos+strLen])
			pos += strLen
		}
	}

	validity := buildValidity(defLevels, n)
	if validity != nil {
		valid := make([]bool, n)
		for i := 0; i < n; i++ {
			valid[i] = defLevels[i] != 0
		}
		return series.NewStringWithValidity(name, values, valid), nil
	}
	return series.NewString(name, values), nil
}
