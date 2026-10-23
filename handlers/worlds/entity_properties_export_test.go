package worlds

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActorPropertiesCapturedBeforeLogin(t *testing.T) {
	w := &worldsHandler{serverState: serverState{entityProperties: map[string][]entity.EntityPropertyDef{}, behaviorPack: behaviourpack.New("test")}}
	_, err := w.packetHandlerPreLogin(&packet.SyncActorProperty{PropertyData: map[string]any{"type": "test:npc", "properties": []any{map[string]any{"name": "test:appearance", "type": int32(entity.PropertyTypeEnum), "enum": []any{"small", "large"}}}}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(w.serverState.entityProperties["test:npc"]) != 1 {
		t.Fatal("login property definitions lost")
	}
}
func TestLoginActorCatalogIsExportedWithoutSpawningActors(t *testing.T) {
	w := &worldsHandler{serverState: serverState{entityProperties: map[string][]entity.EntityPropertyDef{}, behaviorPack: behaviourpack.New("test")}}
	data, err := nbt.MarshalEncoding(map[string]any{"idlist": []map[string]any{{"id": "test:unseen", "rid": int32(1)}, {"id": "minecraft:pig", "rid": int32(2)}}}, nbt.NetworkLittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.packetHandlerPreLogin(&packet.AvailableActorIdentifiers{SerialisedEntityIdentifiers: data}, time.Now()); err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if err = w.serverState.behaviorPack.Save(utils.OSWriter{Base: folder}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(folder, "entities", "test", "unseen.json"))
	if err != nil || !strings.Contains(string(raw), "test:unseen") {
		t.Fatalf("catalog definition omitted: %v", err)
	}
	if _, err = os.Stat(filepath.Join(folder, "entities", "minecraft", "pig.json")); !os.IsNotExist(err) {
		t.Fatal("vanilla actor overwritten")
	}
}
func TestEnumActorPropertyExportsItsStringValue(t *testing.T) {
	defs := []entity.EntityPropertyDef{{Name: "test:appearance", Type: entity.PropertyTypeEnum, Enum: []any{"small", "large"}}}
	values := map[string]any{}
	applyProperties(defs, protocol.EntityProperties{IntegerProperties: []protocol.IntegerEntityProperty{{Index: 0, Value: 1}}}, values, nil)
	if values["test:appearance"] != "large" {
		t.Fatalf("enum index persisted instead of name: %v", values)
	}
}
