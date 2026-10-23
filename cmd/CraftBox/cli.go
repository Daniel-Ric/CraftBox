//go:build !android

package main

import "github.com/Daniel-Ric/CraftBox/ui/cli"

func init() {
	uis["cli"] = &cli.CLI{}
}
