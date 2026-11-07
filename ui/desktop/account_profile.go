package desktop

import (
	"context"
	"errors"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/df-mc/go-xsapi/v2/xal/xsts"
)

type AccountProfile struct {
	Gamertag        string `json:"gamertag"`
	ClassicGamertag string `json:"classicGamertag,omitempty"`
	XUID            string `json:"xuid"`
}

func accountProfileFromClaims(info xsts.UserInfo) AccountProfile {
	name := info.UniqueModernGamerTag
	if name == "" && info.ModernGamerTag != "" {
		name = info.ModernGamerTag
		if info.ModernGamerTagSuffix != "" {
			name += "#" + info.ModernGamerTagSuffix
		}
	}
	if name == "" {
		name = info.GamerTag
	}
	return AccountProfile{Gamertag: name, ClassicGamertag: info.GamerTag, XUID: info.XUID}
}

func (a *App) GetAccountProfile() (*AccountProfile, error) {
	account := auth.Auth.Account()
	if account == nil {
		return nil, errors.New("Sign in to view your Xbox profile")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 12*time.Second)
	defer cancel()
	client, err := account.XBLClient(ctx)
	if auth.Auth.Account() != account {
		return nil, errors.New("Account changed")
	}
	if err != nil {
		return nil, errors.New("Xbox profile unavailable. Try again.")
	}
	profile := accountProfileFromClaims(client.UserInfo())
	return &profile, nil
}
