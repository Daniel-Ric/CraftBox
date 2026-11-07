package scene

import (
	"bytes"

	"github.com/tailscale/hujson"
)

// Resource packs may prefix UTF-8 JSON with a byte-order mark. Remove only
// that leading encoding marker; keep normal JSONC validation and string data.
func standardizePackJSON(raw []byte) ([]byte, error) {
	return hujson.Standardize(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
}
