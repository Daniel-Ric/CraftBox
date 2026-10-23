package protocol

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/google/uuid"
)

const (
	ArmSizeSlim = iota
	ArmSizeWide
)

type Skin struct {
	SkinID string

	PlayFabID string

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

	PieceTintColours []PersonaPieceTintColour

	Trusted bool

	OverrideAppearance bool
	ProfileHash        string
}

func (x *Skin) Marshal(r IO) {
	r.String(&x.SkinID)
	r.String(&x.PlayFabID)
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
	if err := x.validate(); err != nil {
		r.InvalidValue(fmt.Sprintf("Skin %v", x.SkinID), "serialised skin", err.Error())
	}
	r.Bool(&x.PremiumSkin)
	r.Bool(&x.PersonaSkin)
	r.Bool(&x.PersonaCapeOnClassicSkin)
	r.Bool(&x.PrimaryUser)
	r.Bool(&x.OverrideAppearance)
	trusted := "false"
	if x.Trusted {
		trusted = "true"
	}
	r.String(&trusted)
	x.Trusted = strings.EqualFold(trusted, "true")
	r.String(&x.ProfileHash)
}

func (x Skin) validate() error {
	if x.SkinImageHeight*x.SkinImageWidth*4 != uint32(len(x.SkinData)) {
		return fmt.Errorf("expected size of skin is %vx%v (%v bytes total), but got %v bytes", x.SkinImageWidth, x.SkinImageHeight, x.SkinImageHeight*x.SkinImageWidth*4, len(x.SkinData))
	}
	if x.CapeImageHeight*x.CapeImageWidth*4 != uint32(len(x.CapeData)) {
		return fmt.Errorf("expected size of cape is %vx%v (%v bytes total), but got %v bytes", x.CapeImageWidth, x.CapeImageHeight, x.CapeImageHeight*x.CapeImageWidth*4, len(x.CapeData))
	}
	for i, animation := range x.Animations {
		if animation.ImageHeight*animation.ImageWidth*4 != uint32(len(animation.ImageData)) {
			return fmt.Errorf("expected size of animation %v is %vx%v (%v bytes total), but got %v bytes", i, animation.ImageWidth, animation.ImageHeight, animation.ImageHeight*animation.ImageWidth*4, len(animation.ImageData))
		}
	}
	return nil
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
