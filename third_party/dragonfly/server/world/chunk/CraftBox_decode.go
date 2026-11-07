package chunk

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type capturedNetworkEncoding struct {
	networkEncoding
	hashed bool
	blocks BlockRegistry
}

type capturedPersistentEncoding struct{}

func (capturedPersistentEncoding) network() byte { return 0 }
func (capturedPersistentEncoding) encodePalette(buf *bytes.Buffer, p *Palette, pe paletteEncoding) {
	bpe := pe.(BlockPaletteEncoding)
	if p.size != 0 {
		_ = protocol.WriteVarint32(buf, int32(len(p.values)))
	}
	encoder := nbt.NewEncoderWithEncoding(buf, nbt.NetworkLittleEndian)
	for _, rid := range p.values {
		_ = encoder.Encode(bpe.EncodeBlockState(rid))
	}
}
func (capturedPersistentEncoding) decodePalette(buf *bytes.Buffer, size paletteSize, pe paletteEncoding) (*Palette, error) {
	count := int32(1)
	if size != 0 {
		if err := protocol.Varint32(buf, &count); err != nil {
			return nil, err
		}
	}
	if count <= 0 || count > 4096 {
		return nil, fmt.Errorf("invalid persistent palette size %d", count)
	}
	values := make([]uint32, count)
	decoder := nbt.NewDecoderWithEncoding(buf, nbt.NetworkLittleEndian)
	bpe := pe.(BlockPaletteEncoding)
	for i := range values {
		var state map[string]any
		if err := decoder.Decode(&state); err != nil {
			return nil, err
		}
		if name, ok := state["name"].(string); ok && !strings.Contains(name, ":") {
			state["name"] = "minecraft:" + name
		}
		rid, err := bpe.DecodeBlockState(state)
		if err != nil {
			return nil, err
		}
		values[i] = rid
	}
	return newPalette(size, values), nil
}

func (e capturedNetworkEncoding) decodePalette(buf *bytes.Buffer, size paletteSize, pe paletteEncoding) (*Palette, error) {
	p, err := e.networkEncoding.decodePalette(buf, size, pe)
	if err != nil || p == nil || !e.hashed {
		return p, err
	}
	if _, blocks := pe.(BlockPaletteEncoding); blocks {
		for i, hash := range p.values {
			rid, ok := e.blocks.HashToRuntimeID(hash)
			if !ok {
				// Only the capture representation uses this placeholder. The raw
				// network packets and the original wire hashes remain untouched.
				if collisions, hasInfo := e.blocks.(interface{ IsAmbiguousNetworkHash(uint32) bool }); hasInfo && collisions.IsAmbiguousNetworkHash(hash) {
					rid, ok = e.blocks.StateToRuntimeID("minecraft:unknown", nil)
				}
			}
			if !ok {
				return nil, fmt.Errorf("unknown captured block hash %08x", hash)
			}
			p.values[i] = rid
		}
	}
	return p, nil
}

func DecodeCapturedNetwork(br BlockRegistry, data []byte, count int, r cube.Range, hashed bool) (*Chunk, []map[string]any, error) {
	c := New(br, r)
	if count < 0 || count > len(c.sub) {
		return nil, nil, fmt.Errorf("invalid captured subchunk count %d", count)
	}
	buf := bytes.NewBuffer(data)
	e := capturedNetworkEncoding{hashed: hashed, blocks: br}
	for i := 0; i < count; i++ {
		index := byte(i)
		sub, err := decodeSubChunk(buf, c, &index, e)
		if err != nil {
			return nil, nil, err
		}
		if int(index) >= len(c.sub) {
			return nil, nil, fmt.Errorf("captured subchunk index %d outside dimension", index)
		}
		c.sub[index] = sub
	}
	if err := decodeBiomes(buf, c, NetworkEncoding); err != nil {
		return nil, nil, err
	}
	borderCount, err := buf.ReadByte()
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	if int(borderCount) > buf.Len() {
		return nil, nil, io.ErrUnexpectedEOF
	}
	buf.Next(int(borderCount))
	var entities []map[string]any
	decoder := nbt.NewDecoderWithEncoding(buf, nbt.NetworkLittleEndian)
	for buf.Len() != 0 {
		var entity map[string]any
		if err := decoder.Decode(&entity); err != nil {
			return nil, nil, err
		}
		entities = append(entities, entity)
	}
	return c, entities, nil
}

func DecodeCapturedSubChunk(buf *bytes.Buffer, br BlockRegistry, r cube.Range, index *byte, encoding Encoding, hashed bool) (*SubChunk, error) {
	if encoding.network() == 1 {
		encoding = capturedNetworkEncoding{hashed: hashed, blocks: br}
	}
	c := New(br, r)
	sub, err := decodeSubChunk(buf, c, index, encoding)
	if err == nil && int(*index) >= len(c.sub) {
		return nil, fmt.Errorf("captured subchunk index %d outside dimension", *index)
	}
	return sub, err
}
