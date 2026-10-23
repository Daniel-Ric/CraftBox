package dialogue

import "encoding/json"

type Button struct {
	Text string
}

func (b Button) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"button_name": b.Text,
		"text":        "",
		"mode":        0,
		"type":        1,
	})
}
