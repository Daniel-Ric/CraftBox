package debug

type Renderer interface {
	AddDebugShape(shape Shape)

	RemoveDebugShape(shape Shape)

	VisibleDebugShapes() []Shape

	RemoveAllDebugShapes()
}
