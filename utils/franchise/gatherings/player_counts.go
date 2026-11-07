package gatherings

import (
	"context"
	"errors"
	"github.com/google/uuid"
	native "github.com/sandertv/gophertunnel/minecraft/service/gatherings"
	"maps"
	"sync"
	"time"
)

// Shared by source selection and Marketplace through the account's Service.
// No tokens or player metadata are persisted to disk.
type playerCountCache struct {
	mu        sync.Mutex
	pending   chan struct{}
	counts    map[uuid.UUID]int
	err       error
	nextFetch time.Time
}

func (c *playerCountCache) get(ctx context.Context, fetch func() (map[uuid.UUID]int, error)) (map[uuid.UUID]int, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		if time.Now().Before(c.nextFetch) {
			result, err := maps.Clone(c.counts), c.err
			c.mu.Unlock()
			return result, err
		}
		if pending := c.pending; pending != nil {
			c.mu.Unlock()
			select {
			case <-pending:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		c.pending = make(chan struct{})
		c.mu.Unlock()
		counts, err := fetch()
		wait := time.Minute
		var limited *native.PlayerCountRateLimitError
		if errors.As(err, &limited) && limited.RetryAfter > wait {
			wait = limited.RetryAfter
		}
		// A canceled caller must not poison other callers' next attempt.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			wait = 0
		}
		c.mu.Lock()
		c.counts = maps.Clone(counts)
		c.err = err
		c.nextFetch = time.Now().Add(wait)
		close(c.pending)
		c.pending = nil
		c.mu.Unlock()
		return maps.Clone(counts), err
	}
}
