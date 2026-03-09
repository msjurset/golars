package parquet

import (
	"fmt"

	"github.com/golang/snappy"
)

// Codec handles compression/decompression for Parquet pages.
type Codec interface {
	Compress(src []byte) []byte
	Decompress(dst, src []byte) ([]byte, error)
	CodecID() int32
}

// uncompressedCodec passes data through unchanged.
type uncompressedCodec struct{}

func (c *uncompressedCodec) Compress(src []byte) []byte { return src }
func (c *uncompressedCodec) Decompress(dst, src []byte) ([]byte, error) {
	if len(dst) >= len(src) {
		copy(dst, src)
		return dst[:len(src)], nil
	}
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}
func (c *uncompressedCodec) CodecID() int32 { return 0 }

// snappyCodec uses Snappy compression.
type snappyCodec struct{}

func (c *snappyCodec) Compress(src []byte) []byte {
	return snappy.Encode(nil, src)
}

func (c *snappyCodec) Decompress(dst, src []byte) ([]byte, error) {
	return snappy.Decode(dst, src)
}

func (c *snappyCodec) CodecID() int32 { return 2 } // SNAPPY in Parquet

// codecForID returns the codec for the given Parquet codec ID.
func codecForID(id int32) (Codec, error) {
	switch id {
	case 0:
		return &uncompressedCodec{}, nil
	case 2:
		return &snappyCodec{}, nil
	default:
		return nil, fmt.Errorf("golars: parquet: unsupported compression codec %d", id)
	}
}
