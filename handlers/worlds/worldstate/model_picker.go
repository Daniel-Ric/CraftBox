package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/internal/modelpicker"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"maps"
)

func (w *World) PickBlock(eye mgl32.Vec3, pitch, yaw float32) (modelpicker.BlockTarget, error) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	return modelpicker.PickBlock(eye, pitch, yaw, func(pos [3]int) (modelpicker.BlockTarget, bool) {
		if pos[1] < w.dimRange.Min() || pos[1] > w.dimRange.Max() {
			return modelpicker.BlockTarget{Identifier: "minecraft:air"}, true
		}
		ch, known, err := w.loadChunkLocked(world.ChunkPos{int32(pos[0] >> 4), int32(pos[2] >> 4)})
		if err != nil || !known {
			return modelpicker.BlockTarget{}, false
		}
		rid := ch.Block(uint8(pos[0]&15), int16(pos[1]), uint8(pos[2]&15), 0)
		name, states, known := w.BlockRegistry.RuntimeIDToState(rid)
		return modelpicker.BlockTarget{Identifier: name, States: maps.Clone(states), RuntimeID: rid}, known
	})
}
