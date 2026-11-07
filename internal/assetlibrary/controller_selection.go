package assetlibrary

import (
	"regexp"
	"strconv"
	"strings"
)

func normalizeControllerIndex(expression string) string {
	s := unwrapControllerExpression(expression)
	var out strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\'' || c == '"' {
			quote := c
			out.WriteByte(c)
			i++
			for i < len(s) {
				c = s[i]
				out.WriteByte(c)
				i++
				if c == '\\' && i < len(s) {
					out.WriteByte(s[i])
					i++
					continue
				}
				if c == quote {
					break
				}
			}
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' {
			start := i
			i++
			for i < len(s) {
				c = s[i]
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.') {
					break
				}
				i++
			}
			token := strings.ToLower(s[start:i])
			for short, long := range map[string]string{"q.": "query.", "v.": "variable.", "c.": "context."} {
				if strings.HasPrefix(token, short) {
					token = long + token[len(short):]
					break
				}
			}
			out.WriteString(token)
			continue
		}
		out.WriteByte(c)
		i++
	}
	return out.String()
}

func unwrapControllerExpression(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		depth := 0
		whole := true
		for i, c := range s {
			if c == '(' {
				depth++
			}
			if c == ')' {
				depth--
			}
			if depth == 0 && i < len(s)-1 {
				whole = false
				break
			}
		}
		if !whole {
			break
		}
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// Split a top-level ternary without confusing nested expressions or quoted
// property names with the resource-separating colon.
func controllerConditional(expression string) (condition, yes, no string, ok bool) {
	s := unwrapControllerExpression(expression)
	depth := 0
	quote := byte(0)
	question := -1
	nested := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case '?':
			if depth == 0 {
				if question < 0 {
					question = i
				} else {
					nested++
				}
			}
		case ':':
			if depth == 0 && question >= 0 {
				if nested > 0 {
					nested--
				} else {
					return strings.TrimSpace(s[:question]), unwrapControllerExpression(s[question+1 : i]), unwrapControllerExpression(s[i+1:]), true
				}
			}
		}
	}
	return "", "", "", false
}

var selectionQuery = regexp.MustCompile(`^(?:q|query)\.(?:is_[a-z0-9_]+|variant|mark_variant|skin_id|color)$`)
var selectionComparison = regexp.MustCompile(`^((?:q|query)\.[a-z0-9_]+)(==|!=)(-?[0-9]+(?:\.[0-9]+)?)$`)

func controllerSelectionQueries(expression string, index int) map[string]float64 {
	if index < 0 {
		return nil
	}
	s := normalizeControllerIndex(expression)
	value := float64(index)
	for strings.HasPrefix(s, "!") && !strings.HasPrefix(s, "!=") {
		s = unwrapControllerExpression(s[1:])
		if value == 0 {
			value = 1
		} else {
			value = 0
		}
	}
	if m := selectionComparison.FindStringSubmatch(s); m != nil {
		n, _ := strconv.ParseFloat(m[3], 64)
		equal := value != 0
		if m[2] == "!=" {
			equal = !equal
		}
		value = n
		if !equal {
			value = 0
			if n == 0 {
				value = 1
			}
		}
		s = m[1]
	}
	if !selectionQuery.MatchString(s) {
		return nil
	}
	s = strings.Replace(s, "q.", "query.", 1)
	return map[string]float64{s: value}
}

func mergePreviewQueries(target map[string]float64, source map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for k, v := range target {
		out[k] = v
	}
	for k, v := range source {
		out[k] = v
	}
	return out
}
