package handlers

import (
	"encoding/base64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func loginSkin(data *login.ClientData) *protocol.Skin {
	if data.SkinImageWidth < 1 || data.SkinImageHeight < 1 || data.SkinImageWidth > 2048 || data.SkinImageHeight > 2048 {
		return nil
	}
	skin := &protocol.Skin{SkinID: data.SkinID, SkinImageWidth: uint32(data.SkinImageWidth), SkinImageHeight: uint32(data.SkinImageHeight), CapeID: data.CapeID, CapeImageWidth: uint32(data.CapeImageWidth), CapeImageHeight: uint32(data.CapeImageHeight), ArmSize: protocol.ArmSizeWide, PersonaSkin: data.PersonaSkin, GeometryDataEngineVersion: []byte(data.SkinGeometryVersion)}
	if data.ArmSize == "slim" {
		skin.ArmSize = protocol.ArmSizeSlim
	}
	values := []struct {
		target *[]byte
		value  string
	}{{&skin.SkinData, data.SkinData}, {&skin.CapeData, data.CapeData}, {&skin.SkinResourcePatch, data.SkinResourcePatch}, {&skin.SkinGeometry, data.SkinGeometry}, {&skin.AnimationData, data.SkinAnimationData}}
	for _, value := range values {
		decoded, err := base64.StdEncoding.DecodeString(value.value)
		if err != nil {
			return nil
		}
		*value.target = decoded
	}
	if len(skin.SkinData) != data.SkinImageWidth*data.SkinImageHeight*4 {
		return nil
	}
	for _, frame := range data.AnimatedImageData {
		if len(skin.Animations) >= 16 || len(frame.Image) > 12<<20 {
			continue
		}
		if frame.ImageWidth < 1 || frame.ImageHeight < 1 || frame.ImageWidth > 2048 || frame.ImageHeight > 8192 {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(frame.Image)
		if err != nil || len(decoded) != frame.ImageWidth*frame.ImageHeight*4 {
			continue
		}
		skin.Animations = append(skin.Animations, protocol.SkinAnimation{ImageWidth: uint32(frame.ImageWidth), ImageHeight: uint32(frame.ImageHeight), ImageData: decoded, AnimationType: uint32(frame.Type), FrameCount: float32(frame.Frames), ExpressionType: uint32(frame.AnimationExpression)})
	}
	return skin
}

func (s *SkinSaver) equipmentEvents(runtime uint64) []SkinAdd {
	gear, ok := s.gear[runtime]
	if !ok || runtime == 0 {
		return nil
	}
	for _, player := range s.playersById {
		if player.RuntimeID != runtime {
			continue
		}
		equipment := map[string]string{}
		for slot, item := range gear {
			if item.Stack.Count == 0 || item.Stack.NetworkID == 0 {
				equipment[slot] = ""
				continue
			}
			if name := s.itemNames[item.Stack.NetworkID]; name != "" {
				equipment[slot] = name
			}
		}
		if len(equipment) == 0 {
			return nil
		}
		return []SkinAdd{{PlayerName: player.Name, PlayerUUID: player.UUID.String(), XUID: player.XUID, Equipment: equipment}}
	}
	return nil
}
