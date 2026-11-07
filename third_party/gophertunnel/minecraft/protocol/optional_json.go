package protocol

import (
	"bytes"
	"encoding/json"
)

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.set {
		return []byte("null"), nil
	}
	return json.Marshal(o.val)
}
func (o *Optional[T]) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("{}")) {
		var zero Optional[T]
		*o = zero
		return nil
	}
	var value T
	if e := json.Unmarshal(raw, &value); e != nil {
		return e
	}
	*o = Option(value)
	return nil
}
