package connectinfo

import (
	"context"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"strings"
)

type Kind string

const (
	KindServer     Kind = "server"
	KindRealm      Kind = "realm"
	KindExperience Kind = "experience"
	KindReplay     Kind = "replay"
	KindXbox       Kind = "xbox"
	KindMessaging  Kind = "messaging"
	KindSignaling  Kind = "signaling"
	KindGathering  Kind = "gathering"
)

func (k Kind) String() string { return string(k) }

type ParsedConnectInfo struct {
	Kind                                                                          Kind
	RealmName, Experience, Replay, Address, Xbox, Messaging, Gathering, Signaling string
}

func (c *ConnectInfo) Parsed() (*ParsedConnectInfo, error) {
	p, err := parseConnectInfo(c.Value)
	if err != nil {
		return nil, err
	}
	out := &ParsedConnectInfo{}
	switch {
	case p.realmName != "":
		out.Kind, out.RealmName = KindRealm, p.realmName
	case p.experience != "":
		out.Kind, out.Experience = KindExperience, p.experience
	case p.replayName != "":
		out.Kind, out.Replay = KindReplay, p.replayName
	case p.netherNet != "":
		out.Kind, out.Xbox = KindXbox, p.netherNet
	case p.jsonRPC != "":
		out.Kind, out.Messaging = KindMessaging, p.jsonRPC
	case p.signaling != "":
		out.Kind, out.Signaling = KindSignaling, p.signaling
	case p.gatheringName != "":
		out.Kind, out.Gathering = KindGathering, p.gatheringName
	default:
		out.Kind, out.Address = KindServer, p.serverAddress
	}
	return out, nil
}

func realmRoute(address realms.RealmAddress) (string, error) {
	value := strings.TrimSpace(address.Address)
	if value == "" {
		return "", fmt.Errorf("Realm returned an empty address")
	}
	switch p := realms.ParseNetworkProtocol(string(address.NetworkProtocol)); p {
	case realms.NetworkProtocolNetherNetJSONRPC:
		return "nethernet-jsonrpc:" + value, nil
	case realms.NetworkProtocolNetherNet:
		return "nethernet-signaling:" + value, nil
	case "", realms.NetworkProtocolDefault:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported Realm network protocol %q", p)
	}
}
func (c *ConnectInfo) Kind() (Kind, error) {
	p, err := c.Parsed()
	if err != nil {
		return "", err
	}
	return p.Kind, nil
}
func (c *ConnectInfo) Realm(ctx context.Context) (*realms.Realm, error) {
	p, err := c.Parsed()
	if err != nil {
		return nil, err
	}
	if p.Kind != KindRealm {
		return nil, nil
	}
	return c.getRealm(ctx, p.RealmName)
}
