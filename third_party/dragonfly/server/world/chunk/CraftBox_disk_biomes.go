package chunk

import (
	"bytes"
	"fmt"
)

// DiskBiomeSections counts the complete vertical biome storages in disk data.
// Captured dimensions may have a different height from their vanilla dimension.
func DiskBiomeSections(data []byte) (int, error) {
	buf := bytes.NewBuffer(data)
	count := 0
	for buf.Len() != 0 {
		if count == 256 {
			return 0, fmt.Errorf("too many disk biome sections")
		}
		storage, err := decodePalettedStorage(buf, DiskEncoding, BiomePaletteEncoding)
		if err != nil {
			return 0, fmt.Errorf("decode disk biome section %d: %w", count, err)
		}
		if count == 0 && storage == nil {
			return 0, fmt.Errorf("first disk biome section refers to a previous section")
		}
		count++
	}
	return count, nil
}
