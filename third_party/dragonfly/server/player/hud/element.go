package hud

type Element struct {
	element
}

type element uint8

func PaperDoll() Element {
	return Element{0}
}

func Armour() Element {
	return Element{1}
}

func ToolTips() Element {
	return Element{2}
}

func TouchControls() Element {
	return Element{3}
}

func Crosshair() Element {
	return Element{4}
}

func HotBar() Element {
	return Element{5}
}

func Health() Element {
	return Element{6}
}

func ProgressBar() Element {
	return Element{7}
}

func Hunger() Element {
	return Element{8}
}

func AirBubbles() Element {
	return Element{9}
}

func HorseHealth() Element {
	return Element{10}
}

func StatusEffects() Element {
	return Element{11}
}

func ItemText() Element {
	return Element{12}
}

func (s element) Uint8() uint8 {
	return uint8(s)
}

func All() []Element {
	return []Element{
		PaperDoll(), Armour(), ToolTips(), TouchControls(), Crosshair(), HotBar(), Health(),
		ProgressBar(), Hunger(), AirBubbles(), HorseHealth(), StatusEffects(), ItemText(),
	}
}
