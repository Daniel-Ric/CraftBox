package biome

import "image/color"

type SulfurCaves struct{}

func (SulfurCaves) Temperature() float64 {
	return 0.8
}

func (SulfurCaves) Rainfall() float64 {
	return 0.4
}

func (SulfurCaves) Depth() float64 {
	return 0.1
}

func (SulfurCaves) Scale() float64 {
	return 0.2
}

func (SulfurCaves) WaterColour() color.RGBA {
	return color.RGBA{R: 0x60, G: 0xb7, B: 0xff, A: 0xa6}
}

func (SulfurCaves) Tags() []string {
	return []string{"caves", "sulfur_caves", "overworld", "monster"}
}

func (SulfurCaves) String() string {
	return "sulfur_caves"
}

func (SulfurCaves) EncodeBiome() int {
	return 194
}
