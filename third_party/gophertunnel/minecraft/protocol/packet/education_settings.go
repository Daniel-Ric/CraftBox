package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type EducationSettings struct {
	CodeBuilderDefaultURI string

	CodeBuilderTitle string

	CanResizeCodeBuilder bool

	DisableLegacyTitleBar bool

	PostProcessFilter string

	ScreenshotBorderPath string

	CanModifyBlocks protocol.Optional[bool]

	OverrideURI protocol.Optional[string]

	HasQuiz bool

	ExternalLinkSettings protocol.Optional[protocol.EducationExternalLinkSettings]
}

func (*EducationSettings) ID() uint32 {
	return IDEducationSettings
}

func (pk *EducationSettings) Marshal(io protocol.IO) {
	io.String(&pk.CodeBuilderDefaultURI)
	io.String(&pk.CodeBuilderTitle)
	io.Bool(&pk.CanResizeCodeBuilder)
	io.Bool(&pk.DisableLegacyTitleBar)
	io.String(&pk.PostProcessFilter)
	io.String(&pk.ScreenshotBorderPath)
	protocol.OptionalFunc(io, &pk.CanModifyBlocks, io.Bool)
	protocol.OptionalFunc(io, &pk.OverrideURI, io.String)
	io.Bool(&pk.HasQuiz)
	protocol.OptionalMarshaler(io, &pk.ExternalLinkSettings)
}
