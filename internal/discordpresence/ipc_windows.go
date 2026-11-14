//go:build windows

package discordpresence

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func dialIPC() (stream, error) {
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		path, err := windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, err
		}
		h, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
		if err != nil {
			continue
		}
		return os.NewFile(uintptr(h), name), nil
	}
	return nil, fmt.Errorf("Waiting for Discord desktop. Open Discord and enable activity sharing")
}
