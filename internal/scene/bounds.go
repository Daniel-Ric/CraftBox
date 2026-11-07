package scene

import "fmt"

type Bounds struct {
	MinX int32 `json:"minX"`
	MinZ int32 `json:"minZ"`
	MaxX int32 `json:"maxX"`
	MaxZ int32 `json:"maxZ"`
}

func (b Bounds) Contains(x, z int32) bool {
	return x >= b.MinX && x <= b.MaxX && z >= b.MinZ && z <= b.MaxZ
}
func (b Bounds) Validate() error {
	if b.MinX > b.MaxX || b.MinZ > b.MaxZ || b.MinX < -1875000 || b.MinZ < -1875000 || b.MaxX > 1874999 || b.MaxZ > 1874999 {
		return fmt.Errorf("Choose valid chunk bounds within the world limits")
	}
	return nil
}
