package worldstate

import "time"

func (w *World) KeepRemovedSince(since time.Time) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	for _, ent := range w.memState.entities {
		if ent.DeletedDistance > 0 && !ent.DeletedAt.Before(since) {
			ent.DeletedDistance = -1
		}
	}
}
