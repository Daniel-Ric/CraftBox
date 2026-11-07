package worldstate

import "errors"

func (w *World) Discard() error {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if w.discarded {
		return nil
	}
	w.discarded = true
	w.cancelCtx(errors.New("capture deleted"))
	if w.resourcePacksDone != nil {
		for range w.resourcePacksDone {
		}
	}
	if w.provider != nil {
		if err := w.provider.Close(); err != nil {
			return err
		}
		w.provider = nil
	}
	w.memState = newWorldState()
	clear(w.StoredChunks)
	if w.scene != nil && !w.sceneRetained {
		w.scene.Close()
		w.scene = nil
	}
	return nil
}
