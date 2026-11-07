package handlers

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/Daniel-Ric/CraftBox/utils/skinconverter"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

type SkinSaver struct {
	log     *logrus.Entry
	session *proxy.Session

	PlayerNameFilter  *regexp.Regexp
	OnlyIfHasGeometry bool
	TextureOnly       bool

	fs             utils.WriterFS
	playersById    map[uuid.UUID]*skinPlayer
	playersByName  map[string]uuid.UUID
	itemNames      map[int32]string
	gear           map[uint64]map[string]protocol.ItemInstance
	localUUID      uuid.UUID
	localRuntimeID uint64
}

func NewSkinSaver(skinCallback func(SkinAdd), PlayerNameFilter *regexp.Regexp, TextureOnly, Timestamped bool) func() *proxy.Handler {
	return func() *proxy.Handler {
		s := &SkinSaver{
			log: logrus.WithField("part", "SkinSaver"),

			PlayerNameFilter: PlayerNameFilter,
			TextureOnly:      TextureOnly,

			playersById:   make(map[uuid.UUID]*skinPlayer),
			playersByName: make(map[string]uuid.UUID),
		}
		return &proxy.Handler{
			Name: "Skin Saver",
			PlayerDataModifier: func(session *proxy.Session, identity *login.IdentityData, data *login.ClientData) {
				if id, err := uuid.Parse(identity.Identity); err == nil {
					s.localUUID = id
					player := s.AddOrGetPlayer(id, identity.DisplayName)
					player.XUID = identity.XUID
					if skin := loginSkin(data); skin != nil && skinCallback != nil {
						skinCallback(SkinAdd{PlayerName: player.Name, PlayerUUID: id.String(), XUID: identity.XUID, Skin: skin})
					}
				}
			},
			GameDataModifier: func(session *proxy.Session, data *minecraft.GameData) {
				s.session = session
				s.localRuntimeID = data.EntityRuntimeID
				if player := s.playersById[s.localUUID]; player != nil {
					player.RuntimeID = data.EntityRuntimeID
					for _, event := range s.equipmentEvents(data.EntityRuntimeID) {
						if skinCallback != nil {
							skinCallback(event)
						}
					}
				}
			},
			SessionStart: func(session *proxy.Session, hostname string) error {
				s.session = session
				basePath := utils.PathData("skins", utils.MakeValidFilename(hostname))
				if Timestamped {
					ts := time.Now().Format("2006-01-02_15-04-05")
					basePath = path.Join(basePath, ts)
				}
				s.fs = utils.OSWriter{Base: basePath}

				return nil
			},
			PacketCallback: func(session *proxy.Session, pk packet.Packet, toServer bool, timeReceived time.Time, preLogin bool) (packet.Packet, error) {
				for _, s := range s.ProcessPacket(pk) {
					if skinCallback != nil {
						skinCallback(s)
					}
				}
				return pk, nil
			},
		}
	}
}

type skinPlayer struct {
	UUID      uuid.UUID
	RuntimeID uint64
	Name      string
	AltName   string
	nameFinal bool
	Position  mgl32.Vec3
	SkinPack  *skinconverter.SkinPack
	XUID      string

	seenSkins map[string]struct{}

	gone bool
}

func (s *SkinSaver) AddOrGetPlayer(id uuid.UUID, name string) *skinPlayer {
	player, ok := s.playersById[id]
	if !ok {
		player = &skinPlayer{UUID: id, seenSkins: make(map[string]struct{})}
		s.playersById[id] = player
	}
	if player.Name == "" && name != "" {
		player.Name = utils.CleanupName(name)
	}
	return player
}

func (s *SkinSaver) ensureFinalName(player *skinPlayer) {
	if !player.nameFinal {
		player.nameFinal = true
		var name = player.Name
		switch {
		case player.Name != "":
			name = player.Name
		case player.AltName != "":
			name = player.AltName
		default:
			name = player.UUID.String()
		}

		for i := 0; ; i++ {
			var nameIt = name
			if i > 0 {
				nameIt = fmt.Sprintf("%s_%d", name, i)
			}
			if _, ok := s.playersByName[nameIt]; !ok {
				player.Name = nameIt
				break
			}
		}
		s.playersByName[player.Name] = player.UUID
	}
}

func (s *SkinSaver) AddSkin(player *skinPlayer, newSkin *protocol.Skin) (*skinconverter.Skin, bool) {
	s.ensureFinalName(player)

	if s.PlayerNameFilter != nil {
		if !s.PlayerNameFilter.MatchString(player.Name) {
			return nil, false
		}
	}

	skin := &skinconverter.Skin{Skin: newSkin}
	if !skin.HaveGeometry() && s.OnlyIfHasGeometry {
		return nil, false
	}

	sh := skin.Hash()
	_, ok := player.seenSkins[sh]
	if ok {
		return skin, false
	}
	player.seenSkins[sh] = struct{}{}

	if s.TextureOnly {
		skinName := player.Name
		if len(player.seenSkins) > 1 {
			skinName += "-" + strconv.Itoa(len(player.seenSkins))
		}

		err := skinconverter.WriteSkinTexture(s.fs, skinName, skin)
		if err != nil {
			s.log.WithError(err).Error("failed to write skin texture")
			return skin, false
		}

		if skin.HaveCape() {
			err := skinconverter.WriteCapeTexture(s.fs, skinName, skin)
			if err != nil {
				s.log.WithError(err).Error("failed to write cape texture")
				return skin, false
			}
		}

		return skin, true
	} else {
		if player.SkinPack == nil {
			player.SkinPack = skinconverter.NewSkinPack(player.Name)
			creating := fmt.Sprintf("Creating Skinpack for %s", player.Name)
			s.log.Info(creating)
		}
		player.SkinPack.AddSkin(skin)
		addedStr := fmt.Sprintf("Added a skin %s", player.Name)
		s.session.SendPopup(addedStr)
		s.log.Info(addedStr)
		pfs := utils.SubFS(s.fs, player.Name)
		err := player.SkinPack.Save(pfs)
		if err != nil {
			s.log.Error(err)
		}
		return skin, true
	}
}

type SkinAdd struct {
	PlayerName string
	Skin       *protocol.Skin
	PlayerUUID string
	XUID       string
	Equipment  map[string]string
}

func (s *SkinSaver) ProcessPacket(pk packet.Packet) (out []SkinAdd) {
	switch pk := pk.(type) {
	case *packet.ItemRegistry:
		s.itemNames = map[int32]string{}
		for _, entry := range pk.Items {
			s.itemNames[int32(entry.RuntimeID)] = entry.Name
		}
		for runtime := range s.gear {
			out = append(out, s.equipmentEvents(runtime)...)
		}
	case *packet.MobArmourEquipment:
		if len(s.gear) >= 5000 && s.gear[pk.EntityRuntimeID] == nil {
			return
		}
		if s.gear == nil {
			s.gear = map[uint64]map[string]protocol.ItemInstance{}
		}
		s.gear[pk.EntityRuntimeID] = map[string]protocol.ItemInstance{"head": pk.Helmet, "chest": pk.Chestplate, "legs": pk.Leggings, "feet": pk.Boots, "body": pk.Body}
		out = append(out, s.equipmentEvents(pk.EntityRuntimeID)...)
	case *packet.InventoryContent:
		localRuntime := s.localRuntimeID
		if localRuntime == 0 && s.session != nil {
			localRuntime = s.session.Player.RuntimeID
		}
		if pk.WindowID == protocol.WindowIDArmour && s.session != nil && len(pk.Content) >= 4 {
			if s.gear == nil {
				s.gear = map[uint64]map[string]protocol.ItemInstance{}
			}
			s.gear[localRuntime] = map[string]protocol.ItemInstance{"head": pk.Content[0], "chest": pk.Content[1], "legs": pk.Content[2], "feet": pk.Content[3]}
			out = append(out, s.equipmentEvents(localRuntime)...)
		}
	case *packet.InventorySlot:
		localRuntime := s.localRuntimeID
		if localRuntime == 0 && s.session != nil {
			localRuntime = s.session.Player.RuntimeID
		}
		if pk.WindowID == protocol.WindowIDArmour && pk.Slot < 4 && s.session != nil {
			if s.gear == nil {
				s.gear = map[uint64]map[string]protocol.ItemInstance{}
			}
			if s.gear[localRuntime] == nil {
				s.gear[localRuntime] = map[string]protocol.ItemInstance{}
			}
			s.gear[localRuntime][[]string{"head", "chest", "legs", "feet"}[pk.Slot]] = pk.NewItem
			out = append(out, s.equipmentEvents(localRuntime)...)
		}
	case *packet.StartGame:
		if s.session != nil && s.session.Client != nil {
			identity := s.session.Client.IdentityData()
			if id, err := uuid.Parse(identity.Identity); err == nil {
				player := s.AddOrGetPlayer(id, identity.DisplayName)
				player.RuntimeID = pk.EntityRuntimeID
				player.XUID = identity.XUID
			}
		}
	case *packet.MovePlayer:
		var player *skinPlayer
		for _, sp := range s.playersById {
			if sp.RuntimeID == pk.EntityRuntimeID {
				player = sp
				break
			}
		}
		if player == nil {
			return
		} else {
			player.Position = pk.Position
		}
	case *packet.MoveActorAbsolute:
		for _, player := range s.playersById {
			if player.RuntimeID == pk.EntityRuntimeID {
				player.Position = pk.Position
				break
			}
		}

	case *packet.PlayerList:
		for _, playerEntry := range pk.Entries {
			if playerEntry.ActionType == protocol.PlayerListActionRemove {
				continue
			}
			player := s.AddOrGetPlayer(playerEntry.UUID, playerEntry.Username)
			player.XUID = playerEntry.XUID
			skin, wasAdded := s.AddSkin(player, &playerEntry.Skin)
			if wasAdded {
				out = append(out, SkinAdd{
					PlayerName: player.Name,
					Skin:       skin.Skin,
					PlayerUUID: player.UUID.String(), XUID: player.XUID,
				})
			}
			out = append(out, s.equipmentEvents(player.RuntimeID)...)
		}
	case *packet.PlayerSkin:
		player := s.AddOrGetPlayer(pk.UUID, "")
		skin, wasAdded := s.AddSkin(player, &pk.Skin)
		if wasAdded {
			out = append(out, SkinAdd{
				PlayerName: player.Name,
				Skin:       skin.Skin,
				PlayerUUID: player.UUID.String(), XUID: player.XUID,
			})
		}
	case *packet.AddPlayer:
		player := s.AddOrGetPlayer(pk.UUID, "")
		player.AltName = pk.Username
		player.RuntimeID = pk.EntityRuntimeID
		out = append(out, s.equipmentEvents(player.RuntimeID)...)
	case *packet.Animate:
		if pk.EntityRuntimeID == s.session.Player.RuntimeID && pk.ActionType == packet.AnimateActionSwingArm {
			s.stealSkin()
		}
	case *packet.ChangeDimension:
		for _, sp := range s.playersById {
			sp.gone = true
		}
	case *packet.RemoveActor:
		for _, sp := range s.playersById {
			if sp.RuntimeID == uint64(pk.EntityUniqueID) {
				sp.gone = true
				break
			}
		}
	}
	return out
}

var playerBBox = cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3)

var rtTest = false

func (s *SkinSaver) stealSkin() {
	if !rtTest {
		return
	}
	s.log.Debugf("%d", len(s.playersById))

	var dist float64 = 40

	pitch := mgl64.DegToRad(float64(s.session.Player.Pitch))
	yaw := mgl64.DegToRad(float64(s.session.Player.HeadYaw + 90))

	dir := mgl64.Vec3{
		math.Cos(yaw) * math.Cos(pitch),
		math.Sin(-pitch),
		math.Sin(yaw) * math.Cos(pitch),
	}.Normalize()

	pos := s.session.Player.Position
	traceStart := mgl64.Vec3{float64(pos[0]), float64(pos[1]), float64(pos[2])}
	traceEnd := traceStart.Add(dir.Mul(dist))
	s.session.ClientWritePacket(&packet.SpawnParticleEffect{
		Dimension:      0,
		EntityUniqueID: -1,
		Position:       mgl32.Vec3{float32(traceEnd[0]), float32(traceEnd[1]), float32(traceEnd[2])},
		ParticleName:   "hivehub:emote_confounded",
	})

	for _, sp := range s.playersById {
		if sp.gone {
			continue
		}
		pos := mgl64.Vec3{float64(sp.Position[0]), float64(sp.Position[1]) - 1.8, float64(sp.Position[2])}
		bb := playerBBox.Translate(pos)

		res, ok := trace.BBoxIntercept(bb, traceStart, traceEnd)
		if ok {
			fmt.Printf("res: %v\n", res.Position())
			interceptPos := res.Position()

			s.session.ClientWritePacket(&packet.SpawnParticleEffect{
				Dimension:      0,
				EntityUniqueID: -1,
				Position:       mgl32.Vec3{float32(interceptPos[0]), float32(interceptPos[1]), float32(interceptPos[2])},
				ParticleName:   "hivehub:emote_confounded",
			})

			id := uuid.MustParse(s.session.Client.IdentityData().Identity)
			playerSkin := sp.SkinPack.Latest().Skin
			if playerSkin == nil {
				logrus.Warnf("%s has no skin", sp.Name)
				continue
			}

			s.session.ClientWritePacket(&packet.PlayerSkin{
				UUID: id,
				Skin: *playerSkin,
			})

			s.session.Server.WritePacket(&packet.PlayerSkin{
				UUID: id,
				Skin: *playerSkin,
			})
			break
		}
	}
}
