package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
	"math"
	"reflect"
	"sort"
)

func actorKey(p world.ChunkPos) []byte { return append(chunkKey(p), 'e') }
func actorChunk(a Actor) world.ChunkPos {
	return world.ChunkPos{int32(math.Floor(float64(a.Position[0]) / 16)), int32(math.Floor(float64(a.Position[2]) / 16))}
}

func (s *Store) PutActor(a Actor) {
	for _, v := range a.Position {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return
		}
	}
	if math.IsNaN(float64(a.Scale)) || math.IsInf(float64(a.Scale), 0) || a.Scale < 0 || a.Scale > 128 {
		return
	}
	raw, err := json.Marshal(a.Properties)
	if err != nil || len(raw) > 64<<10 {
		return
	}
	a.Properties = nil
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &a.Properties)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.storageError != nil {
		return
	}
	if s.actors == nil {
		s.actors = map[int64]Actor{}
	}
	old, exists := s.actors[a.ID]
	allowed := actorAllowed(a)
	if !allowed {
		if s.excludedActors == nil {
			s.excludedActors = map[int64]bool{}
		}
		if !s.excludedActors[a.ID] {
			s.excludedActors[a.ID] = true
			if len(s.actorLinks[a.ID]) > 0 {
				s.rewriteActorsLocked(s.allActorChunksLocked())
			}
		}
	}
	if (!exists && !allowed) || (exists && allowed && reflect.DeepEqual(old, a)) {
		return
	}
	affected := map[world.ChunkPos]bool{}
	if exists {
		affected[actorChunk(old)] = true
	}
	if allowed {
		s.actors[a.ID] = a
		affected[actorChunk(a)] = true
	} else {
		delete(s.actors, a.ID)
	}
	s.rewriteActorsLocked(affected)
}

func (s *Store) allActorChunksLocked() map[world.ChunkPos]bool {
	out := map[world.ChunkPos]bool{}
	for _, a := range s.actors {
		out[actorChunk(a)] = true
	}
	return out
}
func (s *Store) linkedExcludedLocked(id int64) bool {
	seen := map[int64]bool{}
	pending := []int64{id}
	for len(pending) > 0 && len(seen) < 256 {
		current := pending[0]
		pending = pending[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		if s.excludedActors[current] {
			return true
		}
		for next := range s.actorLinks[current] {
			pending = append(pending, next)
		}
	}
	return false
}

func (s *Store) ActorExcluded(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.linkedExcludedLocked(id)
}

func (s *Store) LinkActor(first, second int64, linked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.storageError != nil {
		return
	}
	if s.actorLinks == nil {
		s.actorLinks = map[int64]map[int64]bool{}
	}
	if s.actorLinks[first] == nil {
		s.actorLinks[first] = map[int64]bool{}
	}
	if s.actorLinks[second] == nil {
		s.actorLinks[second] = map[int64]bool{}
	}
	if linked {
		s.actorLinks[first][second] = true
		s.actorLinks[second][first] = true
	} else {
		delete(s.actorLinks[first], second)
		delete(s.actorLinks[second], first)
	}
	s.rewriteActorsLocked(s.allActorChunksLocked())
}
func (s *Store) rewriteActorsLocked(affected map[world.ChunkPos]bool) {
	if len(affected) == 0 {
		return
	}
	s.sequence++
	batch := new(leveldb.Batch)
	for p := range affected {
		actors := []Actor{}
		for _, e := range s.actors {
			if actorChunk(e) == p && !s.linkedExcludedLocked(e.ID) {
				actors = append(actors, e)
			}
		}
		sort.Slice(actors, func(i, j int) bool { return actors[i].ID < actors[j].ID })
		raw, _ := json.Marshal(actors)
		batch.Put(actorKey(p), raw)
		s.positions[p] = struct{}{}
		s.versions[p] = s.sequence
		if old, ok := s.cache[p]; ok {
			s.cacheBytes -= meshBytes(old)
			delete(s.cache, p)
		}
	}
	if err := s.db.Write(batch, nil); err != nil {
		s.storageError = err
	}
}

func ActorFromNBT(id int64, data map[string]any) Actor {
	typ, _ := data["identifier"].(string)
	a := Actor{ID: id, Type: typ, Position: vectorValue(data["Pos"], [3]float32{}), Scale: 1, Variant: int(numeric(data["Variant"])), MarkVariant: int(numeric(data["MarkVariant"])), NameVisible: truth(data["CustomNameVisible"]), Height: 1.8}
	if owner, ok := data["OwnerNew"].(int64); ok {
		a.OwnerID = owner
	} else {
		a.OwnerID = int64(numeric(data["OwnerNew"]))
	}
	if scale, ok := data["CraftBoxScale"]; ok {
		a.Scale = float32(numeric(scale))
	}
	if height, ok := data["CraftBoxHeight"]; ok {
		a.Height = float32(numeric(height))
	}
	a.Invisible = truth(data["CraftBoxInvisible"])
	a.Excluded = truth(data["CraftBoxExcludeFrom3D"])
	if effects, ok := data["ActiveEffects"].([]any); ok {
		for _, effect := range effects {
			if e, ok := effect.(map[string]any); ok && numeric(e["Id"]) == 14 {
				a.Invisible = true
			}
		}
	}
	a.Name, _ = data["CustomName"].(string)
	if raw, ok := data["RawtextName"].(string); ok && raw != "" {
		a.Name = raw
	}
	if r, ok := data["Rotation"].([]float32); ok && len(r) > 0 {
		a.Yaw = r[0]
	}
	if r, ok := data["Rotation"].([]any); ok && len(r) > 0 {
		a.Yaw = float32(numeric(r[0]))
	}
	a.Properties, _ = data["properties"].(map[string]any)
	return a
}
