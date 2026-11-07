package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

type connectionMeshRegistry struct{ world.BlockRegistry }

func (connectionMeshRegistry) AirRuntimeID() uint32        { return 0 }
func (connectionMeshRegistry) BlockCount() int             { return 3 }
func (connectionMeshRegistry) LightBlock(uint32) uint8     { return 0 }
func (connectionMeshRegistry) FilteringBlock(uint32) uint8 { return 15 }
func (connectionMeshRegistry) RuntimeIDToState(rid uint32) (string, map[string]any, bool) {
	if rid == 1 {
		return "minecraft:cobblestone_wall", nil, true
	}
	if rid == 2 {
		return "minecraft:stone", nil, true
	}
	return "minecraft:air", nil, true
}
func TestStackedStraightWallMeshesHaveNoExtraCentralPost(t *testing.T) {
	reg := connectionMeshRegistry{}
	ch := chunk.New(reg, cube.Range{0, 15})
	for x := uint8(7); x <= 9; x++ {
		ch.SetBlock(x, 1, 8, 0, 1)
		ch.SetBlock(x, 2, 8, 0, 1)
	}
	s := &Store{registry: reg, catalog: mechanicalCatalog(t), chunks: map[world.ChunkPos]*chunk.Chunk{{0, 0}: ch}}
	m := s.mesh(world.ChunkPos{0, 0}, nil)
	for _, part := range m.Parts {
		for i := 0; i < len(part.Positions); i += 3 {
			x, y, z := part.Positions[i], part.Positions[i+1], part.Positions[i+2]
			if x > 8 && x < 9 && y >= 1 && y <= 3 && (z == 8.25 || z == 8.75) {
				t.Fatalf("stacked straight wall gained a center post at %v", part.Positions[i:i+3])
			}
		}
	}
}

func TestWallConnectionsAcrossChunkBoundaryRespectCrop(t *testing.T) {
	reg := connectionMeshRegistry{}
	left := chunk.New(reg, cube.Range{0, 15})
	right := chunk.New(reg, cube.Range{0, 15})
	left.SetBlock(15, 1, 8, 0, 1)
	right.SetBlock(0, 1, 8, 0, 2)
	s := &Store{registry: reg, catalog: mechanicalCatalog(t), chunks: map[world.ChunkPos]*chunk.Chunk{{0, 0}: left, {1, 0}: right}}
	for _, clipped := range []bool{false, true} {
		var keep func(int32, int32) bool
		if clipped {
			keep = func(x, z int32) bool { return x == 0 && z == 0 }
		}
		out := s.mesh(world.ChunkPos{0, 0}, keep)
		maxX := float32(0)
		for _, part := range out.Parts {
			for i := 0; i < len(part.Positions); i += 3 {
				maxX = max(maxX, part.Positions[i])
			}
		}
		want := float32(16)
		if clipped {
			want = 15.75
		}
		if maxX != want {
			t.Fatalf("clipped=%t: wall extends to %v, want %v", clipped, maxX, want)
		}
	}
}
