package desktop

import "golang.org/x/sys/windows"

func openResultDirectory(directory string) error {
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, path, nil, nil, windows.SW_SHOWNORMAL)
}
