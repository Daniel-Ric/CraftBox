package auth

import (
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

type tokenInfo struct {
	*oauth2.Token
	ClientID string
}

func (t *tokenInfo) LiveToken() *oauth2.Token {
	return t.Token
}

func (t *tokenInfo) AuthConfig() *auth.Config {
	switch t.ClientID {
	case auth.AndroidConfig.ClientID:
		return &auth.AndroidConfig
	case auth.IOSConfig.ClientID:
		return &auth.IOSConfig
	case auth.NintendoConfig.ClientID:
		return &auth.NintendoConfig
	case auth.PlayStationConfig.ClientID:
		return &auth.PlayStationConfig
	case "":
		return &auth.AndroidConfig
	}
	return nil
}
