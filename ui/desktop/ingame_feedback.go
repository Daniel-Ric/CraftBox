package desktop

import (
	"github.com/Daniel-Ric/CraftBox/internal/ingamefeedback"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
)

func (a *App) GetIngameHotbarMessages() bool { return proxy.IngameHotbarMessages() }
func (a *App) SetIngameHotbarMessages(enabled bool) error {
	return proxy.SetIngameHotbarMessages(enabled)
}
func (a *App) GetIngameInterfaceSettings() ingamefeedback.Settings {
	return proxy.IngameInterfaceSettings()
}
func (a *App) SetIngameInterfaceSettings(settings ingamefeedback.Settings) error {
	return proxy.SetIngameInterfaceSettings(settings)
}
