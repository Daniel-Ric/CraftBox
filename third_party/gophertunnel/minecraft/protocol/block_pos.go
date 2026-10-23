package protocol

type BlockPos [3]int32

func (pos BlockPos) X() int32 {
	return pos[0]
}

func (pos BlockPos) Y() int32 {
	return pos[1]
}

func (pos BlockPos) Z() int32 {
	return pos[2]
}
