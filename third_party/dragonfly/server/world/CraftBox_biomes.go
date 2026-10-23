package world

import "maps"

type BiomeRegistry struct {
	IDToBiome   map[int]Biome
	NameToBiome map[string]Biome
}

var DefaultBiomes = &BiomeRegistry{IDToBiome: biomes, NameToBiome: biomeByName}

func (r *BiomeRegistry) Clone() *BiomeRegistry {
	return &BiomeRegistry{IDToBiome: maps.Clone(r.IDToBiome), NameToBiome: maps.Clone(r.NameToBiome)}
}

func (r *BiomeRegistry) Register(b Biome) {
	if _, exists := r.IDToBiome[b.EncodeBiome()]; exists {
		panic("duplicate biome ID")
	}
	if r.IDToBiome == nil {
		r.IDToBiome = make(map[int]Biome)
	}
	if r.NameToBiome == nil {
		r.NameToBiome = make(map[string]Biome)
	}
	r.IDToBiome[b.EncodeBiome()] = b
	r.NameToBiome[b.String()] = b
}

func (r *BiomeRegistry) BiomeByID(id int) (Biome, bool) { b, ok := r.IDToBiome[id]; return b, ok }
func (r *BiomeRegistry) BiomeByName(name string) (Biome, bool) {
	b, ok := r.NameToBiome[name]
	return b, ok
}
func (r *BiomeRegistry) Biomes() []Biome {
	biomes := make([]Biome, 0, len(r.IDToBiome))
	for _, b := range r.IDToBiome {
		biomes = append(biomes, b)
	}
	return biomes
}
