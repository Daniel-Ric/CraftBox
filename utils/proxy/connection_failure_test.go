package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
)

func TestConnectionFailurePreservesEndpointAndCause(t *testing.T) {
	ready := make(chan struct{})
	close(ready)
	s := &Session{haveClientData: ready, connectInfo: &connectinfo.ConnectInfo{Value: "https://zerix.ml:19132"}}
	for _, leg := range []string{"client", "server"} {
		err := s.connectionFailure(context.DeadlineExceeded, leg)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("original cause lost")
		}
		var failure *ConnectionFailure
		if !errors.As(err, &failure) || failure.Leg != leg || !failure.LocalReady || failure.Target != "https://zerix.ml:19132" {
			t.Fatalf("missing connection context: %+v", failure)
		}
		if err.Error() != context.DeadlineExceeded.Error() {
			t.Fatal("technical error changed")
		}
		if s.connectionFailure(err, "other") != err {
			t.Fatal("existing context overwritten")
		}
	}
	if s.connectionFailure(nil, "client") != nil {
		t.Fatal("nil became an error")
	}
}

func TestConnectionFailureDoesNotClaimUnverifiedLocalConnection(t *testing.T) {
	err := (&Session{}).connectionFailure(errors.New("dial failed"), "server")
	if err.(*ConnectionFailure).LocalReady {
		t.Fatal("local connection incorrectly marked connected")
	}
}
