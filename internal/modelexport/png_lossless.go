package modelexport

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

var errPNGNoGain = errors.New("PNG recompression would not save space")

type boundedPNGWriter struct {
	bytes.Buffer
	limit int
}

func (w *boundedPNGWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.Len() {
		return 0, errPNGNoGain
	}
	return w.Buffer.Write(p)
}

// Recompress IDAT scanlines directly. No color conversion, resampling, palette
// changes or ancillary metadata removal can occur on this path.
func losslessPNG(ctx context.Context, raw []byte) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if len(raw) < 45 || !bytes.Equal(raw[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return raw, nil
	}
	type chunk struct {
		start, end int
		kind       string
	}
	chunks := []chunk{}
	streams := []io.Reader{}
	firstIDAT := -1
	endedIDAT := false
	seenEnd := false
	imageEnd := len(raw)
	var maxInflated int64
	for at := 8; at < len(raw); {
		if len(chunks) >= 8192 || len(raw)-at < 12 {
			return raw, nil
		}
		n := int64(binary.BigEndian.Uint32(raw[at:]))
		if n > int64(len(raw)-at-12) {
			return raw, nil
		}
		end := at + 12 + int(n)
		kind := string(raw[at+4 : at+8])
		if crc32.ChecksumIEEE(raw[at+4:end-4]) != binary.BigEndian.Uint32(raw[end-4:end]) {
			return raw, nil
		}
		if len(chunks) == 0 {
			if kind != "IHDR" || n != 13 {
				return raw, nil
			}
			width, height := int64(binary.BigEndian.Uint32(raw[at+8:])), int64(binary.BigEndian.Uint32(raw[at+12:]))
			if width < 1 || height < 1 || width > 8192 || height > 8192 {
				return raw, nil
			}
			maxInflated = width*height*8 + height*8 + 1024
		}
		if kind == "dSIG" {
			return raw, nil
		}
		if kind == "IDAT" {
			if endedIDAT {
				return raw, nil
			}
			if firstIDAT < 0 {
				firstIDAT = len(chunks)
			}
			streams = append(streams, bytes.NewReader(raw[at+8:end-4]))
		} else if firstIDAT >= 0 {
			endedIDAT = true
		}
		chunks = append(chunks, chunk{at, end, kind})
		at = end
		if kind == "IEND" {
			if n != 0 || len(raw)-at > 3 || !bytes.Equal(raw[at:], make([]byte, len(raw)-at)) {
				return raw, nil
			}
			imageEnd = at
			seenEnd = true
			break
		}
	}
	if !seenEnd || firstIDAT < 0 {
		return raw, nil
	}
	source, e := zlib.NewReader(io.MultiReader(streams...))
	if e != nil {
		return raw, nil
	}
	defer source.Close()
	compressed := &boundedPNGWriter{limit: len(raw) - 20}
	target, e := zlib.NewWriterLevel(compressed, zlib.BestCompression)
	if e != nil {
		return nil, e
	}
	n, copyErr := io.CopyBuffer(contextWriter{ctx, target}, io.LimitReader(source, maxInflated+1), make([]byte, 64<<10))
	closeErr := target.Close()
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if copyErr != nil || closeErr != nil || n > maxInflated {
		return raw, nil
	}
	var result bytes.Buffer
	result.Grow(len(raw))
	result.Write(raw[:8])
	for i, ch := range chunks {
		if ch.kind != "IDAT" {
			result.Write(raw[ch.start:ch.end])
			continue
		}
		if i != firstIDAT {
			continue
		}
		var head [8]byte
		binary.BigEndian.PutUint32(head[:], uint32(compressed.Len()))
		copy(head[4:], "IDAT")
		result.Write(head[:])
		result.Write(compressed.Bytes())
		crc := crc32.NewIEEE()
		crc.Write(head[4:])
		crc.Write(compressed.Bytes())
		var sum [4]byte
		binary.BigEndian.PutUint32(sum[:], crc.Sum32())
		result.Write(sum[:])
	}
	result.Write(raw[imageEnd:])
	if result.Len() >= len(raw) {
		return raw, nil
	}
	return result.Bytes(), nil
}
