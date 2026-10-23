package customblock

type Material struct {
	texture string

	renderMethod Method

	faceDimming bool

	ambientOcclusion float32
}

func NewMaterial(texture string, method Method) Material {
	m := Material{
		texture:          texture,
		renderMethod:     method,
		faceDimming:      true,
		ambientOcclusion: 1,
	}
	if !method.AmbientOcclusion() {
		m.ambientOcclusion = 0
	}
	return m
}

func (m Material) WithFaceDimming() Material {
	m.faceDimming = true
	return m
}

func (m Material) WithoutFaceDimming() Material {
	m.faceDimming = false
	return m
}

func (m Material) WithAmbientOcclusion() Material {
	m.ambientOcclusion = 1
	return m
}

func (m Material) WithoutAmbientOcclusion() Material {
	m.ambientOcclusion = 0
	return m
}

func (m Material) Encode() map[string]any {
	return map[string]any{
		"texture":           m.texture,
		"render_method":     m.renderMethod.String(),
		"face_dimming":      m.faceDimming,
		"ambient_occlusion": m.ambientOcclusion,
	}
}
