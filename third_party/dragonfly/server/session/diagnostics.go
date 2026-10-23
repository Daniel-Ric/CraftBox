package session

type Diagnostics struct {
	AverageFramesPerSecond float64

	AverageServerSimTickTime float64

	AverageClientSimTickTime float64

	AverageBeginFrameTime float64

	AverageInputTime float64

	AverageRenderTime float64

	AverageEndFrameTime float64

	AverageRemainderTimePercent float64

	AverageUnaccountedTimePercent float64
}
