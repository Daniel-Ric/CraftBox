package scene

import "encoding/base64"

type atlasChange struct {
	revision uint64
	x, y     int
}
type AtlasPatch struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Size int    `json:"size"`
	Data string `json:"data"`
}

func (c *Catalog) recordAtlasChange(x, y int) {
	if c.changeIndex == nil {
		c.changeIndex = map[[2]int]int{}
		for i, v := range c.changes {
			c.changeIndex[[2]int{v.x, v.y}] = i
		}
	}
	key := [2]int{x, y}
	change := atlasChange{revision: c.revision, x: x, y: y}
	if i, ok := c.changeIndex[key]; ok {
		c.changes[i] = change
	} else {
		c.changeIndex[key] = len(c.changes)
		c.changes = append(c.changes, change)
	}
}

func (c *Catalog) atlasPatches(after uint64) []AtlasPatch {
	out := []AtlasPatch{}
	if after == c.revision {
		return out
	}
	if after > c.revision {
		after = 0
	}
	c.ensureAtlas()
	for _, change := range c.changes {
		if change.revision <= after {
			continue
		}
		data := make([]byte, cellSize*cellSize*4)
		for y := 0; y < cellSize; y++ {
			offset := c.atlas.PixOffset(change.x, change.y+y)
			copy(data[y*cellSize*4:], c.atlas.Pix[offset:offset+cellSize*4])
		}
		out = append(out, AtlasPatch{X: change.x, Y: change.y, Size: cellSize, Data: base64.StdEncoding.EncodeToString(data)})
	}
	return out
}
