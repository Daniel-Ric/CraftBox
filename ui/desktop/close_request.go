package desktop

func (a *App) shouldPreventWindowClose() bool {
	if a.shouldPreventSessionClose() {
		return true
	}
	return a.shouldPreventStudioClose()
}

func (a *App) shouldPreventSessionClose() bool {
	a.mu.Lock()
	running, approved, emit := a.state.Running, a.sessionCloseApproved, a.emit
	a.sessionCloseApproved = false
	a.mu.Unlock()
	if running && !approved {
		if emit != nil {
			emit("desktop:close-requested", true)
		}
		return true
	}
	return false
}
