package block

type WallConnectionType struct {
	wallConnectionType
}

func NoWallConnection() WallConnectionType {
	return WallConnectionType{0}
}

func ShortWallConnection() WallConnectionType {
	return WallConnectionType{1}
}

func TallWallConnection() WallConnectionType {
	return WallConnectionType{2}
}

func WallConnectionTypes() []WallConnectionType {
	return []WallConnectionType{NoWallConnection(), ShortWallConnection(), TallWallConnection()}
}

type wallConnectionType uint8

func (w wallConnectionType) Uint8() uint8 {
	return uint8(w)
}

func (w wallConnectionType) String() string {
	switch w {
	case 0:
		return "none"
	case 1:
		return "short"
	case 2:
		return "tall"
	}
	panic("unknown wall connection type")
}

func (w wallConnectionType) Height() float64 {
	switch w {
	case 0:
		return 0
	case 1:
		return 0.75
	case 2:
		return 1
	}
	panic("unknown wall connection type")
}
