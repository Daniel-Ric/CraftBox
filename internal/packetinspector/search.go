package packetinspector

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var boundedNumber = regexp.MustCompile(`^[+-]?[0-9]{1,100}(?:\.[0-9]{1,100})?(?:[eE][+-]?[0-9]{1,2})?$`)
var fieldPredicate = regexp.MustCompile(`^(.+?)(>=|<=|!=|=|>|<)(.*)$`)

func searchMatch(p Packet, search string) bool {
	safe := Redact(p)
	body := readablePayload(safe)
	hay := strings.ToLower(safe.Name + " " + body + " " + safe.Note + " " + safe.Reason + " " + safe.SourceAddress + " " + safe.DestinationAddress)
	var decoded any
	parsed := false
	lookup := func(key string) (any, bool) {
		if !parsed {
			d := json.NewDecoder(strings.NewReader(body))
			d.UseNumber()
			_ = d.Decode(&decoded)
			parsed = true
		}
		return fieldAt(decoded, key)
	}
	matches := func(token string) bool {
		parts := strings.SplitN(token, ":", 2)
		value := strings.Trim(token, "\"'")
		if len(parts) == 2 {
			value = strings.Trim(parts[1], "\"'")
			lower := strings.ToLower(value)
			switch strings.ToLower(parts[0]) {
			case "type", "name":
				if strings.ContainsAny(lower, "*?") {
					ok, _ := path.Match(lower, strings.ToLower(p.Name))
					return ok
				}
				return strings.Contains(strings.ToLower(p.Name), lower)
			case "id":
				n, e := strconv.ParseUint(value, 0, 32)
				return e == nil && p.PacketID == uint32(n)
			case "dir", "direction":
				if lower == "in" {
					lower = "clientbound"
				}
				if lower == "out" {
					lower = "serverbound"
				}
				return strings.HasPrefix(p.Direction, lower)
			case "stage":
				return strings.Contains(p.Stage, lower)
			case "action":
				return strings.Contains(p.Action, lower)
			case "is":
				switch lower {
				case "problem", "partial", "truncated":
					return p.Partial || p.Stage == "gap"
				case "bookmarked":
					return p.Bookmarked
				case "modified":
					return p.Action == "edit"
				case "blocked":
					return p.Action == "block"
				case "redacted":
					return safe.Redacted
				}
				return false
			case "hex", "raw":
				b, e := hex.DecodeString(strings.ReplaceAll(lower, " ", ""))
				return e == nil && len(b) > 0 && bytes.Contains(safe.Raw, b)
			case "bytes", "size":
				return compareSearch(json.Number(strconv.Itoa(p.Bytes)), value)
			case "has":
				v, ok := lookup(value)
				return ok && v != nil
			case "field":
				if m := fieldPredicate.FindStringSubmatch(value); m != nil {
					v, ok := lookup(m[1])
					return ok && compareSearch(v, m[2]+strings.Trim(m[3], "\"'"))
				}
				_, ok := lookup(value)
				return ok
			case "actor", "entity":
				for _, key := range []string{"EntityRuntimeID", "EntityUniqueID", "ActorRuntimeID", "ActorUniqueID", "TargetEntityRuntimeID", "RuntimeEntityID"} {
					if v, ok := lookup(key); ok && compareSearch(v, "="+value) {
						return true
					}
				}
				return false
			case "source":
				return strings.Contains(strings.ToLower(p.SourceAddress), lower)
			case "destination":
				return strings.Contains(strings.ToLower(p.DestinationAddress), lower)
			default:
				return strings.Contains(hay, strings.ToLower(token))
			}
		}
		if strings.HasPrefix(value, "size>") || strings.HasPrefix(value, "size<") {
			return compareSearch(json.Number(strconv.Itoa(p.Bytes)), strings.TrimPrefix(value, "size"))
		}
		return strings.Contains(hay, strings.ToLower(value))
	}
	// OR separates conjunctions; quoted pipe characters stay inside a token.
	group, anyToken := true, false
	for _, token := range searchParts.FindAllString(search, -1) {
		if token == "|" {
			if group && anyToken {
				return true
			}
			group = true
			anyToken = false
			continue
		}
		neg := strings.HasPrefix(token, "-")
		token = strings.TrimPrefix(token, "-")
		m := matches(token)
		if neg {
			m = !m
		}
		group = group && m
		anyToken = true
	}
	return group && anyToken
}
func fieldAt(value any, key string) (any, bool) {
	key = strings.ReplaceAll(strings.ReplaceAll(key, "[", "."), "]", "")
	for _, part := range strings.Split(key, ".") {
		switch x := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = x[part]
			if !ok {
				for k, v := range x {
					if strings.EqualFold(k, part) {
						value = v
						ok = true
						break
					}
				}
			}
			if !ok {
				return nil, false
			}
		case []any:
			n, e := strconv.Atoi(part)
			if e != nil || n < 0 || n >= len(x) {
				return nil, false
			}
			value = x[n]
		default:
			return nil, false
		}
	}
	return value, true
}
func compareSearch(v any, expression string) bool {
	op := "="
	for _, prefix := range []string{">=", "<=", "!=", ">", "<", "="} {
		if strings.HasPrefix(expression, prefix) {
			op = prefix
			expression = strings.TrimPrefix(expression, prefix)
			break
		}
	}
	var text string
	switch x := v.(type) {
	case string:
		text = x
	case json.Number:
		text = string(x)
	default:
		b, _ := json.Marshal(v)
		text = string(b)
	}
	cmp := strings.Compare(text, expression)
	// Rational comparison preserves full 64-bit IDs rather than rounding floats.
	if _, number := v.(json.Number); number && boundedNumber.MatchString(text) && boundedNumber.MatchString(expression) {
		a, okA := new(big.Rat).SetString(text)
		b, okB := new(big.Rat).SetString(expression)
		if okA && okB {
			cmp = a.Cmp(b)
		}
	}
	switch op {
	case "=":
		return cmp == 0
	case "!=":
		return cmp != 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	}
	return false
}
