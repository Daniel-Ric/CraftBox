package protocol

type EducationSharedResourceURI struct {
	ButtonName string

	LinkURI string
}

func (x *EducationSharedResourceURI) Marshal(r IO) {
	r.String(&x.ButtonName)
	r.String(&x.LinkURI)
}

type EducationAgentCapabilities struct {
	CanModifyBlocks Optional[bool]
}

func (x *EducationAgentCapabilities) Marshal(r IO) {
	OptionalFunc(r, &x.CanModifyBlocks, r.Bool)
}

type EducationExternalLinkSettings struct {
	URL string

	DisplayName string
}

func (x *EducationExternalLinkSettings) Marshal(r IO) {
	r.String(&x.URL)
	r.String(&x.DisplayName)
}
