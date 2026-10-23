package protocol

import (
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShapeDataLast = iota
	ShapeDataArrow
	ShapeDataText
	ShapeDataBox
	ShapeDataLine
	ShapeDataSphere
	ShapeDataCylinder
	ShapeDataPyramid
	ShapeDataEllipsoid
	ShapeDataCone
)

func lookupShapeData(shapeDataType uint32, x *ShapeData) bool {
	switch shapeDataType {
	case ShapeDataLast:
		*x = &LastShape{}
	case ShapeDataArrow:
		*x = &ArrowShape{}
	case ShapeDataText:
		*x = &TextShape{}
	case ShapeDataBox:
		*x = &BoxShape{}
	case ShapeDataLine:
		*x = &LineShape{}
	case ShapeDataSphere:
		*x = &SphereShape{}
	case ShapeDataCylinder:
		*x = &CylinderShape{}
	case ShapeDataPyramid:
		*x = &PyramidShape{}
	case ShapeDataEllipsoid:
		*x = &EllipsoidShape{}
	case ShapeDataCone:
		*x = &ConeShape{}
	default:
		return false
	}
	return true
}

func lookupShapeDataType(x ShapeData, shapeDataType *uint32) bool {
	switch x.(type) {
	case *LastShape:
		*shapeDataType = ShapeDataLast
	case *ArrowShape:
		*shapeDataType = ShapeDataArrow
	case *TextShape:
		*shapeDataType = ShapeDataText
	case *BoxShape:
		*shapeDataType = ShapeDataBox
	case *LineShape:
		*shapeDataType = ShapeDataLine
	case *SphereShape:
		*shapeDataType = ShapeDataSphere
	case *CylinderShape:
		*shapeDataType = ShapeDataCylinder
	case *PyramidShape:
		*shapeDataType = ShapeDataPyramid
	case *EllipsoidShape:
		*shapeDataType = ShapeDataEllipsoid
	case *ConeShape:
		*shapeDataType = ShapeDataCone
	default:
		return false
	}
	return true
}

type ShapeData interface {
	Marshal(r IO)
}

type LastShape struct{}

func (shape *LastShape) Marshal(io IO) {}

type LineShape struct {
	LineEndLocation mgl32.Vec3
}

func (shape *LineShape) Marshal(io IO) {
	io.Vec3(&shape.LineEndLocation)
}

type TextShape struct {
	Text string

	UseRotation bool

	BackgroundColour Optional[color.RGBA]

	LineGapHeight float32

	DepthTest bool

	ShowBackface bool

	ShowBackfaceText bool
}

func (shape *TextShape) Marshal(io IO) {
	io.String(&shape.Text)
	io.Bool(&shape.UseRotation)
	OptionalFunc(io, &shape.BackgroundColour, io.BEARGB)
	io.Float32(&shape.LineGapHeight)
	io.Bool(&shape.DepthTest)
	io.Bool(&shape.ShowBackface)
	io.Bool(&shape.ShowBackfaceText)
}

type BoxShape struct {
	BoxBound mgl32.Vec3
}

func (shape *BoxShape) Marshal(io IO) {
	io.Vec3(&shape.BoxBound)
}

type SphereShape struct {
	Segments byte
}

func (shape *SphereShape) Marshal(io IO) {
	io.Uint8(&shape.Segments)
}

type ArrowShape struct {
	ArrowEndLocation Optional[mgl32.Vec3]

	ArrowHeadLength Optional[float32]

	ArrowHeadRadius Optional[float32]

	Segments Optional[byte]
}

func (shape *ArrowShape) Marshal(io IO) {
	OptionalFunc(io, &shape.ArrowEndLocation, io.Vec3)
	OptionalFunc(io, &shape.ArrowHeadLength, io.Float32)
	OptionalFunc(io, &shape.ArrowHeadRadius, io.Float32)
	OptionalFunc(io, &shape.Segments, io.Uint8)
}

type CylinderShape struct {
	RadiusX mgl32.Vec2

	RadiusZ mgl32.Vec2

	Height float32

	NumSegments byte
}

func (shape *CylinderShape) Marshal(io IO) {
	io.Vec2(&shape.RadiusX)
	io.Vec2(&shape.RadiusZ)
	io.Float32(&shape.Height)
	io.Uint8(&shape.NumSegments)
}

type PyramidShape struct {
	Width float32

	Depth Optional[float32]

	Height float32
}

func (shape *PyramidShape) Marshal(io IO) {
	io.Float32(&shape.Width)
	OptionalFunc(io, &shape.Depth, io.Float32)
	io.Float32(&shape.Height)
}

type EllipsoidShape struct {
	Radii mgl32.Vec3

	SegmentsPerAxis byte
}

func (shape *EllipsoidShape) Marshal(io IO) {
	io.Vec3(&shape.Radii)
	io.Uint8(&shape.SegmentsPerAxis)
}

type ConeShape struct {
	Radii mgl32.Vec2

	Height float32

	NumSegments byte
}

func (shape *ConeShape) Marshal(io IO) {
	io.Vec2(&shape.Radii)
	io.Float32(&shape.Height)
	io.Uint8(&shape.NumSegments)
}

const (
	PrimitiveShapeLine uint8 = iota
	PrimitiveShapeBox
	PrimitiveShapeSphere
	PrimitiveShapeCircle
	PrimitiveShapeText
	PrimitiveShapeArrow
	PrimitiveShapeCylinder
	PrimitiveShapePyramid
	PrimitiveShapeEllipsoid
	PrimitiveShapeCone
)

type PrimitiveShape struct {
	NetworkID uint64

	DimensionID Optional[int32]

	AttachedToEntityID Optional[int64]

	Type Optional[uint8]

	Location Optional[mgl32.Vec3]

	Scale Optional[float32]

	Rotation Optional[mgl32.Vec3]

	TotalTimeLeft Optional[float32]

	MaxRenderDistance Optional[float32]

	Colour Optional[color.RGBA]

	ExtraShapeData ShapeData
}

func (x *PrimitiveShape) Marshal(io IO) {
	io.Varuint64(&x.NetworkID)
	OptionalFunc(io, &x.Type, io.Uint8)
	OptionalFunc(io, &x.Location, io.Vec3)
	OptionalFunc(io, &x.Scale, io.Float32)
	OptionalFunc(io, &x.Rotation, io.Vec3)
	OptionalFunc(io, &x.TotalTimeLeft, io.Float32)
	OptionalFunc(io, &x.MaxRenderDistance, io.Float32)
	OptionalFunc(io, &x.Colour, io.BEARGB)
	OptionalFunc(io, &x.DimensionID, io.Varint32)
	OptionalFunc(io, &x.AttachedToEntityID, io.ActorUniqueID)
	io.ShapeData(&x.ExtraShapeData)
}
