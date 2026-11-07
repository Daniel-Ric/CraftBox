package worldstate

import (
	"context"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// SaveStructure never writes to the live world.
func (w *World) SaveStructure(ctx context.Context, name string, b structure.Bounds) (*structure.Session, error) {
	w.stateLock.Lock()
	store := w.scene
	packs := append([]*resource.Pack{}, w.ResourcePacks...)
	entries := append(w.BlockEntries[:0:0], w.BlockEntries...)
	w.stateLock.Unlock()
	return structure.NewCapturedSession(ctx, utils.PathData("structures", "sessions"), store, name, b, packs, entries)
}
