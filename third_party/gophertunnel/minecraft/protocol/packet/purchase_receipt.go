package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PurchaseReceipt struct {
	Receipts []string
}

func (*PurchaseReceipt) ID() uint32 {
	return IDPurchaseReceipt
}

func (pk *PurchaseReceipt) Marshal(io protocol.IO) {
	protocol.FuncSlice(io, &pk.Receipts, io.String)
}
