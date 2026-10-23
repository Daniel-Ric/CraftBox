package scene

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type conditionReader struct {
	tokens []string
	at     int
	states map[string]any
	bad    bool
}

func blockCondition(expression string, states map[string]any) (bool, bool) {
	v, ok := staticExpression(expression, states)
	return truth(v), ok
}

func staticExpression(expression string, states map[string]any) (any, bool) {
	if len(expression) > 4096 {
		return false, false
	}
	p := conditionReader{states: states}
	for i := 0; i < len(expression); {
		ch := expression[i]
		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}
		start := i
		if ch == '\'' || ch == '"' {
			i++
			for i < len(expression) && expression[i] != ch {
				if expression[i] == '\\' {
					i++
				}
				i++
			}
			if i >= len(expression) {
				return false, false
			}
			i++
		} else if unicode.IsLetter(rune(ch)) || ch == '_' {
			i++
			for i < len(expression) && (unicode.IsLetter(rune(expression[i])) || unicode.IsDigit(rune(expression[i])) || strings.ContainsRune("._", rune(expression[i]))) {
				i++
			}
		} else if ch >= '0' && ch <= '9' || ch == '.' {
			i++
			for i < len(expression) && (expression[i] >= '0' && expression[i] <= '9' || expression[i] == '.') {
				i++
			}
		} else {
			i++
			if i < len(expression) && strings.Contains(" == != <= >= && || ", " "+expression[start:i+1]+" ") {
				i++
			}
		}
		p.tokens = append(p.tokens, expression[start:i])
		if len(p.tokens) > 512 {
			return false, false
		}
	}
	value := p.expression(0)
	return value, !p.bad && p.at == len(p.tokens)
}
func truth(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != ""
	case nil:
		return false
	default:
		return numeric(v) != 0
	}
}
func numeric(v any) float64 {
	switch x := v.(type) {
	case bool:
		if x {
			return 1
		}
		return 0
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case uint8:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	}
	return 0
}
func (p *conditionReader) peek() string {
	if p.at >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.at]
}
func (p *conditionReader) take() string {
	s := p.peek()
	if s == "" {
		p.bad = true
	} else {
		p.at++
	}
	return s
}
func precedence(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case "==", "!=", "<", ">", "<=", ">=":
		return 3
	case "+", "-":
		return 4
	case "*", "/", "%":
		return 5
	}
	return -1
}
func (p *conditionReader) expression(min int) any {
	left := p.atom()
	for !p.bad {
		op := p.peek()
		priority := precedence(op)
		if priority < min {
			break
		}
		p.take()
		right := p.expression(priority + 1)
		a, b := numeric(left), numeric(right)
		switch op {
		case "||":
			left = truth(left) || truth(right)
		case "&&":
			left = truth(left) && truth(right)
		case "==", "!=":
			equal := false
			if _, ok := left.(string); ok {
				equal = fmt.Sprint(left) == fmt.Sprint(right)
			} else if _, ok := right.(string); ok {
				equal = false
			} else {
				equal = a == b
			}
			if op == "!=" {
				equal = !equal
			}
			left = equal
		case "<":
			left = a < b
		case ">":
			left = a > b
		case "<=":
			left = a <= b
		case ">=":
			left = a >= b
		case "+":
			left = a + b
		case "-":
			left = a - b
		case "*":
			left = a * b
		case "/":
			if b == 0 {
				p.bad = true
			} else {
				left = a / b
			}
		case "%":
			if b == 0 {
				p.bad = true
			} else {
				left = math.Mod(a, b)
			}
		}
	}
	if min == 0 && p.peek() == "?" {
		p.take()
		yes := p.expression(0)
		if p.take() != ":" {
			p.bad = true
		}
		no := p.expression(0)
		if truth(left) {
			left = yes
		} else {
			left = no
		}
	}
	return left
}
func (p *conditionReader) atom() any {
	token := p.take()
	switch token {
	case "!":
		return !truth(p.atom())
	case "-":
		return -numeric(p.atom())
	case "+":
		return numeric(p.atom())
	case "(":
		v := p.expression(0)
		if p.take() != ")" {
			p.bad = true
		}
		return v
	case "true":
		return true
	case "false":
		return false
	case "query.block_state", "q.block_state", "query.block_property", "q.block_property", "query.property", "q.property":
		if p.take() != "(" {
			p.bad = true
		}
		name := p.take()
		if len(name) < 2 || (name[0] != '\'' && name[0] != '"') {
			p.bad = true
			return nil
		}
		if p.take() != ")" {
			p.bad = true
		}
		v, ok := p.states[name[1:len(name)-1]]
		if !ok {
			p.bad = true
		}
		return v
	}
	if v, ok := p.states[token]; ok {
		return v
	}
	if len(token) >= 2 && (token[0] == '\'' || token[0] == '"') {
		return token[1 : len(token)-1]
	}
	n, err := strconv.ParseFloat(token, 64)
	if err != nil {
		p.bad = true
	}
	return n
}
