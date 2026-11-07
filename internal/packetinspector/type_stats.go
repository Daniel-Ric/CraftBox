package packetinspector

import "sort"

func addTypeStat(types map[string]*TypeStat, p Packet) bool {
	name := p.Name
	other := false
	if types[name] == nil && len(types) >= 4096 {
		name = "Other types (catalog limit)"
		other = true
	}
	stat := types[name]
	if stat == nil {
		stat = &TypeStat{Name: name, PacketID: p.PacketID, First: p.Time, Other: other}
		types[name] = stat
	}
	stat.Count++
	stat.Bytes += int64(p.Bytes)
	stat.Last = p.Time
	if p.Direction == "serverbound" {
		stat.Serverbound++
	} else {
		stat.Clientbound++
	}
	return other
}
func copyTypes(types map[string]*TypeStat) []TypeStat {
	out := make([]TypeStat, 0, len(types))
	for _, stat := range types {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	return out
}
func (s *Store) Types(id string) ([]TypeStat, error) {
	s.mu.Lock()
	entry, e := s.open(id)
	if e != nil {
		s.mu.Unlock()
		return nil, e
	}
	if entry.types != nil {
		out := copyTypes(entry.types)
		s.mu.Unlock()
		return out, nil
	}
	s.mu.Unlock()
	snapshot, e := s.snapshot(id)
	if e != nil {
		return nil, e
	}
	defer s.closeSnapshot(&snapshot)
	types := map[string]*TypeStat{}
	for n := uint64(1); n <= snapshot.meta.Count; n++ {
		p, e := s.read(&snapshot, n)
		if e != nil {
			return nil, e
		}
		addTypeStat(types, p)
	}
	s.mu.Lock()
	if entry, e := s.open(id); e == nil && entry.meta.Count == snapshot.meta.Count {
		entry.types = types
	}
	s.mu.Unlock()
	return copyTypes(types), nil
}
