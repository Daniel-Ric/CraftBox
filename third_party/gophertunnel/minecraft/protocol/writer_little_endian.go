//go:build !armbe && !arm64be && !ppc64 && !mips && !mips64 && !mips64p32 && !ppc && !sparc && !sparc64 && !s390 && !s390x

package protocol

import (
	"encoding/binary"
	"math"
)

func (w *Writer) Uint16(x *uint16) {
	data := w.buf[:2]
	binary.LittleEndian.PutUint16(data, *x)
	_, _ = w.w.Write(data)
}

func (w *Writer) Int16(x *int16) {
	data := w.buf[:2]
	binary.LittleEndian.PutUint16(data, uint16(*x))
	_, _ = w.w.Write(data)
}

func (w *Writer) Uint32(x *uint32) {
	data := w.buf[:4]
	binary.LittleEndian.PutUint32(data, *x)
	_, _ = w.w.Write(data)
}

func (w *Writer) Int32(x *int32) {
	data := w.buf[:4]
	binary.LittleEndian.PutUint32(data, uint32(*x))
	_, _ = w.w.Write(data)
}

func (w *Writer) BEInt32(x *int32) {
	data := w.buf[:4]
	binary.BigEndian.PutUint32(data, uint32(*x))
	_, _ = w.w.Write(data)
}

func (w *Writer) Uint64(x *uint64) {
	data := w.buf[:8]
	binary.LittleEndian.PutUint64(data, *x)
	_, _ = w.w.Write(data)
}

func (w *Writer) Int64(x *int64) {
	data := w.buf[:8]
	binary.LittleEndian.PutUint64(data, uint64(*x))
	_, _ = w.w.Write(data)
}

func (w *Writer) Float32(x *float32) {
	data := w.buf[:4]
	binary.LittleEndian.PutUint32(data, math.Float32bits(*x))
	_, _ = w.w.Write(data)
}

func (w *Writer) Float64(x *float64) {
	data := w.buf[:8]
	binary.LittleEndian.PutUint64(data, math.Float64bits(*x))
	_, _ = w.w.Write(data)
}
