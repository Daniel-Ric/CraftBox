package chunk

var (
	SkyLight skyLight

	BlockLight blockLight
)

type (
	light interface {
		light(sub *SubChunk, x, y, z uint8) uint8
		setLight(sub *SubChunk, x, y, z, v uint8)
	}
	skyLight   struct{}
	blockLight struct{}
)

func (skyLight) light(sub *SubChunk, x, y, z uint8) uint8   { return sub.SkyLight(x, y, z) }
func (skyLight) setLight(sub *SubChunk, x, y, z, v uint8)   { sub.SetSkyLight(x, y, z, v) }
func (blockLight) light(sub *SubChunk, x, y, z uint8) uint8 { return sub.BlockLight(x, y, z) }
func (blockLight) setLight(sub *SubChunk, x, y, z, v uint8) { sub.SetBlockLight(x, y, z, v) }
