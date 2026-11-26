//go:build !windows

package desktop

import (
	"context"
	"errors"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func discoverFlarial() (flarialInstallation, bool) { return flarialInstallation{}, false }

func checkFlarialProcesses() error { return nil }

func flarialMinecraftReady(_ bool) (bool, error) { return false, nil }

func launchFlarial(_ flarialInstallation, _ string) error {
	return errors.New("Flarial capture launch is only available on Windows")
}

func openMinecraftURL(ctx context.Context, value string) error {
	wruntime.BrowserOpenURL(ctx, value)
	return nil
}
