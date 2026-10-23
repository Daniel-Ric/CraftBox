//go:build !armbe && !arm64be && !ppc64 && !mips && !mips64 && !mips64p32 && !ppc && !sparc && !sparc64 && !s390 && !s390x

package protocol

import (
	"encoding/binary"
	"math"
)

func (r *Reader) Uint16(x *uint16) {
	b := r.buf[:2]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = binary.LittleEndian.Uint16(b)
}

func (r *Reader) Int16(x *int16) {
	b := r.buf[:2]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = int16(binary.LittleEndian.Uint16(b))
}

func (r *Reader) Uint32(x *uint32) {
	b := r.buf[:4]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = binary.LittleEndian.Uint32(b)
}

func (r *Reader) Int32(x *int32) {
	b := r.buf[:4]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = int32(binary.LittleEndian.Uint32(b))
}

func (r *Reader) BEInt32(x *int32) {
	b := r.buf[:4]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = int32(binary.BigEndian.Uint32(b))
}

func (r *Reader) Uint64(x *uint64) {
	b := r.buf[:8]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = binary.LittleEndian.Uint64(b)
}

func (r *Reader) Int64(x *int64) {
	b := r.buf[:8]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = int64(binary.LittleEndian.Uint64(b))
}

func (r *Reader) Float32(x *float32) {
	b := r.buf[:4]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func (r *Reader) Float64(x *float64) {
	b := r.buf[:8]
	if _, err := r.r.Read(b); err != nil {
		r.panic(err)
	}
	*x = math.Float64frombits(binary.LittleEndian.Uint64(b))
}
