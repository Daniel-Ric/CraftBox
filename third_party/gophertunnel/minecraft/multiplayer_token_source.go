package minecraft

import (
	"context"
	"crypto/ecdsa"

	"github.com/sandertv/gophertunnel/minecraft/service"
)

type MultiplayerTokenSource interface {
	MultiplayerToken(ctx context.Context, key *ecdsa.PublicKey) (jwt string, err error)
}

type multiplayerTokenSource struct {
	env *service.AuthorizationEnvironment

	src service.TokenSource
}

func (s *multiplayerTokenSource) MultiplayerToken(ctx context.Context, key *ecdsa.PublicKey) (string, error) {
	return s.env.MultiplayerToken(ctx, s.src, key)
}
