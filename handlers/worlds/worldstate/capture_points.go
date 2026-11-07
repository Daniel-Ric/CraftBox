package worldstate

import "github.com/Daniel-Ric/CraftBox/ui/messages"

// Serialize annotation writes with provider creation, save and rename.
func (w *World) AddCapturePoint(point messages.CapturePoint) ([]messages.CapturePoint, error) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	return messages.AddCapturePoint(w.Folder, point)
}
