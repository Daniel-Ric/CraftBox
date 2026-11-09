package modelpicker

import (
	"errors"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"math"
	"sync"
	"time"
)

type Actor struct {
	RuntimeID   uint64     `json:"runtimeId"`
	UniqueID    int64      `json:"-"`
	Identifier  string     `json:"identifier"`
	Position    mgl32.Vec3 `json:"position"`
	Width       float32    `json:"-"`
	Height      float32    `json:"-"`
	Scale       float32    `json:"scale"`
	Variant     int32      `json:"variant"`
	MarkVariant int32      `json:"markVariant"`
	SkinID      int32      `json:"skinId"`
	Distance    float64    `json:"distance"`
}
type Tracker struct {
	mu      sync.Mutex
	actors  map[uint64]Actor
	unique  map[int64]uint64
	hover   uint64
	hovered time.Time
}

func New() *Tracker { return &Tracker{actors: map[uint64]Actor{}, unique: map[int64]uint64{}} }
func (t *Tracker) Hover(runtime uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hover = runtime
	t.hovered = time.Now()
}
func metadata(a *Actor, m map[uint32]any) {
	if v, ok := m[protocol.EntityDataKeyVariant].(int32); ok {
		a.Variant = v
	}
	if v, ok := m[protocol.EntityDataKeyMarkVariant].(int32); ok {
		a.MarkVariant = v
	}
	if v, ok := m[protocol.EntityDataKeySkinID].(int32); ok {
		a.SkinID = v
	}
	if v, ok := m[protocol.EntityDataKeyWidth].(float32); ok && v > 0 && v < 128 {
		a.Width = v
	}
	if v, ok := m[protocol.EntityDataKeyHeight].(float32); ok && v > 0 && v < 128 {
		a.Height = v
	}
	if v, ok := m[protocol.EntityDataKeyScale].(float32); ok && v > 0 && v < 128 {
		a.Scale = v
	}
}
func (t *Tracker) Observe(pk packet.Packet, toServer bool) {
	if toServer {
		if p, ok := pk.(*packet.Interact); ok && p.ActionType == packet.InteractActionMouseOverEntity {
			t.Hover(p.TargetEntityRuntimeID)
		}
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	switch p := pk.(type) {
	case *packet.AddActor:
		if len(t.actors) >= 16384 {
			return
		}
		a := Actor{RuntimeID: p.EntityRuntimeID, UniqueID: p.EntityUniqueID, Identifier: p.EntityType, Position: p.Position, Width: 1, Height: 2, Scale: 1}
		metadata(&a, p.EntityMetadata)
		t.actors[a.RuntimeID] = a
		t.unique[a.UniqueID] = a.RuntimeID
	case *packet.RemoveActor:
		delete(t.actors, t.unique[p.EntityUniqueID])
		delete(t.unique, p.EntityUniqueID)
	case *packet.ChangeDimension:
		t.actors = map[uint64]Actor{}
		t.unique = map[int64]uint64{}
		t.hover = 0
	case *packet.MoveActorAbsolute:
		if a, ok := t.actors[p.EntityRuntimeID]; ok {
			a.Position = p.Position
			t.actors[a.RuntimeID] = a
		}
	case *packet.MoveActorDelta:
		if a, ok := t.actors[p.EntityRuntimeID]; ok {
			if v, present := p.PositionX.Value(); present {
				a.Position[0] = v
			}
			if v, present := p.PositionY.Value(); present {
				a.Position[1] = v
			}
			if v, present := p.PositionZ.Value(); present {
				a.Position[2] = v
			}
			t.actors[a.RuntimeID] = a
		}
	case *packet.SetActorData:
		if a, ok := t.actors[p.EntityRuntimeID]; ok {
			metadata(&a, p.EntityMetadata)
			t.actors[a.RuntimeID] = a
		}
	}
}
func (t *Tracker) Pick(eye mgl32.Vec3, pitch, yaw float32) (Actor, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if a, ok := t.actors[t.hover]; ok && time.Since(t.hovered) < 2*time.Second && a.Position.Sub(eye).Len() < 96 {
		a.Distance = hoverDistance(a, eye, pitch, yaw)
		return a, nil
	}
	p, y := float64(pitch)*math.Pi/180, float64(yaw)*math.Pi/180
	direction := [3]float64{-math.Sin(y) * math.Cos(p), -math.Sin(p), math.Cos(y) * math.Cos(p)}
	nearest := 96.0
	var found Actor
	for _, a := range t.actors {
		width := float64(a.Width*a.Scale) / 2
		height := float64(a.Height * a.Scale)
		minimum := [3]float64{float64(a.Position[0]) - width, float64(a.Position[1]), float64(a.Position[2]) - width}
		maximum := [3]float64{float64(a.Position[0]) + width, float64(a.Position[1]) + height, float64(a.Position[2]) + width}
		lo, hi := 0.0, nearest
		for axis := 0; axis < 3; axis++ {
			origin := float64(eye[axis])
			d := direction[axis]
			if math.Abs(d) < 1e-8 {
				if origin < minimum[axis] || origin > maximum[axis] {
					hi = -1
					break
				}
				continue
			}
			start, end := (minimum[axis]-origin)/d, (maximum[axis]-origin)/d
			if start > end {
				start, end = end, start
			}
			lo = math.Max(lo, start)
			hi = math.Min(hi, end)
		}
		if hi >= lo && lo < nearest {
			nearest = lo
			found = a
			found.Distance = lo
		}
	}
	if found.RuntimeID == 0 {
		return Actor{}, errors.New("No captured entity under the crosshair. Aim at the model and press F8 again")
	}
	return found, nil
}

func hoverDistance(actor Actor, eye mgl32.Vec3, pitch, yaw float32) float64 {
	p, y := float64(pitch)*math.Pi/180, float64(yaw)*math.Pi/180
	direction := [3]float64{-math.Sin(y) * math.Cos(p), -math.Sin(p), math.Cos(y) * math.Cos(p)}
	width, height := float64(actor.Width*actor.Scale)/2, float64(actor.Height*actor.Scale)
	minimum := [3]float64{float64(actor.Position[0]) - width, float64(actor.Position[1]), float64(actor.Position[2]) - width}
	maximum := [3]float64{float64(actor.Position[0]) + width, float64(actor.Position[1]) + height, float64(actor.Position[2]) + width}
	lo, hi := 0.0, 96.0
	for axis, d := range direction {
		origin := float64(eye[axis])
		if math.Abs(d) < 1e-8 {
			if origin < minimum[axis] || origin > maximum[axis] {
				return math.Max(0, float64(actor.Position.Sub(eye).Len())-width)
			}
			continue
		}
		a, b := (minimum[axis]-origin)/d, (maximum[axis]-origin)/d
		if a > b {
			a, b = b, a
		}
		lo = math.Max(lo, a)
		hi = math.Min(hi, b)
	}
	if hi >= lo {
		return lo
	}
	return math.Max(0, float64(actor.Position.Sub(eye).Len())-width)
}
