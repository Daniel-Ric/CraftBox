package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/df-mc/go-xsapi/v2"
	"github.com/df-mc/go-xsapi/v2/mpsd"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/auth"
)


type ClientConfig struct {



	Log *slog.Logger
}


func (conf ClientConfig) New(client *xsapi.Client) *Client {
	if conf.Log == nil {
		conf.Log = slog.Default().With("src", "minecraft/p2p")
	}
	return &Client{
		client: client,
		conf:   conf,
	}
}


func NewClient(client *xsapi.Client) *Client {
	var c ClientConfig
	return c.New(client)
}


type Client struct {
	client *xsapi.Client
	conf   ClientConfig
}


func (c *Client) Worlds(ctx context.Context) ([]World, error) {
	activities, err := c.client.MPSD().Activities(ctx, auth.ServiceConfigID)
	if err != nil {
		return nil, err
	}
	worlds := make([]World, 0, len(activities))
	for _, activity := range activities {
		var world World
		if err := json.Unmarshal(activity.CustomProperties, &world); err != nil {
			c.conf.Log.Error("error decoding world data",
				slog.Any("error", err),
				slog.String("customProperties", string(activity.CustomProperties)),
			)
			continue
		}
		world.client, world.handleID = c, activity.ID
		worlds = append(worlds, world)
	}
	return worlds, nil
}



func (c *Client) Join(ctx context.Context, handleID uuid.UUID) (_ *Session, err error) {
	s, err := c.client.MPSD().Join(ctx, handleID, mpsd.JoinConfig{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err2 := s.Close(); err2 != nil {
				err = errors.Join(err, fmt.Errorf("cleanup session: %w", err2))
			}
		}
	}()
	session := &Session{
		client:  c,
		session: s,

		nonceReceived: make(chan struct{}),

		log: c.conf.Log.With("sessionRef", s.Reference()),
	}
	if err := session.updateWorldData(s.Properties().Custom); err != nil {
		return nil, err
	}
	s.Handle(&handler{session})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-session.nonceReceived:
		return session, nil
	}
}



func (w *World) Join(ctx context.Context) (*Session, error) {
	if w.client == nil {
		return nil, errors.New("minecraft/p2p: client is not bound to world")
	}
	return w.client.Join(ctx, w.handleID)
}



type Session struct {
	client  *Client
	session *mpsd.Session

	world      World
	connection Connection
	worldMu    sync.RWMutex

	nonce         string
	nonceReceived chan struct{}

	log *slog.Logger
}




func (s *Session) World() World {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	return s.world.clone()
}




func (s *Session) updateWorldData(custom json.RawMessage) error {
	var world World
	if err := json.Unmarshal(custom, &world); err != nil {
		return fmt.Errorf("decode custom properties: %w", err)
	}
	if len(world.SupportedConnections) == 0 {
		return errors.New("world has no supported connections")
	}

	s.worldMu.Lock()
	defer s.worldMu.Unlock()

	s.world = world
	s.connection = world.SupportedConnections[0]

	if s.nonce == "" {


		xuid := s.client.client.UserInfo().XUID
		if nonces := world.Nonces; nonces != nil {
			nonce, ok := nonces[xuid]
			if ok {
				if nonce == "" {
					return errors.New("host published empty-nonce for caller")
				}
				s.log.Debug("received nonce from host", slog.String("xuid", xuid), slog.String("nonce", nonce))
				s.nonce = nonce
				close(s.nonceReceived)
			}
		}
	}
	return nil
}




func (s *Session) Connection() Connection {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	return s.connection
}




func (s *Session) Nonce() string {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	return s.nonce
}


func (s *Session) Close() error {
	return s.session.Close()
}


func (s *Session) CloseContext(ctx context.Context) error {
	return s.session.CloseContext(ctx)
}


type handler struct {
	*Session
}


func (h *handler) HandleSessionChange(session *mpsd.Session) {
	if err := h.updateWorldData(session.Properties().Custom); err != nil {
		h.log.Error("error updating world data", slog.Any("error", err))
	}
}
