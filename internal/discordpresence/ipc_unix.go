//go:build !windows

package discordpresence

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func dialIPC() (stream, error) {
	for _, dir := range []string{os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("TMPDIR"), os.Getenv("TMP"), os.Getenv("TEMP"), "/tmp"} {
		if dir == "" {
			continue
		}
		for _, sub := range []string{"", "app/com.discordapp.Discord", "snap.discord"} {
			for i := 0; i < 10; i++ {
				conn, err := net.DialTimeout("unix", filepath.Join(dir, sub, fmt.Sprintf("discord-ipc-%d", i)), 100*time.Millisecond)
				if err == nil {
					return conn, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Waiting for Discord desktop. Open Discord and enable activity sharing")
}
