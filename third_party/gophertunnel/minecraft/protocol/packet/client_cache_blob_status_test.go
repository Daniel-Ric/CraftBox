package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"testing"
)

func blobStatusWire(misses, hits int) []byte {
	var wire bytes.Buffer
	for _, count := range []int{misses, hits} {
		var length [5]byte
		n := binary.PutUvarint(length[:], uint64(count))
		wire.Write(length[:n])
		for i := 0; i < count; i++ {
			_ = binary.Write(&wire, binary.LittleEndian, uint64(i+1))
		}
	}
	return wire.Bytes()
}

func TestClientCacheBlobStatusAcceptsProtocolSizedLists(t *testing.T) {
	for _, count := range []int{0, 1024, 1025, 2054, 4095} {
		for _, hits := range []bool{false, true} {
			t.Run(fmt.Sprintf("count-%d/hits-%t", count, hits), func(t *testing.T) {
				misses, have := count, 1
				if hits {
					misses, have = 1, count
				}
				raw := blobStatusWire(misses, have)
				buffer := bytes.NewBuffer(raw)
				var pk ClientCacheBlobStatus
				pk.Marshal(protocol.NewReader(buffer, 0, true))
				if len(pk.MissHashes) != misses || len(pk.HitHashes) != have || buffer.Len() != 0 {
					t.Fatalf("incorrect lists: misses=%d hits=%d trailing=%d", len(pk.MissHashes), len(pk.HitHashes), buffer.Len())
				}
				var encoded bytes.Buffer
				pk.Marshal(protocol.NewWriter(&encoded, 0))
				if !bytes.Equal(encoded.Bytes(), raw) {
					t.Fatal("cache-status wire order changed")
				}
			})
		}
	}
}

func TestClientCacheBlobStatusRejectsOversizeAndTruncatedLists(t *testing.T) {
	for _, hits := range []bool{false, true} {
		misses, have := 4096, 0
		if hits {
			misses, have = 0, 4096
		}
		for _, raw := range [][]byte{blobStatusWire(misses, have), {0x86, 0x10, 0}} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("accepted invalid blob-status list hits=%t bytes=%d", hits, len(raw))
					}
				}()
				var pk ClientCacheBlobStatus
				pk.Marshal(protocol.NewReader(bytes.NewBuffer(raw), 0, true))
			}()
		}
	}
}

func TestUnrelatedProtocolSlicesKeepDefaultLimit(t *testing.T) {
	raw := []byte{0x81, 0x08}
	raw = append(raw, make([]byte, 1025*8)...)
	var hashes []uint64
	defer func() {
		if recover() == nil {
			t.Fatal("generic slice limit was relaxed")
		}
		if hashes != nil {
			t.Fatal("oversized generic slice allocated")
		}
	}()
	reader := protocol.NewReader(bytes.NewBuffer(raw), 0, true)
	protocol.FuncSlice(reader, &hashes, reader.Uint64)
}
