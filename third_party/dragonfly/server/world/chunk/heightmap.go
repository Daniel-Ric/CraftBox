package chunk

import (
	"math"
)

type HeightMap []int16

func (h HeightMap) At(x, z uint8) int16 {
	return h[(uint16(x)<<4)|uint16(z)]
}

func (h HeightMap) Set(x, z uint8, val int16) {
	h[(uint16(x)<<4)|uint16(z)] = val
}

func (h HeightMap) HighestNeighbour(x, z uint8) int16 {
	highest := int16(math.MinInt16)
	if x != 15 {
		if val := h.At(x+1, z); val > highest {
			highest = val
		}
	}
	if x != 0 {
		if val := h.At(x-1, z); val > highest {
			highest = val
		}
	}
	if z != 15 {
		if val := h.At(x, z+1); val > highest {
			highest = val
		}
	}
	if z != 0 {
		if val := h.At(x, z-1); val > highest {
			highest = val
		}
	}
	return highest
}
