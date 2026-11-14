package desktop

import "github.com/Daniel-Ric/CraftBox/internal/discordpresence"

func (a *App) ConfigureDiscordPresence(settings discordpresence.Settings) {
	a.discordMu.Lock()
	defer a.discordMu.Unlock()
	if a.discordPresence == nil {
		a.discordPresence = discordpresence.NewController(func() string {
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.state.Stopping {
				return "Finishing capture"
			}
			if a.state.Running {
				switch a.state.Command {
				case "worlds":
					return "Capturing worlds"
				case "packs":
					return "Capturing resource packs"
				case "skins":
					return "Capturing skins"
				case "packets":
					return "Inspecting packets"
				}
			}
			return "In the workspace"
		})
	}
	a.discordPresence.Configure(settings)
}

func (a *App) GetDiscordPresenceStatus() discordpresence.Status {
	a.discordMu.Lock()
	defer a.discordMu.Unlock()
	if a.discordPresence == nil {
		return discordpresence.Status{Message: "Disabled"}
	}
	return a.discordPresence.Status()
}

func (a *App) RetryDiscordPresence() {
	a.discordMu.Lock()
	defer a.discordMu.Unlock()
	if a.discordPresence != nil {
		a.discordPresence.Retry()
	}
}

func (a *App) closeDiscordPresence() {
	a.discordMu.Lock()
	defer a.discordMu.Unlock()
	if a.discordPresence != nil {
		a.discordPresence.Close()
		a.discordPresence = nil
	}
}
