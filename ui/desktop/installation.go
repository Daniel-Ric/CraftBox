package desktop

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
)

func portableInstallationID(executable string) (string, error) {
	directory := filepath.Join(filepath.Dir(executable), ".craftbox")
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(directory, "installation-id")
	for attempt := 0; attempt < 20; attempt++ {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			nonce := make([]byte, 16)
			if _, err = rand.Read(nonce); err == nil {
				_, err = file.WriteString(hex.EncodeToString(nonce))
			}
			if err == nil {
				err = file.Sync()
			}
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				os.Remove(path)
				return "", err
			}
		} else if !os.IsExist(err) {
			return "", err
		}
		reader, err := os.Open(path)
		if err != nil {
			return "", err
		}
		raw, err := io.ReadAll(io.LimitReader(reader, 65))
		reader.Close()
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(raw))
		nonce, err := hex.DecodeString(value)
		if err == nil && len(nonce) == 16 {
			return fmt.Sprintf("%x", sha256.Sum256([]byte(installationID(executable)+":"+value))), nil
		}

		if attempt < 19 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return "", fmt.Errorf("Invalid portable installation identity")
}

func (a *App) installationScope() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.installationScopeID != "" {
		return a.installationScopeID
	}
	executable, err := os.Executable()
	if err == nil {
		a.installationScopeID, err = portableInstallationID(executable)
	}
	if err != nil {

		a.installationScopeID = "transient-" + rand.Text()
		diagnostics.Record("warning", "desktop", "installation.identity.unavailable", "Using a temporary first-start identity", map[string]any{"error": err.Error()})
	}
	return a.installationScopeID
}
