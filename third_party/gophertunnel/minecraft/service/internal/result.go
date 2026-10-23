package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Result[T any] struct {
	Data T `json:"result"`
}

type Error struct {
	Namespace string `json:"namespace"`

	Code string `json:"code"`

	Message string `json:"message"`

	CustomData json.RawMessage `json:"customData"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("minecraft/service: %s: %q (%s)", e.Code, e.Message, e.Namespace)
}

func Err(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	e := new(Error)
	if err := json.Unmarshal(body, e); err != nil {
		if len(body) > 0 {
			return fmt.Errorf("%s %s: %s: %q", resp.Request.Method, resp.Request.URL, resp.Status, body)
		}
		return fmt.Errorf("%s %s: %s", resp.Request.Method, resp.Request.URL, resp.Status)
	}
	return e
}

const UserAgent = "libhttpclient/1.0.0.0"
