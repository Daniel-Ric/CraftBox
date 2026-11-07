package structure

import (
	"context"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

func TestNativeColumnsRetainCustomStateAndBlockData(t *testing.T) {
	ctx := context.Background()
	d, _ := New("Native", Vec3{17, 4, 2})
	idx, e := d.AddState(State{Name: "craftbox:custom", States: map[string]any{"color": "red"}})
	if e != nil {
		t.Fatal(e)
	}
	d.Set(Vec3{16, 2, 1}, Cell{[2]int32{idx, -1}, true})
	d.BlockData[Key(Vec3{16, 2, 1})] = map[string]any{"block_entity_data": map[string]any{"id": "Sign", "Text": "hello", "custom": int64(9007199254740993)}}
	i, _ := NewImporter("Again", Bounds{Vec3{-4, 60, 7}, Vec3{12, 63, 8}})
	reg := Registry()
	err := d.Columns(ctx, reg, Vec3{-4, 60, 7}, func(p world.ChunkPos, c *chunk.Column) error { return i.Column(ctx, p, c, reg) })
	if err != nil {
		t.Fatal(err)
	}
	c := i.Doc.Cell(Vec3{16, 2, 1})
	if i.Doc.Palette[c.Layers[0]].Name != "craftbox:custom" {
		t.Fatal("custom block lost")
	}
	be := i.Doc.BlockData[Key(Vec3{16, 2, 1})]["block_entity_data"].(map[string]any)
	if be["custom"] != int64(9007199254740993) {
		t.Fatal("NBT precision lost")
	}
}
func TestMaterialCountsDoorHalvesAndDoubleSlabs(t *testing.T) {
	d, _ := New("Materials", Vec3{3, 1, 1})
	a, _ := d.AddState(State{Name: "minecraft:oak_door", States: map[string]any{"upper_block_bit": uint8(1)}})
	b, _ := d.AddState(State{Name: "minecraft:double_stone_block_slab"})
	d.Set(Vec3{}, Cell{[2]int32{a, -1}, true})
	d.Set(Vec3{1, 0, 0}, Cell{[2]int32{b, -1}, true})
	m, e := CountMaterials(context.Background(), d, nil)
	if e != nil {
		t.Fatal(e)
	}
	if m.Blocks != 2 || len(m.Rows) != 2 || m.Rows[0].Items != 2 || m.Rows[1].Items != 0 {
		t.Fatalf("%+v", m)
	}
}
