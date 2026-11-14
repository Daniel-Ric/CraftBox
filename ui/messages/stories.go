package messages

import "context"

type StoriesAnswer uint8

const (
	StoriesCancel StoriesAnswer = iota
	StoriesOptIn
	StoriesOptOut
)

type EventStoriesPrompt struct {
	Context   context.Context
	RealmName string
	Reply     chan<- StoriesAnswer
}

func AskStoriesPrompt(ctx context.Context, realmName string) StoriesAnswer {
	reply := make(chan StoriesAnswer, 1)
	SendEvent(&EventStoriesPrompt{Context: ctx, RealmName: realmName, Reply: reply})
	select {
	case a := <-reply:
		return a
	case <-ctx.Done():
		return StoriesCancel
	}
}
