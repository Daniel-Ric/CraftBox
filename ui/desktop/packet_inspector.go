package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/handlers"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds"
	"github.com/Daniel-Ric/CraftBox/internal/modelpicker"
	pi "github.com/Daniel-Ric/CraftBox/internal/packetinspector"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func (a *App) packetInspector() *pi.Controller {
	a.packetMu.Lock()
	defer a.packetMu.Unlock()
	if a.packetRecorder == nil {
		a.packetRecorder = pi.NewController(utils.PathData("packet-inspector"))
	}
	return a.packetRecorder
}
func (a *App) GetPacketInspector() (pi.Status, error) { return a.packetInspector().Status() }
func (a *App) ConfigurePacketInspector(settings pi.Settings) error {
	return a.packetInspector().Configure(settings)
}
func (a *App) QueryPackets(q pi.Query) (pi.Page, error) { return a.packetInspector().Store.Query(q) }
func (a *App) GetPacketTypes(capture string) ([]pi.TypeStat, error) {
	return a.packetInspector().Store.Types(capture)
}
func (a *App) GetPacketDetail(capture, id string, revealSecrets bool) (pi.Packet, error) {
	return a.packetInspector().Store.Detail(capture, id, revealSecrets)
}
func (a *App) GetPacketAnalysis(q pi.Query) (pi.Analysis, error) {
	return a.packetInspector().Store.Analyze(q)
}
func (a *App) UpdatePacketCapture(u pi.CaptureUpdate) error {
	return a.packetInspector().Store.Update(u)
}
func (a *App) DeletePacketCapture(id string) error {
	return a.packetInspector().Delete(id)
}
func (a *App) SetPacketRules(rules []pi.Rule) error {
	return a.packetInspector().SetRules(rules)
}
func (a *App) GetPacketRules() []pi.Rule { return a.packetInspector().Engine.Rules() }
func (a *App) packetShield() int32 {
	a.packetMu.Lock()
	read := a.packetShieldReader
	a.packetMu.Unlock()
	if read != nil {
		return read()
	}
	return 0
}
func (a *App) PreviewPacketSend(r pi.SendRequest) (pi.Packet, error) {
	return a.packetInspector().Preview(r, a.packetShield())
}
func (a *App) SendInspectorPacket(r pi.SendRequest) (pi.Packet, error) {
	return a.packetInspector().Send(r, a.packetShield())
}
func (a *App) closePacketInspector() {
	a.packetMu.Lock()
	c := a.packetRecorder
	archive := a.commandArchive
	a.packetMu.Unlock()
	if c != nil {
		c.Close()
	}
	if archive != nil {
		archive.Close()
	}
}
func (a *App) inspectorHandlers(ctx context.Context, command string) []func() *proxy.Handler {
	c := a.packetInspector()
	settings := c.Settings()
	a.packetMu.Lock()
	a.packetModules = settings
	a.packetModulesSet = true
	a.packetMu.Unlock()
	factories := []func() *proxy.Handler{func() *proxy.Handler {
		id := ""
		explorerID := ""
		modelTargets := modelpicker.New()
		return &proxy.Handler{Name: "Packet Inspector", CaptureStatus: func() string {
			if id == "" {
				return "§7Packet Inspector: §cnot ready§r"
			}
			if c.Settings().Recording {
				return "§7Packet Inspector: §arecording§r"
			}
			return "§7Packet Inspector: §epaused§r"
		}, SessionStart: func(s *proxy.Session, name string) error {
			var e error
			id, e = c.Begin(name, protocol.CurrentProtocol, func(pk packet.Packet, toServer bool) error {
				if s.SessionContext().Err() != nil {
					return errors.New("NetherNet session has ended")
				}
				destination := s.Client
				if toServer {
					destination = s.Server
				}
				if destination == nil {
					return errors.New("Selected endpoint is not connected")
				}
				return destination.WritePacket(pk)
			})
			if m, err := a.commandExplorer().Begin(name, id, protocol.CurrentProtocol); err == nil {
				explorerID = m.ID
				a.startModelPicker(s, modelTargets, explorerID)
			} else {
				c.Report("Command Explorer: " + err.Error())
			}
			if e != nil {
				c.Report("Recorder unavailable: " + e.Error())
				return nil
			}
			a.packetMu.Lock()
			a.packetShieldReader = func() int32 {
				if s.Server != nil {
					return s.Server.ShieldID()
				}
				return 0
			}
			a.packetMu.Unlock()
			return nil
		}, PacketWire: func(s *proxy.Session, header packet.Header, payload []byte, toServer bool, stage string, at time.Time, shield int32, sourceAddress, destinationAddress, connectionLeg string) {
			direction := "clientbound"
			if toServer {
				direction = "serverbound"
			}
			var head bytes.Buffer
			_ = header.Write(&head)
			c.Observe(id, pi.Packet{SourceAddress: sourceAddress, DestinationAddress: destinationAddress, ConnectionLeg: connectionLeg, Header: head.Bytes(), PacketID: header.PacketID, Time: at.UnixMilli(), Direction: direction, Stage: stage, Raw: payload, ShieldID: shield, Name: "Pending decode"})
			if explorerID != "" && stage == "received" && !toServer && header.PacketID == packet.IDResourcePackStack {
				if decoded, err := pi.Decode(header.PacketID, payload, shield); err == nil {
					if err = a.commandExplorer().Record(explorerID, decoded, false, at); err != nil {
						c.Report("Command Explorer: " + err.Error())
					}
					a.warmModelPicker()
				}
			}
		}, PacketControl: func(s *proxy.Session, pk packet.Packet, toServer bool) (packet.Packet, error) {
			modelTargets.Observe(pk, toServer)
			if explorerID != "" {
				if err := a.commandExplorer().Record(explorerID, pk, toServer, time.Now()); err != nil {
					c.Report("Command Explorer: " + err.Error())
				}
			}
			return c.Intercept(s.SessionContext(), id, pk, toServer)
		}, PacketCallback: func(s *proxy.Session, pk packet.Packet, toServer bool, at time.Time, preLogin bool) (packet.Packet, error) {
			if preLogin {
				modelTargets.Observe(pk, toServer)
			}
			if preLogin && explorerID != "" && pk.ID() != packet.IDResourcePackStack {
				if err := a.commandExplorer().Record(explorerID, pk, toServer, at); err != nil {
					c.Report("Command Explorer: " + err.Error())
				}
			}
			return pk, nil
		}, OnSessionEnd: func(s *proxy.Session, wg *sync.WaitGroup) {
			if explorerID != "" {
				_ = a.commandExplorer().End(explorerID)
			}
			c.End(id)
			a.packetMu.Lock()
			if a.modelPickSession == explorerID {
				a.modelPickRead = nil
				a.modelPickSession = ""
				a.clearHotbarFeedback = nil
			}
			a.packetShieldReader = nil
			a.packetMu.Unlock()
		}}
	}}
	if settings.CaptureWorlds && command != "worlds" {
		factories = append(factories, worlds.NewWorldsHandler(ctx, worlds.WorldSettings{VoidGen: true, SaveEntities: true, SaveCustomEntities: true, SaveInventories: true, BlockUpdates: true}))
	}
	if settings.CaptureSkins && command != "skins" {
		factories = append(factories, handlers.NewSkinSaver(func(sa handlers.SkinAdd) {
			if sa.Equipment != nil {
				messages.SendEvent(&messages.EventPlayerEquipment{PlayerName: sa.PlayerName, PlayerUUID: sa.PlayerUUID, Equipment: sa.Equipment})
			}
			if sa.Skin != nil {
				messages.SendEvent(&messages.EventPlayerSkin{PlayerName: sa.PlayerName, PlayerUUID: sa.PlayerUUID, XUID: sa.XUID, Skin: *sa.Skin})
			}
		}, nil, false, true))
	}
	return factories
}

type PacketContentLink struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Search   string `json:"search"`
	Relation string `json:"relation"`
}

func (a *App) GetPacketContentLinks(capture, id string) ([]PacketContentLink, error) {
	p, e := a.packetInspector().Store.Detail(capture, id, false)
	out := []PacketContentLink{}
	if e != nil {
		return out, e
	}
	if p.Name == "AvailableCommands" {
		kind := "commands"
		if sessions, err := a.commandExplorer().List(); err == nil {
			for _, session := range sessions {
				if session.Capture == capture {
					out = append(out, PacketContentLink{Kind: kind, ID: session.ID, Name: session.Name, Search: p.Name, Relation: "Related session archive"})
					break
				}
			}
		}
	}
	items, e := a.assets().List()
	if e != nil {
		return out, e
	}
	lower := strings.ToLower(p.Payload)
	for _, item := range items {
		identity := item.PlayerUUID
		if item.Kind == "packs" {
			identity = item.UUID
		}
		if identity != "" && strings.Contains(lower, strings.ToLower(identity)) {
			out = append(out, PacketContentLink{Kind: item.Kind, ID: item.ID, Name: item.Name, Search: identity, Relation: "Identifier in packet"})
		}
	}
	status, _ := a.packetInspector().Status()
	if capture == status.Active && strings.Contains(strings.ToLower(p.Name), "chunk") {
		a.mu.Lock()
		for _, w := range a.state.Worlds {
			out = append(out, PacketContentLink{Kind: "worlds", ID: w.ID, Name: w.Name, Search: p.Name, Relation: "Current capture world; verify coordinates"})
		}
		a.mu.Unlock()
	}
	return out, nil
}
func (a *App) ExtractPacketContent(capture, id string) ([]PacketContentLink, error) {
	p, e := a.packetInspector().Store.Detail(capture, id, true)
	if e != nil {
		return nil, e
	}
	var pk packet.Packet
	if len(p.Raw) > 0 {
		pk, e = pi.Decode(p.PacketID, p.Raw, a.packetShield())
	} else {
		pk, e = pi.DecodeJSON(p.PacketID, p.Payload)
	}
	if e != nil {
		return nil, e
	}
	out := []PacketContentLink{}
	save := func(name, playerUUID string, skin protocol.Skin) error {
		items, e := a.assets().CaptureSkin(context.Background(), name, "Packet Inspector", skin, map[string]any{"playerUUID": playerUUID, "packetCapture": capture, "packetSequence": id})
		for _, item := range items {
			out = append(out, PacketContentLink{Kind: "skins", ID: item.ID, Name: item.Name, Search: playerUUID, Relation: "Extracted from this packet"})
		}
		return e
	}
	switch v := pk.(type) {
	case *packet.AvailableCommands:
		m, err := a.ExtractCommandExplorer(capture)
		if err != nil {
			return nil, err
		}
		kind := "commands"
		out = append(out, PacketContentLink{Kind: kind, ID: m.ID, Name: m.Name, Search: p.Name, Relation: "Extracted session archive"})
	case *packet.PlayerSkin:
		e = save(v.NewSkinName, v.UUID.String(), v.Skin)
	case *packet.PlayerList:
		for _, entry := range v.Entries {
			if len(entry.Skin.SkinData) > 0 {
				if e = save(entry.Username, entry.UUID.String(), entry.Skin); e != nil {
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("%s has no standalone skin; packs and worlds are assembled by their live capture modules", p.Name)
	}
	return out, e
}
func (a *App) writePacketExport(path string, q pi.Query, reveal bool) error {
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		path += ".json"
	}
	f, e := os.CreateTemp(filepath.Dir(path), "craftbox-packets-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	c, e := a.packetInspector().Store.Capture(q.Capture)
	if e != nil {
		f.Close()
		return e
	}
	meta, _ := json.Marshal(c)
	_, e = f.Write(append([]byte(`{"schema":2,"capture":`), meta...))
	if e == nil {
		_, e = f.WriteString(`,"packets":[`)
	}
	first := true
	if e == nil {
		e = a.packetInspector().Store.Export(q, reveal, func(p pi.Packet) error {
			if !first {
				if _, e := f.WriteString(","); e != nil {
					return e
				}
			}
			first = false
			return json.NewEncoder(f).Encode(p)
		})
	}
	if e == nil {
		_, e = f.WriteString("]}")
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
