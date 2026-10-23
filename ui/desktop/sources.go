package desktop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
)

type Source struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Available   bool   `json:"available"`
}

func (a *App) ListSources(kind string) ([]Source, error) {
	account := auth.Auth.Account()
	if account == nil {
		return nil, errors.New("Sign in with your Microsoft account")
	}
	ctx := a.ctx
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
			add(fmt.Sprintf("source:realm:%d", realm.ID), realm.Name, "Realm", target, true)
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
		}
	case "events", "featured":
		client, err := account.Gatherings(ctx)
		if err != nil {
			return nil, err
		}
		token, err := account.MCToken(ctx)
		if err != nil {
			return nil, err
		}
		if kind == "events" {
			list, err := client.GetGatherings(ctx, token)
			if err != nil {
				return nil, err
			}
			for _, event := range list {
				target := &connectinfo.ConnectInfo{}
				target.SetGathering(event)
				add("source:event:"+event.GatheringID, event.Title, event.Description, target, event.IsEnabled && !time.Now().Before(event.StartTimeUtc) && time.Now().Before(event.EndTimeUtc))
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
