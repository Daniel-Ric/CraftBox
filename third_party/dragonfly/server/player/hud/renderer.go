package hud

type Renderer interface {
	ShowHudElement(e Element)

	HideHudElement(e Element)

	HudElementHidden(e Element) bool
}
