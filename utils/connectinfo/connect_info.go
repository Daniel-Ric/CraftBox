package connectinfo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/franchise/gatherings"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/realms"
)

type ErrNotFound struct {
	Name string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found", e.Name)
}

type ConnectInfo struct {
	Value   string
	Account *auth.Account

	gathering  *gatherings.Gathering
	realm      *realms.Realm
	experience *gatherings.FeaturedServer
	world      *p2p.World
}

func (c *ConnectInfo) getGathering(ctx context.Context, name string) (*gatherings.Gathering, error) {
	if c.gathering != nil && c.gathering.Title == name {
		return c.gathering, nil
	}
	if c.Account == nil {
		return nil, errors.New("no account")
	}
	gatheringsService, err := c.Account.Gatherings(ctx)
	if err != nil {
		return nil, err
	}
	mcToken, err := c.Account.MCToken(ctx)
	if err != nil {
		return nil, err
	}
	gatherings, err := gatheringsService.GetGatherings(ctx, mcToken)
	if err != nil {
		return nil, err
	}
	for _, gathering := range gatherings {
		title := strings.ToLower(gathering.Title)
		id := strings.ToLower(gathering.GatheringID)
		if strings.HasPrefix(title, name) || strings.HasPrefix(id, name) {
			return gathering, nil
		}
	}
	return nil, &ErrNotFound{Name: name}
}

func (c *ConnectInfo) getExperience(ctx context.Context, name string) (*gatherings.FeaturedServer, error) {
	if c.experience != nil {
		if c.experience.ExperienceId == name {
			return c.experience, nil
		}
		if strings.EqualFold(c.experience.Name, name) {
			return c.experience, nil
		}
	}
	if c.Account == nil {
		return nil, errors.New("no account")
	}
	gatherings, err := c.Account.Gatherings(ctx)
	if err != nil {
		return nil, err
	}
	mcToken, err := c.Account.MCToken(ctx)
	if err != nil {
		return nil, err
	}
	featuredServers, err := gatherings.GetCatalog(ctx, mcToken)
	if err != nil {
		return nil, err
	}
	for _, experience := range featuredServers {
		if experience.ExperienceId == name || strings.EqualFold(experience.Name, name) {
			c.experience = &experience
			return c.experience, nil
		}
	}
	return nil, &ErrNotFound{Name: name}
}

func (c *ConnectInfo) getRealm(ctx context.Context, name string) (*realms.Realm, error) {
	name = strings.ToLower(name)
	if c.realm != nil && (strings.EqualFold(c.realm.Name, name) || strconv.Itoa(c.realm.ID) == name) {
		return c.realm, nil
	}
	if c.Account == nil {
		return nil, errors.New("no account")
	}
	realmsClient, err := c.Account.Realms(ctx)
	if err != nil {
		return nil, err
	}
	realms, err := realmsClient.Realms(ctx)
	if err != nil {
		return nil, err
	}
	for _, realm := range realms {
		lowerName := strings.ToLower(realm.Name)
		if strings.HasPrefix(lowerName, name) || strconv.Itoa(realm.ID) == name {
			c.realm = &realm
			return &realm, nil
		}
	}
	return nil, &ErrNotFound{Name: name}
}

func (c *ConnectInfo) Name(ctx context.Context) (string, error) {
	info, err := parseConnectInfo(c.Value)
	if err != nil {
		return "", err
	}
	if info.netherNet != "" {
		return info.netherNet, nil
	}
	if info.jsonRPC != "" {
		return info.jsonRPC, nil
	}
	if info.signaling != "" {
		return info.signaling, nil
	}
	if info.serverAddress != "" {
		if u, err := url.Parse(info.serverAddress); err == nil && u.Hostname() != "" {
			host := u.Hostname()
			if port := u.Port(); port != "" && port != "80" && port != "443" && port != "19132" {
				host += "_" + port
			}
			return host, nil
		}
		host, port, err := net.SplitHostPort(info.serverAddress)
		if err != nil {
			host = info.serverAddress
		} else if port != "19132" {
			host += "_" + port
		}

		return host, nil
	}
	if info.replayName != "" {
		return filepath.Base(info.replayName), nil
	}
	if info.realmName != "" {
		realm, err := c.getRealm(ctx, info.realmName)
		if err != nil {
			return "", err
		}
		return realm.Name, nil
	}
	if info.gatheringName != "" {
		gathering, err := c.getGathering(ctx, info.gatheringName)
		if err != nil {
			return "", err
		}
		return gathering.Title, nil
	}
	if info.experience != "" {
		exp, err := c.getExperience(ctx, info.experience)
		var notFound *ErrNotFound
		switch {
		case errors.As(err, &notFound):
			return info.experience, nil
		case err != nil:
			return "", err
		default:
			return exp.Name, nil
		}
	}
	return "invalid", nil
}

func (c *ConnectInfo) Address(ctx context.Context) (string, error) {
	info, err := parseConnectInfo(c.Value)
	if err != nil {
		return "", err
	}
	if info.netherNet != "" {
		return "nethernet:" + info.netherNet, nil
	}
	if info.jsonRPC != "" {
		return "nethernet-jsonrpc:" + info.jsonRPC, nil
	}
	if info.signaling != "" {
		return "nethernet-signaling:" + info.signaling, nil
	}
	if info.serverAddress != "" {
		return info.serverAddress, nil
	}
	if info.replayName != "" {
		return info.replayName, nil
	}
	if info.realmName != "" {
		realm, err := c.getRealm(ctx, info.realmName)
		if err != nil {
			return "", err
		}
		address, err := realm.Address(ctx)
		if err != nil {
			return "", err
		}
		return realmRoute(address)
	}
	if info.gatheringName != "" {
		gathering, err := c.getGathering(ctx, info.gatheringName)
		if err != nil {
			return "", err
		}
		mcToken, err := c.Account.MCToken(ctx)
		if err != nil {
			return "", err
		}
		return gathering.Address(ctx, mcToken)
	}

	if info.experience != "" {
		assignment, err := c.ResolveExperience(ctx)
		if err != nil {
			return "", err
		}
		return assignment.Address()
	}
	return "", errors.New("invalid address")
}

func (c *ConnectInfo) ResolveExperience(ctx context.Context) (*gatherings.ExperienceAssignment, error) {
	info, err := parseConnectInfo(c.Value)
	if err != nil {
		return nil, err
	}
	if info.experience != "" {
		expId, err := uuid.Parse(info.experience)
		if err != nil {
			experience, err := c.getExperience(ctx, info.experience)
			if err != nil {
				return nil, err
			}
			expId, err = uuid.Parse(experience.ExperienceId)
			if err != nil {
				return nil, err
			}
		}
		gatheringsService, err := c.Account.Gatherings(ctx)
		if err != nil {
			return nil, err
		}
		mcToken, err := c.Account.MCToken(ctx)
		if err != nil {
			return nil, err
		}
		address, err := gatheringsService.ResolveExperience(ctx, mcToken, expId)
		if err != nil {
			return nil, fmt.Errorf("JoinExperience: %w", err)
		}
		return address, nil
	}
	return nil, errors.New("not an experience source")
}

func (c *ConnectInfo) IsReplay() bool {
	info, err := parseConnectInfo(c.Value)
	return err == nil && info.replayName != ""
}

func (c *ConnectInfo) SetRealm(realm *realms.Realm) {
	c.Value = "realm:" + realm.Name
	c.realm = realm
}

func (c *ConnectInfo) SetGathering(gathering *gatherings.Gathering) {
	c.Value = "gathering:" + gathering.Title
	c.gathering = gathering
}

func (c *ConnectInfo) SetFeaturedServer(server *gatherings.FeaturedServer) {
	if server.Address != "" {
		c.Value = server.Address
		c.experience = nil
	} else if server.ExperienceId != "" {
		c.Value = "experience:" + server.Name
		c.experience = server
	} else {
		c.Value = server.Address
	}
}

func (c *ConnectInfo) SetWorld(world *p2p.World) {
	c.Value = "nethernet:" + world.HandleID().String()
	c.world = world
}

type parsedConnectInfo struct {
	jsonRPC       string
	signaling     string
	gatheringName string
	realmName     string
	replayName    string
	experience    string
	serverAddress string
	netherNet     string
}

func parseConnectInfo(value string) (*parsedConnectInfo, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("empty source")
	}
	if strings.HasPrefix(value, "nethernet-signaling:") {
		id := strings.TrimPrefix(value, "nethernet-signaling:")
		if id == "" || strings.ContainsAny(id, " /\\\t\r\n") {
			return nil, errors.New("invalid NetherNet signaling ID")
		}
		return &parsedConnectInfo{signaling: id}, nil
	}
	if strings.HasPrefix(value, "nethernet-jsonrpc:") {
		id, err := uuid.Parse(strings.TrimPrefix(value, "nethernet-jsonrpc:"))
		if err != nil || id == uuid.Nil {
			return nil, errors.New("invalid NetherNet messaging ID")
		}
		return &parsedConnectInfo{jsonRPC: id.String()}, nil
	}
	if gatheringRegex.MatchString(value) {
		p := regexGetParams(gatheringRegex, value)
		input := strings.ToLower(p["Title"])
		return &parsedConnectInfo{gatheringName: input}, nil
	}

	if experienceRegex.MatchString(value) {
		p := regexGetParams(experienceRegex, value)
		return &parsedConnectInfo{experience: p["ID"]}, nil
	}

	if realmRegex.MatchString(value) {
		p := regexGetParams(realmRegex, value)
		input := strings.ToLower(p["Name"])
		return &parsedConnectInfo{realmName: input}, nil
	}

	if pcapRegex.MatchString(value) {
		p := regexGetParams(pcapRegex, value)
		input := p["Filename"]
		return &parsedConnectInfo{replayName: input}, nil
	}

	if strings.HasPrefix(value, "nethernet:") || strings.HasPrefix(value, "xbox:") {
		_, raw, _ := strings.Cut(value, ":")
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("nethernet: %s", err)
		}
		return &parsedConnectInfo{netherNet: id.String()}, nil
	}
	for _, prefix := range []string{"realm:", "experience:", "gathering:"} {
		if strings.HasPrefix(value, prefix) {
			return nil, fmt.Errorf("empty source %q", prefix)
		}
	}
	serverAddress, err := serverAddress(value)
	if err != nil {
		return nil, err
	}
	return &parsedConnectInfo{serverAddress: serverAddress}, nil
}
