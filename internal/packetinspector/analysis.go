package packetinspector

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func Fields(payload string) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal([]byte(payload), &m)
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	return m
}
func field(m map[string]json.RawMessage, names ...string) json.RawMessage {
	for _, name := range names {
		for key, value := range m {
			if strings.EqualFold(key, name) {
				return value
			}
		}
	}
	return nil
}
func scalar(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(value, &s) == nil {
		return s
	}
	if len(value) < 128 && value[0] != '{' && value[0] != '[' {
		return string(value)
	}
	return ""
}
func position(raw json.RawMessage) ([3]float64, bool) {
	var a [3]float64
	if len(raw) == 0 {
		return a, false
	}
	var v []float64
	if json.Unmarshal(raw, &v) == nil && len(v) >= 3 {
		copy(a[:], v)
		return a, true
	}
	var m map[string]float64
	if json.Unmarshal(raw, &m) == nil && len(m) > 0 {
		for i, k := range []string{"x", "y", "z"} {
			a[i] = m[k]
		}
		return a, true
	}
	return a, false
}
func (s *Store) Analyze(q Query) (Analysis, error) {
	a, e := s.snapshot(q.Capture)
	out := Analysis{Capture: a.meta, Types: []TypeStat{}, Buckets: []TimeBucket{}, Timeline: []Evidence{}, Entities: []Entity{}, Transactions: []Evidence{}, Chunks: []Evidence{}, Changes: []Evidence{}}
	if e != nil {
		return out, e
	}
	defer s.closeSnapshot(&a)
	types := map[string]*TypeStat{}
	entities := map[string]*Entity{}
	unique := map[string]string{}
	localPlayers := map[string]string{}
	var start, end int64
	count := a.meta.Count
	if q.Before > 0 && q.Before < count {
		count = q.Before
	}
	out.Capture.Count = count
	if count > 0 {
		p, e := s.read(&a, 1)
		if e != nil {
			return out, e
		}
		start = p.Time
		p, e = s.read(&a, count)
		if e != nil {
			return out, e
		}
		end = p.Time
	}
	width := int64(math.Max(1000, float64(end-start)/240))
	buckets := map[int64]*TimeBucket{}
	appendEvidence := func(target *[]Evidence, p Packet, summary string) {
		if len(*target) >= 2000 {
			out.Partial = true
			return
		}
		*target = append(*target, Evidence{ID: p.ID, Time: p.Time, Name: p.Name, Direction: p.Direction, Summary: summary})
	}
	for n := uint64(1); n <= count; n++ {
		p, e := s.read(&a, n)
		if e != nil {
			return out, e
		}
		if q.Name != "" && p.Name != q.Name || q.Direction != "" && p.Direction != q.Direction || q.Stage != "" && p.Stage != q.Stage || !Matches(p, q.Search) {
			continue
		}
		if addTypeStat(types, p) {
			out.Partial = true
		}
		bin := (p.Time - start) / width
		if bin < 0 {
			bin = 0
			out.Partial = true
		}
		if bin > 240 {
			bin = 240
			out.Partial = true
		}
		bucket := buckets[bin]
		if bucket == nil {
			bucket = &TimeBucket{Time: start + bin*width}
			buckets[bin] = bucket
		}
		bucket.Bytes += int64(p.Bytes)
		if p.Direction == "serverbound" {
			bucket.Serverbound++
		} else {
			bucket.Clientbound++
		}
		p = Redact(p)
		name := strings.ToLower(p.Name)
		f := Fields(p.Payload)
		if name == "startgame" {
			localPlayers[p.OriginSession] = scalar(field(f, "EntityRuntimeID"))
		}
		if p.Action != "" && p.Action != "forward" {
			appendEvidence(&out.Changes, p, p.Action+": "+p.Reason)
		}
		if containsAny(name, "login", "disconnect", "transfer", "startgame", "spawn", "respawn", "changedimension", "resourcepack", "networksettings", "playstatus", "violation") {
			appendEvidence(&out.Timeline, p, p.Name)
		}
		if containsAny(name, "itemstack", "inventorytransaction", "modalform", "commandrequest", "commandoutput") {
			ref := scalar(field(f, "RequestID", "RequestId", "FormID", "FormId", "RequestId"))
			appendEvidence(&out.Transactions, p, p.Name+" "+ref)
		}
		if containsAny(name, "levelchunk", "subchunk", "cacheblob", "cachemiss", "chunkradius", "publisherupdate") {
			x := scalar(field(f, "ChunkX", "X"))
			z := scalar(field(f, "ChunkZ", "Z"))
			appendEvidence(&out.Chunks, p, fmt.Sprintf("%s (%s, %s) %d bytes", p.Name, x, z, p.Bytes))
		}
		if !containsAny(name, "actor", "player", "entity", "move") {
			continue
		}
		runtime := scalar(field(f, "EntityRuntimeID", "ActorRuntimeID", "RuntimeEntityID", "RuntimeID"))
		if runtime == "" && name == "playerauthinput" {
			runtime = localPlayers[p.OriginSession]
		}
		uid := scalar(field(f, "EntityUniqueID", "ActorUniqueID", "UniqueEntityID"))
		if runtime != "" && uid != "" {
			if len(unique) < 10000 {
				unique[p.OriginSession+":"+uid] = runtime
			} else {
				out.Partial = true
			}
		}
		if runtime == "" {
			runtime = unique[p.OriginSession+":"+uid]
		}
		if runtime == "" {
			runtime = uid
		}
		if runtime == "" {
			continue
		}
		actorKey := p.OriginSession + ":" + runtime
		actor := entities[actorKey]
		if actor == nil {
			if len(entities) >= 5000 {
				out.Partial = true
				continue
			}
			actor = &Entity{ID: runtime, Partial: !containsAny(name, "addactor", "addplayer"), Positions: [][3]float64{}, Evidence: []string{}}
			actor.Key = actorKey
			actor.Session = p.OriginSession
			entities[actorKey] = actor
		}
		if typ := scalar(field(f, "EntityType", "EntityTypeIdentifier", "Identifier")); typ != "" {
			actor.Type = typ
		}
		if nick := scalar(field(f, "Username", "Name")); nick != "" {
			actor.Name = nick
		}
		if containsAny(name, "removeactor", "removeentity") {
			actor.Removed = true
		}
		if name == "moveactordelta" {
			updated := false
			for i, key := range []string{"PositionX", "PositionY", "PositionZ"} {
				raw := field(f, key)
				if len(raw) > 0 && string(raw) != "null" {
					var value float64
					if json.Unmarshal(raw, &value) == nil {
						actor.Position[i] = value
						updated = true
					}
				}
			}
			if updated {
				actor.Positions = append(actor.Positions, actor.Position)
				if len(actor.Positions) > 128 {
					actor.Positions = actor.Positions[1:]
					out.Partial = true
				}
			}
		}
		if pos, ok := position(field(f, "Position")); ok {
			if name == "moveactordelta" {
				flagsText := scalar(field(f, "Flags"))
				flags, _ := strconv.ParseUint(flagsText, 10, 16)
				for i := 0; i < 3; i++ {
					if flags&(1<<i) != 0 {
						actor.Position[i] = pos[i]
					}
				}
			} else {
				actor.Position = pos
			}
			actor.Positions = append(actor.Positions, actor.Position)
			if len(actor.Positions) > 128 {
				actor.Positions = actor.Positions[1:]
				out.Partial = true
			}
		}
		actor.Evidence = append(actor.Evidence, p.ID)
		if len(actor.Evidence) > 32 {
			actor.Evidence = actor.Evidence[1:]
		}
	}
	for _, t := range types {
		out.Types = append(out.Types, *t)
	}
	sort.Slice(out.Types, func(i, j int) bool { return out.Types[i].Count > out.Types[j].Count })
	for _, b := range buckets {
		out.Buckets = append(out.Buckets, *b)
	}
	sort.Slice(out.Buckets, func(i, j int) bool { return out.Buckets[i].Time < out.Buckets[j].Time })
	for _, actor := range entities {
		out.Entities = append(out.Entities, *actor)
	}
	sort.Slice(out.Entities, func(i, j int) bool { return out.Entities[i].ID < out.Entities[j].ID })
	return out, nil
}
func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
