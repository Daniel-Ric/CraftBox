package packet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"slices"

	"github.com/sandertv/gophertunnel/minecraft/internal"
)

const maxPooledEncoderBufferCap = 1 << 20

type Encoder struct {
	w io.Writer

	header               []byte
	compressionThreshold int
	compression          Compression
	encrypt              *encrypt

	disableEncryption bool
}

func NewEncoder(w io.Writer) *Encoder {
	var batch []byte
	if b, ok := w.(BatchHeaderer); ok {
		batch = b.BatchHeader()
	} else {
		batch = []byte{header}
	}
	var disableEncryption bool
	if d, ok := w.(EncryptionDisabler); ok {
		disableEncryption = d.DisableEncryption()
	}
	return &Encoder{
		w:                 w,
		header:            batch,
		disableEncryption: disableEncryption,
	}
}

func (encoder *Encoder) EnableEncryption(keyBytes [32]byte) {
	if encoder.disableEncryption {
		return
	}
	block, _ := aes.NewCipher(keyBytes[:])
	first12 := append([]byte(nil), keyBytes[:12]...)
	stream := cipher.NewCTR(block, append(first12, 0, 0, 0, 2))
	encoder.encrypt = newEncrypt(keyBytes[:], stream)
}

func (encoder *Encoder) EnableCompression(compression Compression, threshold int) {
	encoder.compression = compression
	encoder.compressionThreshold = threshold
}

func (encoder *Encoder) Encode(packets [][]byte) error {
	buf := internal.BufferPool.Get().(*bytes.Buffer)
	var compressedBuf *bytes.Buffer
	defer func() {

		buf.Reset()
		internal.BufferPool.Put(buf)
		if compressedBuf != nil && compressedBuf.Cap() <= maxPooledEncoderBufferCap {
			compressedBuf.Reset()
			internal.BufferPool.Put(compressedBuf)
		}
	}()

	compression := encoder.compression
	_, _ = buf.Write(encoder.header)
	if compression != nil {
		_ = buf.WriteByte(0)
	}
	batchStart := buf.Len()

	var l [5]byte
	for _, packet := range packets {

		if _, err := buf.Write(l[:putVaruint32(&l, uint32(len(packet)))]); err != nil {
			return fmt.Errorf("encode batch: write packet length: %w", err)
		}
		if _, err := buf.Write(packet); err != nil {
			return fmt.Errorf("encode batch: write packet payload: %w", err)
		}
	}

	data := buf.Bytes()
	if compression != nil {
		batch := data[batchStart:]
		if len(batch) < encoder.compressionThreshold {
			data[len(encoder.header)] = byte(NopCompression.EncodeCompression())
		} else {
			compressedBuf = internal.BufferPool.Get().(*bytes.Buffer)
			_, _ = compressedBuf.Write(encoder.header)
			_ = compressedBuf.WriteByte(byte(compression.EncodeCompression()))
			var err error
			if appender, ok := compression.(appendCompression); ok {
				if n := appender.MaxCompressedLen(len(batch)); n > 0 {
					compressedBuf.Grow(n)
				}
				dst := compressedBuf.Bytes()
				data, err = appender.CompressAppend(dst, batch)
			} else {
				dst := compressedBuf.Bytes()
				var compressed []byte
				compressed, err = compression.Compress(batch)
				data = append(dst, compressed...)
			}
			if err != nil {
				return fmt.Errorf("compress batch: %w", err)
			}
		}
	}

	if encoder.encrypt != nil {

		data = slices.Grow(data, 8)
		data = encoder.encrypt.encrypt(data)
	}
	if _, err := encoder.w.Write(data); err != nil {
		return fmt.Errorf("write batch: %w", err)
	}
	return nil
}

func putVaruint32(b *[5]byte, x uint32) int {
	i := 0
	for x >= 0x80 {
		b[i] = byte(x) | 0x80
		i++
		x >>= 7
	}
	b[i] = byte(x)
	return i + 1
}
