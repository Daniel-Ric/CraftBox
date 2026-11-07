package scene

import (
	"testing"
)

func TestFenceFamiliesRespectAllExplicitConnectionPatterns(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:oak_fence", "minecraft:nether_brick_fence", "minecraft:cobblestone_wall", "minecraft:iron_bars"} {
		for bits := 0; bits < 16; bits++ {
			props := map[string]any{}
			for i, d := range []string{"east", "west", "south", "north"} {
				on := bits&(1<<i) != 0
				if name == "minecraft:cobblestone_wall" {
					value := "none"
					if on {
						value = "short"
					}
					props["wall_connection_type_"+d] = value
					props["wall_post_bit"] = true
				} else {
					props["minecraft:connection_"+d] = on
				}
			}
			m := mechanicalModel(c, name, props)
			neighbors := [4]*blockModel{{name: "minecraft:stone", full: true}, {name: "minecraft:stone", full: true}, {name: "minecraft:stone", full: true}, {name: "minecraft:stone", full: true}}
			boxes := connectedBoxes(&m, neighbors)
			for i := 0; i < 4; i++ {
				axis := 0
				if i >= 2 {
					axis = 2
				}
				reaches := false
				for _, b := range boxes {
					if i%2 == 0 && b.max[axis] == 1 || i%2 == 1 && b.min[axis] == 0 {
						reaches = true
					}
				}
				if reaches != (bits&(1<<i) != 0) {
					t.Fatalf("%s bits %04b direction %d reaches=%t", name, bits, i, reaches)
				}
			}
		}
	}
}
func TestConnectionsRecogniseFullQuadFaces(t *testing.T) {
	c := mechanicalCatalog(t)
	target := c.composterModel(nil)
	for _, kind := range []string{"fence", "wall", "pane"} {
		source := blockModel{name: "minecraft:oak_fence", connect: kind}
		for i := 0; i < 4; i++ {
			if !connectionAllowed(&source, &target, i) {
				t.Fatalf("%s fails to attach to composter direction %d", kind, i)
			}
		}
	}
}

func TestFenceFamiliesInferAllPatternsFromFullBlockNeighbours(t *testing.T) {
	c := mechanicalCatalog(t)
	solid := blockModel{name: "minecraft:stone", full: true}
	for _, name := range []string{"minecraft:oak_fence", "minecraft:spruce_fence", "minecraft:nether_brick_fence", "minecraft:cobblestone_wall", "minecraft:deepslate_brick_wall", "minecraft:iron_bars"} {
		for bits := 0; bits < 16; bits++ {
			m := mechanicalModel(c, name, nil)
			var neighbours [4]*blockModel
			for i := range neighbours {
				if bits&(1<<i) != 0 {
					neighbours[i] = &solid
				}
			}
			boxes := connectedBoxes(&m, neighbours)
			for i := 0; i < 4; i++ {
				axis := 0
				if i >= 2 {
					axis = 2
				}
				reaches := false
				for _, b := range boxes {
					if i%2 == 0 && b.max[axis] == 1 || i%2 == 1 && b.min[axis] == 0 {
						reaches = true
					}
				}
				if reaches != (bits&(1<<i) != 0) {
					t.Fatalf("%s inferred %04b direction %d", name, bits, i)
				}
			}
		}
	}
	wood := mechanicalModel(c, "minecraft:oak_fence", nil)
	spruce := mechanicalModel(c, "minecraft:spruce_fence", nil)
	nether := mechanicalModel(c, "minecraft:nether_brick_fence", nil)
	if !connectionAllowed(&wood, &spruce, 0) || connectionAllowed(&wood, &nether, 0) {
		t.Fatal("wood/nether brick fence compatibility is wrong")
	}
}
func TestWallRaisesTheArmActuallyCoveredFromAbove(t *testing.T) {
	wall := blockModel{name: "minecraft:cobblestone_wall", connect: "wall"}
	solid := blockModel{full: true}
	top := blockModel{boxes: []box{{[3]float32{.5, 0, 0}, [3]float32{1, 1, 1}}}}
	boxes := connectedBoxes(&wall, [4]*blockModel{&solid, &solid, nil, nil}, &top)
	east, west := float32(0), float32(0)
	for _, b := range boxes {
		if b.max[0] == 1 {
			east = b.max[1]
		}
		if b.min[0] == 0 {
			west = b.max[1]
		}
	}
	if east != 1 || west != .8125 {
		t.Fatalf("wrong raised arm east=%v west=%v", east, west)
	}
}
func TestWallBelowChestUsesItsBottomSurface(t *testing.T) {
	c := mechanicalCatalog(t)
	wall := blockModel{name: "minecraft:cobblestone_wall", connect: "wall"}
	solid := blockModel{full: true}
	chest := c.chestModel("minecraft:ender_chest", nil, nil)
	boxes := connectedBoxes(&wall, [4]*blockModel{&solid, &solid, nil, nil}, &chest)
	for _, b := range boxes {
		if (b.min[0] == 0 || b.max[0] == 1) && b.max[1] != 1 {
			t.Fatal("wall leaves a gap below the chest")
		}
	}
}
func TestWallDoesNotAssumeUnknownUpperWallHasPost(t *testing.T) {
	if wallPostForConnections([4]float32{.8125, .8125, 0, 0}, &blockModel{connect: "wall"}) {
		t.Fatal("unknown upper wall forces a post into a straight wall")
	}
}
func TestDripstoneStagesUseSeparateUpAndDownSprites(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, stage := range []string{"tip", "frustum", "middle", "base", "merge"} {
		up := mechanicalModel(c, "minecraft:pointed_dripstone", map[string]any{"dripstone_thickness": stage})
		down := mechanicalModel(c, "minecraft:pointed_dripstone", map[string]any{"dripstone_thickness": stage, "hanging": true})
		if len(up.quads) != 2 || len(down.quads) != 2 || len(up.boxes) > 0 || up.quads[0].texture == down.quads[0].texture || up.quads[0].texture.missing || down.quads[0].texture.missing {
			t.Fatalf("dripstone stage %s not rendered with crossed direction-specific sprites", stage)
		}
	}
}
func TestGrindstoneHasSeparateRollerAndSupports(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, attachment := range []string{"standing", "hanging", "side"} {
		m := mechanicalModel(c, "minecraft:grindstone", map[string]any{"attachment": attachment, "direction": int32(2)})
		if len(m.quads) < 24 || len(m.boxes) > 0 || m.full {
			t.Fatalf("wrong %s grindstone geometry", attachment)
		}
		for _, q := range m.quads {
			if q.texture.missing {
				t.Fatalf("missing grindstone texture %s", attachment)
			}
		}
	}
}
