package world

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"math"
)

type ChunkPos [2]int32

func (p ChunkPos) String() string {
	return fmt.Sprintf("(%v, %v)", p[0], p[1])
}

func (p ChunkPos) X() int32 {
	return p[0]
}

func (p ChunkPos) Z() int32 {
	return p[1]
}

type SubChunkPos [3]int32

func (p SubChunkPos) String() string {
	return fmt.Sprintf("(%v, %v, %v)", p[0], p[1], p[2])
}

func (p SubChunkPos) X() int32 {
	return p[0]
}

func (p SubChunkPos) Y() int32 {
	return p[1]
}

func (p SubChunkPos) Z() int32 {
	return p[2]
}

func chunkPosFromVec3(vec3 mgl64.Vec3) ChunkPos {
	return ChunkPos{int32(math.Floor(vec3[0])) >> 4, int32(math.Floor(vec3[2])) >> 4}
}

func chunkPosFromBlockPos(p cube.Pos) ChunkPos {
	return ChunkPos{int32(p[0] >> 4), int32(p[2] >> 4)}
}
