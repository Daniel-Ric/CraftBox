//go:build android

package main

import "github.com/Daniel-Ric/CraftBox/ui/gui"

func init() { uis["gui"] = &gui.GUI{} }
