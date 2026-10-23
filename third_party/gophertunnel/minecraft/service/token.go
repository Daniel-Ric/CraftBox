package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/df-mc/go-playfab/v2/title"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/service/internal"
	"golang.org/x/text/language"
)

type TokenSource interface {
	ServiceToken(ctx context.Context) (*Token, error)
}

type AuthorizationEnvironment struct {
	ServiceURI *url.URL `json:"serviceUri"`

	Issuer *url.URL `json:"issuer"`

	PlayFabTitleID title.Title `json:"playFabTitleId"`

	EduPlayFabTitleID title.Title `json:"eduPlayFabTitleId"`

	HTTPClient *http.Client `json:"-"`

	KeyRefreshInterval time.Duration `json:"-"`

	verifier *oidc.IDTokenVerifier

	verifierMu sync.Mutex
}

func (e *AuthorizationEnvironment) httpClient() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return http.DefaultClient
}

func (e *AuthorizationEnvironment) ServiceName() string {
	return "auth"
}

func (e *AuthorizationEnvironment) UnmarshalJSON(b []byte) error {
	type Alias AuthorizationEnvironment
	data := struct {
		*Alias
		ServiceURI string `json:"serviceUri"`
		Issuer     string `json:"issuer"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	if data.ServiceURI == "" {
		return errors.New("service: AuthorizationEnvironment.ServiceURI cannot be empty string")
	}
	var err error
	e.ServiceURI, err = url.Parse(data.ServiceURI)
	if err != nil {
		return fmt.Errorf("parse ServiceURI: %w", err)
	}
	if data.Issuer == "" {
		return errors.New("service: AuthorizationEnvironment.Issuer cannot be empty string")
	}
	e.Issuer, err = url.Parse(data.Issuer)
	if err != nil {
		return fmt.Errorf("parse Issuer: %w", err)
	}
	return nil
}

func (e *AuthorizationEnvironment) Token(ctx context.Context, config TokenConfig) (*Token, error) {
	defaultUserConfig(&config.User)
	defaultDeviceConfig(e, &config.Device)
	if config.User.Token == "" {
		return nil, errors.New("minecraft/service: UserConfig.Token is empty")
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(config); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	requestURL := e.ServiceURI.JoinPath("/api/v1.0/session/start").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", internal.UserAgent)

	resp, err := e.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, internal.Err(resp)
	}
	var result internal.Result[*Token]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if result.Data == nil || !result.Data.Valid() {
		return nil, errors.New("minecraft/service: AuthorizationEnvironment: invalid token result")
	}
	if err := decodeClaims(result.Data); err != nil {
		return nil, fmt.Errorf("minecraft/service: decode JWT token claims: %w", err)
	}
	return result.Data, nil
}

func (e *AuthorizationEnvironment) Renew(ctx context.Context, token *Token, user UserConfig) (*Token, error) {
	defaultUserConfig(&user)
	if user.Token == "" {
		return nil, errors.New("minecraft/service: UserConfig.Token is empty")
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(user); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	requestURL := e.ServiceURI.JoinPath("/api/v1.0/session/renew").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	token.SetAuthHeader(req)

	resp, err := e.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, internal.Err(resp)
	}
	var result internal.Result[*Token]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if result.Data == nil || !result.Data.Valid() {
		return nil, errors.New("minecraft/service: invalid renew token result")
	}
	if err := decodeClaims(result.Data); err != nil {
		return nil, fmt.Errorf("minecraft/service: decode JWT token claims: %w", err)
	}
	return result.Data, nil
}

func (e *AuthorizationEnvironment) Verifier() (*oidc.IDTokenVerifier, error) {
	return e.VerifierContext(context.Background())
}

func (e *AuthorizationEnvironment) VerifierContext(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	e.verifierMu.Lock()
	defer e.verifierMu.Unlock()
	if e.verifier != nil {
		return e.verifier, nil
	}

	config, err := e.configuration(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtain OpenID configuration: %w", err)
	}

	refreshInterval := e.KeyRefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = 30 * time.Minute
	}
	keySet := newRefreshingKeySet(ctx, e, config.JWKSURL, refreshInterval, config.Algorithms)

	issuer := e.Issuer.JoinPath().String()
	e.verifier = oidc.NewVerifier(issuer, keySet, &oidc.Config{
		ClientID:             "api://auth-minecraft-services/multiplayer",
		SupportedSigningAlgs: config.Algorithms,
	})
	return e.verifier, nil
}

func (e *AuthorizationEnvironment) configuration(ctx context.Context) (*oidc.ProviderConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.Issuer.JoinPath("/.well-known/openid-configuration").String(), nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", internal.UserAgent)

	resp, err := e.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, internal.Err(resp)
	}
	var config oidc.ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	return &config, nil
}

func (e *AuthorizationEnvironment) MultiplayerToken(ctx context.Context, src TokenSource, key *ecdsa.PublicKey) (string, error) {
	token, err := src.ServiceToken(ctx)
	if err != nil {
		return "", fmt.Errorf("request service token: %w", err)
	}
	b, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("encode public key: %w", err)
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(map[string]any{

		"publicKey": base64.StdEncoding.EncodeToString(b),
	}); err != nil {
		return "", fmt.Errorf("encode request body: %w", err)
	}

	requestURL := e.ServiceURI.JoinPath("/api/v1.0/multiplayer/session/start").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return "", fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	token.SetAuthHeader(req)

	resp, err := e.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", internal.Err(resp)
	}
	var result internal.Result[*multiplayerToken]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response body: %w", err)
	}
	if result.Data == nil || result.Data.SignedToken == "" {
		return "", errors.New("minecraft/service: invalid multiplayer token result")
	}
	return result.Data.SignedToken, nil
}

type multiplayerToken struct {
	IssuedAt time.Time `json:"issuedAt"`

	SignedToken string `json:"signedToken"`

	ValidUntil time.Time `json:"validUntil"`
}

func (t *multiplayerToken) Valid() bool {
	return t.SignedToken != "" && time.Now().Before(t.ValidUntil)
}

func defaultUserConfig(user *UserConfig) {
	if user.LanguageCode == language.Und {
		user.LanguageCode = language.AmericanEnglish
	}
	if user.Language == "" {
		base, _ := user.LanguageCode.Base()
		user.Language = base.String()
	}
	if user.RegionCode == "" {
		region, _ := user.LanguageCode.Region()
		user.RegionCode = region.String()
	}
}

func defaultDeviceConfig(auth *AuthorizationEnvironment, device *DeviceConfig) {
	if device.ApplicationType == "" {
		device.ApplicationType = ApplicationTypeMinecraftPE
	}
	if device.GameVersion == "" {
		device.GameVersion = protocol.CurrentVersion
	}
	if device.ID == "" {
		device.ID = uuid.NewString()
	}
	if device.Memory == "" {
		device.Memory = strconv.FormatUint(16*(1<<30), 10)
	}
	if device.Platform == "" {
		device.Platform = PlatformWindows10
	}
	if device.StorePlatform == "" {
		device.StorePlatform = StorePlatformUWPStore
	}
	if device.Type == "" {
		device.Type = DeviceTypeWindows10
	}
	if device.PlayFabTitleID == "" {
		device.PlayFabTitleID = auth.PlayFabTitleID
	}
}

type Token struct {
	AuthorizationHeader string `json:"authorizationHeader"`

	ValidUntil time.Time `json:"validUntil"`

	Treatments []string `json:"treatments"`

	Configuration map[string]Configuration `json:"configurations"`

	TreatmentContext string `json:"treatmentContext"`

	Claims Claims `json:"-"`
}

func decodeClaims(token *Token) error {
	s := strings.TrimPrefix(token.AuthorizationHeader, "MCToken ")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed JWT: %q", s)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode JWT payload: %w", err)
	}
	if err := json.Unmarshal(payload, &token.Claims); err != nil {
		return fmt.Errorf("decode JWT claims: %w", err)
	}
	if err := token.Claims.Validate(jwt.Expected{}); err != nil {
		return fmt.Errorf("validate JWT claims: %w", err)
	}
	return nil
}

type Claims struct {
	jwt.Claims

	PlayerMessagingID uuid.UUID `json:"pmid"`
}

func (c Claims) Validate(e jwt.Expected) error {
	if c.PlayerMessagingID == uuid.Nil {
		return errors.New("service: Token.Claims.PlayerMessagingID is uuid.Nil")
	}
	return c.Claims.Validate(e)
}

const expirationDelta = time.Minute

func (t *Token) Valid() bool {
	return t.AuthorizationHeader != "" && time.Now().Before(t.ValidUntil.Add(-expirationDelta))
}

func (t *Token) SetAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", t.AuthorizationHeader)
}

type Configuration struct {
	ID string `json:"id"`

	Parameters map[string]string `json:"parameters"`
}

type TokenConfig struct {
	Device DeviceConfig `json:"device,omitempty"`

	User UserConfig `json:"user,omitempty"`
}

type UserConfig struct {
	Language string `json:"language,omitempty"`

	LanguageCode language.Tag `json:"languageCode,omitempty"`

	RegionCode string `json:"regionCode,omitempty"`

	Token string `json:"token,omitempty"`

	TokenType string `json:"tokenType,omitempty"`
}

type DeviceConfig struct {
	ApplicationType string `json:"applicationType,omitempty"`

	Capabilities []string `json:"capabilities"`

	GameVersion string `json:"gameVersion,omitempty"`

	ID string `json:"id,omitempty"`

	HardwareMemoryTier int `json:"hardwareMemoryTier,omitempty"`

	Memory string `json:"memory,omitempty"`

	Platform string `json:"platform,omitempty"`

	PlayFabTitleID title.Title `json:"playFabTitleId,omitempty"`

	StorePlatform string `json:"storePlatform,omitempty"`

	TreatmentOverrides []string `json:"treatmentOverrides,omitempty"`

	Type string `json:"type,omitempty"`
}

const (
	CapabilityRayTracing = "RayTracing"

	CapabilityVibrantVisuals = "VibrantVisuals"
)

const PlatformWindows10 = "Windows10"

const StorePlatformUWPStore = "uwp.store"

const DeviceTypeWindows10 = "Windows10"

const TokenTypePlayFab = "PlayFab"
