package minecraft

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/discovery"
	"github.com/pion/webrtc/v4"
)

func (l *directNetherNetListener) listenDiscovery(n DirectNetherNet, address string, settings webrtc.SettingEngine) error {
	if n.DisableLANDiscovery {
		return nil
	}
	discoveryAddress := n.DiscoveryAddress
	if discoveryAddress == "" {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		discoveryAddress = net.JoinHostPort(host, strconv.Itoa(discovery.DefaultPort))
	}
	d, err := (discovery.ListenConfig{Log: n.Log}).Listen(discoveryAddress)
	if err != nil {
		if n.DiscoveryAddress != "" {
			return fmt.Errorf("listen NetherNet LAN discovery: %w", err)
		}
		logger := n.Log
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn("NetherNet LAN discovery unavailable; direct connections remain available", "address", discoveryAddress, "error", err)
		return nil
	}
	l.discovery = d
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate LAN nonce: %w", err)
	}
	l.nonce = hex.EncodeToString(nonce)
	l.lanListener, err = (nethernet.ListenConfig{AllowAnonymous: true, Log: n.Log, IssueServerIdentity: n.IssueServerIdentity, API: webrtc.NewAPI(webrtc.WithSettingEngine(settings))}).Listen(d)
	if err != nil {
		return fmt.Errorf("listen NetherNet LAN: %w", err)
	}
	return nil
}

func (l *directNetherNetListener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	case c := <-l.incoming:
		return c, nil
	}
}

func (l *directNetherNetListener) acceptFrom(source net.Listener) {
	for {
		c, err := source.Accept()
		if err != nil {
			go l.Close()
			return
		}
		select {
		case l.incoming <- c:
		case <-l.closed:
			_ = c.Close()
			return
		}
	}
}

func (l *directNetherNetListener) PongData(data []byte) {
	l.Listener.PongData(data)
	if l.discovery == nil {
		return
	}
	parts := strings.Split(string(data), ";")
	if len(parts) < 9 {
		return
	}
	number := func(index int) int32 { v, _ := strconv.ParseInt(parts[index], 10, 32); return int32(v) }
	gameType := int32(discovery.GameTypeSurvival)
	switch strings.ToLower(parts[8]) {
	case "creative":
		gameType = discovery.GameTypeCreative
	case "adventure":
		gameType = discovery.GameTypeAdventure
	}
	l.discovery.ServerData(&discovery.ServerData{
		ServerName: parts[1], Protocol: number(2), Version: parts[3], LevelName: parts[7],
		PlayerCount: number(4), MaxPlayerCount: number(5), GameType: gameType,
		AcceptsOnlineAuth: true, AcceptsSelfSignedAuth: true, Nonce: l.nonce, ConnectionType: 4,
	})
}
