package proxy

import (
	"sync"
	"sync/atomic"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var exportProgressSequence atomic.Int64

type exportFeedback struct {
	mu      sync.Mutex
	enabled bool
	visible map[int64]struct{}
	issued  map[int64]struct{}
}

type ExportProgress struct {
	session *Session
	id      int64
}

func (s *Session) ExportProgressEnabled() bool {
	f := &s.exportFeedback
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.enabled
}

func (s *Session) SetExportProgressEnabled(enabled bool) {
	f := &s.exportFeedback
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = enabled
	if !enabled {
		for id := range f.visible {
			s.hideExportProgress(id)
		}
		clear(f.visible)
	}
}

func (s *Session) BeginExportProgress() *ExportProgress {
	return &ExportProgress{session: s, id: -(1 << 60) - exportProgressSequence.Add(1)}
}

func (p *ExportProgress) Update(title string, fraction float32) {
	if p == nil {
		return
	}
	f := &p.session.exportFeedback
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.enabled || p.session.Client == nil {
		return
	}
	title = locale.Text(title)
	if _, visible := f.visible[p.id]; visible {
		_ = p.session.ClientWritePacket(&packet.BossEvent{BossEntityUniqueID: p.id, EventType: packet.BossEventTitle, BossBarTitle: title})
		_ = p.session.ClientWritePacket(&packet.BossEvent{BossEntityUniqueID: p.id, EventType: packet.BossEventHealthPercentage, HealthPercentage: fraction})
		return
	}
	event := &packet.BossEvent{BossEntityUniqueID: p.id, EventType: packet.BossEventShow, BossBarTitle: title, HealthPercentage: fraction, Colour: packet.BossEventColourBlue}
	if err := p.session.ClientWritePacket(event); err == nil {
		if f.visible == nil {
			f.visible = make(map[int64]struct{})
			f.issued = make(map[int64]struct{})
		}
		f.visible[p.id] = struct{}{}
		f.issued[p.id] = struct{}{}
	}
}

func (p *ExportProgress) Close() {
	if p == nil {
		return
	}
	f := &p.session.exportFeedback
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, visible := f.visible[p.id]; visible {
		p.session.hideExportProgress(p.id)
		delete(f.visible, p.id)
	}
}

func (s *Session) hideExportProgress(id int64) {
	_ = s.ClientWritePacket(&packet.BossEvent{BossEntityUniqueID: id, EventType: packet.BossEventHide})
}

func (s *Session) isLocalExportProgress(id int64) bool {
	f := &s.exportFeedback
	f.mu.Lock()
	defer f.mu.Unlock()
	_, issued := f.issued[id]
	return issued
}
