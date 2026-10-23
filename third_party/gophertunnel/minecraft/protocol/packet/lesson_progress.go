package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	LessonActionStart = iota
	LessonActionComplete
	LessonActionRestart
)

type LessonProgress struct {
	Identifier string

	Action int32

	Score int32
}

func (*LessonProgress) ID() uint32 {
	return IDLessonProgress
}

func (pk *LessonProgress) Marshal(io protocol.IO) {
	io.Varint32(&pk.Action)
	io.Varint32(&pk.Score)
	io.String(&pk.Identifier)
}
