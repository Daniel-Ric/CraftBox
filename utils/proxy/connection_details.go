package proxy

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	nethernet "github.com/df-mc/go-nethernet"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func connectionAddress(address net.Addr) string {
	if peer, ok := address.(*nethernet.Addr); ok {
		if peer == nil || peer.SelectedCandidate == nil {
			return "Not available"
		}
		candidate := peer.SelectedCandidate
		if net.ParseIP(candidate.Address) == nil || candidate.Port == 0 {
			return "Not available"
		}
		return net.JoinHostPort(candidate.Address, strconv.Itoa(int(candidate.Port))) + " (ICE)"
	}
	return connectionEndpoint(address.String())
}

// connectionEndpoint deliberately omits URL credentials, paths and query strings.
func connectionEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Not available"
	}
	if !strings.Contains(value, "://") {
		value = "udp://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || strings.ContainsAny(u.Host, "§\r\n\t") {
		return "Not available"
	}
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "udp" {
		return "Not available"
	}
	return u.Scheme + "://" + u.Host
}

// ConnectionDetailsText reports known local state without contacting any service.
func (s *Session) ConnectionDetailsText() string {
	state := "Waiting for server"
	if s.Server != nil {
		state = "Upstream connection established"
	}
	if s.spawned.Load() {
		state = "In game"
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		state = "Session ended"
	}
	lines := []string{"§aConnection details§r", "§7Status: §f" + state,
		fmt.Sprintf("§7Minecraft: §f%s §7· Protocol: §f%d", protocol.CurrentVersion, protocol.CurrentProtocol)}
	if s.connectInfo != nil {
		if p, err := s.connectInfo.Parsed(); err == nil {
			lines = append(lines, "§7Source: §f"+p.Kind.String())
			if p.Address != "" {
				lines = append(lines, "§7Destination: §f"+connectionEndpoint(p.Address))
			}
		}
	}
	appendAddress := func(label string, address net.Addr) {
		if address == nil {
			return
		}
		lines = append(lines, "§7"+label+": §f"+connectionAddress(address))
	}
	if s.Server != nil {
		addr := s.Server.RemoteAddr()
		appendAddress("Remote endpoint", addr)
		if addr != nil {
			network := "Other"
			switch strings.ToLower(addr.Network()) {
			case "udp", "udp4", "udp6", "raknet":
				network = "RakNet / UDP"
			case "nethernet", "webrtc", "p2p":
				network = "NetherNet / WebRTC"
			}
			lines = append(lines, "§7Transport: §f"+network)
		}
		if latency := s.Server.Latency(); latency > 0 {
			lines = append(lines, fmt.Sprintf("§7Measured RTT: §f%s", latency.Round(time.Millisecond)))
		}
	}
	if s.Client != nil {
		appendAddress("Local proxy endpoint", s.Client.LocalAddr())
	}
	return strings.Join(lines, "§r\n") + "§r"
}
