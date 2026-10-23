package block

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
)

type BannerPatternLayer struct {
	Type BannerPatternType

	Colour item.Colour
}

func (b BannerPatternLayer) EncodeNBT() map[string]any {
	return map[string]any{
		"Pattern": bannerPatternID(b.Type),
		"Color":   int32(invertColour(b.Colour)),
	}
}

func (b BannerPatternLayer) DecodeNBT(data map[string]any) any {
	id := nbtconv.String(data, "Pattern")
	pattern, exists := BannerPatternByID(id)
	if !exists {
		panic(fmt.Errorf("unknown banner pattern id %q", id))
	}
	b.Type = pattern
	b.Colour = invertColourID(int16(nbtconv.Int32(data, "Color")))
	return b
}
