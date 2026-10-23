package minecraft

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/df-mc/go-playfab/v2"
	"github.com/df-mc/go-xsapi/v2"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/internal"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/service"
	"golang.org/x/oauth2"
)

type Dialer struct {
	GetClientData    func() login.ClientData
	EarlyConnHandler func(*Conn)

	ErrorLog *slog.Logger

	HTTPClient *http.Client

	ClientData login.ClientData

	IdentityData login.IdentityData

	TokenSource oauth2.TokenSource

	XBLClient *xsapi.Client

	PlayFabClient *playfab.Client

	PacketFunc func(header packet.Header, payload []byte, src, dst net.Addr)

	DownloadResourcePack func(id uuid.UUID, version string, current, total int) bool

	ResourcePackCache ResourcePackCache

	DisconnectOnUnknownPackets bool

	DisconnectOnInvalidPackets bool

	Protocol Protocol

	MaxDecompressedLen int

	FlushRate time.Duration

	EnableClientCache bool

	KeepXBLIdentityData bool

	EnableLegacyAuth bool

	DisablePortFollowing bool
}

func Dial(network, address string) (*Conn, error) {
	var d Dialer
	return d.Dial(network, address)
}

func DialTimeout(network, address string, timeout time.Duration) (*Conn, error) {
	var d Dialer
	return d.DialTimeout(network, address, timeout)
}

func DialContext(ctx context.Context, network, address string) (*Conn, error) {
	var d Dialer
	return d.DialContext(ctx, network, address)
}

func (d Dialer) Dial(network, address string) (*Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d Dialer) DialTimeout(network, address string, timeout time.Duration) (*Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d Dialer) DialContextNetwork(ctx context.Context, network Network, address string) (conn *Conn, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if d.ErrorLog == nil {
		d.ErrorLog = slog.New(internal.DiscardHandler{})
	}
	d.ErrorLog = d.ErrorLog.With("src", "dialer")
	if d.Protocol == nil {
		d.Protocol = DefaultProtocol
	}
	if d.FlushRate == 0 {
		d.FlushRate = time.Second / 20
	}
	if d.HTTPClient == nil {
		d.HTTPClient = http.DefaultClient
	}

	key, err := ecdsa.GenerateKey(elliptic.P384(), cryptorand.Reader)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("generating ECDSA key: %w", err)}
	}
	var (
		chainData, token string
		verifier         *oidc.IDTokenVerifier
	)
	if d.PlayFabClient != nil && d.TokenSource == nil && d.XBLClient == nil {
		return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: errors.New("PlayFabClient requires XBLClient or TokenSource for authenticated login")}
	}
	if d.TokenSource != nil || d.XBLClient != nil {
		ctx = auth.WithContextClient(ctx, d.HTTPClient)
		if d.XBLClient == nil {
			x, ok := d.TokenSource.(xsapi.TokenSource)
			if !ok {
				x = auth.ContextSession(ctx, d.TokenSource)
			}
			d.XBLClient, err = xsapi.ClientConfig{
				HTTPClient: d.HTTPClient,
				RTAMode:    xsapi.RTADisabled,
			}.New(ctx, x)
			if err != nil {
				return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("login to xbox live: %w", err)}
			}
			defer d.XBLClient.Close()
		}
		if !d.EnableLegacyAuth {
			e, err := authEnv(ctx)
			if err != nil {
				return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("request authorization environment: %w", err)}
			}
			verifier, err = e.VerifierContext(ctx)
			if err != nil {
				return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("create OIDC verifier: %w", err)}
			}

			m, ok := d.TokenSource.(MultiplayerTokenSource)
			if !ok {

				if d.PlayFabClient == nil {
					client, err := playfab.LoginWithXbox(ctx, e.PlayFabTitleID, d.XBLClient, playfab.ClientConfig{
						HTTPClient:    d.HTTPClient,
						CreateAccount: true,
					})
					if err != nil {
						return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("login to playfab: %w", err)}
					}
					defer client.Close()

					d.PlayFabClient = client
				}
				m = &multiplayerTokenSource{src: e.TokenSource(d.PlayFabClient, service.TokenConfig{}), env: e}
			}
			token, err = m.MultiplayerToken(ctx, &key.PublicKey)
			if err != nil {
				return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: err}
			}
		}
		chainData, err = auth.RequestMinecraftChain(ctx, d.XBLClient, key)
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("request Minecraft auth chain: %w", err)}
		}
		identityData, err := readChainIdentityData([]byte(chainData))
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: err}
		}
		d.IdentityData = identityData
	}

	if !d.DisablePortFollowing {
		if pong, err := network.PingContext(ctx, address); err == nil {
			address = addressWithPongPort(pong, address)
		}
	}

	var netConn net.Conn
	if i, ok := network.(identityDialer); ok && token != "" {
		netConn, err = i.DialContextIdentity(ctx, address, token, key)
	} else {
		netConn, err = network.DialContext(ctx, address)
	}
	if err != nil {
		return nil, err
	}

	conn = newConn(netConn, key, d.ErrorLog, d.Protocol, d.FlushRate, false)
	conn.pool = conn.proto.Packets(false)
	conn.identityData = d.IdentityData
	conn.clientData = d.ClientData
	if d.GetClientData != nil {
		conn.clientData = d.GetClientData()
	}
	conn.packetFunc = d.PacketFunc
	conn.downloadResourcePack = d.DownloadResourcePack
	conn.resourcePackCache = d.ResourcePackCache
	conn.cacheEnabled = d.EnableClientCache
	conn.disconnectOnInvalidPacket = d.DisconnectOnInvalidPackets
	conn.disconnectOnUnknownPacket = d.DisconnectOnUnknownPackets
	conn.maxDecompressedLen = d.MaxDecompressedLen

	defaultIdentityData(&conn.identityData)
	defaultClientData(address, conn.identityData.DisplayName, &conn.clientData)

	var request []byte
	if chainData == "" && token == "" {

		if !d.KeepXBLIdentityData {
			clearXBLIdentityData(&conn.identityData)
		}
		request = login.EncodeOffline(conn.identityData, conn.clientData, key, d.EnableLegacyAuth)
	} else {

		setAndroidData(&conn.clientData)

		request = login.Encode(chainData, conn.clientData, key, token, d.EnableLegacyAuth)
		identityData, _, _, _ := login.Parse(request, verifier)

		conn.identityData = identityData
	}

	readyForLogin, connected := make(chan struct{}), make(chan struct{})
	if d.EarlyConnHandler != nil {
		d.EarlyConnHandler(conn)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	go listenConn(conn, readyForLogin, connected, cancel)

	conn.expect(packet.IDNetworkSettings, packet.IDPlayStatus)
	if err := conn.WritePacket(&packet.RequestNetworkSettings{ClientProtocol: d.Protocol.ID()}); err != nil {
		return nil, conn.wrap(fmt.Errorf("send request network settings: %w", err), "dial")
	}
	_ = conn.Flush()

	select {
	case <-ctx.Done():
		return nil, conn.wrap(context.Cause(ctx), "dial")
	case <-conn.ctx.Done():
		return nil, conn.closeErr("dial")
	case <-readyForLogin:

		conn.expect(packet.IDServerToClientHandshake, packet.IDPlayStatus, packet.IDResourcePacksInfo)
		if err := conn.WritePacket(&packet.Login{ConnectionRequest: request, ClientProtocol: d.Protocol.ID()}); err != nil {
			return nil, conn.wrap(fmt.Errorf("send login: %w", err), "dial")
		}
		_ = conn.Flush()

		select {
		case <-ctx.Done():
			return nil, conn.wrap(context.Cause(ctx), "dial")
		case <-conn.ctx.Done():
			return nil, conn.closeErr("dial")
		case <-connected:

			return conn, nil
		}
	}
}

func (d Dialer) DialContext(ctx context.Context, network, address string) (conn *Conn, err error) {
	if d.ErrorLog == nil {
		d.ErrorLog = slog.New(internal.DiscardHandler{})
	}
	d.ErrorLog = d.ErrorLog.With("src", "dialer")
	n, ok := networkByID(network, d.ErrorLog)
	if !ok {
		return nil, &net.OpError{Op: "dial", Net: "minecraft", Err: fmt.Errorf("dial: no network under id %v", network)}
	}
	return d.DialContextNetwork(ctx, n, address)
}

func readChainIdentityData(chainData []byte) (login.IdentityData, error) {
	chain := struct{ Chain []string }{}
	if err := json.Unmarshal(chainData, &chain); err != nil {
		return login.IdentityData{}, fmt.Errorf("read chain: read json: %w", err)
	}
	data := chain.Chain[1]
	claims := struct {
		ExtraData login.IdentityData `json:"extraData"`
	}{}
	tok, err := jwt.ParseSigned(data, []jose.SignatureAlgorithm{jose.ES384})
	if err != nil {
		return login.IdentityData{}, fmt.Errorf("read chain: parse jwt: %w", err)
	}
	if err := tok.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return login.IdentityData{}, fmt.Errorf("read chain: read claims: %w", err)
	}
	if claims.ExtraData.Identity == "" {
		return login.IdentityData{}, fmt.Errorf("read chain: no extra data found")
	}
	return claims.ExtraData, nil
}

func listenConn(conn *Conn, readyForLogin, connected chan struct{}, cancel context.CancelCauseFunc) {
	defer func() {
		_ = conn.Close()
	}()
	cancelContext := true
	for {

		receiveErr := false
		if err := conn.dec.DecodeFunc(func(data []byte) error {
			loggedInBefore, readyToLoginBefore := conn.loggedIn, conn.readyToLogin
			if err := conn.receive(data); err != nil {
				receiveErr = true
				return err
			}
			if !readyToLoginBefore && conn.readyToLogin {

				readyForLogin <- struct{}{}
			}
			if !loggedInBefore && conn.loggedIn {

				cancelContext = false
				connected <- struct{}{}
			}
			return nil
		}); err != nil {
			if receiveErr || !errors.Is(err, net.ErrClosed) {
				if cancelContext {
					cancel(err)
				} else {
					conn.log.Error(err.Error())
				}
			}
			return
		}
	}
}

//go:embed skin_resource_patch.json
var skinResourcePatch []byte

//go:embed skin_geometry.json
var skinGeometry []byte

func serverAddress(address string) string {
	if !strings.Contains(address, "://") {
		return address
	}
	u, err := url.Parse(address)
	if err != nil || u.Port() == "" {
		return address
	}
	return address + ":" + u.Port()
}

func defaultClientData(address, username string, d *login.ClientData) {
	d.ServerAddress = serverAddress(address)
	d.ThirdPartyName = username
	if d.DeviceOS == 0 {
		d.DeviceOS = protocol.DeviceAndroid
	}
	if d.DefaultInputMode == 0 {
		d.DefaultInputMode = packet.InputModeTouch
	}
	if d.CurrentInputMode == 0 {
		d.CurrentInputMode = packet.InputModeTouch
	}
	if d.GameVersion == "" {
		d.GameVersion = protocol.CurrentVersion
	}
	if d.ClientRandomID == 0 {
		d.ClientRandomID = rand.Int63()
	}
	if d.DeviceID == "" {
		d.DeviceID = d.ExpectedDeviceIDFormat().Generate()
	}
	if d.LanguageCode == "" {
		d.LanguageCode = "en_GB"
	}
	if d.PlayFabID == "" {
		id := make([]byte, 8)
		_, _ = cryptorand.Read(id)
		d.PlayFabID = hex.EncodeToString(id)
	}

	if d.AnimatedImageData == nil {
		d.AnimatedImageData = make([]login.SkinAnimation, 0)
	}
	if d.PersonaPieces == nil {
		d.PersonaPieces = make([]login.PersonaPiece, 0)
	}
	if d.PieceTintColours == nil {
		d.PieceTintColours = make([]login.PersonaPieceTintColour, 0)
	}
	if d.SelfSignedID == "" {
		d.SelfSignedID = uuid.New().String()
	}
	if d.SkinID == "" {
		d.SkinID = uuid.New().String()
	}
	if d.SkinData == "" {
		d.SkinData = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0, 0, 0, 255}, 32*64))
		d.SkinImageHeight = 32
		d.SkinImageWidth = 64
	}
	if d.SkinResourcePatch == "" {
		d.SkinResourcePatch = base64.StdEncoding.EncodeToString(skinResourcePatch)
	}
	if d.SkinGeometry == "" {
		d.SkinGeometry = base64.StdEncoding.EncodeToString(skinGeometry)
	}
	if d.SkinGeometryVersion == "" {
		d.SkinGeometryVersion = base64.StdEncoding.EncodeToString([]byte("0.0.0"))
	}
}

func setAndroidData(data *login.ClientData) {
	data.DeviceOS = protocol.DeviceAndroid
	data.GameVersion = protocol.CurrentVersion
}

func clearXBLIdentityData(data *login.IdentityData) {
	data.XUID = ""
	data.TitleID = ""
}

func defaultIdentityData(data *login.IdentityData) {
	if data.Identity == "" {
		data.Identity = uuid.New().String()
	}
	if data.DisplayName == "" {
		data.DisplayName = "Steve"
	}
}

func splitPong(s string) []string {
	var runes []rune
	var tokens []string
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\\':
			inEscape = true
		case r == ';':
			tokens = append(tokens, string(runes))
			runes = runes[:0]
		case inEscape:
			inEscape = false
			fallthrough
		default:
			runes = append(runes, r)
		}
	}
	return append(tokens, string(runes))
}

func addressWithPongPort(pong []byte, address string) string {
	frag := splitPong(string(pong))
	if len(frag) > 10 {
		portStr := frag[10]
		port, err := strconv.Atoi(portStr)

		if err != nil || port == 19132 {
			return address
		}

		addressParts := strings.Split(address, ":")
		address = strings.Join(strings.Split(address, ":")[:len(addressParts)-1], ":")
		return address + ":" + portStr
	}
	return address
}
