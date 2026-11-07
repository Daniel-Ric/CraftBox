//go:build !windows

package assetlibrary

func InstalledMinecraft() ([]Installation, error) { return []Installation{}, nil }
