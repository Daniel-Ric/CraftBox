package diagnostics

import (
	"errors"
)

type StartupRecovery struct {
	SessionIDs []string `json:"sessionIds"`
	SafeStart  bool     `json:"safeStart"`
	Confirmed  bool     `json:"confirmed"`
	MapOnly    bool     `json:"mapOnly"`
}

func isUnexpected(s Session) bool {
	return !s.Active && (s.Status == "crashed" || s.Status == "interrupted") &&
		(s.Role == "desktop" || s.Role == "main" || s.Role == "map-window")
}

func startupRecovery(sessions []Session, currentID string) StartupRecovery {
	result := StartupRecovery{SessionIDs: []string{}}
	var previous Session
	failures := 0
	var currentStart string
	for _, s := range sessions {
		if s.ID == currentID {
			currentStart = s.Started
			break
		}
	}
	for _, s := range sessions {
		if s.ID == currentID || currentStart != "" && s.Started >= currentStart ||
			(s.Role != "desktop" && s.Role != "main") {
			continue
		}
		if previous.ID == "" {
			previous = s
		}
		if !isUnexpected(s) || !s.StartupPending {
			break
		}
		failures++
		if failures == 2 {
			break
		}
	}
	result.SafeStart = failures >= 2
	if isUnexpected(previous) && previous.CrashNotice == "" {
		result.SessionIDs = append(result.SessionIDs, previous.ID)
		result.Confirmed = previous.Status == "crashed"
	}
	result.MapOnly = len(result.SessionIDs) == 0
	if previous.ID != "" && !previous.Active {
		for _, s := range sessions {
			if s.Role != "map-window" || !isUnexpected(s) || s.CrashNotice != "" {
				continue
			}
			linked := s.ParentSession == previous.ID
			legacy := s.ParentSession == "" && s.Started >= previous.Started &&
				(currentStart == "" || s.Started < currentStart)
			if linked || legacy {
				result.SessionIDs = append(result.SessionIDs, s.ID)
				if result.MapOnly && s.Status == "crashed" {
					result.Confirmed = true
				}
			}
			if len(result.SessionIDs) == 4 {
				break
			}
		}
	}
	result.MapOnly = result.MapOnly && len(result.SessionIDs) > 0
	return result
}

func (s *Store) StartupRecovery(currentID string) (StartupRecovery, error) {
	sessions, err := s.sessions()
	if err != nil {
		return StartupRecovery{}, err
	}
	return startupRecovery(sessions, currentID), nil
}

func (s *Store) SetCrashNotice(ids []string, notice string) error {
	if len(ids) == 0 || len(ids) > 4 || (notice != "later" && notice != "dismissed") {
		return errors.New("invalid crash notice")
	}
	lock, err := s.storeLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	sessions, err := s.reportSessions(ids)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.CrashNotice == "dismissed" || session.CrashNotice == notice {
			continue
		}
		root, err := s.root.OpenRoot(session.ID)
		if err != nil {
			return err
		}
		session.CrashNotice = notice
		err = saveSessionMeta(root, session)
		root.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) reportSessions(ids []string) ([]Session, error) {
	if len(ids) == 0 || len(ids) > 4 {
		return nil, errors.New("select up to four interrupted sessions")
	}
	sessions, err := s.sessions()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Session, len(sessions))
	for _, session := range sessions {
		if isUnexpected(session) {
			byID[session.ID] = session
		}
	}
	result := make([]Session, 0, len(ids))
	for _, id := range ids {
		session, ok := byID[id]
		if !ok {
			return nil, errors.New("a selected crash session is no longer available")
		}
		result = append(result, session)
		delete(byID, id)
	}
	return result, nil
}

func validDesktopWorkspace(workspace string) bool {
	switch workspace {
	case "worlds", "packs", "skins", "packets", "marketplace", "bedrock", "structures", "render", "cinematic", "home":
		return true
	default:
		return false
	}
}

func (m *Manager) SetDesktopWorkspace(workspace string, ready bool) error {
	if !validDesktopWorkspace(workspace) {
		return errors.New("invalid desktop workspace")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.session.Role != "desktop" {
		return nil
	}
	if m.session.Workspace == workspace && (!ready || !m.session.StartupPending) {
		return nil
	}
	previous := m.session
	m.session.Workspace = workspace
	if ready {
		m.session.StartupPending = false
	}
	if err := m.saveMeta(); err != nil {
		m.session = previous
		return err
	}
	return nil
}
