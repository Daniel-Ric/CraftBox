package proxy

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/messaging"
	"github.com/df-mc/go-nethernet"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/realms"
)

type signalingCloser interface {
	nethernet.Signaling
	io.Closer
}

func (s *Session) joinP2P(ctx context.Context, id uuid.UUID) (*p2p.Session, error) {
	if s.p2pSession != nil {
		return s.p2pSession, nil
	}
	if s.connectInfo.Account == nil {
		return nil, fmt.Errorf("no account")
	}
	client, err := s.connectInfo.Account.XBLClient(ctx)
	if err != nil {
		return nil, err
	}
	session, err := p2p.NewClient(client).Join(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("joining session: %w", err)
	}
	s.p2pSession = session
	return session, nil
}

func (s *Session) resolveServerName(ctx context.Context) error {
	parsed, err := s.connectInfo.Parsed()
	if err != nil {
		return err
	}
	if parsed.Kind == connectinfo.KindXbox {
		id, err := uuid.Parse(parsed.Xbox)
		if err != nil {
			return err
		}
		session, err := s.joinP2P(ctx, id)
		if err != nil {
			return err
		}
		world := session.World()
		s.ServerName = world.WorldName
		if s.ServerName == "" {
			s.ServerName = world.HostName
		}
		if s.ServerName == "" {
			s.ServerName = id.String()
		}
		return nil
	}
	s.ServerName, err = s.connectInfo.Name(ctx)
	if err != nil {
		return err
	}
	if parsed.Kind == connectinfo.KindServer {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		network := minecraft.DirectNetherNet{}
		if s.hostnameAlias != nil {
			network.HTTPClient = s.hostnameAlias.Client
		}
		if pong, err := network.PingContext(pingCtx, parsed.Address); err == nil {
			if name := minecraft.ParsePongData(pong).ServerName; name != "" {
				s.ServerName = name
			}
		}
	}
	return nil
}

func realmSignalingProtocol(address realms.RealmAddress) (realms.NetworkProtocol, error) {
	p := realms.ParseNetworkProtocol(string(address.NetworkProtocol))
	switch p {
	case "", realms.NetworkProtocolDefault:
		return realms.NetworkProtocolDefault, nil
	case realms.NetworkProtocolNetherNet, realms.NetworkProtocolNetherNetJSONRPC:
		return p, nil
	default:
		return "", fmt.Errorf("unsupported Realm network protocol %q", address.NetworkProtocol)
	}
}

func (s *Session) openSignaling(ctx context.Context, jsonRPC bool) (signalingCloser, error) {
	if s.connectInfo.Account == nil {
		return nil, fmt.Errorf("no account")
	}
	src, err := s.connectInfo.Account.TokenSource(ctx)
	if err != nil {
		return nil, err
	}
	if jsonRPC {
		return (messaging.Dialer{}).DialContext(ctx, src)
	}
	return (signaling.Dialer{}).DialContext(ctx, src)
}

func (s *Session) resolveServerAddress(ctx context.Context, parsed *connectinfo.ParsedConnectInfo) (address, nonce string, sig signalingCloser, err error) {
	switch parsed.Kind {
	case connectinfo.KindServer:
		return parsed.Address, "", nil, nil
	case connectinfo.KindRealm:
		realm, err := s.connectInfo.Realm(ctx)
		if err != nil {
			return "", "", nil, err
		}
		if realm == nil {
			return "", "", nil, fmt.Errorf("Realm not found")
		}
		destination, err := realm.Address(ctx)
		if err != nil {
			return "", "", nil, err
		}
		proto, err := realmSignalingProtocol(destination)
		if err != nil {
			return "", "", nil, err
		}
		if destination.Address == "" {
			return "", "", nil, fmt.Errorf("Realm returned an empty address")
		}
		if proto != realms.NetworkProtocolDefault {
			sig, err = s.openSignaling(ctx, proto == realms.NetworkProtocolNetherNetJSONRPC)
		}
		return destination.Address, "", sig, err
	case connectinfo.KindMessaging:
		sig, err = s.openSignaling(ctx, true)
		return parsed.Messaging, "", sig, err
	case connectinfo.KindSignaling:
		sig, err = s.openSignaling(ctx, false)
		return parsed.Signaling, "", sig, err
	case connectinfo.KindXbox:
		id, err := uuid.Parse(parsed.Xbox)
		if err != nil {
			return "", "", nil, err
		}
		session, err := s.joinP2P(ctx, id)
		if err != nil {
			return "", "", nil, err
		}
		switch session.Connection().Type {
		case p2p.ConnectionTypeSignalingOverJSONRPC:
			sig, err = s.openSignaling(ctx, true)
		case p2p.ConnectionTypeSignalingOverWebSocket:
			sig, err = s.openSignaling(ctx, false)
		default:
			return "", "", nil, fmt.Errorf("unsupported Xbox signaling type %d", session.Connection().Type)
		}
		return session.Connection().Address(), session.Nonce(), sig, err
	case connectinfo.KindExperience, connectinfo.KindGathering:
		resolved, err := s.connectInfo.Address(ctx)
		if err != nil {
			return "", "", nil, err
		}
		next, err := (&connectinfo.ConnectInfo{Value: resolved}).Parsed()
		if err != nil {
			return "", "", nil, err
		}
		if next.Kind != connectinfo.KindMessaging && next.Kind != connectinfo.KindServer {
			return "", "", nil, fmt.Errorf("unsupported allocated destination %q", next.Kind)
		}
		return s.resolveServerAddress(ctx, next)
	default:
		return "", "", nil, fmt.Errorf("cannot connect to source kind %q", parsed.Kind)
	}
}
