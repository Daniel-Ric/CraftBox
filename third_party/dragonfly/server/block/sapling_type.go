package block

type SaplingType struct {
	sapling
}

func OakSapling() SaplingType {
	return SaplingType{0}
}

func SpruceSapling() SaplingType {
	return SaplingType{1}
}

func BirchSapling() SaplingType {
	return SaplingType{2}
}

func JungleSapling() SaplingType {
	return SaplingType{3}
}

func AcaciaSapling() SaplingType {
	return SaplingType{4}
}

func DarkOakSapling() SaplingType {
	return SaplingType{5}
}

func CherrySapling() SaplingType {
	return SaplingType{6}
}

func PaleOakSapling() SaplingType {
	return SaplingType{7}
}

func PoplarSapling() SaplingType {
	return SaplingType{8}
}

func SaplingTypes() []SaplingType {
	return []SaplingType{OakSapling(), SpruceSapling(), BirchSapling(), JungleSapling(), AcaciaSapling(),
		DarkOakSapling(), CherrySapling(), PaleOakSapling(), PoplarSapling()}
}

type sapling uint8

func (s sapling) Uint8() uint8 {
	return uint8(s)
}

func (s sapling) Name() string {
	switch s {
	case 0:
		return "Oak Sapling"
	case 1:
		return "Spruce Sapling"
	case 2:
		return "Birch Sapling"
	case 3:
		return "Jungle Sapling"
	case 4:
		return "Acacia Sapling"
	case 5:
		return "Dark Oak Sapling"
	case 6:
		return "Cherry Sapling"
	case 7:
		return "Pale Oak Sapling"
	case 8:
		return "Poplar Sapling"
	}
	panic("unknown sapling type")
}

func (s sapling) String() string {
	return s.Wood().String() + "_sapling"
}

func (s sapling) Wood() WoodType {
	switch s {
	case 0:
		return OakWood()
	case 1:
		return SpruceWood()
	case 2:
		return BirchWood()
	case 3:
		return JungleWood()
	case 4:
		return AcaciaWood()
	case 5:
		return DarkOakWood()
	case 6:
		return CherryWood()
	case 7:
		return PaleOakWood()
	case 8:
		return PoplarWood()
	}
	panic("unknown sapling type")
}

func (s sapling) Leaves() LeavesType {
	switch s {
	case 0:
		return OakLeaves()
	case 1:
		return SpruceLeaves()
	case 2:
		return BirchLeaves()
	case 3:
		return JungleLeaves()
	case 4:
		return AcaciaLeaves()
	case 5:
		return DarkOakLeaves()
	case 6:
		return CherryLeaves()
	case 7:
		return PaleOakLeaves()
	case 8:
		return OrangePoplarLeaves()
	}
	panic("unknown sapling type")
}
