package packetinspector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func jsonChar(r *bufio.Reader) (byte, error) {
	for {
		b, e := r.ReadByte()
		if e != nil {
			return 0, e
		}
		if b != ' ' && b != '\r' && b != '\n' && b != '\t' {
			return b, nil
		}
	}
}
func boundedJSONValue(r *bufio.Reader, limit int) ([]byte, error) {
	first, e := jsonChar(r)
	if e != nil {
		return nil, e
	}
	out := []byte{first}
	depth := 0
	quoted := first == '"'
	escaped := false
	if first == '{' || first == '[' {
		depth = 1
	}
	for {
		if !quoted && depth == 0 && first != '"' {
			b, e := r.ReadByte()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, e
			}
			if b == ',' || b == ']' || b == '}' || b == ' ' || b == '\n' || b == '\r' || b == '\t' {
				_ = r.UnreadByte()
				break
			}
			out = append(out, b)
		} else {
			b, e := r.ReadByte()
			if e != nil {
				return nil, e
			}
			out = append(out, b)
			if quoted {
				if escaped {
					escaped = false
				} else if b == '\\' {
					escaped = true
				} else if b == '"' {
					quoted = false
					if depth == 0 {
						break
					}
				}
			} else {
				if b == '"' {
					quoted = true
				} else if b == '{' || b == '[' {
					depth++
				} else if b == '}' || b == ']' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
		}
		if len(out) > limit {
			return nil, errors.New("JSON value exceeds the per-record size limit")
		}
	}
	if len(out) > limit || !json.Valid(out) {
		return nil, errors.New("Invalid or oversized JSON value")
	}
	return out, nil
}
func (s *Store) importJSON(id string, in io.Reader) error {
	r := bufio.NewReaderSize(io.LimitReader(in, 16<<30), 65536)
	b, e := jsonChar(r)
	if e != nil || b != '{' {
		return errors.New("Capture JSON must be an object")
	}
	seen := map[string]bool{}
	count, schema := 0, 0
	for {
		b, e = jsonChar(r)
		if e != nil {
			return e
		}
		if b == '}' {
			break
		}
		_ = r.UnreadByte()
		raw, e := boundedJSONValue(r, 4096)
		if e != nil {
			return e
		}
		var key string
		if json.Unmarshal(raw, &key) != nil {
			return errors.New("Invalid capture property")
		}
		if seen[key] {
			return errors.New("Duplicate capture property")
		}
		seen[key] = true
		b, e = jsonChar(r)
		if e != nil || b != ':' {
			return errors.New("Missing JSON property separator")
		}
		if key == "packets" {
			b, e = jsonChar(r)
			if e != nil || b != '[' {
				return errors.New("Capture packets must be an array")
			}
			for {
				b, e = jsonChar(r)
				if e != nil {
					return e
				}
				if b == ']' {
					break
				}
				_ = r.UnreadByte()
				raw, e = boundedJSONValue(r, maxRecord)
				if e != nil {
					return e
				}
				var fields map[string]json.RawMessage
				if json.Unmarshal(raw, &fields) != nil {
					return errors.New("Invalid packet record")
				}
				p, e := importedPacket(fields)
				if e != nil {
					return fmt.Errorf("Packet %d: %w", count+1, e)
				}
				if _, e = s.Append(id, p); e != nil {
					return e
				}
				count++
				b, e = jsonChar(r)
				if e != nil {
					return e
				}
				if b == ']' {
					break
				}
				if b != ',' {
					return errors.New("Invalid packet separator")
				}
				b, e = jsonChar(r)
				if e != nil {
					return e
				}
				if b == ']' {
					return errors.New("Trailing packet comma")
				}
				_ = r.UnreadByte()
			}
		} else {
			raw, e = boundedJSONValue(r, 1<<20)
			if e != nil {
				return e
			}
			switch key {
			case "schema":
				if json.Unmarshal(raw, &schema) != nil {
					return errors.New("Invalid schema")
				}
			case "name":
				var name string
				if json.Unmarshal(raw, &name) != nil {
					return errors.New("Invalid capture name")
				}
				if e = s.Update(CaptureUpdate{Capture: id, Name: name}); e != nil {
					return e
				}
			case "capture":
				var c Capture
				if e = json.Unmarshal(raw, &c); e != nil {
					return e
				}
				if e = s.Update(CaptureUpdate{Capture: id, Name: c.Name, Notes: c.Notes}); e != nil {
					return e
				}
				s.mu.Lock()
				a, e := s.open(id)
				if e == nil {
					a.meta.Protocol = c.Protocol
				}
				s.mu.Unlock()
				if e != nil {
					return e
				}
			case "status":
				var status map[string]json.RawMessage
				if json.Unmarshal(raw, &status) != nil {
					return errors.New("Invalid capture status")
				}
				value := scalar(status["protocol"])
				var protocol int
				_ = json.Unmarshal([]byte(value), &protocol)
				s.mu.Lock()
				a, e := s.open(id)
				if e == nil {
					a.meta.Protocol = protocol
				}
				s.mu.Unlock()
				if e != nil {
					return e
				}
			}
		}
		b, e = jsonChar(r)
		if e != nil {
			return e
		}
		if b == '}' {
			break
		}
		if b != ',' {
			return errors.New("Invalid capture property separator")
		}
		b, e = jsonChar(r)
		if e != nil {
			return e
		}
		if b == '}' {
			return errors.New("Trailing capture comma")
		}
		_ = r.UnreadByte()
	}
	if _, e = jsonChar(r); e != io.EOF {
		return errors.New("Trailing data after capture JSON")
	}
	if schema < 1 || schema > 2 || count == 0 {
		return errors.New("Unsupported capture schema or empty capture")
	}
	return nil
}
