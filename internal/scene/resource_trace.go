package scene

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

type ResourceTrace struct {
	Position [3]int32       `json:"position"`
	Block    string         `json:"block"`
	States   map[string]any `json:"states"`
	Face     string         `json:"face"`
	Texture  string         `json:"texture"`
	Pack     string         `json:"pack"`
	Found    bool           `json:"found"`
	Note     string         `json:"note"`
}

func (s *Store) TraceResource(ctx context.Context, position [3]int32, face int) (ResourceTrace, error) {
	result := ResourceTrace{Position: position}
	if face < 0 || face >= len(faceNames) {
		return result, fmt.Errorf("Choose a valid block face")
	}
	if position[1] < -32768 || position[1] > 32767 {
		return result, fmt.Errorf("Position is outside the world height")
	}
	if e := s.acquire(ctx); e != nil {
		return result, e
	}
	defer s.release()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.db == nil {
		return result, fmt.Errorf("Scene is closed")
	}
	p := world.ChunkPos{position[0] >> 4, position[2] >> 4}
	raw, e := s.db.Get(chunkKey(p), nil)
	if e != nil {
		return result, e
	}
	if len(raw) < 8 {
		return result, fmt.Errorf("Chunk data is incomplete")
	}
	r := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
	if int(position[1]) < r[0] || int(position[1]) > r[1] {
		return result, fmt.Errorf("Position is outside the captured height")
	}
	c, e := chunk.NetworkDecode(s.registry, raw[8:], (r.Height()>>4)+1, r)
	if e != nil {
		return result, e
	}
	name, states, ok := s.registry.RuntimeIDToState(c.Block(uint8(position[0]&15), int16(position[1]), uint8(position[2]&15), 0))
	if !ok {
		return result, fmt.Errorf("Block state is not known")
	}
	result.Block = name
	result.States = states
	result.Face = faceNames[face]
	short := strings.TrimPrefix(name, "minecraft:")
	key := short
	if faces := s.catalog.blocks[short]; faces != nil {
		for _, which := range []string{faceNames[face], "side", "all", "up"} {
			if value := faces[which]; value != "" {
				key = value
				break
			}
		}
	}
	texture := s.catalog.terrain[key]
	if texture == "" {
		texture = key
		if !strings.HasPrefix(texture, "textures/") {
			texture = "textures/blocks/" + texture
		}
	}
	result.Texture = texture
	result.Note = "Default face texture mapping"
	if _, ok := s.catalog.definitions[short]; ok {
		result.Note = "Custom block: state permutations and material instances can select additional textures"
	}
	if !fs.ValidPath(texture) {
		return result, nil
	}
	for _, source := range s.catalog.sources {
		for _, ext := range []string{"", ".png", ".tga"} {
			f, e := source.Open(texture + ext)
			if e != nil {
				continue
			}
			i, e := f.Stat()
			f.Close()
			if e != nil || !i.Mode().IsRegular() {
				continue
			}
			result.Found = true
			result.Texture = texture + ext
			result.Pack = "Bundled vanilla assets"
			if b, e := fs.ReadFile(source, "manifest.json"); e == nil {
				var m struct {
					Header struct {
						Name string `json:"name"`
					} `json:"header"`
				}
				if json.Unmarshal(b, &m) == nil && m.Header.Name != "" {
					result.Pack = m.Header.Name
				}
			}
			return result, nil
		}
	}
	result.Note += ". No readable texture was found in the active sources"
	return result, nil
}
