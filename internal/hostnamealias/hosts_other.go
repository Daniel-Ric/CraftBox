//go:build !windows

package hostnamealias

import (
	"context"
	"fmt"
	"os"
)

func acquirePlatform(context.Context, string) (*Lease, error) {
	return nil, fmt.Errorf("destination hostname redirection is currently supported on Windows only")
}

func systemHosts() (string, func(), error) {
	return "", nil, fmt.Errorf("destination hostname aliases are currently supported on Windows only")
}
func atomicReplace(path, tmp string) error { return os.Rename(tmp, path) }
func flushDNS()                            {}
