package scene

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"strings"
)

func BlockAssetGeometry(name string, materials map[string]string) (geometry any, note string, err error) {
	defer func() {
		if recover() != nil {
			geometry = nil
			err = fmt.Errorf("Default block state is unavailable")
		}
	}()
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	reg := world.DefaultBlockRegistry
	var id uint32
	found := false
	for _, block := range reg.Blocks() {
		n, _ := block.EncodeBlock()
		if n == name {
			id = reg.BlockRuntimeID(block)
			found = true
			break
		}
	}
	if !found {
		return nil, "", fmt.Errorf("Block shape is not in the installed definitions")
	}
	store := &Store{registry: reg, catalog: NewCatalog()}
	model := store.model(id)
	if model.approx || model.cross || len(model.quads) > 0 {
		return nil, "", fmt.Errorf("This block needs its specialised mesh or entity model")
	}
	faces := map[string]any{}
	for _, face := range faceNames {
		alias := face
		if materials[face] == "" {
			if face != "up" && face != "down" && materials["side"] != "" {
				alias = "side"
			} else if materials["all"] != "" {
				alias = "all"
			} else {
				alias = "*"
			}
		}
		faces[face] = map[string]any{"uv": []int{0, 0}, "uv_size": []int{16, 16}, "material_instance": alias}
	}
	cubes := []any{}
	for _, box := range model.boxes {
		cubes = append(cubes, map[string]any{"origin": []float32{box.min[0]*16 - 8, box.min[1] * 16, box.min[2]*16 - 8}, "size": []float32{(box.max[0] - box.min[0]) * 16, (box.max[1] - box.min[1]) * 16, (box.max[2] - box.min[2]) * 16}, "uv": faces})
	}
	if len(cubes) == 0 {
		return nil, "", fmt.Errorf("No standalone shape exists for this block")
	}
	return map[string]any{"minecraft:geometry": []any{map[string]any{"description": map[string]any{"identifier": "geometry.preview.native", "texture_width": 16, "texture_height": 16}, "bones": []any{map[string]any{"name": "block", "pivot": []int{0, 0, 0}, "cubes": cubes}}}}}, "Default Minecraft block state; connections and placed-state rotations depend on the world", nil
}
