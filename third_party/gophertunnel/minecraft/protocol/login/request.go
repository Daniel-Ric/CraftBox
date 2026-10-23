package login

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
)

type chain []string

type certificate struct {
	Chain chain `json:"chain"`
}

type request struct {
	Certificate certificate `json:"Certificate"`

	AuthenticationType uint8 `json:"AuthenticationType"`

	Token string `json:"Token"`

	RawToken string `json:"-"`

	Legacy bool `json:"-"`
}

func (r *request) MarshalJSON() ([]byte, error) {
	if r.Legacy {
		return json.Marshal(r.Certificate)
	}

	cert, err := json.Marshal(r.Certificate)
	if err != nil {
		return nil, err
	}

	type Alias request
	return json.Marshal(&struct {
		Certificate string `json:"Certificate"`
		Alias
	}{
		Certificate: string(cert),
		Alias:       (Alias)(*r),
	})
}

func init() {

	const mojangPublicKey = `MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAECRXueJeTDqNRRgJi/vlRufByu/2G0i2Ebt6YMar5QX/R0DIIyrJMcUpruK4QveTfJSTp3Shlq4Gk34cD/4GUWwkv0DVuzeuB+tXija7HBxii03NHDbPAD0AKnLr2wdAp`

	data, _ := base64.StdEncoding.DecodeString(mojangPublicKey)
	publicKey, _ := x509.ParsePKIXPublicKey(data)
	mojangKey = publicKey.(*ecdsa.PublicKey)
}

var mojangKey = new(ecdsa.PublicKey)

type AuthResult struct {
	PublicKey             *ecdsa.PublicKey
	XBOXLiveAuthenticated bool
}

func Parse(request []byte, verifier *oidc.IDTokenVerifier) (IdentityData, ClientData, AuthResult, error) {
	var (
		iData IdentityData
		cData ClientData
		res   AuthResult
		key   = &ecdsa.PublicKey{}
	)
	req, err := parseLoginRequest(request)
	if err != nil {
		return iData, cData, res, fmt.Errorf("parse login request: %w", err)
	}

	var (
		authenticated bool
		t             = time.Now()
	)
	switch {
	case verifier != nil && req.Token != "":

		idt, err := verifier.Verify(context.Background(), req.Token)
		if err != nil {
			return iData, cData, res, fmt.Errorf("verify ID token: %w", err)
		}
		var claims tokenClaims
		if err := idt.Claims(&claims); err != nil {
			return iData, cData, res, fmt.Errorf("parse ID token: %w", err)
		}
		if err := claims.Validate(jwt.Expected{Time: t}); err != nil {
			return iData, cData, res, fmt.Errorf("validate ID token: %w", err)
		}
		if err := ParsePublicKey(claims.ClientPublicKey, key); err != nil {
			return iData, cData, res, fmt.Errorf("parse cpk: %w", err)
		}
		iData = claims.identityData()
		authenticated = iData.XUID != ""

		if authenticated && len(req.Certificate.Chain) > 0 {
			if legacyID, _, legacyAuthed, err := parseLegacyChain(req.Certificate.Chain, t); err == nil && legacyAuthed {
				if legacyID.TitleID != "" && legacyID.XUID == iData.XUID {
					iData.TitleID = legacyID.TitleID
				}
			}
		}
		if err := iData.Validate(); err != nil {
			return iData, cData, res, fmt.Errorf("validate identity data: %w", err)
		}
	case req.Token != "":

		tok, err := jwt.ParseSigned(req.Token, []jose.SignatureAlgorithm{jose.ES384, jose.RS256})
		if err != nil {
			return iData, cData, res, fmt.Errorf("parse unverified token: %w", err)
		}
		var claims tokenClaims
		if err := tok.UnsafeClaimsWithoutVerification(&claims); err != nil {
			return iData, cData, res, fmt.Errorf("parse unverified token claims: %w", err)
		}
		if err := ParsePublicKey(claims.ClientPublicKey, key); err != nil {
			return iData, cData, res, fmt.Errorf("parse cpk: %w", err)
		}
		iData = claims.identityData()
		if err := iData.Validate(); err != nil {
			return iData, cData, res, fmt.Errorf("validate identity data: %w", err)
		}
	default:
		legacyID, legacyKey, legacyAuthed, err := parseLegacyChain(req.Certificate.Chain, t)
		if err != nil {
			return iData, cData, res, err
		}
		iData, key, authenticated = legacyID, legacyKey, legacyAuthed
	}

	if err := parseFullClaim(req.RawToken, key, &cData); err != nil {
		return iData, cData, res, fmt.Errorf("parse client data: %w", err)
	}
	if !strings.Contains(cData.ServerAddress, "://") && strings.Count(cData.ServerAddress, ":") > 1 && cData.ServerAddress[0] != '[' {

		ind := strings.LastIndex(cData.ServerAddress, ":")
		cData.ServerAddress = "[" + cData.ServerAddress[:ind] + "]" + cData.ServerAddress[ind:]
	}
	if err := cData.Validate(); err != nil {
		return iData, cData, res, fmt.Errorf("validate client data: %w", err)
	}
	return iData, cData, AuthResult{PublicKey: key, XBOXLiveAuthenticated: authenticated}, nil
}

func parseLegacyChain(chain []string, now time.Time) (IdentityData, *ecdsa.PublicKey, bool, error) {
	if len(chain) == 0 {
		return IdentityData{}, nil, false, fmt.Errorf("decode chain: no elements")
	}
	key := &ecdsa.PublicKey{}
	tok, err := jwt.ParseSigned(chain[0], []jose.SignatureAlgorithm{jose.ES384})
	if err != nil {
		return IdentityData{}, nil, false, fmt.Errorf("parse token 0: %w", err)
	}

	raw := tok.Headers[0].ExtraHeaders["x5u"]
	if err := parseAsKey(raw, key); err != nil {
		return IdentityData{}, nil, false, fmt.Errorf("parse x5u: %w", err)
	}

	var (
		identityClaims identityClaims
		authenticated  bool
	)
	iss := "Mojang"

	switch len(chain) {
	case 1:

		if err := parseFullClaim(chain[0], key, &identityClaims); err != nil {
			return IdentityData{}, nil, false, err
		}
		if err := identityClaims.Validate(jwt.Expected{Time: now}); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("validate token 0: %w", err)
		}
	case 3:

		var c jwt.Claims
		if err := parseFullClaim(chain[0], key, &c); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("parse token 0: %w", err)
		}
		if err := c.Validate(jwt.Expected{Time: now}); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("validate token 0: %w", err)
		}
		authenticated = key.Equal(mojangKey)

		if err := parseFullClaim(chain[1], key, &c); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("parse token 1: %w", err)
		}
		if err := c.Validate(jwt.Expected{Time: now, Issuer: iss}); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("validate token 1: %w", err)
		}
		if err := parseFullClaim(chain[2], key, &identityClaims); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("parse token 2: %w", err)
		}
		if err := identityClaims.Validate(jwt.Expected{Time: now, Issuer: iss}); err != nil {
			return IdentityData{}, nil, false, fmt.Errorf("validate token 2: %w", err)
		}
		if authenticated != (identityClaims.ExtraData.XUID != "") {
			return IdentityData{}, nil, false, fmt.Errorf("identity data must have an XUID when logged into XBOX Live only")
		}
	default:
		return IdentityData{}, nil, false, fmt.Errorf("unexpected login chain length %v", len(chain))
	}
	return identityClaims.ExtraData, key, authenticated, nil
}

func parseLoginRequest(requestData []byte) (*request, error) {
	buf := bytes.NewBuffer(requestData)
	var chainLength int32
	if err := binary.Read(buf, binary.LittleEndian, &chainLength); err != nil {
		return nil, fmt.Errorf("read chain length: %w", err)
	}
	if chainLength <= 0 {
		return nil, fmt.Errorf("invalid chain length: %d", chainLength)
	}
	chainData := buf.Next(int(chainLength))

	r := struct {
		request
		Certificate string `json:"Certificate"`
		Chain       chain  `json:"chain"`
	}{}
	if err := json.Unmarshal(chainData, &r); err != nil {
		return nil, fmt.Errorf("decode chain data: %w", err)
	}

	if r.Certificate != "" {
		if err := json.Unmarshal([]byte(r.Certificate), &r.request.Certificate); err != nil {
			return nil, fmt.Errorf("decode certificate: %w", err)
		}
	} else {
		r.request.Certificate.Chain = r.Chain
	}

	if r.AuthenticationType == 1 {
		return nil, fmt.Errorf("guest authentication is not supported")
	}

	var rawLength int32
	if err := binary.Read(buf, binary.LittleEndian, &rawLength); err != nil {
		return nil, fmt.Errorf("read raw token length: %w", err)
	}
	r.RawToken = string(buf.Next(int(rawLength)))
	if n := buf.Len(); n != 0 {
		return nil, fmt.Errorf("%d unread bytes", n)
	}
	return &r.request, nil
}

func parseFullClaim(claim string, key *ecdsa.PublicKey, v any) error {
	tok, err := jwt.ParseSigned(claim, []jose.SignatureAlgorithm{jose.ES384})
	if err != nil {
		return fmt.Errorf("error parsing signed token: %w", err)
	}
	var m map[string]any
	if err := tok.Claims(key, v, &m); err != nil {
		return fmt.Errorf("error verifying claims of token: %w", err)
	}
	newKey, present := m["identityPublicKey"]
	if present {
		if err := parseAsKey(newKey, key); err != nil {
			return fmt.Errorf("error parsing identity public key: %w", err)
		}
	}
	return nil
}

func parseAsKey(k any, pub *ecdsa.PublicKey) error {
	kStr, _ := k.(string)
	if err := ParsePublicKey(kStr, pub); err != nil {
		return fmt.Errorf("error parsing public key: %w", err)
	}
	return nil
}

func Encode(loginChain string, data ClientData, key *ecdsa.PrivateKey, token string, legacy bool) []byte {

	cert := &certificate{}
	_ = json.Unmarshal([]byte(loginChain), &cert)

	keyData := MarshalPublicKey(&key.PublicKey)
	tok, _ := jwt.ParseSigned(cert.Chain[0], []jose.SignatureAlgorithm{jose.ES384})

	x5uData := tok.Headers[0].ExtraHeaders["x5u"]
	x5u, _ := x5uData.(string)
	claims := jwt.Claims{
		Expiry:    jwt.NewNumericDate(time.Now().Add(time.Hour * 6)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour * 6)),
	}

	signer, _ := jose.NewSigner(jose.SigningKey{Key: key, Algorithm: jose.ES384}, &jose.SignerOptions{
		ExtraHeaders: map[jose.HeaderKey]any{"x5u": keyData},
	})
	firstJWT, _ := jwt.Signed(signer).Claims(identityPublicKeyClaims{
		Claims:               claims,
		IdentityPublicKey:    x5u,
		CertificateAuthority: true,
	}).Serialize()

	req := &request{
		Certificate: certificate{

			Chain: append(chain{firstJWT}, cert.Chain...),
		},
		Token:  token,
		Legacy: legacy,
	}

	req.RawToken, _ = jwt.Signed(signer).Claims(data).Serialize()

	return encodeRequest(req)
}

func encodeRequest(req *request) []byte {
	chainBytes, _ := json.Marshal(req)

	buf := bytes.NewBuffer(nil)
	_ = binary.Write(buf, binary.LittleEndian, int32(len(chainBytes)))
	_, _ = buf.WriteString(string(chainBytes))

	_ = binary.Write(buf, binary.LittleEndian, int32(len(req.RawToken)))
	_, _ = buf.WriteString(req.RawToken)
	return buf.Bytes()
}

func EncodeOffline(identityData IdentityData, data ClientData, key *ecdsa.PrivateKey, legacy bool) []byte {
	keyData := MarshalPublicKey(&key.PublicKey)
	claims := jwt.Claims{
		Expiry:    jwt.NewNumericDate(time.Now().Add(time.Hour * 6)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour * 6)),
	}

	signer, _ := jose.NewSigner(jose.SigningKey{Key: key, Algorithm: jose.ES384}, &jose.SignerOptions{
		ExtraHeaders: map[jose.HeaderKey]any{"x5u": keyData},
	})

	req := &request{AuthenticationType: 2}
	if legacy {
		chainJWT, _ := jwt.Signed(signer).Claims(identityClaims{
			Claims:            claims,
			ExtraData:         identityData,
			IdentityPublicKey: keyData,
		}).Serialize()
		req.Certificate = certificate{Chain: chain{chainJWT}}
		req.Legacy = true
	} else {
		claims.Audience = jwt.Audience{"api://auth-minecraft-services/multiplayer"}
		req.Certificate = certificate{Chain: chain{""}}
		req.Token, _ = jwt.Signed(signer).Claims(tokenClaims{
			Claims:          claims,
			ClientPublicKey: keyData,
			XUID:            identityData.XUID,
			DisplayName:     identityData.DisplayName,
			Identity:        identityData.Identity,
			PlayFabID:       identityData.PlayFabID,
			PlayFabTitleID:  identityData.PlayFabTitleID,
		}).Serialize()
	}

	req.RawToken, _ = jwt.Signed(signer).Claims(data).Serialize()

	return encodeRequest(req)
}

type tokenClaims struct {
	jwt.Claims

	IdentityProviderType string `json:"ipt"`

	PlayFabID string `json:"mid"`

	PlayFabTitleID string `json:"tid"`

	ClientPublicKey string `json:"cpk"`

	XUID string `json:"xid"`

	DisplayName string `json:"xname"`

	Identity string `json:"leguuid,omitempty"`
}

func (tc tokenClaims) identityData() IdentityData {
	identity := tc.Identity
	if identity == "" {
		if tc.XUID != "" {
			identity = identityFromXUID(tc.XUID).String()
		} else {
			identity = uuid.New().String()
		}
	}
	return IdentityData{
		XUID:           tc.XUID,
		Identity:       identity,
		DisplayName:    tc.DisplayName,
		PlayFabID:      tc.PlayFabID,
		PlayFabTitleID: tc.PlayFabTitleID,
	}
}

func identityFromXUID(xuid string) uuid.UUID {

	hash := md5.New()
	hash.Write([]byte("pocket-auth-1-xuid:"))
	hash.Write([]byte(xuid))
	s := hash.Sum(nil)
	var id uuid.UUID
	copy(id[:], s)
	id[6] = (id[6] & 0x0f) | 0x30
	id[8] = (id[8] & 0x3f) | 0x80
	return id
}

type identityClaims struct {
	jwt.Claims

	ExtraData IdentityData `json:"extraData"`

	IdentityPublicKey string `json:"identityPublicKey"`
}

func (c identityClaims) Validate(e jwt.Expected) error {
	if err := c.Claims.Validate(e); err != nil {
		return err
	}
	return c.ExtraData.Validate()
}

type identityPublicKeyClaims struct {
	jwt.Claims

	IdentityPublicKey    string `json:"identityPublicKey"`
	CertificateAuthority bool   `json:"certificateAuthority,omitempty"`
}

func ParsePublicKey(b64Data string, key *ecdsa.PublicKey) error {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return fmt.Errorf("decode public key data: %w", err)
	}
	publicKey, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("expected ECDSA public key, got %v", key)
	}
	*key = *ecdsaKey
	return nil
}

func MarshalPublicKey(key *ecdsa.PublicKey) string {
	data, _ := x509.MarshalPKIXPublicKey(key)
	return base64.StdEncoding.EncodeToString(data)
}
