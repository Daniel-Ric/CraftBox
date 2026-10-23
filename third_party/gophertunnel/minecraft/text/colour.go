package text

import (
	"fmt"
	"golang.org/x/net/html"
	"regexp"
	"strings"
)

var cleaner = regexp.MustCompile("§[0-9a-v]")

func Clean(s string) string {
	return cleaner.ReplaceAllString(s, "")
}

func ANSI(a ...any) string {
	str := make([]string, len(a))
	for i, v := range a {
		str[i] = minecraftReplacer.Replace(fmt.Sprint(v))
	}
	return strings.Join(str, " ")
}

func Colourf(format string, a ...any) string {
	str := fmt.Sprintf(format, a...)

	e := &enc{w: &strings.Builder{}, first: true}
	t := html.NewTokenizer(strings.NewReader(str))
	for t.Next() != html.ErrorToken {
		e.process(t.Token())
	}
	return e.w.String()
}

type enc struct {
	w           *strings.Builder
	formatStack []string
	first       bool
}

func (e *enc) process(tok html.Token) {
	if e.first {
		e.w.WriteString(Reset)
		e.first = false
	}
	switch tok.Type {
	case html.TextToken:
		e.writeText(tok.Data)
	case html.StartTagToken:
		if format, ok := strMap[tok.Data]; ok {
			e.formatStack = append(e.formatStack, format)
			return
		}

		e.writeText("<" + tok.Data + ">")
	case html.EndTagToken:
		for i, format := range e.formatStack {
			if f, ok := strMap[tok.Data]; ok && f == format {
				e.formatStack = append(e.formatStack[:i], e.formatStack[i+1:]...)
				return
			}
		}

		e.writeText("</" + tok.Data + ">")
	}
}

func (e *enc) writeText(s string) {
	for _, format := range e.formatStack {
		e.w.WriteString(format)
	}
	e.w.WriteString(s)
	if len(e.formatStack) != 0 {
		e.w.WriteString(Reset)
	}
}
