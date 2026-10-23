package trace

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"iter"
	"math"
)

type Result interface {
	BBox() cube.BBox

	Position() mgl64.Vec3

	Face() cube.Face
}

type EntityFilter func(iter.Seq[world.Entity]) iter.Seq[world.Entity]

func Perform(start, end mgl64.Vec3, tx *world.Tx, box cube.BBox, filter EntityFilter) (hit Result, ok bool) {

	TraverseBlocks(start, end, func(pos cube.Pos) (cont bool) {
		b := tx.Block(pos)

		if result, ok := BlockIntercept(pos, tx, b, start, end); ok {
			hit = result
			end = hit.Position()
			return false
		}
		return true
	})

	dist := math.MaxFloat64
	bb := box.Translate(start).Extend(end.Sub(start))
	entities := tx.EntitiesWithin(bb.Grow(8.0))
	if filter != nil {
		entities = filter(entities)
	}
	for entity := range entities {
		if !entity.H().Type().BBox(entity).Translate(entity.Position()).IntersectsWith(bb) {
			continue
		}

		result, ok := EntityIntercept(entity, start, end)
		if !ok {
			continue
		}

		if distance := start.Sub(result.Position()).LenSqr(); distance < dist {
			dist = distance
			hit = result
		}
	}

	return hit, hit != nil
}
