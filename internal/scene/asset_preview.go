package scene

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

type PackPreview struct {
	Name  string
	Files fs.FS
}

func (s *Store) PreviewPacks(ctx context.Context, packs []PackPreview) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("The 3D scene is closed")
	}
	base := s.originalCatalog
	if base == nil {
		base = s.catalog
	}
	next := base
	if len(packs) > 0 {
		next = NewCatalog()
		for i := len(base.sources) - 1; i >= 0; i-- {
			if e := ctx.Err(); e != nil {
				return e
			}
			if e := next.AddPack(base.sources[i]); e != nil {
				return e
			}
		}
		for key, value := range base.definitions {
			if _, ok := next.definitions[key]; !ok {
				next.definitions[key] = value
			}
		}
		names := []string{}
		for i := len(packs) - 1; i >= 0; i-- {
			if e := ctx.Err(); e != nil {
				return e
			}
			if e := next.AddPack(packs[i].Files); e != nil {
				return fmt.Errorf("%s: %w", packs[i].Name, e)
			}
			names = append(names, packs[i].Name)
		}
		next.Source = "Pack preview: " + strings.Join(names, ", ")
	}
	if s.originalCatalog == nil {
		s.originalCatalog = s.catalog
	}
	s.catalog = next
	s.cache = map[world.ChunkPos]ChunkMesh{}
	s.cacheScope = map[world.ChunkPos]string{}
	s.cacheBytes = 0
	s.models = map[uint32]*blockModel{}
	s.decorativeModels = map[string]*blockModel{}
	s.sequence++
	for p := range s.versions {
		s.versions[p] = s.sequence
	}
	return nil
}
