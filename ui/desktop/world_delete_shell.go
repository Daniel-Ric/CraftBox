//go:build gui && !android

package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (a *MapWindowApp) DeleteCapturedWorld(id string) error {
	data, _ := json.Marshal(map[string]string{"worldId": id})
	ctx, cancel := context.WithTimeout(a.ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(a.address, "/map-window")+"/world-delete", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	deleteClient := *a.client
	deleteClient.Timeout = 2 * time.Minute
	response, err := deleteClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	return a.refresh()
}
