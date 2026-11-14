package worldstate

import (
	"math"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl32"
)

func (w *World) NotePlayerPosition(pos mgl32.Vec3) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if w.spawnSet || !w.standsInStoredChunk(pos) {
		return
	}
	w.spawn, w.spawnSet = pos, true
}

func (w *World) standsInStoredChunk(pos mgl32.Vec3) bool {
	cp := world.ChunkPos{int32(math.Floor(float64(pos.X()))) >> 4, int32(math.Floor(float64(pos.Z()))) >> 4}
	if _, ok := w.StoredChunks[cp]; !ok {
		return false
	}
	feet := float64(pos.Y()) - PlayerEyeHeight
	return feet >= float64(w.dimRange.Min()) && feet <= float64(w.dimRange.Max()+1)
}

func (w *World) Spawn() mgl32.Vec3 {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if !w.spawnSet {
		w.spawn = w.fallbackSpawnLocked()
		w.spawnSet = true
	}
	return w.spawn
}

func (w *World) fallbackSpawnLocked() mgl32.Vec3 {
	start := w.StartPos
	if w.standsInStoredChunk(start) || len(w.StoredChunks) == 0 {
		return start
	}
	x, z := int32(math.Floor(float64(start.X()))), int32(math.Floor(float64(start.Z())))
	cp := world.ChunkPos{x >> 4, z >> 4}
	if _, ok := w.StoredChunks[cp]; !ok {

		best := math.MaxFloat64
		for p := range w.StoredChunks {
			dx, dz := float64(p[0]-cp[0]), float64(p[1]-cp[1])
			if d := dx*dx + dz*dz; d < best {
				best, cp = d, p
			}
		}
		x, z = cp[0]<<4+8, cp[1]<<4+8
	}
	y := float32(w.dimRange.Max() + 1)
	if ch, ok, err := w.loadChunkLocked(cp); ok && err == nil {
		var br chunk.BlockRegistry = world.DefaultBlockRegistry
		if w.BlockRegistry != nil {
			br = w.BlockRegistry
		}
		air := br.AirRuntimeID()
		for by := ch.Range().Max(); by >= ch.Range().Min(); by-- {
			if ch.SubChunk(int16(by)).Empty() {
				continue
			}
			if ch.Block(uint8(x&15), int16(by), uint8(z&15), 0) != air {
				y = float32(by + 1)
				break
			}
		}
	}
	return mgl32.Vec3{float32(x) + 0.5, y + PlayerEyeHeight, float32(z) + 0.5}
}
