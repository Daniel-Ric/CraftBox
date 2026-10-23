package messages

import "github.com/Daniel-Ric/CraftBox/internal/scene"

type EventSceneAvailable struct {
	WorldID  string
	Store    *scene.Store
	Retained bool
}
