package discordpresence

import (
	"sync"
	"time"
)

type Status struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
	Activity  string `json:"activity"`
	StartedAt int64  `json:"startedAt"`
}

type Controller struct {
	mu       sync.Mutex
	settings Settings
	status   Status
	wake     chan struct{}
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
	activity func() string
	started  time.Time
}

func NewController(activity func() string) *Controller {
	c := &Controller{wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), activity: activity, started: time.Now(), status: Status{Message: "Disabled"}}
	go c.run()
	return c
}

func (c *Controller) Configure(s Settings) {
	c.mu.Lock()
	c.settings = s
	c.mu.Unlock()
	c.Retry()
}
func (c *Controller) Retry() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}
func (c *Controller) Status() Status {
	c.mu.Lock()
	s := c.status
	c.mu.Unlock()
	s.Activity = c.activity()
	s.StartedAt = c.started.Unix()
	return s
}
func (c *Controller) setStatus(connected bool, message string) {
	c.mu.Lock()
	c.status = Status{Connected: connected, Message: message}
	c.mu.Unlock()
}
func (c *Controller) Close() { c.once.Do(func() { close(c.stop) }); <-c.done }

func (c *Controller) run() {
	defer close(c.done)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	var rpc *client
	disconnect := func() {
		if rpc != nil {
			_ = rpc.SetActivity(nil)
			_ = rpc.conn.Close()
			rpc = nil
		}
	}
	defer disconnect()
	for {
		select {
		case <-c.stop:
			return
		case <-c.wake:
		case <-ticker.C:
		}
		c.mu.Lock()
		s := c.settings
		c.mu.Unlock()
		if !s.Enabled {
			disconnect()
			c.setStatus(false, "Disabled")
			continue
		}
		if err := s.Validate(); err != nil {
			disconnect()
			c.setStatus(false, err.Error())
			continue
		}
		if rpc == nil {
			conn, err := dialIPC()
			if err != nil {
				c.setStatus(false, err.Error())
				continue
			}
			rpc, err = handshake(conn)
			if err != nil {
				_ = conn.Close()
				c.setStatus(false, err.Error())
				continue
			}
		}
		a := s.Activity(c.activity(), c.started)
		if err := rpc.SetActivity(&a); err != nil {
			_ = rpc.conn.Close()
			rpc = nil
			c.setStatus(false, err.Error())
			continue
		}
		c.setStatus(true, "Connected to Discord")
	}
}
