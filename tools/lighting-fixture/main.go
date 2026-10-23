package main

import (
	"context"
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"os"
	"path/filepath"
)

func main() {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	rid := func(name string) uint32 {
		r, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{})
		if !ok {
			panic(name)
		}
		return r
	}
	catalog := scene.NewCatalog()
	if err := catalog.AddPack(os.DirFS("assets/vanilla-3d")); err != nil {
		panic(err)
	}
	store := scene.NewStore(reg, catalog)
	defer store.Close()
	c := chunk.New(reg, cube.Range{0, 15})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			c.SetBlock(uint8(x), 0, uint8(z), 0, rid("white_concrete"))
		}
	}
	for z := 1; z < 15; z++ {
		for y := 1; y < 6; y++ {
			c.SetBlock(8, int16(y), uint8(z), 0, rid("white_concrete"))
		}
	}
	c.SetBlock(4, 3, 7, 0, rid("glowstone"))
	c.SetBlock(12, 1, 12, 0, rid("redstone_lamp"))
	store.Put(world.ChunkPos{0, 0}, c)
	page, err := store.Page(context.Background(), scene.PageRequest{Limit: 8}, nil)
	if err != nil {
		panic(err)
	}
	if len(page.Chunks) != 1 || len(page.Chunks[0].Lights) != 1 {
		panic("missing authoritative fixture emitter")
	}
	data, err := json.Marshal([]scene.Page{page})
	if err != nil {
		panic(err)
	}
	path := "tmp/lighting/scene-pages.json"
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}
	if err = os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}
