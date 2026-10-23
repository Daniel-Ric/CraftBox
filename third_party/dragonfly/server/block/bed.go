package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type Bed struct {
	transparent
	sourceWaterDisplacer

	Colour item.Colour

	Facing cube.Direction

	Head bool

	Occupied bool

	Sleeper *world.EntityHandle
}

func (Bed) MaxCount() int {
	return 1
}

func (Bed) Model() world.BlockModel {
	return model.Bed{}
}

func (Bed) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (b Bed) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, nothingEffective, oneOf(b)).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		headSide, _, ok := b.head(pos, tx)
		if !ok {
			return
		}

		s := headSide.Sleeper
		if s == nil {
			return
		}

		ent, ok := s.Entity(tx)
		if !ok {
			return
		}

		sleeper, ok := ent.(world.Sleeper)
		if ok {
			sleeper.Wake()
		}
	})
}

func (b Bed) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	if pos, _, used = firstReplaceable(tx, pos, face, b); !used {
		return
	}
	if !supportedFromBelow(pos, tx) {
		return
	}

	b.Facing = user.Rotation().Direction()

	side, sidePos := b, pos.Side(b.Facing.Face())
	side.Head = true

	if !replaceableWith(tx, sidePos, side) {
		return
	}
	if !supportedFromBelow(sidePos, tx) {
		return
	}

	ctx.IgnoreBBox = true
	place(tx, sidePos, side, user, ctx)
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

func (b Bed) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	s, ok := u.(world.Sleeper)
	if !ok {
		return false
	}

	w := tx.World()
	if w.Dimension() != world.Overworld {
		tx.SetBlock(pos, nil, nil)
		ExplosionConfig{
			SpawnFire: true,
		}.Explode(tx, world.BlockExplosionSource{
			Block:         b,
			Pos:           pos,
			ExplosionSize: 5,
		})
		return true
	}

	_, sidePos, ok := b.side(pos, tx)
	if !ok {
		return false
	}

	userPos := s.Position()
	if sidePos.Vec3Middle().Sub(userPos).Len() > 2 && pos.Vec3Middle().Sub(userPos).Len() > 2 {
		s.Messaget(chat.MessageBedTooFar)
		return true
	}

	headSide, headPos, ok := b.head(pos, tx)
	if !ok {
		return false
	}

	if _, safeSpawn := b.SafeSpawn(pos, tx); !safeSpawn {
		s.Messaget(chat.MessageBedObstructed)
		return false
	}

	if _, ok = tx.Liquid(headPos); ok {
		return false
	}

	previousSpawn := w.PlayerSpawn(s.UUID())
	if previousSpawn != headPos {
		w.SetPlayerSpawn(s.UUID(), headPos)
		s.Messaget(chat.MessageRespawnPointSet)
	}

	time := w.Time() % world.TimeFull
	if !tx.Thundering() {
		if !tx.Raining() && (time <= world.TimeSleep || time >= world.TimeWake) {
			s.Messaget(chat.MessageNoSleep)
			return true
		}
		if time <= world.TimeSleepWithRain || time >= world.TimeWakeWithRain {
			s.Messaget(chat.MessageNoSleep)
			return true
		}
	}
	if headSide.Sleeper != nil {
		s.Messaget(chat.MessageBedIsOccupied)
		return true
	}

	s.Sleep(headPos)
	return true
}

func (b Bed) SleepingEntity() *world.EntityHandle {
	return b.Sleeper
}

func (b Bed) StartSleeping(pos cube.Pos, tx *world.Tx, e *world.EntityHandle) {
	b.setOccupied(pos, tx, true)
	b.Sleeper = e
	b.Occupied = true
	tx.SetBlock(pos, b, nil)
}

func (b Bed) StopSleeping(pos cube.Pos, tx *world.Tx) {
	b.setOccupied(pos, tx, false)
	b.Sleeper = nil
	b.Occupied = false
	tx.SetBlock(pos, b, nil)
}

func (b Bed) setOccupied(pos cube.Pos, tx *world.Tx, occupied bool) {
	if side, sidePos, ok := b.side(pos, tx); ok {
		side.Occupied = occupied
		tx.SetBlock(sidePos, side, nil)
	}
}

func (b Bed) EntityLand(_ cube.Pos, _ *world.Tx, e world.Entity, distance *float64) {
	if _, ok := e.(fallDistanceEntity); ok {
		*distance *= 0.5
	}
	if v, ok := e.(velocityEntity); ok {
		vel := v.Velocity()
		vel[1] = vel[1] * -2 / 3
		v.SetVelocity(vel)
	}
}

type velocityEntity interface {
	Velocity() mgl64.Vec3

	SetVelocity(mgl64.Vec3)
}

func (b Bed) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if _, _, ok := b.side(pos, tx); !ok {
		breakBlockNoDrops(b, pos, tx)
	}
}

func (b Bed) EncodeItem() (name string, meta int16) {
	return "minecraft:bed", int16(b.Colour.Uint8())
}

func (b Bed) EncodeBlock() (name string, properties map[string]interface{}) {
	return "minecraft:bed", map[string]interface{}{
		"direction":      int32(horizontalDirection(b.Facing)),
		"occupied_bit":   boolByte(b.Occupied),
		"head_piece_bit": boolByte(b.Head),
	}
}

func (b Bed) EncodeNBT() map[string]interface{} {
	return map[string]interface{}{
		"id":    "Bed",
		"color": b.Colour.Uint8(),
	}
}

func (b Bed) DecodeNBT(data map[string]interface{}) interface{} {
	b.Colour = item.Colours()[nbtconv.Uint8(data, "color")]
	return b
}

func (b Bed) head(pos cube.Pos, tx *world.Tx) (Bed, cube.Pos, bool) {
	headSide, headPos, ok := b.side(pos, tx)
	if !ok {
		return Bed{}, cube.Pos{}, false
	}
	if b.Head {
		return b, pos, true
	}
	return headSide, headPos, true
}

func (b Bed) side(pos cube.Pos, tx *world.Tx) (Bed, cube.Pos, bool) {
	sidePos := bedSide(pos, b.Facing, b.Head)
	o, ok := tx.Block(sidePos).(Bed)
	return o, sidePos, ok
}

func bedSide(pos cube.Pos, facing cube.Direction, head bool) cube.Pos {
	face := facing.Face()
	if head {
		face = face.Opposite()
	}
	return pos.Side(face)
}

func allBeds() (beds []world.Block) {
	for _, d := range cube.Directions() {
		for _, head := range []bool{false, true} {
			for _, occupied := range []bool{false, true} {
				beds = append(beds, Bed{Facing: d, Head: head, Occupied: occupied})
			}
		}
	}
	return
}

func (Bed) CanRespawnOn() bool {
	return true
}

var bedOffsets = map[cube.Face][]cube.Pos{
	cube.FaceNorth: {{-1, 0, 0}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {1, 0, 0}, {1, 0, -1}, {1, 0, -2}, {0, 0, -2}, {-1, 0, -2}, {-1, 0, -1}, {0, 1, -1}, {0, 1, 0}},
	cube.FaceEast:  {{0, 0, -1}, {-1, 0, -1}, {-1, 0, 0}, {-1, 0, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {2, 0, 1}, {2, 0, 0}, {2, 0, -1}, {1, 0, -1}, {1, 1, 0}, {0, 1, 0}},
	cube.FaceSouth: {{1, 0, 0}, {1, 0, -1}, {0, 0, -1}, {-1, 0, -1}, {-1, 0, 0}, {-1, 0, 1}, {-1, 0, 2}, {0, 0, 2}, {1, 0, 2}, {1, 0, 1}, {0, 1, 1}, {0, 1, 0}},
	cube.FaceWest:  {{0, 0, 1}, {1, 0, 1}, {1, 0, 0}, {1, 0, -1}, {1, 0, -1}, {0, 0, -1}, {-1, 0, -1}, {-2, 0, -1}, {-2, 0, 0}, {-2, 0, 1}, {-1, 0, 1}, {-1, 1, 0}, {0, 1, 0}},
}

func (b Bed) SafeSpawn(pos cube.Pos, tx *world.Tx) (cube.Pos, bool) {
	_, headPos, ok := b.head(pos, tx)
	if !ok {
		return cube.Pos{}, false
	}

	heelPos := headPos.Side(b.Facing.Opposite().Face())

	for _, offset := range bedOffsets[b.Facing.Face()] {
		offsetPos := heelPos.Add(offset)

		if _, solidBlock := tx.Block(offsetPos).Model().(model.Solid); solidBlock {
			if diffuser, ok := tx.Block(offsetPos).(LightDiffuser); !ok || diffuser.LightDiffusionLevel() != 0 {
				continue
			}
		}

		if _, emptyBlock := tx.Block(offsetPos.Side(cube.FaceDown)).Model().(model.Empty); emptyBlock {
			continue
		}

		return heelPos.Add(offset), true
	}
	return cube.Pos{}, false
}

func supportedFromBelow(pos cube.Pos, tx *world.Tx) bool {
	below := pos.Side(cube.FaceDown)
	return tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx)
}
