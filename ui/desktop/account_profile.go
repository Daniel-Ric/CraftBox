package desktop

import (
	"context"
	"errors"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/df-mc/go-xsapi/v2/social"
	"github.com/df-mc/go-xsapi/v2/xal/xsts"
)

type AccountProfile struct {
	Gamertag         string `json:"gamertag"`
	ClassicGamertag  string `json:"classicGamertag,omitempty"`
	XUID             string `json:"xuid"`
	AvatarURL        string `json:"avatarURL,omitempty"`
	Gamerscore       string `json:"gamerscore,omitempty"`
	Bio              string `json:"bio,omitempty"`
	Presence         string `json:"presence,omitempty"`
	AccountTier      string `json:"accountTier,omitempty"`
	DetailsAvailable bool   `json:"detailsAvailable"`
	Friends          int    `json:"friends"`
	Followers        int    `json:"followers"`
	Following        int    `json:"following"`
	PlayFabID        string `json:"playFabID,omitempty"`
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
	profile.PlayFabID = account.CachedPlayFabID()
	user, profileErr := client.Social().UserByXUID(ctx, profile.XUID)
	if auth.Auth.Account() != account {
		return nil, errors.New("Account changed")
	}
	if profileErr == nil {
		enrichAccountProfile(&profile, user)
	}
	return &profile, nil
}

func enrichAccountProfile(profile *AccountProfile, user social.User) {
	if user.XUID != profile.XUID {
		return
	}
	profile.DetailsAvailable = true
	profile.Gamerscore = user.GamerScore.String()
	profile.Bio = user.Detail.Bio
	profile.Presence = user.PresenceText
	profile.AccountTier = user.Detail.AccountTier
	profile.Friends, profile.Followers, profile.Following = user.Detail.FriendCount, user.Detail.FollowerCount, user.Detail.FollowingCount
	if imageURL, err := url.Parse(user.DisplayPictureRawURL); err == nil && imageURL.Scheme == "https" && imageURL.Host != "" && imageURL.User == nil {
		profile.AvatarURL = imageURL.String()
	}
}

func (a *App) GetAccountSignInProfiles() ([]string, error) {
	entries, err := os.ReadDir(utils.PathData(""))
	if err != nil {
		return nil, errors.New("Saved sign-ins unavailable")
	}
	names := []string{""}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, "token-") && strings.HasSuffix(name, ".json") {
			name = strings.TrimSuffix(strings.TrimPrefix(name, "token-"), ".json")
			if name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\:`) {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names, nil
}
