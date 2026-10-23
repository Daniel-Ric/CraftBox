package title

import (
	"fmt"
	"strings"
	"time"
)

type Title struct {
	text, subtitle, actionText                string
	fadeInDuration, fadeOutDuration, duration time.Duration
}

func New(text ...any) Title {
	return Title{
		text:            format(text),
		fadeInDuration:  time.Second / 20,
		fadeOutDuration: time.Second / 20,
		duration:        time.Second * 2,
	}
}

func (title Title) Text() string {
	return title.text
}

func (title Title) WithSubtitle(text ...any) Title {
	title.subtitle = format(text)
	return title
}

func (title Title) Subtitle() string {
	return title.subtitle
}

func (title Title) WithActionText(text ...any) Title {
	title.actionText = format(text)
	return title
}

func (title Title) ActionText() string {
	return title.actionText
}

func (title Title) Duration() time.Duration {
	return title.duration
}

func (title Title) WithDuration(d time.Duration) Title {
	title.duration = d
	return title
}

func (title Title) WithFadeInDuration(d time.Duration) Title {
	title.fadeInDuration = d
	return title
}

func (title Title) FadeInDuration() time.Duration {
	return title.fadeInDuration
}

func (title Title) WithFadeOutDuration(d time.Duration) Title {
	title.fadeOutDuration = d
	return title
}

func (title Title) FadeOutDuration() time.Duration {
	return title.fadeOutDuration
}

func format(a []any) string {
	return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintln(a...), "\n"), "\n")
}
