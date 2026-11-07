package packetinspector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var secretKey = regexp.MustCompile(`(?i)(token|password|secret|key$|keydata|authorization|certificate|jwt|signature|salt|^chain$)`)
var secretAssignment = regexp.MustCompile(`(?i)((?:[a-z_]*token|password|secret|[a-z_]*key|authorization|jwt|signature|salt)\s*[:=]\s*)[^,\s)\]}]+`)

func sensitiveName(name string) bool {
	name = strings.ToLower(name)
	return name == "login" || strings.Contains(name, "handshake") || strings.Contains(name, "authentication")
}
func redactJSON(text string) (string, bool) {
	if text == "" {
		return text, false
	}
	d := json.NewDecoder(strings.NewReader(text))
	d.UseNumber()
	var v any
	if d.Decode(&v) != nil {
		return `{"redacted":true,"reason":"Unstructured payload; enable sensitive data to inspect the original"}`, true
	}
	changed := false
	var walk func(any) any
	walk = func(v any) any {
		switch t := v.(type) {
		case string:
			if secretAssignment.MatchString(t) {
				changed = true
				return secretAssignment.ReplaceAllString(t, "${1}[hidden]")
			}
		case map[string]any:
			for k, x := range t {
				if secretKey.MatchString(k) {
					t[k] = "[hidden]"
					changed = true
				} else {
					t[k] = walk(x)
				}
			}
		case []any:
			for i, x := range t {
				t[i] = walk(x)
			}
		}
		return v
	}
	walk(v)
	if !changed {
		return text, false
	}
	raw, _ := json.Marshal(v)
	return string(raw), true
}
func Redact(p Packet) Packet {
	p.Decoded = readablePayload(p)
	if p.Partial && len(p.Raw) > 0 {
		p.Raw = nil
		p.Redacted = true
		p.Reason = "Undecoded wire bytes hidden; enable sensitive data to inspect them"
	}
	if sensitiveName(p.Name) || sensitivePacketID(p.PacketID) {
		p.Payload = `{"redacted":true,"reason":"Enable Show sensitive data to inspect this packet"}`
		p.Decoded = p.Payload
		p.Before = ""
		p.Raw = nil
		p.Redacted = true
		p.Reason = "Sensitive packet details hidden"
		return p
	}
	var changed, before bool
	p.Payload, changed = redactJSON(p.Payload)
	var decodedChanged bool
	p.Decoded, decodedChanged = redactJSON(p.Decoded)
	if decodedChanged {
		// Canonical JSON may retain the same secret inside a base64 byte field.
		// A redacted record is not editable/sendable, so expose only its safe view.
		p.Payload = p.Decoded
	}
	changed = changed || decodedChanged
	p.Before = readablePayload(Packet{Payload: p.Before})
	p.Before, before = redactJSON(p.Before)
	if changed || before {
		p.Raw = nil
		p.Redacted = true
	}
	return p
}

var searchParts = regexp.MustCompile(`(?:[^\s"'|]+|"[^"]*"|'[^']*')+|\|`)

func Matches(p Packet, search string) bool {
	if strings.TrimSpace(search) == "" {
		return true
	}
	return searchMatch(p, search)
}

func (s *Store) snapshot(id string) (archive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleting[id] {
		return archive{}, fmt.Errorf("This recording is being deleted")
	}
	a, e := s.open(id)
	if e != nil {
		return archive{}, e
	}
	out := *a
	out.data, e = os.Open(filepath.Join(s.Root, id, "packets.bin"))
	if e != nil {
		return archive{}, e
	}
	out.index, e = os.Open(filepath.Join(s.Root, id, "packets.idx"))
	if e != nil {
		out.data.Close()
		return archive{}, e
	}
	if s.readers[id] == 0 {
		s.readCancel[id] = make(chan struct{})
		s.readDone[id] = make(chan struct{})
	}
	s.readers[id]++
	out.readCancel = s.readCancel[id]
	out.annotations = map[string]Annotation{}
	for k, v := range a.annotations {
		out.annotations[k] = v
	}
	return out, nil
}
func (s *Store) closeSnapshot(a *archive) {
	a.data.Close()
	a.index.Close()
	s.mu.Lock()
	s.readers[a.meta.ID]--
	if s.readers[a.meta.ID] == 0 {
		close(s.readDone[a.meta.ID])
		delete(s.readDone, a.meta.ID)
		delete(s.readCancel, a.meta.ID)
		delete(s.readers, a.meta.ID)
	}
	s.mu.Unlock()
}
func (s *Store) Query(q Query) (Page, error) {
	out := Page{Rows: []Packet{}}
	a, e := s.snapshot(q.Capture)
	if e != nil {
		return out, e
	}
	defer s.closeSnapshot(&a)
	out.Capture = a.meta
	if len(q.Search) > 2048 {
		return out, fmt.Errorf("Search exceeds 2048 characters")
	}
	if q.Limit <= 0 {
		q.Limit = 200
	}
	if q.Limit > 400 {
		q.Limit = 400
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	count := a.meta.Count
	if q.Before > 0 && q.Before < count {
		count = q.Before
	}
	if q.Search == "" && q.Name == "" && q.Direction == "" && q.Stage == "" && q.Action == "" && !q.Bookmarked {
		start := q.After + 1
		if start > count {
			return out, nil
		}
		out.Total = count - start + 1
		for i := uint64(q.Offset); i < out.Total && len(out.Rows) < q.Limit; i++ {
			n := start + i
			if q.Reverse {
				n = count - i
			}
			p, e := s.read(&a, n)
			if e != nil {
				return out, e
			}
			p = Redact(p)
			p.Summary = packetSummary(p)
			p.Decoded = ""
			p.Raw = nil
			p.Payload = ""
			p.Before = ""
			out.Rows = append(out.Rows, p)
		}
		return out, nil
	}
	for i := uint64(1); i <= count; i++ {
		n := i
		if q.Reverse {
			n = count - i + 1
		}
		if n <= q.After {
			continue
		}
		p, e := s.read(&a, n)
		if e != nil {
			return out, e
		}
		if q.Name != "" && p.Name != q.Name || q.Direction != "" && p.Direction != q.Direction || q.Stage != "" && p.Stage != q.Stage || q.Action != "" && p.Action != q.Action || q.Bookmarked && !p.Bookmarked || !Matches(p, q.Search) {
			continue
		}
		out.Total++
		if out.Total <= uint64(q.Offset) || len(out.Rows) >= q.Limit {
			continue
		}
		p = Redact(p)
		p.Summary = packetSummary(p)
		p.Decoded = ""
		p.Raw = nil
		p.Payload = ""
		p.Before = ""
		out.Rows = append(out.Rows, p)
	}
	return out, nil
}
func (s *Store) Export(q Query, reveal bool, write func(Packet) error) error {
	a, e := s.snapshot(q.Capture)
	if e != nil {
		return e
	}
	defer s.closeSnapshot(&a)
	for n := uint64(1); n <= a.meta.Count; n++ {
		if q.Before > 0 && n > q.Before {
			break
		}
		p, e := s.read(&a, n)
		if e != nil {
			return e
		}
		if n <= q.After || q.Name != "" && p.Name != q.Name || q.Direction != "" && p.Direction != q.Direction || q.Stage != "" && p.Stage != q.Stage || q.Action != "" && p.Action != q.Action || q.Bookmarked && !p.Bookmarked || !Matches(p, q.Search) {
			continue
		}
		if !reveal {
			p = Redact(p)
		}
		if e = write(p); e != nil {
			return e
		}
	}
	return nil
}
func RawBase64(value string) ([]byte, error) { return base64.StdEncoding.DecodeString(value) }
