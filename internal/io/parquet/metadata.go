// Package parquet provides a minimal Apache Parquet reader and writer for
// the golars DataFrame library. It supports PLAIN encoding with no compression.
package parquet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Parquet magic bytes.
var magic = [4]byte{'P', 'A', 'R', '1'}

// Parquet physical types.
const (
	TypeBoolean   = 0
	TypeInt32     = 1
	TypeInt64     = 2
	TypeFloat     = 4
	TypeDouble    = 5
	TypeByteArray = 6
)

// Repetition types.
const (
	RepRequired = 0
	RepOptional = 1
)

// Codec types.
const (
	CodecUncompressed = 0
	CodecSnappy       = 2
)

// Page types.
const (
	PageDataPage = 0
)

// Encoding types.
const (
	EncodingPlain = 0
)

// Thrift compact protocol type IDs.
const (
	thriftBool   = 1
	thriftI32    = 5
	thriftI64    = 6
	thriftBinary = 8
	thriftList   = 12
	thriftStruct = 12
	// In compact protocol, list uses type 12 for struct elements.
	// The actual list header type id is 15 in the field header.
	thriftListField = 15
)

// zigzagEncode encodes a signed int64 as a zigzag-encoded uint64.
func zigzagEncode(n int64) uint64 {
	return uint64((n << 1) ^ (n >> 63))
}

// zigzagDecode decodes a zigzag-encoded uint64 to a signed int64.
func zigzagDecode(n uint64) int64 {
	return int64((n >> 1) ^ -(n & 1))
}

// appendVarint appends a varint-encoded uint64 to a buffer.
func appendVarint(buf *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}

// readVarint reads a varint-encoded uint64 from a reader.
func readVarint(r io.ByteReader) (uint64, error) {
	var result uint64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("golars: parquet: reading varint: %w", err)
		}
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("golars: parquet: varint overflow")
		}
	}
}

// ---------------------------------------------------------------------------
// Thrift compact protocol writer
// ---------------------------------------------------------------------------

type thriftWriter struct {
	buf     *bytes.Buffer
	lastFid int
}

func newThriftWriter() *thriftWriter {
	return &thriftWriter{buf: &bytes.Buffer{}}
}

func (w *thriftWriter) writeFieldHeader(fid int, ftype byte) {
	delta := fid - w.lastFid
	if delta > 0 && delta <= 15 {
		w.buf.WriteByte(byte(delta<<4) | ftype)
	} else {
		w.buf.WriteByte(ftype)
		appendVarint(w.buf, zigzagEncode(int64(fid)))
	}
	w.lastFid = fid
}

func (w *thriftWriter) writeBool(fid int, v bool) {
	// In compact protocol, bool true is type 1, false is type 2.
	if v {
		w.writeFieldHeader(fid, 1)
	} else {
		w.writeFieldHeader(fid, 2)
	}
}

func (w *thriftWriter) writeI32(fid int, v int32) {
	w.writeFieldHeader(fid, 5)
	appendVarint(w.buf, zigzagEncode(int64(v)))
}

func (w *thriftWriter) writeI64(fid int, v int64) {
	w.writeFieldHeader(fid, 6)
	appendVarint(w.buf, zigzagEncode(v))
}

func (w *thriftWriter) writeBinary(fid int, v []byte) {
	w.writeFieldHeader(fid, 8)
	appendVarint(w.buf, uint64(len(v)))
	w.buf.Write(v)
}

func (w *thriftWriter) writeListBegin(fid int, elemType byte, size int) {
	// Field header uses type 12 for the list field? No, list field type = 15 in Thrift compact.
	// Actually in compact protocol the field type for list is 12... let me re-check.
	// Compact protocol type mapping:
	//   1 = bool true, 2 = bool false, 3 = i8, 4 = i16, 5 = i32, 6 = i64,
	//   7 = double, 8 = binary, 9 = list, 10 = set, 11 = map, 12 = struct
	// Wait, that's different. Let me use the correct compact protocol type IDs.
	w.writeFieldHeader(fid, 9) // 9 = list in compact protocol
	if size < 15 {
		w.buf.WriteByte(byte(size<<4) | elemType)
	} else {
		w.buf.WriteByte(0xf0 | elemType)
		appendVarint(w.buf, uint64(size))
	}
}

func (w *thriftWriter) writeStructBegin(fid int) {
	w.writeFieldHeader(fid, 12) // 12 = struct
	w.lastFid = 0               // reset for nested struct
}

func (w *thriftWriter) writeStructEnd() {
	// Does not write anything for struct end per se in compact protocol; we
	// just need a stop byte which the caller writes via writeStop.
}

func (w *thriftWriter) writeStop() {
	w.buf.WriteByte(0)
}

func (w *thriftWriter) bytes() []byte {
	return w.buf.Bytes()
}

// ---------------------------------------------------------------------------
// Thrift compact protocol reader
// ---------------------------------------------------------------------------

type thriftReader struct {
	r       *bytes.Reader
	lastFid int
}

func newThriftReader(data []byte) *thriftReader {
	return &thriftReader{r: bytes.NewReader(data)}
}

// readFieldHeader reads the next field header. Returns fid=0 and ftype=0 for
// the stop marker.
func (r *thriftReader) readFieldHeader() (fid int, ftype int, err error) {
	b, err := r.r.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("golars: parquet: reading field header: %w", err)
	}
	if b == 0 {
		return 0, 0, nil // stop
	}
	ftype = int(b & 0x0f)
	delta := int(b >> 4)
	if delta == 0 {
		// Read the field id as a zigzag varint.
		v, err := readVarint(r.r)
		if err != nil {
			return 0, 0, err
		}
		fid = int(zigzagDecode(v))
	} else {
		fid = r.lastFid + delta
	}
	r.lastFid = fid
	return fid, ftype, nil
}

func (r *thriftReader) readBool(ftype int) bool {
	// In compact protocol, bool is encoded in the type:
	// type 1 = true, type 2 = false.
	return ftype == 1
}

func (r *thriftReader) readI32() (int32, error) {
	v, err := readVarint(r.r)
	if err != nil {
		return 0, err
	}
	return int32(zigzagDecode(v)), nil
}

func (r *thriftReader) readI64() (int64, error) {
	v, err := readVarint(r.r)
	if err != nil {
		return 0, err
	}
	return zigzagDecode(v), nil
}

func (r *thriftReader) readBinary() ([]byte, error) {
	length, err := readVarint(r.r)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	_, err = io.ReadFull(r.r, data)
	if err != nil {
		return nil, fmt.Errorf("golars: parquet: reading binary data: %w", err)
	}
	return data, nil
}

func (r *thriftReader) readListHeader() (elemType int, size int, err error) {
	b, err := r.r.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("golars: parquet: reading list header: %w", err)
	}
	elemType = int(b & 0x0f)
	size = int(b >> 4)
	if size == 15 {
		v, err := readVarint(r.r)
		if err != nil {
			return 0, 0, err
		}
		size = int(v)
	}
	return elemType, size, nil
}

// skip skips over a value of the given compact protocol type.
func (r *thriftReader) skip(ftype int) error {
	switch ftype {
	case 1, 2: // bool true/false — no extra bytes
		return nil
	case 3: // i8
		_, err := r.r.ReadByte()
		return err
	case 4: // i16
		_, err := readVarint(r.r)
		return err
	case 5: // i32
		_, err := readVarint(r.r)
		return err
	case 6: // i64
		_, err := readVarint(r.r)
		return err
	case 7: // double — 8 bytes
		tmp := make([]byte, 8)
		_, err := io.ReadFull(r.r, tmp)
		return err
	case 8: // binary
		_, err := r.readBinary()
		return err
	case 9: // list
		elemType, size, err := r.readListHeader()
		if err != nil {
			return err
		}
		for i := 0; i < size; i++ {
			if err := r.skip(elemType); err != nil {
				return err
			}
		}
		return nil
	case 12: // struct
		return r.skipStruct()
	default:
		return fmt.Errorf("golars: parquet: cannot skip thrift type %d", ftype)
	}
}

func (r *thriftReader) skipStruct() error {
	savedFid := r.lastFid
	r.lastFid = 0
	for {
		_, ftype, err := r.readFieldHeader()
		if err != nil {
			return err
		}
		if ftype == 0 {
			break // stop byte
		}
		if err := r.skip(ftype); err != nil {
			return err
		}
	}
	r.lastFid = savedFid
	return nil
}

// ---------------------------------------------------------------------------
// Metadata structures
// ---------------------------------------------------------------------------

// schemaElement represents a Parquet schema element.
type schemaElement struct {
	Name           string
	NumChildren    int32
	HasNumChildren bool
	Type           int32
	HasType        bool
	RepetitionType int32
	HasRepetition  bool
}

// columnMetaData holds metadata for a single column chunk.
type columnMetaData struct {
	Type                  int32
	PathInSchema          []string
	Codec                 int32
	NumValues             int64
	TotalUncompressedSize int64
	TotalCompressedSize   int64
	DataPageOffset        int64
}

// columnChunk represents a column chunk in a row group.
type columnChunk struct {
	FileOffset int64
	MetaData   columnMetaData
}

// rowGroup represents a row group in the file.
type rowGroup struct {
	Columns       []columnChunk
	TotalByteSize int64
	NumRows       int64
}

// fileMetaData is the top-level file metadata.
type fileMetaData struct {
	Version   int32
	Schema    []schemaElement
	NumRows   int64
	RowGroups []rowGroup
}

// pageHeader describes a data page.
type pageHeader struct {
	Type               int32
	UncompressedSize   int32
	CompressedSize     int32
	DataPageNumValues  int32
	DataPageEncoding   int32
	DataPageDefEnc     int32
	DataPageRepEnc     int32
	HasDataPageHeader  bool
}

// ---------------------------------------------------------------------------
// Metadata encoding
// ---------------------------------------------------------------------------

func encodeFileMetaData(md *fileMetaData) []byte {
	w := newThriftWriter()

	// field 1: version (i32)
	w.writeI32(1, md.Version)

	// field 2: schema (list<SchemaElement>)
	w.writeListBegin(2, 12, len(md.Schema)) // 12 = struct
	for _, se := range md.Schema {
		encodeSchemaElement(w, &se)
	}

	// field 3: num_rows (i64)
	w.writeI64(3, md.NumRows)

	// field 4: row_groups (list<RowGroup>)
	w.writeListBegin(4, 12, len(md.RowGroups))
	for _, rg := range md.RowGroups {
		encodeRowGroup(w, &rg)
	}

	w.writeStop()
	return w.bytes()
}

func encodeSchemaElement(w *thriftWriter, se *schemaElement) {
	savedFid := w.lastFid
	w.lastFid = 0

	// field 1: name (binary)
	w.writeBinary(1, []byte(se.Name))

	// field 2: num_children (optional i32)
	if se.HasNumChildren {
		w.writeI32(2, se.NumChildren)
	}

	// field 3: type (optional i32)
	if se.HasType {
		w.writeI32(3, se.Type)
	}

	// field 7: repetition_type (optional i32)
	if se.HasRepetition {
		w.writeI32(7, se.RepetitionType)
	}

	w.writeStop()
	w.lastFid = savedFid
}

func encodeRowGroup(w *thriftWriter, rg *rowGroup) {
	savedFid := w.lastFid
	w.lastFid = 0

	// field 1: columns (list<ColumnChunk>)
	w.writeListBegin(1, 12, len(rg.Columns))
	for _, cc := range rg.Columns {
		encodeColumnChunk(w, &cc)
	}

	// field 2: total_byte_size (i64)
	w.writeI64(2, rg.TotalByteSize)

	// field 3: num_rows (i64)
	w.writeI64(3, rg.NumRows)

	w.writeStop()
	w.lastFid = savedFid
}

func encodeColumnChunk(w *thriftWriter, cc *columnChunk) {
	savedFid := w.lastFid
	w.lastFid = 0

	// field 2: file_offset (i64)
	w.writeI64(2, cc.FileOffset)

	// field 3: meta_data (struct ColumnMetaData)
	w.writeStructBegin(3)
	encodeColumnMetaDataFields(w, &cc.MetaData)

	w.writeStop() // end of ColumnChunk struct
	w.lastFid = savedFid
}

func encodeColumnMetaDataFields(w *thriftWriter, cmd *columnMetaData) {
	// field 1: type (i32)
	w.writeI32(1, cmd.Type)

	// field 2: encodings (list<i32>) — we write PLAIN (0)
	w.writeListBegin(2, 5, 1) // 5 = i32
	appendVarint(w.buf, zigzagEncode(int64(EncodingPlain)))

	// field 3: path_in_schema (list<binary>)
	w.writeListBegin(3, 8, len(cmd.PathInSchema)) // 8 = binary
	for _, p := range cmd.PathInSchema {
		appendVarint(w.buf, uint64(len(p)))
		w.buf.Write([]byte(p))
	}

	// field 4: codec (i32)
	w.writeI32(4, cmd.Codec)

	// field 5: num_values (i64)
	w.writeI64(5, cmd.NumValues)

	// field 6: total_uncompressed_size (i64)
	w.writeI64(6, cmd.TotalUncompressedSize)

	// field 7: total_compressed_size (i64)
	w.writeI64(7, cmd.TotalCompressedSize)

	// field 8: data_page_offset (i64)
	w.writeI64(8, cmd.DataPageOffset)

	w.writeStop() // end of ColumnMetaData struct
}

func encodePageHeader(ph *pageHeader) []byte {
	w := newThriftWriter()

	// field 1: type (i32)
	w.writeI32(1, ph.Type)

	// field 2: uncompressed_page_size (i32)
	w.writeI32(2, ph.UncompressedSize)

	// field 3: compressed_page_size (i32)
	w.writeI32(3, ph.CompressedSize)

	// field 5: data_page_header (struct, optional)
	if ph.HasDataPageHeader {
		w.writeStructBegin(5)
		// field 1: num_values
		w.writeI32(1, ph.DataPageNumValues)
		// field 2: encoding
		w.writeI32(2, ph.DataPageEncoding)
		// field 3: definition_level_encoding
		w.writeI32(3, ph.DataPageDefEnc)
		// field 4: repetition_level_encoding
		w.writeI32(4, ph.DataPageRepEnc)
		w.writeStop() // end DataPageHeader
	}

	w.writeStop()
	return w.bytes()
}

// ---------------------------------------------------------------------------
// Metadata decoding
// ---------------------------------------------------------------------------

func decodeFileMetaData(data []byte) (*fileMetaData, error) {
	r := newThriftReader(data)
	md := &fileMetaData{}

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return nil, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // version
			md.Version, err = r.readI32()
		case 2: // schema
			md.Schema, err = decodeSchemaList(r)
		case 3: // num_rows
			md.NumRows, err = r.readI64()
		case 4: // row_groups
			md.RowGroups, err = decodeRowGroupList(r)
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return nil, err
		}
	}
	return md, nil
}

func decodeSchemaList(r *thriftReader) ([]schemaElement, error) {
	_, size, err := r.readListHeader()
	if err != nil {
		return nil, err
	}
	elements := make([]schemaElement, size)
	for i := 0; i < size; i++ {
		elements[i], err = decodeSchemaElement(r)
		if err != nil {
			return nil, err
		}
	}
	return elements, nil
}

func decodeSchemaElement(r *thriftReader) (schemaElement, error) {
	se := schemaElement{}
	savedFid := r.lastFid
	r.lastFid = 0

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return se, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // name
			b, err := r.readBinary()
			if err != nil {
				return se, err
			}
			se.Name = string(b)
		case 2: // num_children
			se.NumChildren, err = r.readI32()
			se.HasNumChildren = true
		case 3: // type
			se.Type, err = r.readI32()
			se.HasType = true
		case 7: // repetition_type
			se.RepetitionType, err = r.readI32()
			se.HasRepetition = true
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return se, err
		}
	}

	r.lastFid = savedFid
	return se, nil
}

func decodeRowGroupList(r *thriftReader) ([]rowGroup, error) {
	_, size, err := r.readListHeader()
	if err != nil {
		return nil, err
	}
	groups := make([]rowGroup, size)
	for i := 0; i < size; i++ {
		groups[i], err = decodeRowGroup(r)
		if err != nil {
			return nil, err
		}
	}
	return groups, nil
}

func decodeRowGroup(r *thriftReader) (rowGroup, error) {
	rg := rowGroup{}
	savedFid := r.lastFid
	r.lastFid = 0

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return rg, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // columns
			rg.Columns, err = decodeColumnChunkList(r)
		case 2: // total_byte_size
			rg.TotalByteSize, err = r.readI64()
		case 3: // num_rows
			rg.NumRows, err = r.readI64()
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return rg, err
		}
	}

	r.lastFid = savedFid
	return rg, nil
}

func decodeColumnChunkList(r *thriftReader) ([]columnChunk, error) {
	_, size, err := r.readListHeader()
	if err != nil {
		return nil, err
	}
	chunks := make([]columnChunk, size)
	for i := 0; i < size; i++ {
		chunks[i], err = decodeColumnChunk(r)
		if err != nil {
			return nil, err
		}
	}
	return chunks, nil
}

func decodeColumnChunk(r *thriftReader) (columnChunk, error) {
	cc := columnChunk{}
	savedFid := r.lastFid
	r.lastFid = 0

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return cc, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 2: // file_offset
			cc.FileOffset, err = r.readI64()
		case 3: // meta_data
			cc.MetaData, err = decodeColumnMetaData(r)
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return cc, err
		}
	}

	r.lastFid = savedFid
	return cc, nil
}

func decodeColumnMetaData(r *thriftReader) (columnMetaData, error) {
	cmd := columnMetaData{}
	savedFid := r.lastFid
	r.lastFid = 0

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return cmd, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // type
			cmd.Type, err = r.readI32()
		case 2: // encodings — skip the list
			err = r.skip(ftype)
		case 3: // path_in_schema
			cmd.PathInSchema, err = decodeBinaryList(r)
		case 4: // codec
			cmd.Codec, err = r.readI32()
		case 5: // num_values
			cmd.NumValues, err = r.readI64()
		case 6: // total_uncompressed_size
			cmd.TotalUncompressedSize, err = r.readI64()
		case 7: // total_compressed_size
			cmd.TotalCompressedSize, err = r.readI64()
		case 8: // data_page_offset
			cmd.DataPageOffset, err = r.readI64()
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return cmd, err
		}
	}

	r.lastFid = savedFid
	return cmd, nil
}

func decodeBinaryList(r *thriftReader) ([]string, error) {
	_, size, err := r.readListHeader()
	if err != nil {
		return nil, err
	}
	result := make([]string, size)
	for i := 0; i < size; i++ {
		b, err := r.readBinary()
		if err != nil {
			return nil, err
		}
		result[i] = string(b)
	}
	return result, nil
}

func decodePageHeader(data []byte) (*pageHeader, int, error) {
	r := newThriftReader(data)
	ph := &pageHeader{}

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return nil, 0, err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // type
			ph.Type, err = r.readI32()
		case 2: // uncompressed_page_size
			ph.UncompressedSize, err = r.readI32()
		case 3: // compressed_page_size
			ph.CompressedSize, err = r.readI32()
		case 5: // data_page_header
			ph.HasDataPageHeader = true
			err = decodeDataPageHeaderInto(r, ph)
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return nil, 0, err
		}
	}

	consumed := len(data) - r.r.Len()
	return ph, consumed, nil
}

func decodeDataPageHeaderInto(r *thriftReader, ph *pageHeader) error {
	savedFid := r.lastFid
	r.lastFid = 0

	for {
		fid, ftype, err := r.readFieldHeader()
		if err != nil {
			return err
		}
		if ftype == 0 {
			break
		}

		switch fid {
		case 1: // num_values
			ph.DataPageNumValues, err = r.readI32()
		case 2: // encoding
			ph.DataPageEncoding, err = r.readI32()
		case 3: // definition_level_encoding
			ph.DataPageDefEnc, err = r.readI32()
		case 4: // repetition_level_encoding
			ph.DataPageRepEnc, err = r.readI32()
		default:
			err = r.skip(ftype)
		}
		if err != nil {
			return err
		}
	}

	r.lastFid = savedFid
	return nil
}

// writeLE32 writes a little-endian int32 to a writer.
func writeLE32(w io.Writer, v int32) error {
	return binary.Write(w, binary.LittleEndian, v)
}
