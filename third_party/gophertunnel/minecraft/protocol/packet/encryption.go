package packet

import (
	"bytes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type encrypt struct {
	sendCounter uint64
	buf         [8]byte
	keyBytes    []byte
	stream      cipher.Stream
}

func newEncrypt(keyBytes []byte, stream cipher.Stream) *encrypt {
	return &encrypt{keyBytes: keyBytes, stream: stream}
}

func (encrypt *encrypt) encrypt(data []byte) []byte {

	binary.LittleEndian.PutUint64(encrypt.buf[:], encrypt.sendCounter)
	encrypt.sendCounter++

	hash := sha256.New()
	hash.Write(encrypt.buf[:])
	hash.Write(data[1:])
	hash.Write(encrypt.keyBytes)

	var digest [sha256.Size]byte
	data = append(data, hash.Sum(digest[:0])[:8]...)

	encrypt.stream.XORKeyStream(data[1:], data[1:])
	return data
}

func (encrypt *encrypt) decrypt(data []byte) {
	encrypt.stream.XORKeyStream(data, data)
}

func (encrypt *encrypt) verify(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("encrypted packet must be at least 8 bytes long, got %v", len(data))
	}
	sum := data[len(data)-8:]

	binary.LittleEndian.PutUint64(encrypt.buf[:], encrypt.sendCounter)
	encrypt.sendCounter++

	hash := sha256.New()
	hash.Write(encrypt.buf[:])
	hash.Write(data[:len(data)-8])
	hash.Write(encrypt.keyBytes)
	var digest [sha256.Size]byte
	ourSum := hash.Sum(digest[:0])[:8]

	if !bytes.Equal(sum, ourSum) {

		return fmt.Errorf("invalid checksum of packet %v: expected %x, got %x", encrypt.sendCounter-1, [8]byte(ourSum), sum)
	}
	return nil
}
