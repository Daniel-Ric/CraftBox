package internal

import (
	"log/slog"
	"sync"

	"github.com/df-mc/go-nethernet"
)




func NewNotifier(log *slog.Logger) *Notifier {
	return &Notifier{
		notifiers: make(map[nethernet.Notifier]struct{}),
		log:       log,
	}
}



type Notifier struct {
	notifiers map[nethernet.Notifier]struct{}
	mu        sync.RWMutex
	log       *slog.Logger
}




func (n *Notifier) Register(notifier nethernet.Notifier) (stop func()) {
	n.mu.Lock()
	n.notifiers[notifier] = struct{}{}
	n.mu.Unlock()

	return func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.stop(notifier)
	}
}




func (n *Notifier) Signal(signal *nethernet.Signal) {
	n.mu.RLock()
	for notifier := range n.notifiers {
		notifier.NotifySignal(signal)
	}
	n.mu.RUnlock()
}



func (n *Notifier) stop(notifier nethernet.Notifier) {
	delete(n.notifiers, notifier)
}


func (n *Notifier) Close() error {
	n.mu.Lock()
	for notifier := range n.notifiers {
		n.stop(notifier)
	}
	n.mu.Unlock()

	return nil
}
