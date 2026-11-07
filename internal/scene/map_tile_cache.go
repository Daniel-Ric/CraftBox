package scene

import (
	"container/list"
	"github.com/df-mc/dragonfly/server/world"
)

const mapTileCacheBudget = 32 << 20

type mapTileKey struct {
	Pos       world.ChunkPos
	Version   uint64
	Pixels, Y int
	Slice     bool
}
type mapTileEntry struct {
	key   mapTileKey
	tile  MapTile
	bytes int
}
type mapTileCache struct {
	entries     map[mapTileKey]*list.Element
	order       list.List
	bytes, hits int
}

func (c *mapTileCache) get(key mapTileKey) (MapTile, bool) {
	if entry := c.entries[key]; entry != nil {
		c.hits++
		c.order.MoveToBack(entry)
		return entry.Value.(mapTileEntry).tile, true
	}
	return MapTile{}, false
}
func (c *mapTileCache) put(key mapTileKey, tile MapTile) {
	if c.entries == nil {
		c.entries = map[mapTileKey]*list.Element{}
	}
	size := len(tile.Image) + len(tile.Heights)*2 + len(tile.Biomes)*4 + len(tile.Blocks)*16 + len(tile.BiomeNames)*64 + 128
	for _, s := range tile.Blocks {
		size += len(s)
	}
	for _, s := range tile.BiomeNames {
		size += len(s)
	}
	if size > mapTileCacheBudget {
		return
	}
	if old := c.entries[key]; old != nil {
		c.bytes -= old.Value.(mapTileEntry).bytes
		c.order.Remove(old)
		delete(c.entries, key)
	}
	for c.bytes+size > mapTileCacheBudget || len(c.entries) >= 1024 {
		old := c.order.Front()
		if old == nil {
			break
		}
		item := old.Value.(mapTileEntry)
		c.bytes -= item.bytes
		delete(c.entries, item.key)
		c.order.Remove(old)
	}
	c.entries[key] = c.order.PushBack(mapTileEntry{key, tile, size})
	c.bytes += size
}
