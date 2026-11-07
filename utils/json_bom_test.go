package utils

import "testing"

func TestParsePackJSONWithBOM(t *testing.T) {
	var doc struct {
		Client struct {
			Description struct {
				Identifier string `json:"identifier"`
			} `json:"description"`
		} `json:"minecraft:client_entity"`
	}
	raw := append([]byte{0xef, 0xbb, 0xbf}, []byte(`{"minecraft:client_entity":{"description":{"identifier":"goe_frm:buffalo"}}}`)...)
	if err := ParseJson(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Client.Description.Identifier != "goe_frm:buffalo" {
		t.Fatal("lost entity identifier")
	}
}
