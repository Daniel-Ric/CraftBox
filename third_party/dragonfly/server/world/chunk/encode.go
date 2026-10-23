package chunk

import (
	"bytes"
	"sync"
)

const (
	SubChunkVersion = 9

	CurrentBlockVersion int32 = 18168865
)

var (
	pool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
)

type (
	SerialisedData struct {
		SubChunks [][]byte

		Biomes []byte
	}

	blockEntry struct {
		Name    string         `nbt:"name"`
		State   map[string]any `nbt:"states"`
		Version int32          `nbt:"version"`
	}
)

func Encode(c *Chunk, e Encoding) SerialisedData {
	d := SerialisedData{SubChunks: make([][]byte, len(c.sub))}
	for i := range c.sub {
		d.SubChunks[i] = EncodeSubChunk(c, e, i)
	}
	d.Biomes = EncodeBiomes(c, e)
	return d
}

func EncodeSubChunk(c *Chunk, e Encoding, ind int) []byte {
	buf := pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		pool.Put(buf)
	}()

	s := c.sub[ind]
	_, _ = buf.Write([]byte{SubChunkVersion, byte(len(s.storages)), uint8(ind + (c.r[0] >> 4))})
	for _, storage := range s.storages {
		encodePalettedStorage(buf, storage, nil, e, BlockPaletteEncoding{Blocks: c.br})
	}
	sub := make([]byte, buf.Len())
	_, _ = buf.Read(sub)
	return sub
}

func EncodeBiomes(c *Chunk, e Encoding) []byte {
	buf := pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		pool.Put(buf)
	}()

	var previous *PalettedStorage
	for _, b := range c.biomes {
		encodePalettedStorage(buf, b, previous, e, BiomePaletteEncoding)
		previous = b
	}
	biomes := make([]byte, buf.Len())
	_, _ = buf.Read(biomes)
	return biomes
}

func encodePalettedStorage(buf *bytes.Buffer, storage, previous *PalettedStorage, e Encoding, pe paletteEncoding) {
	if storage.Equal(previous) {
		_, _ = buf.Write([]byte{0x7f<<1 | e.network()})
		return
	}
	b := make([]byte, len(storage.indices)*4+1)
	b[0] = byte(storage.bitsPerIndex<<1) | e.network()

	for i, v := range storage.indices {

		b[i*4+1], b[i*4+2], b[i*4+3], b[i*4+4] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	_, _ = buf.Write(b)

	e.encodePalette(buf, storage.palette, pe)
}
