package behaviourpack

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"testing"
	"testing/fstest"
)

func TestClientEntityCatalogIncludesUnseenTypesAndKeepsCapturedProperties(t *testing.T) {
	bp := New("test")
	meta := protocol.NewEntityMetadata()
	meta.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasGravity)
	bp.AddEntity("test:seen", nil, meta, []entity.EntityPropertyDef{{Name: "test:variant", Type: entity.PropertyTypeEnum, Enum: []any{"a", "b"}}})
	assets := fstest.MapFS{"entity/unseen.json": {Data: []byte(`{"format_version":"1.10.0","minecraft:client_entity":{"description":{"identifier":"test:unseen"}}}`)}, "entity/seen.json": {Data: []byte(`{"minecraft:client_entity":{"description":{"identifier":"test:seen"}}}`)}, "entity/vanilla.json": {Data: []byte(`{"minecraft:client_entity":{"description":{"identifier":"minecraft:pig"}}}`)}}
	if err := bp.IncludeClientEntities(assets); err != nil {
		t.Fatal(err)
	}
	if _, ok := bp.entities["test:unseen"]; !ok {
		t.Fatal("unseen custom client entity omitted")
	}
	if _, ok := bp.entities["minecraft:pig"]; ok {
		t.Fatal("vanilla behavior replaced")
	}
	if bp.entities["test:seen"].MinecraftEntity.Description.Properties["test:variant"].Type != "enum" {
		t.Fatal("captured properties lost")
	}
	if bp.entities["test:seen"].MinecraftEntity.Components["minecraft:physics"].(map[string]any)["has_gravity"] != true {
		t.Fatal("captured physics replaced")
	}
}
