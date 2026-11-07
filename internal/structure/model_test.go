package structure

import (
	"bytes"
	"context"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestBedrockStructureFixturePreservesLayersAndNBT(t *testing.T) {
	// Z is fastest, followed by Y, followed by X. Do not use the Java YZX order.
	fixture := map[string]any{"format_version": int32(1), "size": []int32{2, 2, 2}, "structure_world_origin": []int32{-17, 64, 30}, "structure": map[string]any{
		"block_indices": [][]int32{{0, 1, -1, 0, 1, 0, 0, 1}, {-1, 2, -1, -1, -1, -1, -1, -1}},
		"entities":      []map[string]any{{"identifier": "minecraft:armor_stand", "Pos": []float32{-16.5, 64, 30.5}, "UniqueID": int64(9007199254740993)}},
		"palette": map[string]any{"default": map[string]any{"block_palette": []map[string]any{
			{"name": "minecraft:air", "states": map[string]any{}, "version": int32(18168865)},
			{"name": "minecraft:oak_stairs", "states": map[string]any{"weirdo_direction": int32(0), "upside_down_bit": uint8(1)}, "version": int32(18168865)},
			{"name": "minecraft:water", "states": map[string]any{"liquid_depth": int32(0)}, "version": int32(18168865)},
		}, "block_position_data": map[string]any{"1": map[string]any{"block_entity_data": map[string]any{"id": "Chest", "x": int32(-17), "y": int32(64), "z": int32(31), "Items": []map[string]any{{"Name": "minecraft:diamond", "Count": uint8(3)}}}, "custom": int64(9007199254740993)}}}},
	}}
	var wire bytes.Buffer
	if err := nbt.NewEncoderWithEncoding(&wire, nbt.LittleEndian).Encode(fixture); err != nil {
		t.Fatal(err)
	}
	doc, err := ReadMCStructure(context.Background(), bytes.NewReader(wire.Bytes()), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if cell := doc.Cell(Vec3{0, 0, 1}); cell.Layers != [2]int32{1, 2} || !cell.Known {
		t.Fatalf("wrong order/layers: %+v", cell)
	}
	if doc.Cell(Vec3{0, 1, 0}).Layers[0] != -1 {
		t.Fatal("structure void became air")
	}
	if doc.Palette[1].States["upside_down_bit"] != uint8(1) {
		t.Fatal("lost byte state type")
	}
	if doc.BlockData[Key(Vec3{0, 0, 1})]["custom"] != int64(9007199254740993) {
		t.Fatal("lost block metadata precision")
	}
	var out bytes.Buffer
	if err := WriteMCStructure(context.Background(), &out, doc, ExportOptions{Entities: true, Inventories: true, BlockData: true}); err != nil {
		t.Fatal(err)
	}
	reread, err := ReadMCStructure(context.Background(), &out, "again")
	if err != nil {
		t.Fatal(err)
	}
	if reread.Cell(Vec3{0, 0, 1}).Layers != doc.Cell(Vec3{0, 0, 1}).Layers || len(reread.Entities) != 1 {
		t.Fatal("Bedrock roundtrip lost content")
	}
}

func TestEditUndoTransformAndUnknown(t *testing.T) {
	d, err := New("House", Vec3{3, 2, 4})
	if err != nil {
		t.Fatal(err)
	}
	history := NewHistory(d)
	state := State{Name: "minecraft:oak_stairs", States: map[string]any{"weirdo_direction": int32(0), "upside_down_bit": uint8(0)}}
	ctx := context.Background()
	if err := history.Apply(ctx, Edit{Kind: "fill", Bounds: []Bounds{{Min: Vec3{0, 0, 0}, Max: Vec3{0, 0, 0}}}, Block: state}); err != nil {
		t.Fatal(err)
	}
	if history.Doc.Cell(Vec3{}).Layers[0] == 0 {
		t.Fatal("fill failed")
	}
	if !history.Undo() || history.Doc.Cell(Vec3{}).Layers[0] != 0 {
		t.Fatal("undo lost original")
	}
	if !history.Redo() {
		t.Fatal("redo failed")
	}
	if err := history.Apply(ctx, Edit{Kind: "rotate", Turns: 1}); err != nil {
		t.Fatal(err)
	}
	if history.Doc.Size != (Vec3{4, 2, 3}) {
		t.Fatal("rotated size incorrect")
	}
	c := history.Doc.Cell(Vec3{3, 0, 0})
	s := history.Doc.Palette[c.Layers[0]]
	if s.States["weirdo_direction"] != int32(2) {
		t.Fatalf("stairs not rotated: %#v", s)
	}
	// Unknown capture cells cannot turn into missing-block reports or ordinary air.
	target, _ := New("Partial", history.Doc.Size)
	target.DefaultKnown = false
	comparison, err := Compare(ctx, history.Doc, target, Vec3{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Unknown == 0 || comparison.Missing != 0 {
		t.Fatalf("unknown treated as air: %+v", comparison)
	}
}

func TestBedrockLimitsAndPlacementModes(t *testing.T) {
	if _, err := New("Huge", Vec3{1 << 30, 1 << 30, 1 << 30}); err == nil {
		t.Fatal("unbounded allocation accepted")
	}
	source, _ := New("Source", Vec3{2, 1, 1})
	h := NewHistory(source)
	if err := h.Apply(context.Background(), Edit{Kind: "fill", Bounds: []Bounds{{Min: Vec3{}, Max: Vec3{0, 0, 0}}}, Block: State{Name: "minecraft:stone"}}); err != nil {
		t.Fatal(err)
	}
	target, _ := New("Target", Vec3{2, 1, 1})
	th := NewHistory(target)
	if err := th.Apply(context.Background(), Edit{Kind: "fill", Block: State{Name: "minecraft:gold_block"}}); err != nil {
		t.Fatal(err)
	}
	placed, _, err := Place(context.Background(), h.Doc, th.Doc, Vec3{}, "ignore-air", true)
	if err != nil {
		t.Fatal(err)
	}
	if placed.Palette[placed.Cell(Vec3{1, 0, 0}).Layers[0]].Name != "minecraft:gold_block" {
		t.Fatal("ignored air overwrote target")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	before := h.Doc
	if err := h.Apply(cancelled, Edit{Kind: "fill", Block: State{Name: "minecraft:dirt"}}); err == nil || h.Doc != before {
		t.Fatal("cancelled edit was partially committed")
	}
}
