package subcommands

import (
	"context"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
)

type CaptureSettings struct {
	ProxySettings proxy.ProxySettings
}

type CaptureCMD struct{}

func (CaptureCMD) Name() string {
	return "capture"
}

func (CaptureCMD) Description() string {
	return locale.Loc("capture_synopsis", nil)
}

func (CaptureCMD) Settings() any {
	return new(CaptureSettings)
}

func (CaptureCMD) Run(ctx context.Context, settings any) error {
	captureSettings := settings.(*CaptureSettings)

	captureSettings.ProxySettings.Capture = true
	p, err := proxy.New(ctx, captureSettings.ProxySettings)
	if err != nil {
		return err
	}

	return p.Run(ctx, true)
}

func init() {
	commands.RegisterCommand(&CaptureCMD{})
}
