package scene

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

type GLBProgress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Bytes uint64 `json:"bytes"`
	Stage string `json:"stage"`
}

func (s *Store) WriteGLB(ctx context.Context, out io.Writer, directory string, keep func(int32, int32) bool, terrainOnly bool, progress func(GLBProgress)) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	s.mu.Lock()
	if s.closed || s.storageError != nil {
		err := fmt.Errorf("3D cache unavailable: %v", s.storageError)
		s.mu.Unlock()
		return err
	}
	positions := []world.ChunkPos{}
	versions := make(map[world.ChunkPos]uint64, len(s.versions))
	for p, v := range s.versions {
		versions[p] = v
	}
	for p := range s.positions {
		if keep == nil || keep(p[0], p[1]) {
			positions = append(positions, p)
		}
	}
	snapshot, err := s.db.GetSnapshot()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	defer snapshot.Release()
	sort.Slice(positions, func(i, j int) bool {
		if positions[i][0] == positions[j][0] {
			return positions[i][1] < positions[j][1]
		}
		return positions[i][0] < positions[j][0]
	})
	w, err := NewGLBWriter(ctx, directory)
	if err != nil {
		return err
	}
	defer w.Close()
	scope := w.bin.Name()
	for i, p := range positions {
		if err := ctx.Err(); err != nil {
			return err
		}
		mesh, err := s.loadMesh(ctx, snapshot, p, versions, keep, scope, terrainOnly)
		if err != nil {
			return err
		}
		if err := w.AddChunk(mesh); err != nil {
			return err
		}
		if progress != nil {
			progress(GLBProgress{Done: i + 1, Total: len(positions), Bytes: w.Bytes(), Stage: "geometry"})
		}
	}
	atlas, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s.catalog.atlasURL(), "data:image/png;base64,"))
	if err != nil {
		return err
	}
	if progress != nil {
		progress(GLBProgress{Done: len(positions), Total: len(positions), Bytes: w.Bytes(), Stage: "writing"})
	}
	return w.Finish(out, atlas, s.catalog.Source)
}
