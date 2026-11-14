package protocol

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/google/uuid"
)

const (
	ArmSizeSlim = iota
	ArmSizeWide
)

type Skin struct {
	SkinID string

	SkinResourcePatch []byte

	SkinImageWidth, SkinImageHeight uint32

	SkinData []byte

	Animations []SkinAnimation

	CapeImageWidth, CapeImageHeight uint32

	CapeData []byte

	SkinGeometry []byte

	AnimationData []byte

	GeometryDataEngineVersion []byte

	PremiumSkin bool

	PersonaSkin bool

	PersonaCapeOnClassicSkin bool

	PrimaryUser bool

	CapeID string

	FullID     string
	SkinColour color.RGBA

	ArmSize uint8

	PersonaPieces []PersonaPiece

	PieceTintColours   []PersonaPieceTintColour
	Trusted            Optional[bool]
	OverrideAppearance bool
	ProfileHash        string
}

func (x *Skin) Marshal(r IO) {
	r.String(&x.SkinID)
	r.ByteSlice(&x.SkinResourcePatch)
	r.Uint32(&x.SkinImageWidth)
	r.Uint32(&x.SkinImageHeight)
	r.ByteSlice(&x.SkinData)
	Slice(r, &x.Animations)
	r.Uint32(&x.CapeImageWidth)
	r.Uint32(&x.CapeImageHeight)
	r.ByteSlice(&x.CapeData)
	r.ByteSlice(&x.SkinGeometry)
	r.ByteSlice(&x.GeometryDataEngineVersion)
	r.ByteSlice(&x.AnimationData)
	r.String(&x.CapeID)
	r.String(&x.FullID)
	r.Uint8(&x.ArmSize)
	r.BEARGB(&x.SkinColour)
	Slice(r, &x.PersonaPieces)
	Slice(r, &x.PieceTintColours)
	x.repairDimensions()
	if err := x.validate(); err != nil {
		r.InvalidValue(fmt.Sprintf("Skin %v", x.SkinID), "serialised skin", err.Error())
	}
	r.Bool(&x.PremiumSkin)
	r.Bool(&x.PersonaSkin)
	r.Bool(&x.PersonaCapeOnClassicSkin)
	r.Bool(&x.PrimaryUser)
	r.Bool(&x.OverrideAppearance)
	trustedStr := trustedSkinToString(x.Trusted)
	r.String(&trustedStr)
	trustedSkinFromString(r, &x.Trusted, trustedStr)
	r.String(&x.ProfileHash)
}

func trustedSkinToString(x Optional[bool]) string {
	t, ok := x.Value()
	switch {
	case !ok:
		return "Unset"
	case t:
		return "True"
	default:
		return "False"
	}
}

func trustedSkinFromString(r IO, x *Optional[bool], s string) {
	switch {
	case strings.EqualFold(s, "unset"):
		*x = Optional[bool]{}
	case strings.EqualFold(s, "true"):
		*x = Option(true)
	case strings.EqualFold(s, "false"):
		*x = Option(false)
	default:
		r.InvalidValue(s, "trusted", "unknown trusted skin flag")
	}
}

func (x Skin) validate() error {
	if !imageSizeMatches(x.SkinImageWidth, x.SkinImageHeight, len(x.SkinData)) {
		return fmt.Errorf("invalid skin image: %vx%v dimensions, %v bytes", x.SkinImageWidth, x.SkinImageHeight, len(x.SkinData))
	}
	if !imageSizeMatches(x.CapeImageWidth, x.CapeImageHeight, len(x.CapeData)) {
		return fmt.Errorf("invalid cape image: %vx%v dimensions, %v bytes", x.CapeImageWidth, x.CapeImageHeight, len(x.CapeData))
	}
	for i, animation := range x.Animations {
		if !imageSizeMatches(animation.ImageWidth, animation.ImageHeight, len(animation.ImageData)) {
			return fmt.Errorf("invalid animation %v: %vx%v dimensions, %v bytes", i, animation.ImageWidth, animation.ImageHeight, len(animation.ImageData))
		}
	}
	return nil
}

func imageSizeMatches(width, height uint32, length int) bool {
	return length%4 == 0 && uint64(width)*uint64(height) == uint64(length/4)
}

// Match upstream's tolerance for incorrectly advertised dimensions, without
// guessing non-square images or allocating unbounded cape buffers.
func (x *Skin) repairDimensions() {
	const maxRepairBytes = 4 * 1024 * 1024
	expectedPixels := uint64(x.SkinImageWidth) * uint64(x.SkinImageHeight)
	if uint64(len(x.SkinData)/4) > expectedPixels && len(x.SkinData) <= maxRepairBytes && len(x.SkinData)%4 == 0 {
		side := uint32(math.Sqrt(float64(len(x.SkinData) / 4)))
		if uint64(side)*uint64(side)*4 == uint64(len(x.SkinData)) {
			x.SkinImageWidth, x.SkinImageHeight = side, side
		}
	}
	capePixels := uint64(x.CapeImageWidth) * uint64(x.CapeImageHeight)
	if capePixels <= maxRepairBytes/4 {
		cape := capePixels * 4
		if uint64(len(x.CapeData)) > cape {
			x.CapeData = x.CapeData[:int(cape)]
		}
		if len(x.CapeData) == 0 && cape > 0 {
			x.CapeData = make([]byte, int(cape))
		}
	}
}

const (
	SkinAnimationHead = iota + 1
	SkinAnimationBody32x32
	SkinAnimationBody128x128
)

const (
	ExpressionTypeLinear = iota
	ExpressionTypeBlinking
)

type SkinAnimation struct {
	ImageWidth, ImageHeight uint32

	ImageData []byte

	AnimationType uint32

	FrameCount float32

	ExpressionType uint32
}

func (x *SkinAnimation) Marshal(r IO) {
	r.Uint32(&x.ImageWidth)
	r.Uint32(&x.ImageHeight)
	r.ByteSlice(&x.ImageData)
	r.Varuint32(&x.AnimationType)
	r.Float32(&x.FrameCount)
	r.Varuint32(&x.ExpressionType)
}

const (
	PieceTypeUnknown = iota
	PieceTypeSkeleton
	PieceTypeBody
	PieceTypeSkin
	PieceTypeBottom
	PieceTypeFeet
	PieceTypeDress
	PieceTypeTop
	PieceTypeHighPants
	PieceTypeHands
	PieceTypeOuterwear
	PieceTypeFacialHair
	PieceTypeMouth
	PieceTypeEyes
	PieceTypeHair
	PieceTypeHood
	PieceTypeBack
	PieceTypeFaceAccessory
	PieceTypeHead
	PieceTypeLegs
	PieceTypeLeftLeg
	PieceTypeRightLeg
	PieceTypeArms
	PieceTypeLeftArm
	PieceTypeRightArm
	PieceTypeCapes
	PieceTypeClassicSkin
	PieceTypeEmote
	PieceTypeUnsupported
)

type PersonaPiece struct {
	PieceID string

	PieceType uint32

	PackID uuid.UUID

	Default bool

	ProductID string
}

func (x *PersonaPiece) Marshal(r IO) {
	r.String(&x.PieceID)
	r.Uint32(&x.PieceType)
	r.UUID(&x.PackID)
	r.Bool(&x.Default)
	r.String(&x.ProductID)
}

type PersonaPieceTintColour struct {
	PieceType string

	Colours [4]color.RGBA
}

func (x *PersonaPieceTintColour) Marshal(r IO) {
	wireType := personaPieceTintWireType(x.PieceType)
	r.String(&wireType)
	if wireType != personaPieceTintWireType(x.PieceType) {
		x.PieceType = personaPieceTintLoginType(wireType)
	}
	for i := range x.Colours {
		r.BEARGB(&x.Colours[i])
	}
}

func personaPieceTintWireType(pieceType string) string {
	if pieceType == "persona_hand" {
		return "hands"
	}
	return strings.TrimPrefix(pieceType, "persona_")
}

func personaPieceTintLoginType(pieceType string) string {
	if pieceType == "hands" {
		return "persona_hand"
	}
	if pieceType == "unsupported" {
		return pieceType
	}
	return "persona_" + pieceType
}
