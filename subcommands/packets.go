package subcommands

import (
	"context"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
)

type PacketInspectorSettings struct{ ProxySettings proxy.ProxySettings }
type PacketInspectorCMD struct{}

func (PacketInspectorCMD) Name() string        { return "packets" }
func (PacketInspectorCMD) Description() string { return "Inspect a NetherNet session" }
func (PacketInspectorCMD) Settings() any       { return new(PacketInspectorSettings) }
func (PacketInspectorCMD) Run(ctx context.Context, settings any) error {
	if settings.(*PacketInspectorSettings).ProxySettings.ConnectInfo == nil {
		return fmt.Errorf("Select a NetherNet source")
	}
	if settings.(*PacketInspectorSettings).ProxySettings.ConnectInfo.IsReplay() {
		return fmt.Errorf("Import the capture in Packet Inspector for offline playback")
	}
	p, e := proxy.New(ctx, settings.(*PacketInspectorSettings).ProxySettings)
	if e != nil {
		return e
	}
	return p.Run(ctx, true)
}
func init() { commands.RegisterCommand(&PacketInspectorCMD{}) }
