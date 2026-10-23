package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/df-mc/go-playfab/v2"
)

func (e *AuthorizationEnvironment) TokenSource(client *playfab.Client, config TokenConfig) TokenSource {
	defaultUserConfig(&config.User)
	defaultDeviceConfig(e, &config.Device)

	return &tokenSource{
		client: client,
		env:    e,
		config: config,
	}
}

type tokenSource struct {
	client *playfab.Client
	env    *AuthorizationEnvironment
	config TokenConfig

	token *Token
	mu    sync.Mutex
}

func (s *tokenSource) ServiceToken(ctx context.Context) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != nil && s.token.Valid() {
		return s.token, nil
	}

	ticket, err := s.client.SessionTicket(ctx)
	if err != nil {
		return nil, fmt.Errorf("request session ticket: %w", err)
	}
	s.config.User.TokenType = TokenTypePlayFab
	s.config.User.Token = ticket

	if tokenRenewable(s.token) {
		if token, err := s.env.Renew(ctx, s.token, s.config.User); err == nil {
			s.token = token
			return s.token, nil
		}
	}

	token, err := s.env.Token(ctx, s.config)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	s.token = token
	return s.token, nil
}

func tokenRenewable(tok *Token) bool {
	return tok != nil && tok.AuthorizationHeader != "" && time.Now().Before(tok.ValidUntil)
}
