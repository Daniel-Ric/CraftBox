package packetinspector

import (
	"fmt"
	"time"
)

type triggerHistory struct {
	session string
	packet  Packet
	cost    int64
}
type triggerWindow struct {
	session, capture string
	until            int64
}

func (c *Controller) triggerPacket(q queuedPacket, p Packet, d Decision) {
	if p.Stage != "received" {
		return
	}
	c.triggerMu.Lock()
	defer c.triggerMu.Unlock()
	defer func() { c.windowCount.Store(int32(len(c.windows))) }()
	c.expireTriggersLocked(p.Time)
	kept := c.windows[:0]
	for _, window := range c.windows {
		if window.session == q.capture {
			if _, e := c.Store.Append(window.capture, p); e != nil {
				c.Report("Trigger recording: " + e.Error())
				_ = c.Store.Finish(window.capture)
				continue
			}
		}
		kept = append(kept, window)
	}
	c.windows = kept
	if d.Action == "trigger" && len(c.windows) < 8 {
		key := q.capture + ":" + d.Rule
		if p.Time >= c.triggerCooldown[key] {
			c.windowCount.Add(1)
			name := d.Rule
			for _, rule := range c.Engine.Rules() {
				if rule.ID == d.Rule && rule.Label != "" {
					name = rule.Label
					break
				}
			}
			capture, e := c.Store.Create("Trigger: "+name, "NetherNet trigger", currentProtocol())
			if e != nil {
				c.Report(e.Error())
			} else {
				c.triggerCooldown[key] = p.Time + int64(max(d.CooldownMS, 1000))
				for _, entry := range c.history {
					if entry.session == q.capture && entry.packet.Time >= p.Time-int64(d.PreMS) {
						if _, e = c.Store.Append(capture.ID, entry.packet); e != nil {
							c.Report(e.Error())
							break
						}
					}
				}
				marked := p
				marked.Bookmarked = true
				marked.Note = "Trigger " + name
				entry, e := c.Store.Append(capture.ID, marked)
				if e == nil {
					_ = c.Store.Update(CaptureUpdate{Capture: capture.ID, Packet: entry.ID, Annotation: Annotation{Note: marked.Note, Bookmarked: true}})
				} else {
					c.Report(e.Error())
				}
				if d.PostMS > 0 && e == nil {
					c.windows = append(c.windows, triggerWindow{session: q.capture, capture: capture.ID, until: p.Time + int64(d.PostMS)})
				} else {
					_ = c.Store.Finish(capture.ID)
				}
			}
		}
	}
	cost := int64(len(p.Raw) + len(p.Payload) + 512)
	if cost <= 16<<20 {
		c.history = append(c.history, triggerHistory{session: q.capture, packet: p, cost: cost})
		c.historyBytes += cost
	}
	for len(c.history) > 0 && (c.historyBytes > 16<<20 || p.Time-c.history[0].packet.Time > 30000) {
		c.historyBytes -= c.history[0].cost
		c.history[0] = triggerHistory{}
		c.history = c.history[1:]
	}
}
func (c *Controller) expireTriggersLocked(at int64) {
	kept := c.windows[:0]
	for _, window := range c.windows {
		if at >= window.until {
			if e := c.Store.Finish(window.capture); e != nil {
				c.Report(fmt.Sprintf("Trigger finalization: %v", e))
			}
		} else {
			kept = append(kept, window)
		}
	}
	c.windows = kept
	c.windowCount.Store(int32(len(c.windows)))
}
func (c *Controller) expireTriggers() {
	if c.queued.Load() > 0 {
		return
	}
	c.triggerMu.Lock()
	defer c.triggerMu.Unlock()
	c.expireTriggersLocked(time.Now().UnixMilli())
}
func (c *Controller) finishTriggers(session string) {
	c.triggerMu.Lock()
	defer c.triggerMu.Unlock()
	kept := c.windows[:0]
	for _, window := range c.windows {
		if session == "" || window.session == session {
			_ = c.Store.Finish(window.capture)
		} else {
			kept = append(kept, window)
		}
	}
	c.windows = kept
	c.history = nil
	c.historyBytes = 0
	c.triggerCooldown = map[string]int64{}
	c.windowCount.Store(int32(len(c.windows)))
}
