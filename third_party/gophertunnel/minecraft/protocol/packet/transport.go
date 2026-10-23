package packet

type BatchHeaderer interface {
	BatchHeader() []byte
}

type EncryptionDisabler interface {
	DisableEncryption() bool
}

type PacketReader interface {
	ReadPacket() ([]byte, error)
}

type TransportCapabilities interface {
	BatchHeaderer
	EncryptionDisabler
	PacketReader
}
