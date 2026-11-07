package packetinspector

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type sendPreview struct {
	Hash    [32]byte
	Expires time.Time
}

func sendHash(r SendRequest, p Packet) [32]byte {
	raw, _ := json.Marshal(struct {
		Generation, Direction string
		ID                    uint32
		Raw                   []byte
	}{r.Generation, r.Direction, r.PacketID, p.Raw})
	return sha256.Sum256(raw)
}
func (c *Controller) Preview(r SendRequest, shield int32) (Packet, error) {
	_, p, e := PrepareSend(r, shield)
	if e != nil {
		return p, e
	}
	var token [16]byte
	if _, e = rand.Read(token[:]); e != nil {
		return p, e
	}
	p.PreviewToken = hex.EncodeToString(token[:])
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.previews) >= 64 {
		c.previews = map[string]sendPreview{}
	}
	c.previews[p.PreviewToken] = sendPreview{Hash: sendHash(r, p), Expires: time.Now().Add(3 * time.Minute)}
	return p, nil
}
func (c *Controller) consumePreview(r SendRequest, p Packet) error {
	preview, ok := c.previews[r.PreviewToken]
	if !ok || time.Now().After(preview.Expires) || preview.Hash != sendHash(r, p) {
		return errors.New("Preview this exact packet and destination again before sending")
	}
	delete(c.previews, r.PreviewToken)
	return nil
}
