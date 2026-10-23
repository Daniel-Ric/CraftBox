package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

func TestLightSourcesFollowBlockStateAndSurvivePacking(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	lit, ok := reg.StateToRuntimeID("minecraft:lit_redstone_lamp", map[string]any{})
	if !ok {
		t.Fatal("missing lit lamp")
	}
	off, ok := reg.StateToRuntimeID("minecraft:redstone_lamp", map[string]any{})
	if !ok {
		t.Fatal("missing off lamp")
	}
	c := chunk.New(reg, cube.Range{0, 15})
	c.SetBlock(2, 3, 4, 0, lit)
	c.SetBlock(5, 3, 4, 0, off)
	store := newTestStore(t, reg, NewCatalog())
	store.Put(world.ChunkPos{-2, 1}, c)
	snap, err := store.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	lights := snap.Chunks[0].Lights
	if len(lights) != 1 || lights[0].Level != 15 || lights[0].Position != ([3]float32{2.5, 3.5, 4.5}) {
		t.Fatalf("wrong emitters: %+v", lights)
	}
	packed := packChunk(snap.Chunks[0])
	if len(packed.Lights) != 1 || packed.Lights[0] != lights[0] {
		t.Fatal("emitter lost in compact page")
	}
	c.SetBlock(2, 3, 4, 0, off)
	store.Put(world.ChunkPos{-2, 1}, c)
	changed, err := store.Snapshot(snap.Sequence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed.Chunks[0].Lights) != 0 {
		t.Fatal("switched-off lamp still illuminates")
	}
}

func TestCapturedEmissionOverridesRegistryAndRespectsPermutations(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	top, ok := reg.StateToRuntimeID("minecraft:torch", map[string]any{"torch_facing_direction": "top"})
	if !ok {
		t.Fatal("missing torch state")
	}
	wall, ok := reg.StateToRuntimeID("minecraft:torch", map[string]any{"torch_facing_direction": "north"})
	if !ok {
		t.Fatal("missing wall torch state")
	}
	catalog := NewCatalog()
	if err := catalog.AddBlockDefinition([]byte(`{"minecraft:block":{"description":{"identifier":"minecraft:torch"},"components":{"minecraft:light_emission":0},"permutations":[{"condition":"query.block_state('torch_facing_direction') == 'top'","components":{"minecraft:light_emission":9}}]}}`)); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t, reg, catalog)
	if got := store.emissionLevel(top, "minecraft:torch"); got != 9 {
		t.Fatalf("active permutation: %d", got)
	}
	if got := store.emissionLevel(wall, "minecraft:torch"); got != 0 {
		t.Fatalf("explicit zero ignored: %d", got)
	}
}

func TestUnknownBlockEmissionDoesNotPanic(t *testing.T){
 reg:=world.DefaultBlockRegistry;reg.Finalize();s:=newTestStore(t,reg,NewCatalog())
 if level:=s.emissionLevel(^uint32(0),"unknown");level!=0{t.Fatal(level)}
}
