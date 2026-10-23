package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

var TokenSource = AndroidConfig.WriterTokenSource(os.Stdout)

func WriterTokenSource(w io.Writer) oauth2.TokenSource {
	return AndroidConfig.WriterTokenSource(w)
}

func (conf Config) WriterTokenSource(w io.Writer) oauth2.TokenSource {
	return &tokenSource{w: w, conf: conf}
}

type tokenSource struct {
	w    io.Writer
	t    oauth2.TokenSource
	mu   sync.Mutex
	conf Config
}

func (src *tokenSource) Token() (*oauth2.Token, error) {
	if src.conf.ClientID == "" {
		panic(fmt.Errorf("minecraft/auth: tokenSource: missing ClientID; construct via Config.WriterTokenSource (or AndroidConfig.WriterTokenSource)"))
	}

	src.mu.Lock()
	defer src.mu.Unlock()
	if src.t == nil {
		t, err := src.conf.RequestLiveTokenWriter(src.w)
		if err != nil {
			return nil, err
		}
		src.t = src.conf.TokenSource(context.Background(), t)
	}
	tok, err := src.t.Token()
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func RefreshTokenSource(t *oauth2.Token) oauth2.TokenSource {
	return RefreshTokenSourceWriter(t, os.Stdout)
}

func (conf Config) RefreshTokenSource(t *oauth2.Token) oauth2.TokenSource {
	return conf.RefreshTokenSourceWriter(t, os.Stdout)
}

func RefreshTokenSourceWriter(t *oauth2.Token, w io.Writer) oauth2.TokenSource {
	return AndroidConfig.RefreshTokenSourceWriter(t, w)
}

func (conf Config) RefreshTokenSourceWriter(t *oauth2.Token, w io.Writer) oauth2.TokenSource {
	var src oauth2.TokenSource
	if t != nil {
		src = conf.TokenSource(context.Background(), t)
	}
	return &tokenSource{w: w, conf: conf, t: src}
}

func RequestLiveToken() (*oauth2.Token, error) {
	return RequestLiveTokenWriter(os.Stdout)
}

func (conf Config) RequestLiveToken() (*oauth2.Token, error) {
	return conf.RequestLiveTokenWriter(os.Stdout)
}

func RequestLiveTokenWriter(w io.Writer) (*oauth2.Token, error) {
	return AndroidConfig.RequestLiveTokenWriter(w)
}

func (conf Config) RequestLiveTokenWriter(w io.Writer) (*oauth2.Token, error) {
	return conf.RequestLiveTokenContext(context.Background(), w)
}

func RequestLiveTokenContext(ctx context.Context, w io.Writer) (*oauth2.Token, error) {
	return AndroidConfig.RequestLiveTokenContext(ctx, w)
}

func (conf Config) RequestLiveTokenContext(ctx context.Context, w io.Writer) (*oauth2.Token, error) {
	d, err := conf.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("start device auth: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Authenticate at %v using the code %v.\n", d.VerificationURI, d.UserCode)

	token, err := conf.DeviceAccessToken(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("poll device token: %w", err)
	}
	_, _ = w.Write([]byte("Authentication successful.\n"))
	return token, nil
}
