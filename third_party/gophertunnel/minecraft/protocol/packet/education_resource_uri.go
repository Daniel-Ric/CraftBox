package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type EducationResourceURI struct {
	Resource protocol.EducationSharedResourceURI
}

func (*EducationResourceURI) ID() uint32 {
	return IDEducationResourceURI
}

func (pk *EducationResourceURI) Marshal(io protocol.IO) {
	protocol.Single(io, &pk.Resource)
}
