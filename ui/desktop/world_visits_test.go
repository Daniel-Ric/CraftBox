package desktop

import (
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"image"
	"testing"
)

func TestSameNamedWorldVisitsKeepIndependentMapsAndExports(t *testing.T) {
	a := NewApp()
	for _, id := range []string{"first", "second"} {
		a.HandleEvent(&messages.EventWorldOpened{WorldID: id, WorldName: "Lobby"})
		a.HandleEvent(&messages.EventMapTiles{WorldID: id, Tiles: []messages.MapTile{{Pos: protocol.ChunkPos{0, 0}, Img: *image.NewRGBA(image.Rect(0, 0, 16, 16))}}})
		a.HandleEvent(&messages.EventFinishedSavingWorld{WorldID: id, WorldName: "Lobby", Filepath: id + ".mcworld", ImagePath: id + ".png", Chunks: 1})
	}
	s := a.Snapshot()
	if len(s.Worlds) != 2 || s.Worlds[0].Path == s.Worlds[1].Path || s.Worlds[0].ImagePath == s.Worlds[1].ImagePath {
		t.Fatalf("visits merged: %+v", s.Worlds)
	}
	if len(a.GetMapTiles()) != 2 {
		t.Fatal("older map was lost or same coordinates overwritten")
	}
	a.HandleEvent(&messages.EventResetMap{WorldID: "second"})
	if tiles := a.GetMapTiles(); len(tiles) != 1 || tiles[0].WorldID != "first" {
		t.Fatalf("reset erased archived world: %+v", tiles)
	}
}
