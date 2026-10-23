package worldstate

import (
	"context"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorldSaveExcludesPlayersButPreservesCustomNPC(t *testing.T) {
	messages.SetEventHandler(func(any) error { return nil })
	blocks := world.DefaultBlockRegistry.Clone()
	blocks.Finalize()
	stone, _ := blocks.StateToRuntimeID("minecraft:stone", map[string]any{})
	w, err := New(context.Background(), nil, func(world.ChunkPos, *chunk.Chunk) {})
	if err != nil {
		t.Fatal(err)
	}
	w.SetDimension(world.Overworld)
	w.BlockRegistry = blocks
	w.SetTime(time.Now(), 0)
	w.Open("test", t.TempDir())
	terrain := chunk.New(blocks, world.Overworld.Range())
	terrain.SetBlock(8, 70, 8, 0, stone)
	if err = w.StoreChunk(world.ChunkPos{0, 0}, &Chunk{Chunk: terrain, BlockEntities: map[cube.Pos]map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	w.AddPlayer(&packet.AddPlayer{UUID: uuid.New(), EntityRuntimeID: 1, Position: mgl32.Vec3{8, 71, 8}})
	bp := behaviourpack.New("test")
	for i, identifier := range []string{"minecraft:player", "CraftBox_player:old", "test:npc"} {
		w.StoreEntity(uint64(i+2), &entity.Entity{RuntimeID: uint64(i + 2), UniqueID: int64(i + 2), EntityType: identifier, Position: mgl32.Vec3{8, 71, 8}, Metadata: protocol.NewEntityMetadata()})
		bp.AddEntity(identifier, nil, protocol.NewEntityMetadata(), nil)
	}
	if err = w.Save(proxy.Player{Position: mgl32.Vec3{8, 71, 8}}, map[string]any{"Pos": []float32{8, 71, 8}}, bp, nil, false, nil, minecraft.GameData{}, "test", false, 0); err != nil {
		t.Fatal(err)
	}
	db, err := mcdb.Config{Blocks: blocks}.Open(w.Folder)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	column, err := db.LoadColumn(world.ChunkPos{0, 0}, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	if len(column.Entities) != 1 || column.Entities[0].Data["identifier"] != "test:npc" {
		t.Fatalf("player saved or NPC removed: %v", column.Entities)
	}
	err = filepath.WalkDir(filepath.Join(w.Folder, "behavior_packs"), func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.Contains(path, "CraftBox_player") {
			t.Fatal("player definition exported")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
