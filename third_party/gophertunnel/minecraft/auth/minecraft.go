package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/df-mc/go-xsapi/v2"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

var minecraftAuthURL = &url.URL{
	Scheme: "https",
	Host:   "multiplayer.minecraft.net",
	Path:   "/authentication",
}

func RequestMinecraftChain(ctx context.Context, client *xsapi.Client, key *ecdsa.PrivateKey) (string, error) {
	data, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}

	body := `{"identityPublicKey":"` + base64.StdEncoding.EncodeToString(data) + `"}`
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, minecraftAuthURL.String(), strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("POST %v: %w", minecraftAuthURL, err)
	}

	request.Header.Set("User-Agent", "MCPE/Android")
	request.Header.Set("Client-Version", protocol.CurrentVersion)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient().Do(xsapi.WithoutAuthHeaders(request, "Signature"))
	if err != nil {
		return "", fmt.Errorf("POST %v: %w", minecraftAuthURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST %v: %v", minecraftAuthURL, resp.Status)
	}
	data, err = io.ReadAll(resp.Body)
	return string(data), err
}
