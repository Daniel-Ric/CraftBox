package proxy

import "errors"

// ConnectionFailure keeps the original cause and records which proxy leg failed.
// The labels describe a connection, not the health of the remote server.
type ConnectionFailure struct {
	Leg        string `json:"leg"`
	Target     string `json:"target"`
	LocalReady bool   `json:"localReady"`
	Cause      error  `json:"-"`
}

func (e *ConnectionFailure) Error() string { return e.Cause.Error() }
func (e *ConnectionFailure) Unwrap() error { return e.Cause }

func (s *Session) connectionFailure(err error, leg string) error {
	if err == nil {
		return nil
	}
	var existing *ConnectionFailure
	if errors.As(err, &existing) {
		return err
	}
	info := &ConnectionFailure{Leg: leg, Cause: err}
	if s.connectInfo != nil {
		info.Target = s.connectInfo.Value
	}
	select {
	case <-s.haveClientData:
		info.LocalReady = true
	default:
	}
	return info
}
