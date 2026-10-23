package block

type LeavesType struct {
	leavesType
}

func OakLeaves() LeavesType {
	return LeavesType{0}
}

func SpruceLeaves() LeavesType {
	return LeavesType{1}
}

func BirchLeaves() LeavesType {
	return LeavesType{2}
}

func JungleLeaves() LeavesType {
	return LeavesType{3}
}

func AcaciaLeaves() LeavesType {
	return LeavesType{4}
}

func DarkOakLeaves() LeavesType {
	return LeavesType{5}
}

func MangroveLeaves() LeavesType {
	return LeavesType{6}
}

func CherryLeaves() LeavesType {
	return LeavesType{7}
}

func PaleOakLeaves() LeavesType {
	return LeavesType{8}
}

func AzaleaLeaves() LeavesType {
	return LeavesType{9}
}

func FloweringAzaleaLeaves() LeavesType {
	return LeavesType{10}
}

func OrangePoplarLeaves() LeavesType {
	return LeavesType{11}
}

func RedPoplarLeaves() LeavesType {
	return LeavesType{12}
}

func YellowPoplarLeaves() LeavesType {
	return LeavesType{13}
}

func PoplarLeavesTypes() []LeavesType {
	return []LeavesType{OrangePoplarLeaves(), RedPoplarLeaves(), YellowPoplarLeaves()}
}

func LeavesTypes() []LeavesType {
	return []LeavesType{
		OakLeaves(),
		SpruceLeaves(),
		BirchLeaves(),
		JungleLeaves(),
		AcaciaLeaves(),
		DarkOakLeaves(),
		MangroveLeaves(),
		CherryLeaves(),
		PaleOakLeaves(),
		AzaleaLeaves(),
		FloweringAzaleaLeaves(),
		OrangePoplarLeaves(),
		RedPoplarLeaves(),
		YellowPoplarLeaves(),
	}
}

func WoodLeavesTypes() []LeavesType {
	return []LeavesType{
		OakLeaves(),
		SpruceLeaves(),
		BirchLeaves(),
		JungleLeaves(),
		AcaciaLeaves(),
		DarkOakLeaves(),
		MangroveLeaves(),
		CherryLeaves(),
		PaleOakLeaves(),
		OrangePoplarLeaves(),
		RedPoplarLeaves(),
		YellowPoplarLeaves(),
	}
}

type leavesType uint8

func (t leavesType) Uint8() uint8 {
	return uint8(t)
}

func (t leavesType) String() string {
	switch t {
	case 9:
		return "azalea_leaves"
	case 10:
		return "azalea_leaves_flowered"
	case 11:
		return "orange_poplar_leaves"
	case 12:
		return "red_poplar_leaves"
	case 13:
		return "yellow_poplar_leaves"
	}
	if wood, ok := t.Wood(); ok {
		return wood.String() + "_leaves"
	}
	panic("unknown leaves type")
}

func (t leavesType) Sapling() (SaplingType, bool) {
	switch t {
	case 0:
		return OakSapling(), true
	case 1:
		return SpruceSapling(), true
	case 2:
		return BirchSapling(), true
	case 3:
		return JungleSapling(), true
	case 4:
		return AcaciaSapling(), true
	case 5:
		return DarkOakSapling(), true
	case 7:
		return CherrySapling(), true
	case 8:
		return PaleOakSapling(), true
	case 11, 12, 13:
		return PoplarSapling(), true
	default:
		return SaplingType{}, false
	}
}

func (t leavesType) Wood() (WoodType, bool) {
	switch t {
	case 0:
		return OakWood(), true
	case 1:
		return SpruceWood(), true
	case 2:
		return BirchWood(), true
	case 3:
		return JungleWood(), true
	case 4:
		return AcaciaWood(), true
	case 5:
		return DarkOakWood(), true
	case 6:
		return MangroveWood(), true
	case 7:
		return CherryWood(), true
	case 8:
		return PaleOakWood(), true
	case 11, 12, 13:
		return PoplarWood(), true
	default:
		return WoodType{}, false
	}
}
