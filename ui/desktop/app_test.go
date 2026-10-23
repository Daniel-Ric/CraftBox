package desktop

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestResourcePackProgressAcceptsVersionedTransferID(t *testing.T) {
	a := NewApp()
	id := uuid.New()
	a.HandleEvent(&messages.EventInitialPacksInfo{Packs: []protocol.TexturePackInfo{{UUID: id, Size: 1000}}})
	a.HandleEvent(&messages.EventPackDownloadProgress{ID: id.String() + "_1.0.0", BytesAdded: 250})
	if got := a.Snapshot().Packs[0].Received; got != 250 {
		t.Fatalf("download progress lost: %d", got)
	}
	a.HandleEvent(&messages.EventFinishedPack{ID: id.String(), Name: "Completed"})
	p := a.Snapshot().Packs[0]
	if !p.Done || p.Size != 1000 || p.Received != 1000 || len(a.Snapshot().Packs) != 1 {
		t.Fatalf("completion mismatch: %+v", p)
	}
}

func TestCancelLoginCancelsAutomaticSessionAuthentication(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	if err := a.StartCommand(StartRequest{Command: c.Name(), Target: "localhost"}); err != nil {
		t.Fatal(err)
	}
	defer a.StopCommand()
	a.HandleEvent(&messages.EventDisplayAuthCode{AuthCode: "test-code", URI: "https://example.test"})
	a.CancelLogin()
	select {
	case <-c.stopped:
	case <-time.After(time.Second):
		t.Fatal("automatic authentication session was not cancelled")
	}
}

func TestNewSessionClearsOldMapAndPublishesReset(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	a.state.Revision = 20
	a.tiles[[2]int32{1, 2}] = Tile{X: 1, Z: 2}
	reset := false
	a.emit = func(name string, _ any) {
		if name == "desktop:reset-map" {
			reset = true
		}
	}
	if err := a.StartCommand(StartRequest{Command: c.Name(), Target: "localhost"}); err != nil {
		t.Fatal(err)
	}
	defer a.StopCommand()
	if !reset || len(a.GetMapTiles()) != 0 || a.Snapshot().Revision <= 20 {
		t.Fatal("new session did not clear the map and preserve event ordering")
	}
}

func TestAuthenticationBlocksCommandStartup(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	a.state.Authenticating = true
	if err := a.StartCommand(StartRequest{Command: c.Name(), Target: "localhost"}); err == nil {
		a.StopCommand()
		t.Fatal("command overlapped authentication")
	}
	if !a.Snapshot().Authenticating {
		t.Fatal("authentication state was discarded")
	}
}

func TestUnsignedNamedAccountDoesNotFallBackToDefaultLogin(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	a.state.Account = "named-account"
	if err := a.StartCommand(StartRequest{Command: c.Name(), Target: "localhost"}); err == nil {
		a.StopCommand()
		t.Fatal("unsigned named account could fall back to default")
	}
}

func TestPackWithoutIconDoesNotCrash(t *testing.T) {
	var icon *image.RGBA
	a := NewApp()
	if err := a.HandleEvent(&messages.EventFinishedPack{ID: "test", Name: "Pack", Icon: icon}); err != nil {
		t.Fatal(err)
	}
	if len(a.Snapshot().Packs) != 1 {
		t.Fatal("pack without icon was lost")
	}
}

func TestLogHistoryIsBufferedAndBounded(t *testing.T) {
	a := NewApp()
	for i := 0; i < 2005; i++ {
		a.appendLog(LogEntry{Message: "message", Source: "proxy"})
	}
	history := a.GetLogs()
	if len(history) != 2000 {
		t.Fatalf("log count: %d", len(history))
	}
	history[0].Source = "changed"
	if a.GetLogs()[0].Source != "proxy" {
		t.Fatal("log snapshot shared mutable storage")
	}
}

type testSettings struct {
	Address *connectinfo.ConnectInfo `flag:"address"`
	Save    bool                     `flag:"save" default:"true"`
	Radius  int                      `flag:"radius" default:"8"`
}
type testCommand struct {
	started chan *testSettings
	stopped chan struct{}
}

func (*testCommand) Name() string        { return "desktop-test" }
func (*testCommand) Description() string { return "test" }
func (*testCommand) Settings() any       { return &testSettings{} }
func (c *testCommand) Run(ctx context.Context, settings any) error {
	c.started <- settings.(*testSettings)
	<-ctx.Done()
	close(c.stopped)
	return ctx.Err()
}
func TestCommandSettingsDefaultsAndCancellation(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	a.ctx = context.Background()
	err := a.StartCommand(StartRequest{Command: c.Name(), Target: "localhost:19132", Values: map[string]string{"radius": "12"}})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-c.started:
		if !s.Save || s.Radius != 12 || s.Address.Value != "localhost:19132" {
			t.Fatalf("settings: %+v", s)
		}
	case <-time.After(time.Second):
		t.Fatal("command did not start")
	}
	if err := a.StartCommand(StartRequest{Command: c.Name(), Target: "other:19132"}); err == nil {
		t.Fatal("overlapping session accepted")
	}
	a.StopCommand()
	select {
	case <-c.stopped:
	case <-time.After(time.Second):
		t.Fatal("command was not cancelled")
	}
}
func TestInvalidSettingsDoNotStartCommand(t *testing.T) {
	c := &testCommand{started: make(chan *testSettings, 1), stopped: make(chan struct{})}
	commands.Registered[c.Name()] = c
	defer delete(commands.Registered, c.Name())
	a := NewApp()
	a.ctx = context.Background()
	for _, r := range []StartRequest{
		{Command: c.Name(), Target: "localhost", Values: map[string]string{"radius": "invalid"}},
		{Command: c.Name()},
		{Command: c.Name(), Target: "localhost", Values: map[string]string{"missing": "true"}},
	} {
		if err := a.StartCommand(r); err == nil {
			t.Fatalf("accepted invalid request: %+v", r)
		}
	}
	if len(c.started) != 0 {
		t.Fatal("invalid request executed")
	}
}
func TestEventsBeforeFrontendMountRemainAvailable(t *testing.T) {
	a := NewApp()
	a.HandleEvent(&messages.EventConnectStateUpdate{State: messages.ConnectStateListening, ListenAddr: "0.0.0.0:19132"})
	a.HandleEvent(&messages.EventFinishedSavingWorld{WorldName: "CubeCraft", Filepath: "worlds/test.mcworld", Chunks: 275, Entities: 208})
	s := a.Snapshot()
	if s.Connection != int(messages.ConnectStateListening) || s.ListenAddr != "0.0.0.0:19132" || len(s.Worlds) != 1 || s.Worlds[0].Chunks != 275 {
		t.Fatalf("lost initial state: %+v", s)
	}
	s.Worlds[0].Name = "changed"
	if a.Snapshot().Worlds[0].Name != "CubeCraft" {
		t.Fatal("snapshot leaked mutable state")
	}
}
