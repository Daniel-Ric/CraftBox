package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

type snowSignalRegistry struct{ connectionMeshRegistry }

func (snowSignalRegistry) BlockCount() int { return 4 }
func (snowSignalRegistry) RuntimeIDToState(id uint32) (string, map[string]any, bool) {
	if id == 3 {
		return "minecraft:snow", nil, true
	}
	return connectionMeshRegistry{}.RuntimeIDToState(id)
}
func TestSnowSurfaceSignalsIgnoreBuriedDepth(t *testing.T) {
	for _, depth := range []int16{1, 32} {
		reg := snowSignalRegistry{}
		ch := chunk.New(reg, cube.Range{0, 63})
		for x := uint8(0); x < 16; x++ {
			for z := uint8(0); z < 16; z++ {
				for y := int16(0); y < depth; y++ {
					ch.SetBlock(x, y, z, 0, 2)
				}
				ch.SetBlock(x, depth, z, 0, 3)
			}
		}
		s := &Store{registry: reg, catalog: mechanicalCatalog(t), chunks: map[world.ChunkPos]*chunk.Chunk{{0, 0}: ch}}
		signals := packChunk(s.mesh(world.ChunkPos{0, 0}, nil)).Signals
		if signals["surface"] != 256 || signals["snowSurface"] != 256 {
			t.Fatalf("depth %d: %v", depth, signals)
		}
	}
}

func TestSceneSignalsDistinguishMaterialsFromEvents(t *testing.T) {
	for _, name := range []string{"minecraft:orange_concrete", "minecraft:orange_terracotta", "minecraft:quartz_block"} {
		if got := sceneBlockSignals(name); len(got) != 0 {
			t.Fatalf("%s has semantic decoration signals: %v", name, got)
		}
	}
	for name, want := range map[string]string{"minecraft:powder_snow": "snow", "minecraft:packed_ice": "ice", "minecraft:lit_pumpkin": "jackolanterns", "minecraft:web": "webs", "hive:easter_egg_blue": "eggs"} {
		found := false
		for _, key := range sceneBlockSignals(name) {
			if key == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing %s", name, want)
		}
	}
}
