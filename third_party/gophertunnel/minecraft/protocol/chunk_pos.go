package protocol

type ChunkPos [2]int32

func (pos ChunkPos) X() int32 {
	return pos[0]
}

func (pos ChunkPos) Z() int32 {
	return pos[1]
}

type SubChunkPos [3]int32

func (pos SubChunkPos) X() int32 {
	return pos[0]
}

func (pos SubChunkPos) Y() int32 {
	return pos[1]
}

func (pos SubChunkPos) Z() int32 {
	return pos[2]
}
