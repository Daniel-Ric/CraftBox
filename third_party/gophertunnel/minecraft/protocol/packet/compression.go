package packet

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/s2"
	"github.com/sandertv/gophertunnel/minecraft/internal"
)

type Compression interface {
	EncodeCompression() uint16

	Compress(decompressed []byte) ([]byte, error)

	Decompress(compressed []byte, limit int) ([]byte, error)
}

type appendCompression interface {
	CompressAppend(dst, decompressed []byte) ([]byte, error)
	MaxCompressedLen(decompressedLen int) int
}

type appendDecompression interface {
	DecompressAppend(dst, compressed []byte, limit int) ([]byte, error)
}

var (
	NopCompression nopCompression

	FlateCompression flateCompression

	SnappyCompression snappyCompression

	DefaultCompression Compression = FlateCompression
)

type (
	nopCompression struct{}

	flateCompression struct{}

	snappyCompression struct{}
)

var (
	flateDecompressPool = sync.Pool{
		New: func() any { return flate.NewReader(bytes.NewReader(nil)) },
	}
	flateCompressPool = sync.Pool{
		New: func() any {
			w, _ := flate.NewWriter(io.Discard, 6)
			return w
		},
	}
)

func (nopCompression) EncodeCompression() uint16 {
	return CompressionAlgorithmNone
}

func (nopCompression) Compress(decompressed []byte) ([]byte, error) {
	return decompressed, nil
}

func (nopCompression) Decompress(compressed []byte, limit int) ([]byte, error) {
	if len(compressed) > limit {
		return nil, fmt.Errorf("nop decompression: size %d exceeds limit %d", len(compressed), limit)
	}
	return compressed, nil
}

func (flateCompression) EncodeCompression() uint16 {
	return CompressionAlgorithmFlate
}

func (flateCompression) Compress(decompressed []byte) ([]byte, error) {
	compressed := internal.BufferPool.Get().(*bytes.Buffer)
	w := flateCompressPool.Get().(*flate.Writer)

	defer func() {

		compressed.Reset()
		internal.BufferPool.Put(compressed)
		flateCompressPool.Put(w)
	}()

	w.Reset(compressed)
	if _, err := w.Write(decompressed); err != nil {
		return nil, fmt.Errorf("compress flate: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close flate writer: %w", err)
	}
	return append([]byte(nil), compressed.Bytes()...), nil
}

func (c flateCompression) Decompress(compressed []byte, limit int) ([]byte, error) {
	pooled := getDecompressBuffer()
	defer putDecompressBuffer(pooled)

	data, err := c.DecompressAppend(*pooled, compressed, limit)
	if err != nil {
		return nil, err
	}
	*pooled = data
	return append([]byte(nil), data...), nil
}

func (flateCompression) DecompressAppend(dst, compressed []byte, limit int) ([]byte, error) {
	limit = max(limit, 0)
	hint := len(compressed)
	if hint > math.MaxInt/2 {
		hint = math.MaxInt
	} else {
		hint *= 2
	}
	hint = max(hint, 32*1024)
	if limit != math.MaxInt {
		hint = min(hint, limit+1)
	}
	dst = slices.Grow(dst, hint)

	r := flateDecompressPool.Get().(io.ReadCloser)
	defer func() {
		_ = r.Close()
		flateDecompressPool.Put(r)
	}()

	if err := r.(flate.Resetter).Reset(bytes.NewReader(compressed), nil); err != nil {
		return nil, fmt.Errorf("reset flate: %w", err)
	}
	decompressed, err := appendReader(dst, r, limit)
	if err != nil {
		return nil, fmt.Errorf("decompress flate: %w", err)
	}
	return decompressed, nil
}

func (snappyCompression) EncodeCompression() uint16 {
	return CompressionAlgorithmSnappy
}

func (snappyCompression) Compress(decompressed []byte) ([]byte, error) {

	return s2.EncodeSnappy(nil, decompressed), nil
}

func (snappyCompression) CompressAppend(dst, decompressed []byte) ([]byte, error) {

	n := s2.MaxEncodedLen(len(decompressed))
	if n < 0 {
		panic(s2.ErrTooLarge)
	}
	offset := len(dst)
	dst = slices.Grow(dst, n)
	encoded := s2.EncodeSnappy(dst[offset:offset:cap(dst)], decompressed)
	return dst[:offset+len(encoded)], nil
}

func (snappyCompression) MaxCompressedLen(decompressedLen int) int {
	return s2.MaxEncodedLen(decompressedLen)
}

func (c snappyCompression) Decompress(compressed []byte, limit int) ([]byte, error) {
	return c.DecompressAppend(nil, compressed, limit)
}

func (snappyCompression) DecompressAppend(dst, compressed []byte, limit int) ([]byte, error) {
	decodedLen, err := s2.DecodedLen(compressed)
	if err != nil {
		return nil, fmt.Errorf("snappy decoded length: %w", err)
	}
	if decodedLen > limit {
		return nil, fmt.Errorf("snappy decoded size %d exceeds limit %d", decodedLen, limit)
	}
	offset := len(dst)
	dst = slices.Grow(dst, decodedLen)
	decompressed, err := s2.Decode(dst[offset:offset:cap(dst)], compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress snappy: %w", err)
	}
	return append(dst[:offset], decompressed...), nil
}

func appendReader(dst []byte, r io.Reader, limit int) ([]byte, error) {
	const chunkSize = 32 * 1024

	limit = max(min(limit, math.MaxInt-1), 0)
	start := len(dst)
	for {
		remaining := limit + 1 - (len(dst) - start)
		if cap(dst) == len(dst) {
			dst = slices.Grow(dst, min(chunkSize, remaining))
		}
		readLen := min(cap(dst)-len(dst), chunkSize, remaining)

		n := len(dst)
		dst = dst[:n+readLen]
		read, err := r.Read(dst[n:])
		dst = dst[:n+read]
		if len(dst)-start > limit {
			return nil, fmt.Errorf("size exceeds limit %d", limit)
		}
		if err == io.EOF {
			return dst, nil
		}
		if err != nil {
			return nil, err
		}
		if read == 0 {
			return nil, io.ErrNoProgress
		}
	}
}

func init() {
	RegisterCompression(flateCompression{})
	RegisterCompression(snappyCompression{})
}

var compressions = map[uint16]Compression{}

func RegisterCompression(compression Compression) {
	compressions[compression.EncodeCompression()] = compression
}

func CompressionByID(id uint16) (Compression, bool) {
	c, ok := compressions[id]
	if !ok {
		c = DefaultCompression
	}
	return c, ok
}
