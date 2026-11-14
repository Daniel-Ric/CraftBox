package worlds

import (
	"image/color"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type customBiome struct {
	name  string
	biome protocol.BiomeDefinition
	pk    *packet.BiomeDefinitionList
}

func (c *customBiome) Temperature() float64 {
	return float64(c.biome.Temperature)
}

func (c *customBiome) Rainfall() float64 { return float64(c.biome.Downfall) }
func (c *customBiome) Depth() float64    { return float64(c.biome.Depth) }
func (c *customBiome) Scale() float64    { return float64(c.biome.Scale) }
func (c *customBiome) WaterColour() color.RGBA {
	v := uint32(c.biome.MapWaterColour)
	if v == 0 {
		return color.RGBA{}
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}
func (c *customBiome) Tags() []string {
	return nil
}

func (c *customBiome) String() string {
	return c.name
}

func (c *customBiome) EncodeBiome() int {
	return int(c.biome.BiomeID)
}
