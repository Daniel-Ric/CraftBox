package worlds

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"maps"
	"math"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

func (w *worldsHandler) packetHandlerPreLogin(_pk packet.Packet, toServer bool, timeReceived time.Time) (packet.Packet, error) {
	switch pk := _pk.(type) {
	case *packet.SyncActorProperty:
		w.handleSyncActorProperty(pk)
	case *packet.AvailableActorIdentifiers:
		if toServer {
			break
		}
		if err := w.handleActorCatalog(pk); err != nil {
			return nil, err
		}
	case *packet.GameRulesChanged:
		w.rememberGameRules(pk.GameRules)
		var haveGameRule = false
		for i, gameRule := range pk.GameRules {
			if gameRule.Name == "showCoordinates" {
				haveGameRule = true
				pk.GameRules[i] = protocol.GameRule{
					Name:  "showCoordinates",
					Value: true,
				}
				break
			}
		}
		if !haveGameRule {
			pk.GameRules = append(pk.GameRules, protocol.GameRule{
				Name:  "showCoordinates",
				Value: true,
			})
		}
	case *packet.StartGame:
		if !w.serverState.haveStartGame {
			w.rememberGameRules(pk.GameRules)
			mode, difficulty, commands := pk.WorldGameMode, pk.Difficulty, pk.CommandsEnabled
			w.captureSettingsMu.Lock()
			w.serverState.captureMode = &mode
			w.serverState.captureDifficulty = &difficulty
			w.serverState.captureCommands = &commands
			w.serverState.rainLevel, w.serverState.lightningLevel = pk.RainLevel, pk.LightningLevel
			w.captureSettingsMu.Unlock()
			w.serverState.useHashedRids = pk.UseBlockNetworkIDHashes

			var haveGameRule = false
			for i, gameRule := range pk.GameRules {
				if gameRule.Name == "showCoordinates" {
					haveGameRule = true
					gameRule.Value = true
					pk.GameRules[i] = gameRule
					break
				}
			}
			if !haveGameRule {
				pk.GameRules = append(pk.GameRules, protocol.GameRule{
					Name:  "showCoordinates",
					Value: true,
				})
			}

			br := world.NewBlockRegistry()
			if len(pk.Blocks) > 0 {
				w.log.Info(locale.Loc("using_customblocks", nil))
				for _, be := range pk.Blocks {
					w.serverState.behaviorPack.AddBlock(be)
				}
				err := AddCustomBlocks(br, pk.Blocks)
				if err != nil {
					return nil, fmt.Errorf("initialise world block registry: %w", err)
				}
				w.serverState.customBlocks = pk.Blocks
			}
			br.Finalize()
			if hashes, ok := br.(interface{ NetworkHashCollisionCount() int }); ok && pk.UseBlockNetworkIDHashes && hashes.NetworkHashCollisionCount() > 0 {
				w.log.Warnf("%d ambiguous block network hashes: affected cells will be captured as minecraft:unknown; other terrain is preserved", hashes.NetworkHashCollisionCount())
			}
			w.serverState.blocks = br
			w.serverState.worldName = pk.WorldName

			dim, _ := world.DimensionByID(int(pk.Dimension))
			w.worldStateMu.Lock()
			if pk.WorldName != "" {
				w.worldState.Name = pk.WorldName
			}
			w.worldState.SetDimension(dim)
			w.worldState.SetTime(timeReceived, int(pk.Time))
			w.openWorldState()
			w.serverState.haveStartGame = true
			w.worldStateMu.Unlock()
		}

	case *packet.DimensionData:
		for _, dd := range pk.Definitions {
			if dd.Name == "minecraft:overworld" {
				w.serverState.dimensions[0] = dd
			}
		}

	case *packet.ItemRegistry:
		w.serverState.itemNames = make(map[int32]string, len(pk.Items))
		for _, entry := range pk.Items {
			w.serverState.itemNames[int32(entry.RuntimeID)] = entry.Name
		}
		world.InsertCustomItems(pk.Items)
		for _, ie := range pk.Items {
			w.serverState.behaviorPack.AddItem(ie)
		}
		w.serverState.behaviorPack.ApplyComponentEntries(pk.Items)

	case *packet.BiomeDefinitionList:
		for _, biome := range pk.BiomeDefinitions {
			biomeName := pk.StringList[biome.NameIndex]
			sp := strings.SplitN(biomeName, ":", 2)
			if len(sp) > 1 {
				ns := sp[0]
				if ns == "minecraft" {
					biomeName = sp[1]
				}
			}
			_, ok := w.serverState.biomes.BiomeByName(biomeName)
			if !ok {
				w.serverState.biomes.Register(&customBiome{name: biomeName, biome: biome, pk: pk})
				w.serverState.behaviorPack.AddBiome(biomeName, biome)
			}
		}

	}

	return _pk, nil
}

func (w *worldsHandler) packetHandlerIngame(_pk packet.Packet, toServer bool, timeReceived time.Time) (packet.Packet, error) {
	switch pk := _pk.(type) {
	case *packet.GameRulesChanged:
		if !toServer {
			w.rememberGameRules(pk.GameRules)
		}
	case *packet.SetDifficulty:
		if !toServer {
			value := int32(pk.Difficulty)
			w.captureSettingsMu.Lock()
			w.serverState.captureDifficulty = &value
			w.captureSettingsMu.Unlock()
		}
	case *packet.SetDefaultGameType:
		if !toServer {
			value := pk.GameType
			w.captureSettingsMu.Lock()
			w.serverState.captureMode = &value
			w.captureSettingsMu.Unlock()
		}
	case *packet.LevelEvent:
		if !toServer {
			w.rememberWeatherEvent(pk)
		}
	case *packet.RequestChunkRadius:
		pk.ChunkRadius = w.settings.ChunkRadius

	case *packet.ChunkRadiusUpdated:
		w.serverState.realChunkRadius = pk.ChunkRadius
		pk.ChunkRadius = w.settings.ChunkRadius

	case *packet.SetCommandsEnabled:
		if !toServer {
			enabled := pk.Enabled
			w.captureSettingsMu.Lock()
			w.serverState.captureCommands = &enabled
			w.captureSettingsMu.Unlock()
			w.currentWorld(func(state *worldstate.World) { state.SetCommandsEnabled(enabled) })
		}
		pk.Enabled = true

	case *packet.SetTime:
		w.currentWorld(func(world *worldstate.World) {
			world.SetTime(timeReceived, int(pk.Time))
		})

	case *packet.ChangeDimension:
		dim, _ := world.DimensionByID(int(pk.Dimension))
		w.SaveAndReset(false, dim)

	case *packet.LevelChunk:
		err := w.handleLevelChunk(pk, timeReceived)
		if err != nil {
			w.log.Errorf("processLevelChunk %s", err)
		}

	case *packet.SubChunk:
		if err := w.processSubChunk(pk); err != nil {
			w.log.WithField("packet", "SubChunk").Error(err)
		}

	case *packet.BlockActorData:
		p := pk.Position
		pos := cube.Pos{int(p.X()), int(p.Y()), int(p.Z())}
		w.currentWorld(func(world *worldstate.World) {
			world.SetBlockNBT(pos, pk.NBTData, false)
		})

	case *packet.ClientBoundMapItemData:
		if !toServer && pk.MapID != ViewMapID {
			w.storeKnownMap(pk)
			w.currentWorld(func(world *worldstate.World) { world.StoreMap(pk) })
		}

	case *packet.MapInfoRequest:
		if pk.MapID == ViewMapID {
			w.mapUI.SchedRedraw()
			_pk = nil
		}
	case *packet.Animate:
		if toServer && pk.ActionType == packet.AnimateActionSwingArm && w.mapUI.Enabled() {
			w.mapUI.ChangeZoom()
			w.session.SendPopup(locale.Loc("zoom_level", locale.Strmap{"Level": w.mapUI.Zoom()}))
		}

	case *packet.SpawnParticleEffect:
		if w.scripting != nil {
			w.scripting.OnSpawnParticle(pk.ParticleName, pk.Position, timeReceived)
		}

	case *packet.AddPlayer:
		if !toServer {
			w.rememberCommandEntities([]string{"minecraft:player"})
		}
		w.currentWorld(func(world *worldstate.World) {
			world.AddPlayer(pk)
		})

	case *packet.PlayerList:
		for _, player := range pk.Entries {
			if player.ActionType == protocol.PlayerListActionAdd {
				w.serverState.playerSkins[player.UUID] = &player.Skin
				if w.serverState.playerXUIDs == nil {
					w.serverState.playerXUIDs = map[uuid.UUID]string{}
				}
				w.serverState.playerXUIDs[player.UUID] = player.XUID
				if w.serverState.playerUniqueIDs == nil {
					w.serverState.playerUniqueIDs = map[uuid.UUID]int64{}
				}
				w.serverState.playerUniqueIDs[player.UUID] = player.EntityUniqueID
				w.currentWorld(func(world *worldstate.World) { world.RegisterScenePlayerID(player.EntityUniqueID) })
			}
		}

	case *packet.PlayerSkin:
		w.serverState.playerSkins[pk.UUID] = &pk.Skin

	case *packet.SyncActorProperty:
		w.handleSyncActorProperty(pk)
	case *packet.AvailableActorIdentifiers:
		if toServer {
			break
		}
		if err := w.handleActorCatalog(pk); err != nil {
			return nil, err
		}

	case *packet.AddActor:
		if !toServer {
			w.rememberCommandEntities([]string{pk.EntityType})
		}
		if !w.settings.SaveEntities || (!w.settings.SaveCustomEntities && entity.IsCustomType(pk.EntityType) && !(w.serverState.isHive && pk.EntityType == worldstate.HiveTextEntity)) {
			break
		}
		w.currentWorld(func(world *worldstate.World) {
			err := world.ActEntity(pk.EntityRuntimeID, true, func(ent *entity.Entity) error {
				isNew := ent.EntityType == ""

				if ent.DeletedDistance > 0 && len(w.serverState.entityRenderDistances) < 200 {
					playerPos := w.session.Player.Position
					dist3d := ent.Position.Sub(playerPos).Len()
					diff := float32(math.Abs(float64(ent.DeletedDistance - dist3d)))
					if diff < 10 {
						w.serverState.entityRenderDistances = append(w.serverState.entityRenderDistances, dist3d)
					}
				}
				ent.DeletedDistance = -1
				ent.LastTeleport = 0

				prevPosition := ent.Position

				ent.UniqueID = pk.EntityUniqueID
				ent.EntityType = pk.EntityType
				ent.Position = pk.Position
				ent.Pitch = pk.Pitch
				ent.Yaw = pk.Yaw
				ent.HeadYaw = pk.HeadYaw
				ent.Velocity = pk.Velocity

				maps.Copy(ent.Metadata, pk.EntityMetadata)
				properties := w.serverState.entityProperties[ent.EntityType]
				changes := applyProperties(properties, pk.EntityProperties, ent.Properties, ent.PropertiesOverridden)

				if w.scripting != nil {
					if !w.scripting.OnEntityAdd(ent, isNew, timeReceived) {
						logrus.Infof("Ignoring Entity: %s %d", ent.EntityType, ent.UniqueID)
						return worldstate.ErrIgnoreEntity
					}
					for name, previous := range changes {
						apply := w.scripting.OnEntityPropertyChange(ent, name, previous, ent.Properties[name])
						if !apply {
							if previous == nil {
								delete(ent.Properties, name)
							} else {
								ent.Properties[name] = previous
							}
						}
					}
				}

				if !isNew {
					w.onEntityUpdate(ent, &prevPosition, changes)
				}

				for _, el := range pk.EntityLinks {
					world.AddEntityLink(el)
				}
				w.serverState.behaviorPack.AddEntity(pk.EntityType, pk.Attributes, ent.Metadata, properties)
				return nil
			})
			if err != nil {
				logrus.Errorf("AddActor: %s", err)
			}
		})

	case *packet.MovePlayer:
		entityRenderDistance := w.serverState.getEntityRenderDistance()
		if pk.EntityRuntimeID == w.session.Player.RuntimeID && pk.Mode == packet.MoveModeTeleport {
			w.currentWorld(func(world *worldstate.World) {
				world.PlayerMove(w.session.Player.TeleportLocation, entityRenderDistance, w.session.Player.Teleports)
			})
		}
		if pk.EntityRuntimeID != w.session.Player.RuntimeID {
			w.currentWorld(func(state *worldstate.World) {
				state.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
					prev := ent.Position
					ent.Position = pk.Position.Sub(mgl32.Vec3{0, worldstate.PlayerEyeHeight, 0})
					ent.Pitch, ent.Yaw, ent.HeadYaw = pk.Pitch, pk.Yaw, pk.HeadYaw
					ent.HasMoved = true
					w.onEntityUpdate(ent, &prev, nil)
					return nil
				})
			})
		}

	case *packet.RemoveActor:
		entityRenderDistance := w.serverState.getEntityRenderDistance()
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(world.GetEntityRuntimeID(pk.EntityUniqueID), false, func(ent *entity.Entity) error {
				playerPos := w.session.Player.Position
				dist3d := ent.Position.Sub(playerPos).Len()

				if w.session.Player.Teleports == ent.LastTeleport {
					dist3d := w.session.Player.TeleportLocation.Sub(ent.Position).Len()
					if dist3d < entityRenderDistance+3 {
						ent.DeletedDistance = entityRenderDistance
					}
				} else {
					ent.DeletedDistance = dist3d
				}
				w.onEntityUpdate(ent, nil, nil)
				return nil
			})
		})

	case *packet.SetActorData:
		w.currentWorld(func(world *worldstate.World) {
			if pk.EntityRuntimeID == w.session.Player.RuntimeID {
				properties := w.serverState.entityProperties["minecraft:player"]
				changed := applyProperties(properties, pk.EntityProperties, w.serverState.playerPropertyValues, nil)
				_ = changed
			} else {
				world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
					properties := w.serverState.entityProperties[ent.EntityType]

					maps.Copy(ent.Metadata, pk.EntityMetadata)
					w.serverState.behaviorPack.AddEntity(ent.EntityType, nil, ent.Metadata, properties)
					changes := applyProperties(properties, pk.EntityProperties, ent.Properties, ent.PropertiesOverridden)
					if w.scripting != nil {
						for name, previous := range changes {
							apply := w.scripting.OnEntityPropertyChange(ent, name, previous, ent.Properties[name])
							if !apply {
								if previous == nil {
									delete(ent.Properties, name)
								} else {
									ent.Properties[name] = previous
								}
							}
						}
					}
					w.onEntityUpdate(ent, nil, changes)
					return nil
				})
			}
		})

	case *packet.PlayerUpdateEntityOverrides:
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(world.GetEntityRuntimeID(pk.EntityUniqueID), false, func(ent *entity.Entity) error {
				var changes = make(map[string]any)
				if pk.Type == packet.PlayerUpdateEntityOverridesTypeClearAll {

				} else {
					properties := w.serverState.entityProperties[ent.EntityType]
					property := properties[pk.PropertyIndex]
					prevValue := ent.Properties[property.Name]
					var value any
					switch pk.Type {
					case packet.PlayerUpdateEntityOverridesTypeRemove:
						oldValue := ent.PropertiesOverridden[property.Name]
						delete(ent.PropertiesOverridden, property.Name)
						value = oldValue
					case packet.PlayerUpdateEntityOverridesTypeInt:
						value = pk.IntValue
						if property.Type == entity.PropertyTypeBool {
							value = value == 1
						}
					case packet.PlayerUpdateEntityOverridesTypeFloat:
						value = pk.FloatValue
					}
					if w.scripting != nil {
						apply := w.scripting.OnEntityPropertyChange(ent, property.Name, prevValue, value)
						if !apply {
							return nil
						}
					}
					if value != nil {
						if _, ok := ent.PropertiesOverridden[property.Name]; !ok {
							ent.PropertiesOverridden[property.Name] = prevValue
						}
						ent.Properties[property.Name] = value
					}
					changes[property.Name] = prevValue
				}

				w.onEntityUpdate(ent, nil, changes)
				return nil
			})
		})

	case *packet.SetActorMotion:
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
				ent.Velocity = pk.Velocity
				w.onEntityUpdate(ent, nil, nil)
				return nil
			})

		})

	case *packet.MoveActorDelta:
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
				prevPosition := ent.Position
				if posX, ok := pk.PositionX.Value(); ok {
					ent.Position[0] = posX
				}
				if posY, ok := pk.PositionY.Value(); ok {
					if ent.EntityType == worldstate.PlayerEntityType {
						posY -= worldstate.PlayerEyeHeight
					}
					ent.Position[1] = posY
				}
				if posZ, ok := pk.PositionZ.Value(); ok {
					ent.Position[2] = posZ
				}

				if rotationX, ok := pk.RotationX.Value(); ok {
					ent.Pitch = rotationX
				}
				if rotationY, ok := pk.RotationY.Value(); ok {
					ent.Yaw = rotationY
				}
				if rotationYHead, ok := pk.RotationYHead.Value(); ok {
					ent.HeadYaw = rotationYHead
				}
				if !ent.Velocity.ApproxEqual(mgl32.Vec3{}) {
					ent.HasMoved = true
				}
				w.onEntityUpdate(ent, &prevPosition, nil)
				return nil
			})
		})

	case *packet.MoveActorAbsolute:
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
				prevPosition := ent.Position
				ent.Position = pk.Position
				if ent.EntityType == worldstate.PlayerEntityType {
					ent.Position[1] -= worldstate.PlayerEyeHeight
				}
				ent.Pitch = pk.Rotation.X()
				ent.Yaw = pk.Rotation.Y()
				if !ent.Velocity.ApproxEqual(mgl32.Vec3{}) {
					ent.HasMoved = true
				}
				w.onEntityUpdate(ent, &prevPosition, nil)
				return nil
			})
		})

	case *packet.MobEquipment:
		if pk.EntityRuntimeID == w.session.Player.RuntimeID {
			if pk.NewItem.Stack.NBTData["map_uuid"] == int64(ViewMapID) {
				_pk = nil
			}
		}
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
				window, ok := ent.Inventory[pk.WindowID]
				if !ok {
					window = make(map[byte]protocol.ItemInstance)
					ent.Inventory[pk.WindowID] = window
				}
				window[pk.HotBarSlot] = pk.NewItem
				item := pk.NewItem
				if pk.WindowID == protocol.WindowIDOffHand {
					ent.OffHand = &item
				} else {
					ent.MainHand = &item
				}
				w.onEntityUpdate(ent, nil, nil)
				return nil
			})
		})

	case *packet.MobArmourEquipment:
		w.currentWorld(func(world *worldstate.World) {
			world.ActEntity(pk.EntityRuntimeID, false, func(ent *entity.Entity) error {
				ent.Helmet = &pk.Helmet
				ent.Chestplate = &pk.Chestplate
				ent.Leggings = &pk.Leggings
				ent.Boots = &pk.Boots
				ent.Body = &pk.Body
				w.onEntityUpdate(ent, nil, nil)
				return nil
			})
		})

	case *packet.SetActorLink:
		w.currentWorld(func(world *worldstate.World) {
			world.AddEntityLink(pk.EntityLink)
		})

	case *packet.InventoryTransaction:
		var drop bool
		for _, action := range pk.Actions {
			if action.OldItem.StackNetworkID == mapItem.StackNetworkID {
				drop = true
			}
			if action.NewItem.StackNetworkID == mapItem.StackNetworkID {
				drop = true
			}
		}
		if drop {
			_pk = nil
		}

	case *packet.ItemStackRequest:
		var requests []protocol.ItemStackRequest
		for _, request := range pk.Requests {
			for _, action := range request.Actions {
				if containsStackNetworkID(action, mapItem.StackNetworkID) {
					continue
				}
			}
			requests = append(requests, request)
		}
		pk.Requests = requests

	case *packet.ContainerOpen:

		existing, ok := w.serverState.openItemContainers[pk.WindowID]
		if !ok {
			existing = &itemContainer{}
		}
		w.serverState.openItemContainers[pk.WindowID] = &itemContainer{
			OpenPacket: pk,
			Content:    existing.Content,
		}

	case *packet.InventoryContent:
		switch pk.WindowID {
		case 0:
			w.serverState.playerInventory = pk.Content
		case protocol.WindowIDOffHand:
			if len(pk.Content) > 0 {
				w.mapUI.rememberOffhand(pk.Content[0])
			} else {
				w.mapUI.rememberOffhand(protocol.ItemInstance{})
			}
			if w.mapUI.Enabled() {
				_pk = nil
			}
		default:

			existing, ok := w.serverState.openItemContainers[byte(pk.WindowID)]
			if ok {
				existing.Content = pk
			}
		}

	case *packet.InventorySlot:
		switch pk.WindowID {
		case 0:
			if w.serverState.playerInventory == nil {
				w.serverState.playerInventory = make([]protocol.ItemInstance, 36)
			}
			w.serverState.playerInventory[pk.Slot] = pk.NewItem
		case protocol.WindowIDOffHand:
			w.mapUI.rememberOffhand(pk.NewItem)
			if w.mapUI.Enabled() {
				_pk = nil
			}
		default:

			existing, ok := w.serverState.openItemContainers[byte(pk.WindowID)]
			if ok {
				existing.Content.Content[pk.Slot] = pk.NewItem
			}
		}

	case *packet.ContainerClose:

		existing, ok := w.serverState.openItemContainers[byte(pk.WindowID)]

		switch pk.WindowID {
		case protocol.WindowIDArmour:
		case protocol.WindowIDOffHand:
		case protocol.WindowIDUI:
		case protocol.WindowIDInventory:
			if !ok {
				break
			}

		default:
			if !ok {
				logrus.Warn(locale.Loc("warn_window_closed_not_open", nil))
				break
			}

			if existing.Content == nil {
				break
			}

			var captureErr error
			w.currentWorld(func(world *worldstate.World) {
				captureErr = w.captureContainerInventory(world, existing)
			})
			if captureErr != nil {
				logrus.WithError(captureErr).Warn("Container contents could not be captured; preserving existing block-entity data")
				break
			}

			w.session.SendMessage(locale.Loc("saved_block_inv", nil))

			delete(w.serverState.openItemContainers, byte(pk.WindowID))
		}
	}

	if w.settings.BlockUpdates {
		switch pk := _pk.(type) {
		case *packet.UpdateBlock:
			w.currentWorld(func(world *worldstate.World) {
				rid, name, properties, found := world.BlockByID(pk.NewBlockRuntimeID)
				if !found {
					return
				}
				if w.scripting != nil {
					if !w.scripting.OnBlockUpdate(name, properties, pk.Position, timeReceived) {
						return
					}
				}
				world.QueueBlockUpdate(pk.Position, rid, uint8(pk.Layer))
			})

		case *packet.UpdateBlockSynced:
			w.currentWorld(func(world *worldstate.World) {
				rid, name, properties, found := world.BlockByID(pk.NewBlockRuntimeID)
				if !found {
					return
				}
				if w.scripting != nil {
					if !w.scripting.OnBlockUpdate(name, properties, pk.Position, timeReceived) {
						return
					}
				}
				world.QueueBlockUpdate(pk.Position, rid, uint8(pk.Layer))
			})

		case *packet.UpdateSubChunkBlocks:
			w.currentWorld(func(world *worldstate.World) {
				for _, block := range pk.Blocks {
					rid, name, properties, found := world.BlockByID(block.BlockRuntimeID)
					if !found {
						break
					}
					if w.scripting != nil {
						if !w.scripting.OnBlockUpdate(name, properties, block.BlockPos, timeReceived) {
							return
						}
					}
					world.QueueBlockUpdate(block.BlockPos, rid, uint8(0))
				}
			})
		}
	}

	return _pk, nil
}

func (w *worldsHandler) packetHandler(_ *proxy.Session, pk packet.Packet, toServer bool, timeReceived time.Time, preLogin bool) (packet.Packet, error) {
	w.worldStateMu.Lock()
	deleted := w.worldState != nil && messages.WorldDeleted(w.worldState.ID)
	w.worldStateMu.Unlock()
	if deleted {
		if _, transition := pk.(*packet.ChangeDimension); !transition {
			return pk, nil
		}
	}
	if w.scripting != nil {
		if !w.scripting.OnPacket(pk, toServer, timeReceived) {
			return nil, nil
		}
	}
	if preLogin {
		return w.packetHandlerPreLogin(pk, toServer, timeReceived)
	}
	return w.packetHandlerIngame(pk, toServer, timeReceived)
}

func (w *worldsHandler) handleSyncActorProperty(pk *packet.SyncActorProperty) {
	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()
	entityType, ok := pk.PropertyData["type"].(string)
	if !ok {
		return
	}
	properties, ok := pk.PropertyData["properties"].([]any)
	if !ok {
		return
	}

	var propertiesOut = make([]entity.EntityPropertyDef, 0, len(properties))
	for _, property := range properties {
		property := property.(map[string]any)
		propertyName, ok := property["name"].(string)
		if !ok {
			continue
		}
		propertyType, ok := property["type"].(int32)
		if !ok {
			continue
		}

		var prop entity.EntityPropertyDef
		prop.Name = propertyName
		prop.Type = propertyType

		switch propertyType {
		case entity.PropertyTypeInt:
			min, ok := property["min"].(int32)
			if !ok {
				continue
			}
			max, ok := property["max"].(int32)
			if !ok {
				continue
			}
			prop.Min = float32(min)
			prop.Max = float32(max)
		case entity.PropertyTypeFloat:
			min, ok := property["min"].(float32)
			if !ok {
				continue
			}
			max, ok := property["max"].(float32)
			if !ok {
				continue
			}
			prop.Min = min
			prop.Max = max
		case entity.PropertyTypeBool:
		case entity.PropertyTypeEnum:
			prop.Enum, _ = property["enum"].([]any)
		default:
			fmt.Printf("Unknown property type %d", propertyType)
		}
		propertiesOut = append(propertiesOut, prop)
	}
	w.serverState.entityProperties[entityType] = propertiesOut
	w.serverState.behaviorPack.SetEntityProperties(entityType, propertiesOut)
	if entityType == "minecraft:player" {
		w.serverState.behaviorPack.SetPlayerProperties(propertiesOut)
	}
}

func applyProperties(defs []entity.EntityPropertyDef, props protocol.EntityProperties, out, overrides map[string]any) (changed map[string]any) {
	changed = make(map[string]any)

	updateVal := func(name string, value any) {
		prev, ok := overrides[name]
		if !ok {
			prev = out[name]
			out[name] = value
		} else {
			overrides[name] = value
		}
		if prev != out[name] {
			changed[name] = prev
		}
	}

	for _, prop := range props.IntegerProperties {
		if int(prop.Index) > len(defs)-1 {
			logrus.Errorf("entity property index more than there are properties, BUG %v", prop)
			continue
		}
		propType := defs[prop.Index]
		var value any = prop.Value
		if propType.Type == entity.PropertyTypeBool {
			value = prop.Value == 1
		}
		if propType.Type == entity.PropertyTypeEnum {
			if prop.Value < 0 || int(prop.Value) >= len(propType.Enum) {
				logrus.Warnf("Invalid enum index for entity property %s", propType.Name)
				continue
			}
			value = propType.Enum[prop.Value]
		}
		updateVal(propType.Name, value)
	}
	for _, prop := range props.FloatProperties {
		if int(prop.Index) > len(defs)-1 {
			logrus.Errorf("entity property index more than there are properties, BUG %v", prop)
			continue
		}
		propType := defs[prop.Index]
		updateVal(propType.Name, prop.Value)
	}
	return changed
}

func (w *worldsHandler) handleActorCatalog(pk *packet.AvailableActorIdentifiers) error {
	var catalog map[string]any
	if err := nbt.UnmarshalEncoding(pk.SerialisedEntityIdentifiers, &catalog, nbt.NetworkLittleEndian); err != nil {
		return fmt.Errorf("decode entity catalog: %w", err)
	}
	entries, _ := catalog["idlist"].([]any)
	ids := make([]string, 0, len(entries))
	w.worldStateMu.Lock()
	for _, item := range entries {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := entry["id"].(string)
		ids = append(ids, id)
		w.serverState.behaviorPack.SetEntityProperties(id, w.serverState.entityProperties[id])
	}
	w.worldStateMu.Unlock()
	w.rememberCommandEntities(ids)
	return nil
}

func (w *worldsHandler) onEntityUpdate(
	ent *entity.Entity,
	prevPosition *mgl32.Vec3,
	changedProperties map[string]any,
) {
	if w.scripting != nil {
		w.scripting.OnEntityUpdate(ent, prevPosition, changedProperties, w.session.Now())
	}
}

func containsStackNetworkID(action protocol.StackRequestAction, target int32) bool {
	switch action := action.(type) {
	case *protocol.TakeStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.PlaceStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.SwapStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.DropStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.DestroyStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.ConsumeStackRequestAction:
		return action.Source.StackNetworkID == target
	case *protocol.CreateStackRequestAction:
		return false
	case *protocol.LabTableCombineStackRequestAction:
		return false
	case *protocol.BeaconPaymentStackRequestAction:
		return false
	case *protocol.MineBlockStackRequestAction:
		return action.StackNetworkID == target
	case *protocol.CraftRecipeStackRequestAction:
		return false
	case *protocol.AutoCraftRecipeStackRequestAction:
		return false
	case *protocol.CraftCreativeStackRequestAction:
		return false
	case *protocol.CraftRecipeOptionalStackRequestAction:
		return false
	case *protocol.CraftGrindstoneRecipeStackRequestAction:
		return false
	case *protocol.CraftLoomRecipeStackRequestAction:
		return false
	case *protocol.CraftNonImplementedStackRequestAction:
		return false
	case *protocol.CraftResultsDeprecatedStackRequestAction:
		return false
	}
	return false
}
