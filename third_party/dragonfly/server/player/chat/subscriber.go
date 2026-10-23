package chat

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"strings"
)

type Subscriber interface {
	UUID() uuid.UUID

	Message(a ...any)
}

type Translator interface {
	Messaget(t Translation, a ...any)
}

type StdoutSubscriber struct{}

var id = uuid.New()

func (c StdoutSubscriber) UUID() uuid.UUID {
	return id
}

func (c StdoutSubscriber) Message(a ...any) {
	s := make([]string, len(a))
	for i, b := range a {
		s[i] = fmt.Sprint(b)
	}
	t := text.ANSI(strings.Join(s, " "))
	if !strings.HasSuffix(t, "\n") {
		fmt.Println(t)
		return
	}
	fmt.Print(t)
}
