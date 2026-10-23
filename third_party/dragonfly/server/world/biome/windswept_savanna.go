package biome

import "image/color"

type WindsweptSavanna struct{}

func (WindsweptSavanna) Temperature() float64 {
	return 2
}

func (WindsweptSavanna) Rainfall() float64 {
	return 0
}

func (WindsweptSavanna) Depth() float64 {
	return 0.3625
}

func (WindsweptSavanna) Scale() float64 {
	return 1.225
}

func (WindsweptSavanna) WaterColour() color.RGBA {
	return color.RGBA{R: 0x60, G: 0xb7, B: 0xff, A: 0xa6}
}

func (WindsweptSavanna) Tags() []string {
	return []string{"animal", "monster", "mutated", "overworld", "savanna", "spawns_savanna_mobs", "spawns_warm_variant_farm_animals"}
}

func (WindsweptSavanna) String() string {
	return "savanna_mutated"
}

func (WindsweptSavanna) EncodeBiome() int {
	return 163
}
