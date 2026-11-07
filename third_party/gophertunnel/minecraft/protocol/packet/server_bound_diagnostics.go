package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerBoundDiagnostics struct {
	AverageFramesPerSecond float32

	AverageServerSimTickTime float32

	AverageClientSimTickTime float32

	AverageBeginFrameTime float32

	AverageInputTime float32

	AverageRenderTime float32

	AverageEndFrameTime float32

	AverageRemainderTimePercent float32

	AverageUnaccountedTimePercent float32

	MemoryCategoryValues []protocol.MemoryCategoryCounter

	EntityDiagnostics []protocol.EntityDiagnosticTimingInfo

	SystemDiagnostics []protocol.SystemDiagnosticTimingInfo

	SystemCategories protocol.Optional[[]protocol.SystemCategory]

	WhiskerScopes []protocol.WhiskerScopeDataSummary
}

func (*ServerBoundDiagnostics) ID() uint32 {
	return IDServerBoundDiagnostics
}

func (pk *ServerBoundDiagnostics) Marshal(io protocol.IO) {
	io.Float32(&pk.AverageFramesPerSecond)
	io.Float32(&pk.AverageServerSimTickTime)
	io.Float32(&pk.AverageClientSimTickTime)
	io.Float32(&pk.AverageBeginFrameTime)
	io.Float32(&pk.AverageInputTime)
	io.Float32(&pk.AverageRenderTime)
	io.Float32(&pk.AverageEndFrameTime)
	io.Float32(&pk.AverageRemainderTimePercent)
	io.Float32(&pk.AverageUnaccountedTimePercent)
	protocol.Slice(io, &pk.MemoryCategoryValues)
	protocol.Slice(io, &pk.EntityDiagnostics)
	protocol.Slice(io, &pk.SystemDiagnostics)
	protocol.OptionalFunc(io, &pk.SystemCategories, func(s *[]protocol.SystemCategory) {
		protocol.Slice(io, s)
	})
	protocol.Slice(io, &pk.WhiskerScopes)
}
