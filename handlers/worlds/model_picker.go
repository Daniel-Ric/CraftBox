package worlds

import (
	"errors"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/internal/modelpicker"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
)

func (w *worldsHandler) pickBlock(session *proxy.Session) (modelpicker.BlockTarget, error) {
	player := session.PlayerSnapshot()
	var target modelpicker.BlockTarget
	err := errors.New("No active terrain capture is available for block picking")
	w.currentWorld(func(world *worldstate.World) {
		target, err = world.PickBlock(player.Position, player.Pitch, player.Yaw)
	})
	return target, err
}
