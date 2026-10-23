package session

import "github.com/df-mc/dragonfly/server/world"

func (s *Session) ViewLayer() *world.ViewLayer {
	return s.viewLayer
}

func (s *Session) ViewNameTag(entity world.Entity, nameTag string) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewNameTag(entity, nameTag)
}

func (s *Session) ViewPublicNameTag(entity world.Entity) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewPublicNameTag(entity)
}

func (s *Session) ViewAlwaysShowNameTag(entity world.Entity, alwaysShow bool) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewAlwaysShowNameTag(entity, alwaysShow)
}

func (s *Session) ViewPublicAlwaysShowNameTag(entity world.Entity) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewPublicAlwaysShowNameTag(entity)
}

func (s *Session) ViewScoreTag(entity world.Entity, scoreTag string) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewScoreTag(entity, scoreTag)
}

func (s *Session) ViewPublicScoreTag(entity world.Entity) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewPublicScoreTag(entity)
}

func (s *Session) ViewVisibility(entity world.Entity, level world.VisibilityLevel) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.ViewVisibility(entity, level)
}

func (s *Session) RemoveViewLayer(entity world.Entity) {
	if s.viewLayer == nil {
		return
	}
	s.viewLayer.Remove(entity)
}

func (s *Session) ViewLayerEntityChanged(e world.Entity) {
	if s.entityHidden(e) || !s.viewingEntity(e.H()) {
		return
	}
	s.ViewEntityState(e)
}

func (s *Session) viewingEntity(handle *world.EntityHandle) bool {
	s.entityMutex.RLock()
	_, ok := s.entityRuntimeIDs[handle]
	s.entityMutex.RUnlock()
	return ok
}
