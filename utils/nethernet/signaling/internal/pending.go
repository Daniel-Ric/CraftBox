package internal

import (
	"sync"

	"github.com/google/uuid"
)


func NewPendingMap() *PendingMap {
	return &PendingMap{expected: make(map[uuid.UUID]chan error)}
}




type PendingMap struct {
	expected map[uuid.UUID]chan error
	mu       sync.Mutex
}



func (i *PendingMap) Add(id uuid.UUID) <-chan error {
	ch := make(chan error, 1)
	i.mu.Lock()
	i.expected[id] = ch
	i.mu.Unlock()
	return ch
}





func (i *PendingMap) Remove(id uuid.UUID) {
	i.mu.Lock()
	ch, ok := i.expected[id]
	delete(i.expected, id)
	i.mu.Unlock()
	if ok {
		close(ch)
	}
}




func (i *PendingMap) Done(id uuid.UUID, err error) bool {
	i.mu.Lock()
	ch, ok := i.expected[id]
	delete(i.expected, id)
	i.mu.Unlock()
	if ok {
		ch <- err
		close(ch)
	}
	return ok
}
