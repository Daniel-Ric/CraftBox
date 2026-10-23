package customblock

type Method struct {
	renderMethod
}

func OpaqueRenderMethod() Method {
	return Method{0}
}

func AlphaTestRenderMethod() Method {
	return Method{1}
}

func BlendRenderMethod() Method {
	return Method{2}
}

func DoubleSidedRenderMethod() Method {
	return Method{3}
}

type renderMethod uint8

func (m renderMethod) Uint8() uint8 {
	return uint8(m)
}

func (m renderMethod) String() string {
	switch m {
	case 0:
		return "opaque"
	case 1:
		return "alpha_test"
	case 2:
		return "blend"
	case 3:
		return "double_sided"
	}
	panic("should never happen")
}

func (m renderMethod) AmbientOcclusion() bool {
	return m != 1 && m != 2
}
