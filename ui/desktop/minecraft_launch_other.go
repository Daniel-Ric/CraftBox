//go:build !windows

package desktop

import (
	"context"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func openMinecraftURL(ctx context.Context, value string) error {
	wruntime.BrowserOpenURL(ctx, value)
	return nil
}
