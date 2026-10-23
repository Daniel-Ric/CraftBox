package protocol

type CacheBlob struct {
	Hash uint64

	Payload []byte
}

func (x *CacheBlob) Marshal(r IO) {
	r.Uint64(&x.Hash)
	r.ByteSlice(&x.Payload)
}
