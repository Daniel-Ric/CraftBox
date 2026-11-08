package desktop

import (
	"context"

	"golang.org/x/sys/windows"
)

func openMinecraftURL(_ context.Context, value string) error {
	uri, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	// The browser launcher uses SW_SHOWNORMAL, which restores stale normal
	// coordinates even for a maximized Minecraft window. Let Windows maximize
	// against the monitor's work area, including its taskbar and DPI settings.
	return windows.ShellExecute(0, nil, uri, nil, nil, windows.SW_SHOWMAXIMIZED)
}
