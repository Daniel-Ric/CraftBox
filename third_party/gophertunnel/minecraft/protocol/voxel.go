package protocol

type VoxelCells struct {
	XSize uint8

	YSize uint8

	ZSize uint8

	Storage []uint8
}

func (x *VoxelCells) Marshal(r IO) {
	r.Uint8(&x.XSize)
	r.Uint8(&x.YSize)
	r.Uint8(&x.ZSize)
	FuncSlice(r, &x.Storage, r.Uint8)
}

type VoxelShapeNameEntry struct {
	Name string

	ID uint16
}

func (x *VoxelShapeNameEntry) Marshal(r IO) {
	r.String(&x.Name)
	r.Uint16(&x.ID)
}

type VoxelShape struct {
	Cells VoxelCells

	XCoordinates []float32

	YCoordinates []float32

	ZCoordinates []float32
}

func (x *VoxelShape) Marshal(r IO) {
	Single(r, &x.Cells)
	FuncSlice(r, &x.XCoordinates, r.Float32)
	FuncSlice(r, &x.YCoordinates, r.Float32)
	FuncSlice(r, &x.ZCoordinates, r.Float32)
}
