//go:build gui && !android

package main

import "github.com/Daniel-Ric/CraftBox/ui/desktop"

func init() {
	runMapWindow = desktop.RunMapWindow
	uis["gui"] = &desktop.GUI{}
}
