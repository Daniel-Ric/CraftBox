package packet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"math"
	"sync"
)

type Decoder struct {
	r   io.Reader
	buf []byte

	pr PacketReader

	header []byte

	decompress         bool
	compression        Compression
	maxDecompressedLen int
	encrypt            *encrypt

	disableEncryption bool

	checkPacketLimit bool
}

func NewDecoder(reader io.Reader) *Decoder {
	var batch []byte
	if b, ok := reader.(BatchHeaderer); ok {
		batch = b.BatchHeader()
	} else {
		batch = []byte{header}
	}
	var disableEncryption bool
	if d, ok := reader.(EncryptionDisabler); ok {
		disableEncryption = d.DisableEncryption()
	}
	if pr, ok := reader.(PacketReader); ok {
		return &Decoder{
			checkPacketLimit:   true,
			pr:                 pr,
			header:             batch,
			maxDecompressedLen: DefaultMaxDecompressedLen,
			disableEncryption:  disableEncryption,
		}
	}
	return &Decoder{
		r:                  reader,
		buf:                make([]byte, 1024*1024*3),
		header:             batch,
		maxDecompressedLen: DefaultMaxDecompressedLen,
		checkPacketLimit:   true,
		disableEncryption:  disableEncryption,
	}
}

func (decoder *Decoder) EnableEncryption(keyBytes [32]byte) {
	if decoder.disableEncryption {
		return
	}
	block, _ := aes.NewCipher(keyBytes[:])
	first12 := append([]byte(nil), keyBytes[:12]...)
	stream := cipher.NewCTR(block, append(first12, 0, 0, 0, 2))
	decoder.encrypt = newEncrypt(keyBytes[:], stream)
}

func (decoder *Decoder) EnableCompression(compression Compression, maxDecompressedLen int) {
	decoder.decompress = true
	decoder.compression = compression
	if maxDecompressedLen == 0 {
		maxDecompressedLen = DefaultMaxDecompressedLen
	} else if maxDecompressedLen < 0 {
		maxDecompressedLen = math.MaxInt
	}
	decoder.maxDecompressedLen = maxDecompressedLen
}

func (decoder *Decoder) DisableBatchPacketLimit() {
	decoder.checkPacketLimit = false
}

const (
	header = 0xfe

	maximumInBatch = 812

	maxPooledDecompressBufferCap = 1 << 20

	DefaultMaxDecompressedLen       = 16 * 1024 * 1024
	DefaultClientMaxDecompressedLen = 64 * 1024 * 1024
)

var decompressBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 32*1024)
		return &b
	},
}

func (decoder *Decoder) Decode() (packets [][]byte, err error) {
	data, pooled, err := decoder.readBatch()
	if pooled != nil {
		defer putDecompressBuffer(pooled)
	}
	if err != nil {
		return nil, err
	}
	data = bytes.Clone(data)
	err = decoder.walkBatch(data, func(packet []byte) error {
		packets = append(packets, packet)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return packets, nil
}

func (decoder *Decoder) DecodeFunc(f func(packet []byte) error) error {
	data, pooled, err := decoder.readBatch()
	if pooled != nil {
		defer putDecompressBuffer(pooled)
	}
	if err != nil {
		return err
	}
	if err := decoder.walkBatch(data, nil); err != nil {
		return err
	}
	return decoder.walkBatch(data, f)
}

func (decoder *Decoder) readBatch() (data []byte, pooled *[]byte, err error) {
	if decoder.pr == nil {
		var n int
		n, err = decoder.r.Read(decoder.buf)
		data = decoder.buf[:n]
	} else {
		data, err = decoder.pr.ReadPacket()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read batch: %w", err)
	}

	if len(data) == 0 {
		return nil, nil, nil
	}
	h := data[:min(len(decoder.header), len(data))]
	if !bytes.Equal(h, decoder.header) {
		return nil, nil, fmt.Errorf("decode batch: invalid header %x: expected %x", h, decoder.header)
	}
	data = data[len(decoder.header):]

	if decoder.encrypt != nil {
		decoder.encrypt.decrypt(data)
		if err := decoder.encrypt.verify(data); err != nil {

			return nil, nil, fmt.Errorf("verify batch: %w", err)
		}
		data = data[:len(data)-8]
	}

	if decoder.decompress {
		if len(data) == 0 {
			return nil, nil, fmt.Errorf("decompress batch: missing compression algorithm")
		}
		if data[0] == 0xff {
			data = data[1:]
		} else {
			compression, ok := CompressionByID(uint16(data[0]))
			if !ok {
				return nil, nil, fmt.Errorf("decompress batch: unknown compression algorithm %v", data[0])
			}
			if compression != decoder.compression {
				return nil, nil, fmt.Errorf("decompress batch: unexpected compression algorithm: got %v, expected %v", compression, decoder.compression)
			}
			data, pooled, err = decoder.decompressBatch(compression, data[1:])
			if err != nil {
				return nil, pooled, fmt.Errorf("decompress batch: %w", err)
			}
		}
	}

	if len(data) > decoder.maxDecompressedLen {
		return nil, pooled, fmt.Errorf("decode batch: size %v exceeds limit %v", len(data), decoder.maxDecompressedLen)
	}
	return data, pooled, nil
}

func (decoder *Decoder) decompressBatch(compression Compression, compressed []byte) ([]byte, *[]byte, error) {
	if decompressor, ok := compression.(appendDecompression); ok {
		pooled := getDecompressBuffer()
		data, err := decompressor.DecompressAppend(*pooled, compressed, decoder.maxDecompressedLen)
		if err != nil {
			return nil, pooled, err
		}
		*pooled = data
		return data, pooled, nil
	}

	data, err := compression.Decompress(compressed, decoder.maxDecompressedLen)
	if err != nil {
		return nil, nil, err
	}
	return data, nil, nil
}

func (decoder *Decoder) walkBatch(data []byte, f func(packet []byte) error) error {
	var packetCount int
	for len(data) != 0 {
		length, n, err := readPacketLength(data)
		if err != nil {
			return fmt.Errorf("decode batch: read packet length: %w", err)
		}
		data = data[n:]
		if length == 0 {
			return fmt.Errorf("decode batch: empty packet")
		}
		if length > uint32(len(data)) {
			return fmt.Errorf("decode batch: packet length %v exceeds remaining %v", length, len(data))
		}
		if packetCount >= maximumInBatch && decoder.checkPacketLimit {
			return fmt.Errorf("decode batch: number of packets exceeds max=%v", maximumInBatch)
		}
		if f != nil {
			if err := f(data[:length:length]); err != nil {
				return err
			}
		}
		packetCount++
		data = data[length:]
	}
	return nil
}

func getDecompressBuffer() *[]byte {
	b := decompressBufferPool.Get().(*[]byte)
	*b = (*b)[:0]
	return b
}

func putDecompressBuffer(b *[]byte) {
	if cap(*b) <= maxPooledDecompressBufferCap {
		*b = (*b)[:0]
		decompressBufferPool.Put(b)
	}
}

func readPacketLength(data []byte) (uint32, int, error) {
	var length uint32
	for i := range 5 {
		if i >= len(data) {
			return 0, 0, io.ErrUnexpectedEOF
		}
		b := data[i]
		length |= uint32(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return length, i + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("varuint32 did not terminate after 5 bytes")
}
