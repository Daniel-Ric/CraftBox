package desktop

import "github.com/Daniel-Ric/CraftBox/internal/terminal"

func (a *App) terminalManager() *terminal.Manager {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminals == nil {
		a.terminals = terminal.New()
	}
	return a.terminals
}
func (a *App) CreateTerminal(r terminal.Request) (terminal.Info, error) {
	return a.terminalManager().Create(r)
}
func (a *App) ReadTerminal(id string) (terminal.Output, error) { return a.terminalManager().Read(id) }
func (a *App) WriteTerminal(id, data string) error             { return a.terminalManager().Write(id, data) }
func (a *App) ResizeTerminal(id string, cols, rows int) error {
	return a.terminalManager().Resize(id, cols, rows)
}
func (a *App) CloseTerminal(id string) error { return a.terminalManager().Close(id) }
func (a *App) closeTerminals()               { a.terminalManager().Shutdown() }
