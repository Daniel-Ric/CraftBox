package chat

import (
	"github.com/google/uuid"
	"sync"
)

var Global = New()

type Chat struct {
	m           sync.Mutex
	subscribers map[uuid.UUID]Subscriber
}

func New() *Chat {
	return &Chat{subscribers: map[uuid.UUID]Subscriber{}}
}

func (chat *Chat) Write(p []byte) (n int, err error) {
	return chat.WriteString(string(p))
}

func (chat *Chat) WriteString(s string) (n int, err error) {
	chat.m.Lock()
	defer chat.m.Unlock()
	for _, subscriber := range chat.subscribers {
		subscriber.Message(s)
	}
	return len(s), nil
}

func (chat *Chat) Writet(t Translation, a ...any) {
	chat.m.Lock()
	defer chat.m.Unlock()
	for _, subscriber := range chat.subscribers {
		if translator, ok := subscriber.(Translator); ok {
			translator.Messaget(t, a...)
			continue
		}
		subscriber.Message(t.F(a...).String())
	}
}

func (chat *Chat) Subscribe(s Subscriber) {
	chat.m.Lock()
	defer chat.m.Unlock()
	chat.subscribers[s.UUID()] = s
}

func (chat *Chat) Subscribed(s Subscriber) bool {
	chat.m.Lock()
	defer chat.m.Unlock()
	_, ok := chat.subscribers[s.UUID()]
	return ok
}

func (chat *Chat) Unsubscribe(s Subscriber) {
	chat.m.Lock()
	defer chat.m.Unlock()
	delete(chat.subscribers, s.UUID())
}

func (chat *Chat) Close() error {
	chat.m.Lock()
	chat.subscribers = nil
	chat.m.Unlock()
	return nil
}
