package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sirupsen/logrus"
)

var ErrNotLoggedIn = errors.New("not Logged In")

var defaultDeviceType = &auth.AndroidConfig

type authSrv struct {
	log     *logrus.Entry
	env     string
	handler messages.AuthCodeHandler
	account atomic.Pointer[Account]

	authCtxCancel atomic.Pointer[context.CancelFunc]
}

var Auth *authSrv = &authSrv{
	log: logrus.WithField("part", "Auth"),
}

func (a *authSrv) SetEnv(env string) {
	a.env = env
}

func (a *authSrv) setAccount(acc *Account) *Account {
	prevAcc := a.account.Swap(acc)
	if prevAcc != nil {
		prevAcc.Close()
	}
	return prevAcc
}

func (a *authSrv) LoadAccount(name string) (err error) {
	tokenInfo, err := readAuth[tokenInfo](tokenFileName(name))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, errors.ErrUnsupported) {
		return nil
	}
	if err != nil {
		return err
	}
	a.setAccount(&Account{
		token: tokenInfo,
		name:  name,
		env:   a.env,
	})
	return nil
}

func (a *authSrv) LoggedIn() bool {
	return a.account.Load() != nil
}

func (a *authSrv) SelectAccount(name string) error {
	if err := a.LoadAccount(name); err != nil {
		return err
	}
	if current := a.Account(); current != nil && current.Name() != name {
		a.setAccount(nil)
	}
	return nil
}

func (a *authSrv) SetHandler(handler messages.AuthCodeHandler) (err error) {
	a.handler = handler
	return nil
}

func (a *authSrv) Login(ctx context.Context, conf *auth.Config, name string) (err error) {
	if conf == nil {
		conf = &auth.AndroidConfig
	}
	device, err := conf.DeviceAuth(ctx)
	if err != nil {
		return fmt.Errorf("start device auth: %w", err)
	}
	if a.handler != nil {
		a.handler.DisplayCode(device.VerificationURI, device.UserCode)
	} else {
		fmt.Printf("Authenticate at %v using the code %v.\n", device.VerificationURI, device.UserCode)
	}
	liveToken, err := conf.DeviceAccessToken(ctx, device)
	if err != nil {
		return err
	}
	if a.handler != nil {
		a.handler.Successful()
	}
	tokenInfo := tokenInfo{
		Token:    liveToken,
		ClientID: conf.ClientID,
	}
	if err = writeAuth(tokenFileName(name), tokenInfo); err != nil {
		return err
	}
	a.setAccount(&Account{
		token: &tokenInfo,
		name:  name,
		env:   a.env,
	})
	return nil
}

func (a *authSrv) Logout() {
	prevAcc := a.setAccount(nil)
	os.Remove(tokenFileName(prevAcc.name))
	os.Remove(chainFileName(prevAcc.name))
}

func (a *authSrv) Account() *Account {
	return a.account.Load()
}

func (a *authSrv) RequestLogin(name string) error {
	return a.RequestLoginContext(context.Background(), name)
}

func (a *authSrv) RequestLoginContext(parent context.Context, name string) error {
	ctx, cancel := context.WithCancel(parent)
	a.authCtxCancel.Store(&cancel)
	defer cancel()
	err := a.Login(ctx, defaultDeviceType, name)
	messages.SendEvent(&messages.EventAuthFinished{
		Error: err,
	})
	return err
}

func (a *authSrv) CancelLogin() {
	cancel := a.authCtxCancel.Swap(nil)
	if cancel != nil {
		(*cancel)()
	}
}
