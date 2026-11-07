package desktop

import (
	"context"
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/sourceprobe"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
	"github.com/google/uuid"
	nativeGatherings "github.com/sandertv/gophertunnel/minecraft/service/gatherings"
	"github.com/sirupsen/logrus"
)

type Source struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Address           string              `json:"address"`
	Available         bool                `json:"available"`
	OnDemand          bool                `json:"onDemand,omitempty"`
	Version           string              `json:"version,omitempty"`
	Protocol          *int                `json:"protocol,omitempty"`
	Players           *int                `json:"players,omitempty"`
	MaxPlayers        *int                `json:"maxPlayers,omitempty"`
	PlayerCountError  string              `json:"playerCountError,omitempty"`
	Host              string              `json:"host,omitempty"`
	Joinability       string              `json:"joinability,omitempty"`
	Route             string              `json:"route,omitempty"`
	Hardcore          bool                `json:"hardcore,omitempty"`
	EditorWorld       bool                `json:"editorWorld,omitempty"`
	CrossPlayDisabled bool                `json:"crossPlayDisabled,omitempty"`
	StartsAt          string              `json:"startsAt,omitempty"`
	EndsAt            string              `json:"endsAt,omitempty"`
	Probe             *sourceprobe.Result `json:"probe,omitempty"`
}

func (a *App) ListSources(kind string) ([]Source, error) {
	account := auth.Auth.Account()
	if account == nil {
		return nil, errors.New("Sign in with your Microsoft account")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 25*time.Second)
	defer cancel()
	sources := []Source{}
	targets := make(map[string]*connectinfo.ConnectInfo)
	add := func(id, name, description string, target *connectinfo.ConnectInfo, available bool) {
		targets[id] = target
		sources = append(sources, Source{ID: id, Name: name, Description: description, Address: target.Value, Available: available})
	}
	switch kind {
	case "realms":
		client, err := account.Realms(ctx)
		if err != nil {
			return nil, err
		}
		list, err := client.Realms(ctx)
		if err != nil {
			return nil, err
		}
		for _, realm := range list {
			target := &connectinfo.ConnectInfo{}
			target.SetRealm(&realm)
			add(fmt.Sprintf("source:realm:%d", realm.ID), realm.Name, realm.MOTD, target, !realm.Expired && strings.EqualFold(realm.State, "OPEN"))
			s := &sources[len(sources)-1]
			s.Host = realm.Owner
			s.Joinability = realm.State
			if realm.Expired {
				s.Joinability = "Expired subscription"
			}
			s.Route = "NetherNet · Realms"
			if realm.MaxPlayers > 0 {
				n := realm.MaxPlayers
				s.MaxPlayers = &n
			}
		}
	case "worlds":
		client, err := account.XBLClient(ctx)
		if err != nil {
			return nil, err
		}
		list, err := p2p.NewClient(client).Worlds(ctx)
		if err != nil {
			return nil, err
		}
		for _, world := range list {
			target := &connectinfo.ConnectInfo{}
			target.SetWorld(&world)
			add("source:world:"+world.HandleID().String(), world.WorldName, world.HostName+" · "+world.Version, target, true)
			s := &sources[len(sources)-1]
			s.Version = world.Version
			if world.Protocol > 0 {
				s.Protocol = &world.Protocol
			}
			s.Players = world.MemberCount
			s.MaxPlayers = world.MaxMemberCount
			if s.Players != nil && *s.Players < 0 {
				s.Players = nil
			}
			if s.MaxPlayers != nil && *s.MaxPlayers < 0 {
				s.MaxPlayers = nil
			}
			s.Host = world.HostName
			s.Joinability = world.Joinability
			s.Hardcore = world.Hardcore
			s.EditorWorld = world.EditorWorld
			s.CrossPlayDisabled = world.CrossPlayDisabled
			s.Route = "NetherNet · Xbox signaling"
		}
	case "events", "experiences", "featured":
		client, err := account.Gatherings(ctx)
		if err != nil {
			return nil, err
		}
		token, err := account.MCToken(ctx)
		if err != nil {
			return nil, err
		}
		if kind != "featured" {
			list, err := client.GetExperiences(ctx, token)
			if err != nil {
				return nil, err
			}
			countCtx, stopCounts := context.WithTimeout(ctx, 8*time.Second)
			counts, countErr := client.GetPlayerCounts(countCtx, token)
			stopCounts()
			countLog := logrus.WithField("part", "Sources").WithField("kind", kind)
			if countErr != nil {
				countLog.WithError(countErr).Warn("Could not fetch experience player counts")
			} else {
				countLog.WithFields(logrus.Fields{"records": len(counts), "experiences": len(list)}).Debug("Fetched experience player counts")
			}
			for _, experience := range list {
				target := &connectinfo.ConnectInfo{}
				target.SetFeaturedServer(&experience)
				add("source:experience:"+experience.ExperienceId, experience.Name, experience.Description, target, true)
				s := &sources[len(sources)-1]
				s.OnDemand = true
				if countErr != nil {
					s.PlayerCountError = "Experience counts unavailable"
					var limited *nativeGatherings.PlayerCountRateLimitError
					if errors.As(countErr, &limited) {
						s.PlayerCountError = "Minecraft is temporarily limiting player-count requests. Please try again later."
					}
				} else if id, err := uuid.Parse(experience.ExperienceId); err == nil {
					if n, ok := counts[id]; ok {
						s.Players = &n
					}
				}
				s.Route = "NetherNet · On-demand experience"
			}
		} else {
			list, err := client.GetFeaturedServers(ctx, token)
			if err != nil {
				return nil, err
			}
			for _, server := range list {
				target := &connectinfo.ConnectInfo{}
				target.SetFeaturedServer(&server)
				add("source:featured:"+server.Name, server.Name, server.Address, target, true)
				sources[len(sources)-1].Route = "NetherNet · HTTP signaling"
			}
		}
	default:
		return nil, fmt.Errorf("unknown source kind %q", kind)
	}
	a.mu.Lock()
	if auth.Auth.Account() != account {
		a.mu.Unlock()
		return nil, errors.New("The account changed while loading. Reload the sources")
	}
	for key, value := range targets {
		a.sources[key] = value
	}
	a.mu.Unlock()
	return sources, nil
}
func (a *App) Login() error {
	a.mu.Lock()
	if a.state.Running || a.state.Authenticating || a.accountBusy {
		a.mu.Unlock()
		return errors.New("End the active session or sign-in first")
	}
	a.state.Authenticating = true
	name := a.state.Account
	ctx, cancel := context.WithCancel(a.ctx)
	done := make(chan struct{})
	a.loginDone = done
	a.loginCancel = cancel
	a.mu.Unlock()
	a.publish()
	go func() {
		defer close(done)
		defer cancel()
		err := auth.Auth.RequestLoginContext(ctx, name)
		a.mu.Lock()
		a.state.Authenticating = false
		a.state.AuthCode = ""
		a.state.AuthURI = ""
		a.loginDone = nil
		a.loginCancel = nil
		a.mu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			a.reportError(err)
		}
		a.publish()
	}()
	return nil
}
func (a *App) CancelLogin() {
	a.mu.Lock()
	cancel := a.loginCancel
	automatic := a.state.Running && a.state.Authenticating
	if a.loginDone == nil && !automatic {
		a.state.Authenticating = false
	}
	a.state.AuthCode = ""
	a.state.AuthURI = ""
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if automatic {
		a.StopCommand()
	}
	a.publish()
}
func (a *App) beginAccountChange() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.state.Running || a.state.Authenticating || a.accountBusy {
		return errors.New("End the active session or sign-in first")
	}
	a.accountBusy = true
	return nil
}
func (a *App) endAccountChange() { a.mu.Lock(); a.accountBusy = false; a.mu.Unlock(); a.publish() }
func (a *App) Logout() error {
	if err := a.beginAccountChange(); err != nil {
		return err
	}
	defer a.endAccountChange()
	if auth.Auth.LoggedIn() {
		auth.Auth.Logout()
	}
	a.mu.Lock()
	a.sources = make(map[string]*connectinfo.ConnectInfo)
	a.mu.Unlock()
	a.publish()
	return nil
}
func (a *App) SwitchAccount(name string) error {
	name = strings.TrimSpace(name)
	if strings.ContainsAny(name, "/\\:") || name == "." || name == ".." {
		return errors.New("Invalid account name")
	}
	if err := a.beginAccountChange(); err != nil {
		return err
	}
	defer a.endAccountChange()
	if err := auth.Auth.SelectAccount(name); err != nil {
		return err
	}
	a.mu.Lock()
	a.state.Account = name
	a.sources = make(map[string]*connectinfo.ConnectInfo)
	a.mu.Unlock()
	a.publish()
	return nil
}
