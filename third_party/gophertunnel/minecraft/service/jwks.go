package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/sandertv/gophertunnel/minecraft/service/internal"
)

type refreshingKeySet struct {
	env *AuthorizationEnvironment

	jwksURL         string
	refreshInterval time.Duration
	sigAlgs         []jose.SignatureAlgorithm

	ctx context.Context

	mu sync.RWMutex

	inflight *refreshingKeySetInflight

	cachedKeys []jose.JSONWebKey
	lastFetch  time.Time
}

type refreshingKeySetInflight struct {
	doneCh chan struct{}
	keys   []jose.JSONWebKey
	err    error
}

func newRefreshingKeySet(ctx context.Context, env *AuthorizationEnvironment, jwksURL string, refreshInterval time.Duration, supportedSigningAlgs []string) *refreshingKeySet {
	sigAlgs := make([]jose.SignatureAlgorithm, 0, len(supportedSigningAlgs))
	for _, alg := range supportedSigningAlgs {
		if alg == "" {
			continue
		}
		sigAlgs = append(sigAlgs, jose.SignatureAlgorithm(alg))
	}
	if len(sigAlgs) == 0 {

		sigAlgs = []jose.SignatureAlgorithm{jose.RS256}
	}

	return &refreshingKeySet{
		env:             env,
		jwksURL:         jwksURL,
		refreshInterval: refreshInterval,
		sigAlgs:         sigAlgs,
		ctx:             context.WithoutCancel(ctx),
	}
}

func (r *refreshingKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	jws, err := jose.ParseSigned(jwt, r.sigAlgs)
	if err != nil {
		return nil, fmt.Errorf("minecraft/service: malformed jwt: %w", err)
	}

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	keys, lastFetch := r.keysFromCache()
	if payload, ok := r.tryVerify(jws, keys, keyID); ok {
		return payload, nil
	}

	needsRefresh := len(keys) == 0
	if !needsRefresh && keyID != "" && !r.knowsKeyID(keys, keyID) && time.Since(lastFetch) >= r.refreshInterval {
		needsRefresh = true
	}
	if needsRefresh {
		keys, err := r.keysFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("minecraft/service: fetch jwks: %w", err)
		}
		if payload, ok := r.tryVerify(jws, keys, keyID); ok {
			return payload, nil
		}
	}

	return nil, errors.New("minecraft/service: failed to verify id token signature")
}

func (r *refreshingKeySet) keysFromCache() (keys []jose.JSONWebKey, lastFetch time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cachedKeys, r.lastFetch
}

func (r *refreshingKeySet) keysFromRemote(ctx context.Context) ([]jose.JSONWebKey, error) {

	r.mu.Lock()
	if r.inflight == nil {
		r.inflight = &refreshingKeySetInflight{doneCh: make(chan struct{})}
		inflight := r.inflight

		go func() {
			keys, err := r.updateKeys()

			inflight.keys = keys
			inflight.err = err
			close(inflight.doneCh)

			r.mu.Lock()
			defer r.mu.Unlock()

			if err == nil {
				r.cachedKeys = keys
				r.lastFetch = time.Now()
			}
			r.inflight = nil
		}()
	}
	inflight := r.inflight
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-inflight.doneCh:
		return inflight.keys, inflight.err
	}
}

func (r *refreshingKeySet) updateKeys() ([]jose.JSONWebKey, error) {
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", internal.UserAgent)

	resp, err := r.env.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, internal.Err(resp)
	}

	keyset, err := decodeKeySet(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode key set: %w", err)
	}
	return keyset.Keys, nil
}

func (r *refreshingKeySet) knowsKeyID(keys []jose.JSONWebKey, keyID string) bool {
	for _, key := range keys {
		if key.KeyID == keyID {
			return true
		}
	}
	return false
}

func (r *refreshingKeySet) tryVerify(jws *jose.JSONWebSignature, keys []jose.JSONWebKey, keyID string) ([]byte, bool) {
	for _, key := range keys {
		if keyID == "" || key.KeyID == keyID {
			if payload, err := jws.Verify(&key); err == nil {
				return payload, true
			}
		}
	}
	return nil, false
}

func decodeKeySet(r io.Reader) (*jose.JSONWebKeySet, error) {
	var data struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	set := &jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(data.Keys)),
	}
	for i, key := range data.Keys {
		x5t, ok := key["x5t"].(string)
		if !ok {
			return nil, errors.New("no x5t found in jwk")
		}

		var fingerprint []byte
		switch len(x5t) {
		case 40:
			var err error
			fingerprint, err = hex.DecodeString(x5t)
			if err != nil {
				return nil, fmt.Errorf("decode x5t hex: %w", err)
			}
			key["x5t"] = base64.RawURLEncoding.EncodeToString(fingerprint)
		case 27:
			var err error
			fingerprint, err = base64.RawURLEncoding.DecodeString(x5t)
			if err != nil {
				return nil, fmt.Errorf("decode x5t base64: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid x5t length: %d", len(x5t))
		}
		if n := len(fingerprint); n != 20 {
			return nil, fmt.Errorf("fingerprint is not 20 bytes long: %d", n)
		}

		b, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("encode reformatted jwk: %w", err)
		}
		var jwk jose.JSONWebKey
		if err := jwk.UnmarshalJSON(b); err != nil {
			return nil, fmt.Errorf("decode reformatted jwk: %w", err)
		}
		set.Keys[i] = jwk
	}
	return set, nil
}
