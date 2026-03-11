package json

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/msjurset/golars/internal/array"
	"github.com/msjurset/golars/internal/bitmap"
	"github.com/msjurset/golars/internal/dtype"
	"github.com/msjurset/golars/internal/series"
)

const writerBufSize = 65536

// WriteFile writes Series columns as a JSON array-of-objects to a file.
func WriteFile(path string, columns []*series.Series) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: json: %w", err)
	}
	defer f.Close()
	return Write(f, columns)
}

// Write writes Series columns as a JSON array of objects to an io.Writer.
func Write(w io.Writer, columns []*series.Series) error {
	if len(columns) == 0 {
		_, err := w.Write([]byte("[]\n"))
		return err
	}

	bw := bufio.NewWriterSize(w, writerBufSize)

	height := columns[0].Len()
	writers := makeColumnWriters(columns)

	bw.WriteByte('[')
	for i := 0; i < height; i++ {
		if i > 0 {
			bw.WriteByte(',')
		}
		bw.WriteByte('{')
		for j, cw := range writers {
			if j > 0 {
				bw.WriteByte(',')
			}
			bw.WriteString(cw.quotedName)
			bw.WriteByte(':')
			cw.writeValue(bw, i)
		}
		bw.WriteByte('}')
	}
	bw.WriteByte(']')
	bw.WriteByte('\n')

	return bw.Flush()
}

// WriteNDJSONFile writes Series columns as NDJSON to a file.
func WriteNDJSONFile(path string, columns []*series.Series) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("golars: ndjson: %w", err)
	}
	defer f.Close()
	return WriteNDJSON(f, columns)
}

// WriteNDJSON writes Series columns as newline-delimited JSON to an io.Writer.
func WriteNDJSON(w io.Writer, columns []*series.Series) error {
	if len(columns) == 0 {
		return nil
	}

	bw := bufio.NewWriterSize(w, writerBufSize)

	height := columns[0].Len()
	writers := makeColumnWriters(columns)

	for i := 0; i < height; i++ {
		bw.WriteByte('{')
		for j, cw := range writers {
			if j > 0 {
				bw.WriteByte(',')
			}
			bw.WriteString(cw.quotedName)
			bw.WriteByte(':')
			cw.writeValue(bw, i)
		}
		bw.WriteByte('}')
		bw.WriteByte('\n')
	}

	return bw.Flush()
}

// columnWriter holds pre-computed state for writing a single column's values.
type columnWriter struct {
	quotedName string
	writeValue func(w *bufio.Writer, i int)
}

// makeColumnWriters creates optimized per-column writers that access underlying
// arrays directly, avoiding per-cell interface dispatch and method overhead.
func makeColumnWriters(columns []*series.Series) []columnWriter {
	writers := make([]columnWriter, len(columns))
	// Reusable scratch buffer for strconv.Append* functions.
	buf := make([]byte, 0, 64)

	for ci, c := range columns {
		name := c.Name()
		qn := strconv.Quote(name)
		writers[ci].quotedName = qn

		arr := c.Array()
		dt := c.DataType()
		validity := arr.Validity()

		switch dt {
		case dtype.Int8:
			ta := arr.(*array.TypedArray[int8])
			vals := ta.Values()
			writers[ci].writeValue = makeIntWriter(vals, validity, buf)
		case dtype.Int16:
			ta := arr.(*array.TypedArray[int16])
			vals := ta.Values()
			writers[ci].writeValue = makeIntWriter(vals, validity, buf)
		case dtype.Int32:
			ta := arr.(*array.TypedArray[int32])
			vals := ta.Values()
			writers[ci].writeValue = makeIntWriter(vals, validity, buf)
		case dtype.Int64:
			ta := arr.(*array.TypedArray[int64])
			vals := ta.Values()
			writers[ci].writeValue = makeIntWriter(vals, validity, buf)
		case dtype.UInt8:
			ta := arr.(*array.TypedArray[uint8])
			vals := ta.Values()
			writers[ci].writeValue = makeUintWriter(vals, validity, buf)
		case dtype.UInt16:
			ta := arr.(*array.TypedArray[uint16])
			vals := ta.Values()
			writers[ci].writeValue = makeUintWriter(vals, validity, buf)
		case dtype.UInt32:
			ta := arr.(*array.TypedArray[uint32])
			vals := ta.Values()
			writers[ci].writeValue = makeUintWriter(vals, validity, buf)
		case dtype.UInt64:
			ta := arr.(*array.TypedArray[uint64])
			vals := ta.Values()
			writers[ci].writeValue = makeUintWriter(vals, validity, buf)
		case dtype.Float32:
			ta := arr.(*array.TypedArray[float32])
			vals := ta.Values()
			writers[ci].writeValue = makeFloat32Writer(vals, validity, buf)
		case dtype.Float64:
			ta := arr.(*array.TypedArray[float64])
			vals := ta.Values()
			writers[ci].writeValue = makeFloat64Writer(vals, validity, buf)
		case dtype.Boolean:
			ba := arr.(*array.BooleanArray)
			writers[ci].writeValue = makeBoolWriter(ba, validity)
		case dtype.String:
			sa := arr.(*array.StringArray)
			writers[ci].writeValue = makeStringWriter(sa, validity)
		case dtype.Date:
			ta := arr.(*array.TypedArray[int32])
			vals := ta.Values()
			writers[ci].writeValue = makeDateWriter(vals, validity, buf)
		case dtype.DateTime:
			ta := arr.(*array.TypedArray[int64])
			vals := ta.Values()
			writers[ci].writeValue = makeDateTimeWriter(vals, validity, buf)
		case dtype.Time:
			ta := arr.(*array.TypedArray[int64])
			vals := ta.Values()
			writers[ci].writeValue = makeTimeWriter(vals, validity, buf)
		case dtype.Duration:
			ta := arr.(*array.TypedArray[int64])
			vals := ta.Values()
			writers[ci].writeValue = makeDurationWriter(vals, validity, buf)
		default:
			writers[ci].writeValue = func(w *bufio.Writer, i int) {
				w.WriteString("null")
			}
		}
	}
	return writers
}

type signedInt interface {
	~int8 | ~int16 | ~int32 | ~int64
}

type unsignedInt interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

func makeIntWriter[T signedInt](vals []T, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			b := strconv.AppendInt(buf[:0], int64(vals[i]), 10)
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		b := strconv.AppendInt(buf[:0], int64(vals[i]), 10)
		w.Write(b)
	}
}

func makeUintWriter[T unsignedInt](vals []T, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			b := strconv.AppendUint(buf[:0], uint64(vals[i]), 10)
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		b := strconv.AppendUint(buf[:0], uint64(vals[i]), 10)
		w.Write(b)
	}
}

func makeFloat64Writer(vals []float64, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			writeFloat64(w, vals[i], buf)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		writeFloat64(w, vals[i], buf)
	}
}

func makeFloat32Writer(vals []float32, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			writeFloat32(w, vals[i], buf)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		writeFloat32(w, vals[i], buf)
	}
}

func writeFloat64(w *bufio.Writer, v float64, buf []byte) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		w.WriteString("null")
		return
	}
	b := strconv.AppendFloat(buf[:0], v, 'f', -1, 64)
	w.Write(b)
}

func writeFloat32(w *bufio.Writer, v float32, buf []byte) {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		w.WriteString("null")
		return
	}
	b := strconv.AppendFloat(buf[:0], f, 'f', -1, 32)
	w.Write(b)
}

func makeBoolWriter(ba *array.BooleanArray, validity *bitmap.Bitmap) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			if ba.Value(i) {
				w.WriteString("true")
			} else {
				w.WriteString("false")
			}
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		if ba.Value(i) {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
	}
}

func makeStringWriter(sa *array.StringArray, validity *bitmap.Bitmap) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			writeJSONStringBytes(w, sa.ValueBytes(i))
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		writeJSONStringBytes(w, sa.ValueBytes(i))
	}
}

// writeJSONStringBytes writes a JSON-escaped string from raw bytes without
// allocating a Go string. It handles the JSON escape sequences for control
// characters, backslash, and double quote.
func writeJSONStringBytes(w *bufio.Writer, b []byte) {
	w.WriteByte('"')
	start := 0
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= 0x20 && c != '"' && c != '\\' {
			continue
		}
		if start < i {
			w.Write(b[start:i])
		}
		switch c {
		case '"':
			w.WriteString(`\"`)
		case '\\':
			w.WriteString(`\\`)
		case '\n':
			w.WriteString(`\n`)
		case '\r':
			w.WriteString(`\r`)
		case '\t':
			w.WriteString(`\t`)
		case '\b':
			w.WriteString(`\b`)
		case '\f':
			w.WriteString(`\f`)
		default:
			// Control characters U+0000 through U+001F
			w.WriteString(`\u00`)
			w.WriteByte("0123456789abcdef"[c>>4])
			w.WriteByte("0123456789abcdef"[c&0xf])
		}
		start = i + 1
	}
	if start < len(b) {
		w.Write(b[start:])
	}
	w.WriteByte('"')
}

// Temporal type writers

func makeDateWriter(vals []int32, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			t := time.Unix(int64(vals[i])*86400, 0).UTC()
			b := t.AppendFormat(buf[:0], `"2006-01-02"`)
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		t := time.Unix(int64(vals[i])*86400, 0).UTC()
		b := t.AppendFormat(buf[:0], `"2006-01-02"`)
		w.Write(b)
	}
}

func makeDateTimeWriter(vals []int64, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			us := vals[i]
			sec := us / 1_000_000
			nsec := (us % 1_000_000) * 1000
			t := time.Unix(sec, nsec).UTC()
			b := t.AppendFormat(buf[:0], `"2006-01-02T15:04:05.000000"`)
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		us := vals[i]
		sec := us / 1_000_000
		nsec := (us % 1_000_000) * 1000
		t := time.Unix(sec, nsec).UTC()
		b := t.AppendFormat(buf[:0], `"2006-01-02T15:04:05.000000"`)
		w.Write(b)
	}
}

func makeTimeWriter(vals []int64, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			ns := vals[i]
			h := ns / 3_600_000_000_000
			ns %= 3_600_000_000_000
			m := ns / 60_000_000_000
			ns %= 60_000_000_000
			s := ns / 1_000_000_000
			ns %= 1_000_000_000
			b := buf[:0]
			b = append(b, '"')
			b = appendTwoDigits(b, int(h))
			b = append(b, ':')
			b = appendTwoDigits(b, int(m))
			b = append(b, ':')
			b = appendTwoDigits(b, int(s))
			if ns > 0 {
				b = append(b, '.')
				b = strconv.AppendInt(b, ns, 10)
			}
			b = append(b, '"')
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		ns := vals[i]
		h := ns / 3_600_000_000_000
		ns %= 3_600_000_000_000
		m := ns / 60_000_000_000
		ns %= 60_000_000_000
		s := ns / 1_000_000_000
		ns %= 1_000_000_000
		b := buf[:0]
		b = append(b, '"')
		b = appendTwoDigits(b, int(h))
		b = append(b, ':')
		b = appendTwoDigits(b, int(m))
		b = append(b, ':')
		b = appendTwoDigits(b, int(s))
		if ns > 0 {
			b = append(b, '.')
			b = strconv.AppendInt(b, ns, 10)
		}
		b = append(b, '"')
		w.Write(b)
	}
}

func makeDurationWriter(vals []int64, validity *bitmap.Bitmap, buf []byte) func(*bufio.Writer, int) {
	if validity == nil {
		return func(w *bufio.Writer, i int) {
			b := buf[:0]
			b = append(b, '"')
			b = strconv.AppendInt(b, vals[i], 10)
			b = append(b, "us\""...)
			w.Write(b)
		}
	}
	return func(w *bufio.Writer, i int) {
		if !validity.IsSet(i) {
			w.WriteString("null")
			return
		}
		b := buf[:0]
		b = append(b, '"')
		b = strconv.AppendInt(b, vals[i], 10)
		b = append(b, "us\""...)
		w.Write(b)
	}
}

func appendTwoDigits(b []byte, v int) []byte {
	if v < 10 {
		b = append(b, '0')
	}
	return strconv.AppendInt(b, int64(v), 10)
}
