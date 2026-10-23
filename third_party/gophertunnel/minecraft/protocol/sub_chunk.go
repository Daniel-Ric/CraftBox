package protocol

const (
	HeightMapDataNone = iota
	HeightMapDataHasData
	HeightMapDataTooHigh
	HeightMapDataTooLow
	HeightMapDataAllCopied
)

const (
	SubChunkResultUndefined = iota
	SubChunkResultSuccess
	SubChunkResultChunkNotFound
	SubChunkResultInvalidDimension
	SubChunkResultPlayerNotFound
	SubChunkResultIndexOutOfBounds
	SubChunkResultSuccessAllAir
)

type SubChunkEntry struct {
	Offset SubChunkOffset

	Result byte

	RawPayload Optional[[]byte]

	HeightMapType byte

	HeightMapData Optional[HeightMap]

	RenderHeightMapType byte

	RenderHeightMapData Optional[HeightMap]

	BlobHash Optional[uint64]
}

func (x *SubChunkEntry) Marshal(r IO) {
	Single(r, &x.Offset)
	r.Uint8(&x.Result)
	OptionalFunc(r, &x.RawPayload, r.ByteSlice)
	r.Uint8(&x.HeightMapType)

	OptionalMarshaler(r, &x.HeightMapData)
	r.Uint8(&x.RenderHeightMapType)

	OptionalMarshaler(r, &x.RenderHeightMapData)
	OptionalFunc(r, &x.BlobHash, r.Uint64)
}

type SubChunkOffset [3]int8

func (x *SubChunkOffset) Marshal(r IO) {
	r.Int8(&x[0])
	r.Int8(&x[1])
	r.Int8(&x[2])
}

type HeightMap [16][16]int8

func (x *HeightMap) Marshal(r IO) {
	for z := range x {
		n := uint32(16)
		r.Varuint32(&n)
		if n != 16 {
			r.InvalidValue(n, "height map row", "must hold 16 heights")
			return
		}
		for i := range x[z] {
			r.Int8(&x[z][i])
		}
	}
}
