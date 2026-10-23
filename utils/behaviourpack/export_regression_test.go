package behaviourpack

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"reflect"
	"testing"
)

func TestCapturedBlockModernizesComponentsWithoutLosingVisuals(t *testing.T) {
	input := protocol.BlockEntry{Name: "test:block", Properties: map[string]any{"components": map[string]any{"minecraft:block_light_emission": map[string]any{"value": int32(7)}, "minecraft:unit_cube": map[string]any{}, "minecraft:transformation": map[string]any{"RY": int32(1)}, "minecraft:geometry": map[string]any{"identifier": "geometry.test", "bone_visibility": map[string]any{"top": "query.block_property('state') == 1"}}, "minecraft:material_instances": map[string]any{"materials": map[string]any{"*": map[string]any{"texture": "test", "ambient_occlusion": uint8(1)}}}}, "permutations": []any{map[string]any{"condition": "query.block_property('state') == 1", "components": map[string]any{}}}}}
	block, version := parseBlock(input)
	if version != "1.26.30" {
		t.Fatalf("downgraded version %s", version)
	}
	if block.Permutations[0].Condition != "query.block_state('state') == 1" {
		t.Fatal("legacy query retained")
	}
	if block.Components["minecraft:light_emission"] != int32(7) {
		t.Fatal("legacy light component retained")
	}
	if _, ok := block.Components["minecraft:unit_cube"]; ok {
		t.Fatal("obsolete unit_cube retained")
	}
	geometry, ok := block.Components["minecraft:geometry"].(map[string]any)
	if !ok || geometry["bone_visibility"] == nil {
		t.Fatal("geometry properties discarded")
	}
	transform := block.Components["minecraft:transformation"].(map[string][]float32)
	if !reflect.DeepEqual(transform["scale"], []float32{1, 1, 1}) {
		t.Fatal("default scale collapses geometry")
	}
	material := block.Components["minecraft:material_instances"].(map[string]any)["*"].(map[string]any)
	if material["ambient_occlusion"] != float32(1) {
		t.Fatal("numeric AO required for modern format")
	}
}
func TestCapturedEntitiesHaveStablePhysicsAndFractionalAttributes(t *testing.T) {
	bp := New("test")
	meta := protocol.NewEntityMetadata()
	meta.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasGravity)
	meta.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasCollision)
	bp.AddEntity("test:mob", []protocol.AttributeValue{{Name: "minecraft:knockback_resistance", Value: .75, Max: 1}}, meta, nil)
	c := bp.entities["test:mob"].MinecraftEntity.Components
	physics := c["minecraft:physics"].(map[string]any)
	if physics["has_gravity"] != true || physics["has_collision"] != true {
		t.Fatal("mobile actor has no gravity/collision")
	}
	resistance := c["minecraft:knockback_resistance"].(map[string]float32)
	if resistance["value"] != float32(.75) {
		t.Fatal("attribute precision lost")
	}
	bp.AddEntity("test:display", nil, protocol.NewEntityMetadata(), nil)
	static := bp.entities["test:display"].MinecraftEntity.Components
	if static["minecraft:knockback_resistance"].(map[string]float32)["value"] != 1 {
		t.Fatal("gravity-free display can be launched by punches")
	}
}
func TestWorldPackSnapshotExcludesPlayersAndHasUniqueIdentity(t *testing.T) {
	bp := New("server")
	bp.AddEntity("CraftBox_player:old", nil, protocol.NewEntityMetadata(), nil)
	bp.AddEntity("test:npc", nil, protocol.NewEntityMetadata(), nil)
	one, two := bp.WorldSnapshot("one", false), bp.WorldSnapshot("two", false)
	if one.Manifest.Header.UUID == two.Manifest.Header.UUID || one.Manifest.Header.UUID == bp.Manifest.Header.UUID {
		t.Fatal("world-specific definitions share pack identity")
	}
	if _, ok := one.entities["CraftBox_player:old"]; ok {
		t.Fatal("stale player exported")
	}
	one.AddEntity("test:new", nil, protocol.NewEntityMetadata(), nil)
	if _, ok := bp.entities["test:new"]; ok {
		t.Fatal("world save changed session pack")
	}
	if _, ok := one.entities["test:npc"]; !ok {
		t.Fatal("NPC removed with players")
	}
}
