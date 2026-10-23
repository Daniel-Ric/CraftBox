package packet

import (
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ClientBoundDebugRendererClear = iota
	ClientBoundDebugRendererAddCube
)

type ClientBoundDebugRenderer struct {
	Type uint32

	Data protocol.Optional[DebugMarkerData]
}

type DebugMarkerData struct {
	Text string

	Position mgl32.Vec3

	Colour color.RGBA

	Duration uint64
}

func (x *DebugMarkerData) Marshal(io protocol.IO) {
	io.String(&x.Text)
	io.Vec3(&x.Position)
	io.BEARGB(&x.Colour)
	io.Uint64(&x.Duration)
}

func (*ClientBoundDebugRenderer) ID() uint32 {
	return IDClientBoundDebugRenderer
}

func (pk *ClientBoundDebugRenderer) Marshal(io protocol.IO) {
	typStr := clientBoundDebugRenderToString(pk.Type)
	io.String(&typStr)
	clientBoundDebugRenderFromString(io, &pk.Type, typStr)
	protocol.OptionalMarshaler(io, &pk.Data)
}

func clientBoundDebugRenderToString(x uint32) string {
	switch x {
	case ClientBoundDebugRendererClear:
		return "cleardebugmarkers"
	case ClientBoundDebugRendererAddCube:
		return "adddebugmarkercube"
	default:
		return "unknown"
	}
}

func clientBoundDebugRenderFromString(io protocol.IO, x *uint32, s string) {
	switch s {
	case "cleardebugmarkers":
		*x = ClientBoundDebugRendererClear
	case "adddebugmarkercube":
		*x = ClientBoundDebugRendererAddCube
	default:
		io.InvalidValue(s, "type", "unknown type")
	}
}
