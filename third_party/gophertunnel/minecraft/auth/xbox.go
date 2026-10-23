package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/df-mc/go-xsapi/v2/xal"
	"github.com/df-mc/go-xsapi/v2/xal/sisu"
	"github.com/df-mc/go-xsapi/v2/xal/xasd"
	"github.com/df-mc/go-xsapi/v2/xal/xsts"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type XBLToken struct {
	AuthorizationToken *xsts.Token
}

func (t XBLToken) SetAuthHeader(r *http.Request) {
	t.AuthorizationToken.SetAuthHeader(r)
}

func (t XBLToken) Valid() bool {
	return t.AuthorizationToken != nil && t.AuthorizationToken.Valid()
}

type contextKey struct{}

var tokenCacheContextKey contextKey

type XBLTokenCache struct {
	conf Config

	device xasd.TokenSource

	session *sisu.Session

	sessionMu sync.RWMutex
}

func (cache *XBLTokenCache) Session() *sisu.Session {
	cache.sessionMu.RLock()
	defer cache.sessionMu.RUnlock()
	return cache.session
}

func (cache *XBLTokenCache) Device() xasd.TokenSource {
	if cache.device == nil {
		return cache.Session()
	}
	return cache.device
}

func ContextSession(ctx context.Context, src oauth2.TokenSource) *sisu.Session {
	if cache, ok := ctx.Value(tokenCacheContextKey).(*XBLTokenCache); ok {
		cache.sessionMu.Lock()
		defer cache.sessionMu.Unlock()
		if cache.session == nil {
			cache.session = cache.conf.New(src, &sisu.SessionConfig{
				DeviceTokenSource: cache.device,
				HTTPClient:        ContextClient(ctx),
			})
		}
		return cache.session
	}
	return AndroidConfig.New(src, &sisu.SessionConfig{HTTPClient: ContextClient(ctx)})
}

func ContextClient(ctx context.Context) *http.Client {
	if client, ok := contextClient(ctx); ok {
		return client
	}
	return http.DefaultClient
}

func contextClient(ctx context.Context) (*http.Client, bool) {
	if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && client != nil {
		return client, true
	}
	if client, ok := ctx.Value(xal.HTTPClient).(*http.Client); ok && client != nil {
		return client, true
	}
	return nil, false
}

func WithContextClient(ctx context.Context, client *http.Client) context.Context {
	if existing, ok := contextClient(ctx); ok {
		client = existing
	}
	if client == nil {
		return ctx
	}
	if c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); !ok || c == nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	}
	if c, ok := ctx.Value(xal.HTTPClient).(*http.Client); !ok || c == nil {
		ctx = context.WithValue(ctx, xal.HTTPClient, client)
	}
	return ctx
}

func (conf Config) NewTokenCache() *XBLTokenCache {
	return &XBLTokenCache{
		conf:   conf,
		device: xasd.ReuseTokenSource(conf.Config.Config, nil, nil),
	}
}

func ReuseTokenCache(session *sisu.Session) *XBLTokenCache {
	return AndroidConfig.ReuseTokenCache(session)
}

func (conf Config) ReuseTokenCache(session *sisu.Session) *XBLTokenCache {
	if session == nil {
		panic("auth: ReuseTokenCache: session cannot be nil")
	}
	return &XBLTokenCache{
		conf:    conf,
		session: session,
	}
}

func WithXBLTokenCache(parent context.Context, cache *XBLTokenCache) context.Context {
	return context.WithValue(parent, tokenCacheContextKey, cache)
}

type Config struct {
	sisu.Config
}

var (
	AndroidConfig = Config{
		sisu.Config{
			Config: xal.Config{

				Device: xal.Device{
					Type:    xal.DeviceTypeAndroid,
					Version: "13",
				},
				UserAgent: "XAL Android 2025.04.20250326.000",
				TitleID:   1739947436,
				Sandbox:   "RETAIL",
			},
			ClientID:    "0000000048183522",
			RedirectURI: "ms-xal-0000000048183522://auth",
		},
	}

	IOSConfig = Config{
		sisu.Config{
			Config: xal.Config{
				Device: xal.Device{
					Type:    xal.DeviceTypeIOS,
					Version: "15.6.1",
				},
				UserAgent: "XAL iOS 2021.11.20211021.000",
				TitleID:   1810924247,
				Sandbox:   "RETAIL",
			},
			ClientID:    "000000004c17c01a",
			RedirectURI: "ms-xal-000000004c17c01a://auth",
		},
	}

	Win32Config = Config{
		sisu.Config{
			Config: xal.Config{

				Device: xal.Device{
					Type:    xal.DeviceTypeWin32,
					Version: "10.0.28000",
				},
				UserAgent: "XAL GRTS 2025.11.20251105.000",
				TitleID:   896928775,
				Sandbox:   "RETAIL",
			},
			ClientID:    "0000000040159362",
			RedirectURI: "ms-xal-0000000040159362://auth",
		},
	}

	NintendoConfig = Config{
		sisu.Config{
			Config: xal.Config{
				Device: xal.Device{
					Type:    xal.DeviceTypeNintendo,
					Version: "0.0.0",
				},
				UserAgent: "XAL",
				TitleID:   2047319603,
				Sandbox:   "RETAIL",
			},
			ClientID: "00000000441cc96b",
		},
	}

	PlayStationConfig = Config{
		sisu.Config{
			Config: xal.Config{
				Device: xal.Device{
					Type:    xal.DeviceTypePlayStation,
					Version: "10.0.0",
				},
				UserAgent: "XAL",
				Sandbox:   "RETAIL",
				TitleID:   2044456598,
			},
			ClientID: "000000004827c78e",
		},
	}

	ServiceConfigID = uuid.MustParse("4fc10100-5f7a-4470-899b-280835760c07")

	PreviewServiceConfigID = uuid.MustParse("00000000-0000-0000-0000-0000717d695f")
)

func RequestXBLToken(ctx context.Context, liveToken *oauth2.Token, relyingParty string) (*XBLToken, error) {
	var conf Config
	if cache, ok := ctx.Value(tokenCacheContextKey).(*XBLTokenCache); ok {
		conf = cache.conf
	} else {
		conf = AndroidConfig
	}
	return conf.RequestXBLToken(ctx, liveToken, relyingParty)
}

func (conf Config) RequestXBLToken(ctx context.Context, liveToken *oauth2.Token, relyingParty string) (*XBLToken, error) {
	var s *sisu.Session
	if cache, ok := ctx.Value(tokenCacheContextKey).(*XBLTokenCache); ok {
		if cache.conf != conf {
			return nil, errors.New("auth: Config.RequestXBLToken: config mismatch")
		}
		cache.sessionMu.Lock()
		if cache.session == nil {
			cache.session = conf.New(conf.TokenSource(context.WithoutCancel(ctx), liveToken), &sisu.SessionConfig{
				DeviceTokenSource: cache.device,
				HTTPClient:        ContextClient(ctx),
			})
		}
		s = cache.session
		cache.sessionMu.Unlock()
	} else {

		s = conf.New(conf.TokenSource(context.WithoutCancel(ctx), liveToken), &sisu.SessionConfig{
			HTTPClient: ContextClient(ctx),
		})
	}
	token, err := s.XSTSToken(ctx, relyingParty)
	if err != nil {
		return nil, err
	}
	return newXBLToken(token)
}

func newXBLToken(token *xsts.Token) (*XBLToken, error) {
	if !token.Valid() {
		return nil, errors.New("auth: xsts token is invalid")
	}

	return &XBLToken{AuthorizationToken: token}, nil
}
