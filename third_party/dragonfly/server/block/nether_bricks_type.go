package block

type NetherBricksType struct {
	netherBricks
}

type netherBricks uint8

func NormalNetherBricks() NetherBricksType {
	return NetherBricksType{0}
}

func RedNetherBricks() NetherBricksType {
	return NetherBricksType{1}
}

func CrackedNetherBricks() NetherBricksType {
	return NetherBricksType{2}
}

func ChiseledNetherBricks() NetherBricksType {
	return NetherBricksType{3}
}

func (n netherBricks) Uint8() uint8 {
	return uint8(n)
}

func (n netherBricks) Name() string {
	switch n {
	case 0:
		return "Nether Bricks"
	case 1:
		return "Red Nether Bricks"
	case 2:
		return "Cracked Nether Bricks"
	case 3:
		return "Chiseled Nether Bricks"
	}
	panic("unknown nether brick type")
}

func (n netherBricks) String() string {
	switch n {
	case 0:
		return "nether_brick"
	case 1:
		return "red_nether_brick"
	case 2:
		return "cracked_nether_bricks"
	case 3:
		return "chiseled_nether_bricks"
	}
	panic("unknown nether brick type")
}

func NetherBricksTypes() []NetherBricksType {
	return []NetherBricksType{NormalNetherBricks(), RedNetherBricks(), CrackedNetherBricks(), ChiseledNetherBricks()}
}
